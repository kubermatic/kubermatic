//go:build e2e

/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package opa

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	constrainttemplatev1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	credentials jig.AWSCredentials

	logOptions            = utils.DefaultLogOptions
	ctKind                = "RequiredLabels"
	defaultConstraintName = "testconstraint"

	//go:embed constraint_template.yaml
	testCT string
)

func init() {
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

func TestOPAIntegration(t *testing.T) {
	ctx := context.Background()
	rawLogger := log.NewFromOptions(logOptions)
	logger := rawLogger.Sugar()

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))

	if err := credentials.Parse(); err != nil {
		t.Fatalf("Failed to get credentials: %v", err)
	}

	seedClient, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("failed to get client for seed cluster: %v", err)
	}

	utilruntime.Must(constrainttemplatev1.AddToScheme(seedClient.Scheme()))

	// create test environment
	testJig := jig.NewAWSCluster(seedClient, logger, credentials, 1, nil)
	testJig.ClusterJig.WithTestName("opa")

	_, cluster, err := testJig.Setup(ctx, jig.WaitForReadyPods)
	defer testJig.Cleanup(ctx, t, true)
	if err != nil {
		t.Fatalf("failed to setup test environment: %v", err)
	}

	// enable OPA
	logger.Info("Enabling OPA...")
	if err := setOPAIntegration(ctx, seedClient, cluster, true); err != nil {
		t.Fatalf("Failed to enable OPA integration: %v", err)
	}

	logger.Info("Waiting for cluster to healthy after enabling OPA...")
	if err := testJig.WaitForHealthyControlPlane(ctx, 2*time.Minute); err != nil {
		t.Fatalf("Cluster did not get healthy: %v", err)
	}

	// Create CT
	logger.Info("Creating Constraint Template (CT)...")
	ct, err := createTestConstraintTemplate(ctx, seedClient)
	if err != nil {
		t.Fatalf("error creating Constraint Template: %v", err)
	}
	logger = logger.With("template", ct.Name)
	logger.Info("Created Constraint Template")

	logger.Info("Creating client for user cluster...")
	userClient, err := testJig.ClusterClient(ctx)
	if err != nil {
		t.Fatalf("error creating user cluster client: %v", err)
	}

	utilruntime.Must(constrainttemplatev1.AddToScheme(userClient.Scheme()))

	logger.Info("Waiting for CT to be synced...")
	if err := waitForCTSync(ctx, userClient, logger, ct.Name, false); err != nil {
		t.Fatal(err)
	}
	logger.Info("Constraint template synced to user cluster.")

	// Create Default Constraint
	logger.Info("Creating Default Constraint...")
	defaultConstraint, err := createConstraint(ctx, seedClient, defaultConstraintName, ctKind)
	if err != nil {
		t.Fatalf("error creating Default Constraint: %v", err)
	}

	logger.Info("Waiting for Default Constraint sync...")
	if err := waitForConstraintSync(ctx, seedClient, logger, defaultConstraint.Name, defaultConstraint.Namespace, false); err != nil {
		t.Fatal(err)
	}
	logger.Info("Synced Default Constraint to KKP namespace.")

	if err := waitForConstraintSync(ctx, seedClient, logger, defaultConstraint.Name, cluster.Status.NamespaceName, false); err != nil {
		t.Fatal(err)
	}
	logger.Info("Synced Default Constraint to user cluster namespace.")

	// Test if constraint works
	logger.Info("Testing if Constraint works by creating policy-breaking ConfigMap...")
	if err := testConstraintForConfigMap(ctx, userClient, logger); err != nil {
		t.Fatal(err)
	}

	logger.Info("Testing if Constraint lets policy-aligned ConfigMap through...")
	cm := genTestConfigMap()
	cm.Labels = map[string]string{"gatekeeper": "true"}
	if err := userClient.Create(ctx, cm); err != nil {
		t.Fatalf("error creating policy-aligned ConfigMap on user cluster: %v", err)
	}

	// Delete constraint
	logger.Info("Deleting Constraint...")
	if err := seedClient.Delete(ctx, defaultConstraint); err != nil {
		t.Fatalf("error deleting Constraint: %v", err)
	}

	logger.Info("Waiting for Constraint sync delete...")
	if err := waitForConstraintSync(ctx, seedClient, logger, defaultConstraint.Name, defaultConstraint.Namespace, true); err != nil {
		t.Fatal(err)
	}
	logger.Info("Synced Default Constraint to KKP namespace.")

	if err := waitForConstraintSync(ctx, seedClient, logger, defaultConstraint.Name, cluster.Status.NamespaceName, true); err != nil {
		t.Fatal(err)
	}
	logger.Info("Synced Default Constraint to user cluster namespace.")

	// Check that constraint does not work
	logger.Info("Testing if policy breaking ConfigMap can now be created...")
	cmBreaking := genTestConfigMap()
	if err := userClient.Create(ctx, cmBreaking); err != nil {
		t.Fatalf("error creating policy-breaking configmap on user cluster after deleting constraint: %v", err)
	}

	// Delete CT
	logger.Info("Deleting Constraint Template...")
	if err := seedClient.Delete(ctx, ct); err != nil {
		t.Fatalf("error deleting Constraint Template: %v", err)
	}

	// Check that CT is removed
	logger.Info("Waiting for Constraint Template delete sync...")
	if err := waitForCTSync(ctx, userClient, logger, ct.Name, true); err != nil {
		t.Fatal(err)
	}

	// Disable OPA Integration
	logger.Info("Disabling OPA...")
	if err := setOPAIntegration(ctx, seedClient, cluster, false); err != nil {
		t.Fatalf("failed to disable OPA integration: %v", err)
	}

	// Check that cluster is healthy
	logger.Info("Waiting for cluster to healthy after disabling OPA...")
	if err := testJig.WaitForHealthyControlPlane(ctx, 2*time.Minute); err != nil {
		t.Fatalf("Cluster did not get healthy: %v", err)
	}
}

func setOPAIntegration(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, enabled bool) error {
	oldCluster := cluster.DeepCopy()
	cluster.Spec.OPAIntegration = &kubermaticv1.OPAIntegrationSettings{
		Enabled: enabled,
	}

	return client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}

func genTestConfigMap() *corev1.ConfigMap {
	cm := &corev1.ConfigMap{}
	cm.Namespace = corev1.NamespaceDefault
	cm.Name = fmt.Sprintf("test-cm-%d", rand.Int())
	return cm
}

func testConstraintForConfigMap(ctx context.Context, userClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	return wait.PollLog(ctx, log, 3*time.Second, 2*time.Minute, func(ctx context.Context) (error, error) {
		cm := genTestConfigMap()
		err := userClient.Create(ctx, cm)
		if err == nil {
			return errors.New("successfully created ConfigMap, but should have been prevented"), nil
		}

		if !strings.Contains(err.Error(), "you must provide labels") {
			return fmt.Errorf("expected error regarding labels, but got: %w", err), nil
		}

		return nil, nil
	})
}

func waitForCTSync(ctx context.Context, userClient ctrlruntimeclient.Client, log *zap.SugaredLogger, ctName string, deleted bool) error {
	return wait.PollLog(ctx, log, 3*time.Second, 1*time.Minute, func(ctx context.Context) (error, error) {
		gatekeeperCT := &constrainttemplatev1.ConstraintTemplate{}
		err := userClient.Get(ctx, types.NamespacedName{Name: ctName}, gatekeeperCT)

		if deleted {
			if apierrors.IsNotFound(err) {
				return nil, nil
			}

			return fmt.Errorf("expected NotFound error, but got: %w", err), nil
		}

		if err != nil {
			return fmt.Errorf("failed to get Constraint Template: %w", err), nil
		}

		return nil, nil
	})
}

func waitForConstraintSync(ctx context.Context, client ctrlruntimeclient.Client, log *zap.SugaredLogger, cName, namespace string, deleted bool) error {
	return wait.PollLog(ctx, log, 3*time.Second, 2*time.Minute, func(ctx context.Context) (error, error) {
		constraint := &kubermaticv1.Constraint{}
		err := client.Get(ctx, types.NamespacedName{Name: cName, Namespace: namespace}, constraint)

		if deleted {
			if apierrors.IsNotFound(err) {
				return nil, nil
			}

			return fmt.Errorf("expected NotFound error, but got: %w", err), nil
		}

		if err != nil {
			return fmt.Errorf("failed to get Constraint: %w", err), nil
		}

		return nil, nil
	})
}

func createTestConstraintTemplate(ctx context.Context, client ctrlruntimeclient.Client) (*kubermaticv1.ConstraintTemplate, error) {
	var ct *kubermaticv1.ConstraintTemplate
	err := yaml.Unmarshal([]byte(testCT), &ct)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling Constraint Template: %w", err)
	}

	return ct, client.Create(ctx, ct)
}

func createConstraint(ctx context.Context, client ctrlruntimeclient.Client, name, ctKind string) (*kubermaticv1.Constraint, error) {
	constraint := &kubermaticv1.Constraint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: jig.KubermaticNamespace(),
		},
		Spec: kubermaticv1.ConstraintSpec{
			ConstraintType: ctKind,
			Match: kubermaticv1.Match{
				Kinds: []kubermaticv1.Kind{{
					APIGroups: []string{""},
					Kinds:     []string{"ConfigMap"},
				}},
			},
			Parameters: kubermaticv1.Parameters{
				"labels": json.RawMessage(`["gatekeeper"]`),
			},
		},
	}

	return constraint, client.Create(ctx, constraint)
}
