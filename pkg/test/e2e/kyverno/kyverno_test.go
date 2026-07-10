//go:build e2e && ee

/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package kyverno

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kyvernocontroller "k8c.io/kubermatic/v2/pkg/ee/kyverno"
	commonseedresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/common"
	userclusterresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/user-cluster"
	policybindingcontroller "k8c.io/kubermatic/v2/pkg/ee/policy-binding-controller"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	"k8c.io/kubermatic/v2/pkg/validation"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	waitInterval = 3 * time.Second
	waitTimeout  = 5 * time.Minute

	requiredLabelKey   = "kyverno-e2e"
	requiredLabelValue = "enabled"
	policyDenyMessage  = "the kyverno-e2e=enabled label is required"
)

var (
	credentials jig.AWSCredentials
	logOptions  = utils.DefaultLogOptions
)

func init() {
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

func TestPolicyTemplateFixtures(t *testing.T) {
	testCases := []struct {
		name       string
		namespaced bool
		enforced   bool
	}{
		{name: "cluster-wide"},
		{name: "cluster-wide-enforced", enforced: true},
		{name: "namespaced", namespaced: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			template, err := newPolicyTemplate("kyverno-e2e-fixture", "kyverno-e2e-fixture", testCase.namespaced, testCase.enforced)
			if err != nil {
				t.Fatalf("failed to build PolicyTemplate fixture: %v", err)
			}
			if validationErrors := validation.ValidatePolicyTemplate(template); len(validationErrors) > 0 {
				t.Fatalf("PolicyTemplate fixture is invalid: %v", validationErrors.ToAggregate())
			}
		})
	}
}

//nolint:gocyclo // The phases form one ordered lifecycle and must stop at the first failed prerequisite.
func TestKyvernoIntegration(t *testing.T) {
	ctx := context.Background()
	rawLogger := log.NewFromOptions(logOptions)
	logger := rawLogger.Sugar()
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))

	if err := credentials.Parse(); err != nil {
		t.Fatalf("failed to get credentials: %v", err)
	}

	seedClient, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("failed to get client for seed cluster: %v", err)
	}

	testJig := jig.NewAWSCluster(seedClient, logger, credentials, 1, nil)
	testJig.ClusterJig.WithTestName("kyverno")
	testJig.ClusterJig.WithPatch(func(spec *kubermaticv1.ClusterSpec) *kubermaticv1.ClusterSpec {
		spec.Kyverno = &kubermaticv1.KyvernoSettings{Enabled: true}
		return spec
	})

	_, cluster, err := testJig.Setup(ctx, jig.WaitForReadyPods)
	defer testJig.Cleanup(ctx, t, true)
	if err != nil {
		t.Fatalf("failed to setup test environment: %v", err)
	}

	logger.Info("Waiting for Kyverno controllers to become healthy...")
	if err := testJig.WaitForKyvernoHealthy(ctx, 10*time.Minute); err != nil {
		t.Fatalf("Kyverno controllers did not become healthy: %v", err)
	}
	if err := waitForSeedKyvernoControllersReady(ctx, seedClient, logger, cluster.Status.NamespaceName); err != nil {
		t.Fatalf("failed to verify seed-side Kyverno controllers: %v", err)
	}
	if err := waitForUserClusterControllerKyvernoFlag(ctx, seedClient, logger, cluster.Status.NamespaceName, true); err != nil {
		t.Fatalf("user-cluster-controller-manager was not started with Kyverno enabled: %v", err)
	}
	if err := waitForClusterKyvernoFinalizer(ctx, seedClient, logger, cluster.Name, true); err != nil {
		t.Fatalf("Kyverno cleanup finalizer was not added to the cluster: %v", err)
	}

	logger.Info("Creating client for user cluster...")
	userClient, err := testJig.ClusterClient(ctx)
	if err != nil {
		t.Fatalf("failed to create user cluster client: %v", err)
	}
	if err := kyvernov1.Install(userClient.Scheme()); err != nil {
		t.Fatalf("failed to add Kyverno APIs to user cluster client scheme: %v", err)
	}

	logger.Info("Waiting for Kyverno CRDs and user-cluster resources...")
	if err := waitForKyvernoUserClusterResources(ctx, userClient, logger, cluster.Status.NamespaceName); err != nil {
		t.Fatalf("Kyverno user-cluster resources did not become ready: %v", err)
	}

	suffix := rand.String(6)
	clusterPolicyNamespace := "kyverno-e2e-cluster-" + suffix
	namespacedPolicyNamespace := "kyverno-e2e-namespaced-" + suffix
	if err := userClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: clusterPolicyNamespace}}); err != nil {
		t.Fatalf("failed to create ClusterPolicy test namespace: %v", err)
	}

	clusterTemplateName := "kyverno-e2e-cluster-" + suffix
	namespacedTemplateName := "kyverno-e2e-namespaced-" + suffix

	logger.Info("Creating cluster-wide and namespaced PolicyTemplates and PolicyBindings...")
	clusterTemplate, err := newPolicyTemplate(clusterTemplateName, clusterPolicyNamespace, false, true)
	if err != nil {
		t.Fatalf("failed to build enforced cluster-wide PolicyTemplate: %v", err)
	}
	if err := seedClient.Create(ctx, clusterTemplate); err != nil {
		t.Fatalf("failed to create enforced cluster-wide PolicyTemplate: %v", err)
	}
	clusterBinding := newPolicyBinding(cluster.Status.NamespaceName, clusterTemplateName, "", false)
	namespacedTemplate, namespacedBinding, err := createPolicyPair(ctx, seedClient, cluster.Status.NamespaceName, namespacedTemplateName, namespacedPolicyNamespace, true)
	if err != nil {
		t.Fatalf("failed to create namespaced policy pair: %v", err)
	}

	if err := waitForActivePolicyBinding(ctx, seedClient, logger, ctrlruntimeclient.ObjectKeyFromObject(clusterBinding), true); err != nil {
		t.Fatalf("cluster-wide PolicyBinding did not become active: %v", err)
	}
	if err := waitForActivePolicyBinding(ctx, seedClient, logger, ctrlruntimeclient.ObjectKeyFromObject(namespacedBinding), false); err != nil {
		t.Fatalf("namespaced PolicyBinding did not become active: %v", err)
	}
	if err := waitForClusterPolicyReady(ctx, userClient, logger, clusterTemplateName); err != nil {
		t.Fatalf("generated ClusterPolicy did not become ready: %v", err)
	}
	if err := waitForPolicyReady(ctx, userClient, logger, namespacedPolicyNamespace, namespacedTemplateName); err != nil {
		t.Fatalf("generated namespaced Policy did not become ready: %v", err)
	}

	logger.Info("Verifying ClusterPolicy admission enforcement...")
	if err := verifyPolicyAdmission(ctx, userClient, logger, clusterPolicyNamespace); err != nil {
		t.Fatalf("ClusterPolicy admission verification failed: %v", err)
	}
	logger.Info("Verifying namespaced Policy admission enforcement...")
	if err := verifyPolicyAdmission(ctx, userClient, logger, namespacedPolicyNamespace); err != nil {
		t.Fatalf("namespaced Policy admission verification failed: %v", err)
	}

	logger.Info("Deleting the namespaced PolicyBinding...")
	if err := seedClient.Delete(ctx, namespacedBinding); err != nil {
		t.Fatalf("failed to delete namespaced PolicyBinding: %v", err)
	}
	if err := waitForObjectDeleted(ctx, seedClient, logger, namespacedBinding); err != nil {
		t.Fatalf("namespaced PolicyBinding was not deleted: %v", err)
	}
	if err := waitForObjectDeleted(ctx, userClient, logger, &kyvernov1.Policy{ObjectMeta: metav1.ObjectMeta{Name: namespacedTemplateName, Namespace: namespacedPolicyNamespace}}); err != nil {
		t.Fatalf("generated namespaced Policy was not deleted with its binding: %v", err)
	}

	logger.Info("Deleting the cluster-wide PolicyTemplate...")
	if err := waitForPolicyTemplateFinalizers(ctx, seedClient, logger, clusterTemplate.Name); err != nil {
		t.Fatalf("PolicyTemplate cleanup finalizers were not established: %v", err)
	}
	if err := seedClient.Delete(ctx, clusterTemplate); err != nil {
		t.Fatalf("failed to delete cluster-wide PolicyTemplate: %v", err)
	}
	if err := waitForObjectDeleted(ctx, seedClient, logger, clusterBinding); err != nil {
		t.Fatalf("dependent PolicyBinding was not deleted with its PolicyTemplate: %v", err)
	}
	if err := waitForObjectDeleted(ctx, userClient, logger, &kyvernov1.ClusterPolicy{ObjectMeta: metav1.ObjectMeta{Name: clusterTemplateName}}); err != nil {
		t.Fatalf("generated ClusterPolicy was not deleted with its PolicyTemplate: %v", err)
	}
	if err := waitForObjectDeleted(ctx, seedClient, logger, clusterTemplate); err != nil {
		t.Fatalf("PolicyTemplate did not finish deletion: %v", err)
	}

	// Keep active bindings across disable/re-enable to verify both the seed-side
	// cleanup fallback and the restarted user-cluster controller.
	disableClusterTemplateName := "kyverno-e2e-disable-cluster-" + suffix
	disableNamespacedTemplateName := "kyverno-e2e-disable-namespaced-" + suffix
	disableClusterTemplate, disableClusterBinding, err := createPolicyPair(ctx, seedClient, cluster.Status.NamespaceName, disableClusterTemplateName, clusterPolicyNamespace, false)
	if err != nil {
		t.Fatalf("failed to create cluster-wide disable test policy pair: %v", err)
	}
	disableNamespacedTemplate, disableNamespacedBinding, err := createPolicyPair(ctx, seedClient, cluster.Status.NamespaceName, disableNamespacedTemplateName, namespacedPolicyNamespace, true)
	if err != nil {
		t.Fatalf("failed to create namespaced disable test policy pair: %v", err)
	}

	if err := waitForActivePolicyBinding(ctx, seedClient, logger, ctrlruntimeclient.ObjectKeyFromObject(disableClusterBinding), false); err != nil {
		t.Fatalf("cluster-wide disable test binding did not become active: %v", err)
	}
	if err := waitForActivePolicyBinding(ctx, seedClient, logger, ctrlruntimeclient.ObjectKeyFromObject(disableNamespacedBinding), false); err != nil {
		t.Fatalf("namespaced disable test binding did not become active: %v", err)
	}
	if err := waitForClusterPolicyReady(ctx, userClient, logger, disableClusterTemplateName); err != nil {
		t.Fatalf("disable test ClusterPolicy did not become ready: %v", err)
	}
	if err := waitForPolicyReady(ctx, userClient, logger, namespacedPolicyNamespace, disableNamespacedTemplateName); err != nil {
		t.Fatalf("disable test Policy did not become ready: %v", err)
	}

	logger.Info("Disabling Kyverno and verifying cleanup...")
	if err := setKyvernoEnabled(ctx, seedClient, cluster.Name, false); err != nil {
		t.Fatalf("failed to disable Kyverno: %v", err)
	}
	if err := waitForInactivePolicyBinding(ctx, seedClient, logger, ctrlruntimeclient.ObjectKeyFromObject(disableClusterBinding)); err != nil {
		t.Fatalf("cluster-wide PolicyBinding did not become inactive after disabling Kyverno: %v", err)
	}
	if err := waitForInactivePolicyBinding(ctx, seedClient, logger, ctrlruntimeclient.ObjectKeyFromObject(disableNamespacedBinding)); err != nil {
		t.Fatalf("namespaced PolicyBinding did not become inactive after disabling Kyverno: %v", err)
	}
	if err := waitForObjectDeleted(ctx, userClient, logger, &kyvernov1.ClusterPolicy{ObjectMeta: metav1.ObjectMeta{Name: disableClusterTemplateName}}); err != nil {
		t.Fatalf("ClusterPolicy remained after disabling Kyverno: %v", err)
	}
	if err := waitForObjectDeleted(ctx, userClient, logger, &kyvernov1.Policy{ObjectMeta: metav1.ObjectMeta{Name: disableNamespacedTemplateName, Namespace: namespacedPolicyNamespace}}); err != nil {
		t.Fatalf("namespaced Policy remained after disabling Kyverno: %v", err)
	}
	if err := waitForKyvernoCRDsRemoved(ctx, userClient, logger); err != nil {
		t.Fatalf("Kyverno CRDs remained after disabling Kyverno: %v", err)
	}
	if err := waitForObjectDeleted(ctx, userClient, logger, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: cluster.Status.NamespaceName}}); err != nil {
		t.Fatalf("Kyverno user-cluster namespace remained after disabling Kyverno: %v", err)
	}
	if err := waitForSeedKyvernoControllersRemoved(ctx, seedClient, logger, cluster.Status.NamespaceName); err != nil {
		t.Fatalf("seed-side Kyverno controllers remained after disabling Kyverno: %v", err)
	}
	if err := waitForUserClusterControllerKyvernoFlag(ctx, seedClient, logger, cluster.Status.NamespaceName, false); err != nil {
		t.Fatalf("user-cluster-controller-manager still had Kyverno enabled: %v", err)
	}
	if err := waitForClusterKyvernoFinalizer(ctx, seedClient, logger, cluster.Name, false); err != nil {
		t.Fatalf("Kyverno cleanup finalizer remained after disable cleanup: %v", err)
	}

	logger.Info("Re-enabling Kyverno and verifying reconciliation of existing PolicyBindings...")
	if err := setKyvernoEnabled(ctx, seedClient, cluster.Name, true); err != nil {
		t.Fatalf("failed to re-enable Kyverno: %v", err)
	}
	if err := testJig.WaitForKyvernoHealthy(ctx, 10*time.Minute); err != nil {
		t.Fatalf("Kyverno controllers did not recover after re-enabling: %v", err)
	}
	if err := waitForSeedKyvernoControllersReady(ctx, seedClient, logger, cluster.Status.NamespaceName); err != nil {
		t.Fatalf("seed-side Kyverno controllers were not restored: %v", err)
	}
	if err := waitForUserClusterControllerKyvernoFlag(ctx, seedClient, logger, cluster.Status.NamespaceName, true); err != nil {
		t.Fatalf("user-cluster-controller-manager was not restarted with Kyverno enabled: %v", err)
	}
	if err := waitForKyvernoUserClusterResources(ctx, userClient, logger, cluster.Status.NamespaceName); err != nil {
		t.Fatalf("Kyverno user-cluster resources were not restored: %v", err)
	}
	if err := waitForClusterKyvernoFinalizer(ctx, seedClient, logger, cluster.Name, true); err != nil {
		t.Fatalf("Kyverno cleanup finalizer was not restored: %v", err)
	}
	if err := waitForActivePolicyBinding(ctx, seedClient, logger, ctrlruntimeclient.ObjectKeyFromObject(disableClusterBinding), false); err != nil {
		t.Fatalf("cluster-wide PolicyBinding did not reactivate: %v", err)
	}
	if err := waitForActivePolicyBinding(ctx, seedClient, logger, ctrlruntimeclient.ObjectKeyFromObject(disableNamespacedBinding), false); err != nil {
		t.Fatalf("namespaced PolicyBinding did not reactivate: %v", err)
	}
	if err := waitForClusterPolicyReady(ctx, userClient, logger, disableClusterTemplateName); err != nil {
		t.Fatalf("ClusterPolicy was not restored after re-enabling Kyverno: %v", err)
	}
	if err := waitForPolicyReady(ctx, userClient, logger, namespacedPolicyNamespace, disableNamespacedTemplateName); err != nil {
		t.Fatalf("namespaced Policy was not restored after re-enabling Kyverno: %v", err)
	}

	logger.Info("Deleting the user cluster while Kyverno policies are active...")
	clusterNamespace := cluster.Status.NamespaceName
	if err := testJig.ClusterJig.Delete(ctx, true); err != nil {
		t.Fatalf("Kyverno resources blocked user cluster deletion: %v", err)
	}
	if err := waitForObjectDeleted(ctx, seedClient, logger, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: clusterNamespace}}); err != nil {
		t.Fatalf("user cluster namespace remained after cluster deletion: %v", err)
	}

	for _, template := range []*kubermaticv1.PolicyTemplate{namespacedTemplate, disableClusterTemplate, disableNamespacedTemplate} {
		if err := seedClient.Delete(ctx, template); err != nil && !apierrors.IsNotFound(err) {
			t.Errorf("failed to delete PolicyTemplate %s during test cleanup: %v", template.Name, err)
			continue
		}
		if err := waitForObjectDeleted(ctx, seedClient, logger, template); err != nil {
			t.Errorf("PolicyTemplate %s remained after test cleanup: %v", template.Name, err)
		}
	}
}
func createPolicyPair(ctx context.Context, client ctrlruntimeclient.Client, bindingNamespace, name, policyNamespace string, namespaced bool) (*kubermaticv1.PolicyTemplate, *kubermaticv1.PolicyBinding, error) {
	template, err := newPolicyTemplate(name, policyNamespace, namespaced, false)
	if err != nil {
		return nil, nil, err
	}
	if err := client.Create(ctx, template); err != nil {
		return nil, nil, fmt.Errorf("failed to create PolicyTemplate %s: %w", name, err)
	}

	binding := newPolicyBinding(bindingNamespace, name, policyNamespace, namespaced)
	if err := client.Create(ctx, binding); err != nil {
		return nil, nil, fmt.Errorf("failed to create PolicyBinding %s/%s: %w", bindingNamespace, name, err)
	}

	return template, binding, nil
}

func newPolicyBinding(bindingNamespace, name, policyNamespace string, namespaced bool) *kubermaticv1.PolicyBinding {
	binding := &kubermaticv1.PolicyBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: bindingNamespace,
		},
		Spec: kubermaticv1.PolicyBindingSpec{
			PolicyTemplateRef: corev1.ObjectReference{Name: name},
		},
	}
	if namespaced {
		binding.Spec.KyvernoPolicyNamespace = &kubermaticv1.KyvernoPolicyNamespace{Name: policyNamespace}
	}
	return binding
}

func newPolicyTemplate(name, policyNamespace string, namespaced, enforced bool) (*kubermaticv1.PolicyTemplate, error) {
	pattern, err := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"labels": map[string]string{requiredLabelKey: requiredLabelValue},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal validation pattern: %w", err)
	}

	resourceDescription := kyvernov1.ResourceDescription{Kinds: []string{"ConfigMap"}}
	if !namespaced {
		resourceDescription.Namespaces = []string{policyNamespace}
	}

	failureAction := kyvernov1.Enforce
	background := false
	policySpec, err := json.Marshal(kyvernov1.Spec{
		Background: &background,
		Rules: []kyvernov1.Rule{{
			Name: "require-e2e-label",
			MatchResources: kyvernov1.MatchResources{
				Any: kyvernov1.ResourceFilters{{ResourceDescription: resourceDescription}},
			},
			Validation: &kyvernov1.Validation{
				FailureAction: &failureAction,
				Message:       policyDenyMessage,
				RawPattern:    &apiextensionsv1.JSON{Raw: pattern},
			},
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Kyverno policy spec: %w", err)
	}

	return &kubermaticv1.PolicyTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: kubermaticv1.PolicyTemplateSpec{
			Title:            "Kyverno e2e required label",
			Description:      "Requires a label on ConfigMaps created by the Kyverno integration e2e test.",
			Visibility:       kubermaticv1.PolicyTemplateVisibilityGlobal,
			Enforced:         enforced,
			NamespacedPolicy: namespaced,
			PolicySpec:       runtime.RawExtension{Raw: policySpec},
		},
	}, nil
}

func setKyvernoEnabled(ctx context.Context, client ctrlruntimeclient.Client, clusterName string, enabled bool) error {
	cluster := &kubermaticv1.Cluster{}
	if err := client.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	oldCluster := cluster.DeepCopy()
	cluster.Spec.Kyverno = &kubermaticv1.KyvernoSettings{Enabled: enabled}
	if err := client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
		return fmt.Errorf("failed to patch Kyverno enabled=%t: %w", enabled, err)
	}

	return nil
}

func waitForSeedKyvernoControllersReady(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, namespace string) error {
	deployments := map[string]int32{
		commonseedresources.KyvernoAdmissionControllerDeploymentName:  commonseedresources.KyvernoAdmissionControllerReplicas,
		commonseedresources.KyvernoBackgroundControllerDeploymentName: commonseedresources.KyvernoBackgroundControllerReplicas,
		commonseedresources.KyvernoCleanupControllerDeploymentName:    commonseedresources.KyvernoCleanupControllerReplicas,
		commonseedresources.KyvernoReportsControllerDeploymentName:    commonseedresources.KyvernoReportsControllerReplicas,
	}

	return wait.PollLog(ctx, logger, waitInterval, 10*time.Minute, func(ctx context.Context) (error, error) {
		for name, expectedReady := range deployments {
			deployment := &appsv1.Deployment{}
			if err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, deployment); err != nil {
				return fmt.Errorf("failed to get Deployment %s/%s: %w", namespace, name, err), nil
			}
			if deployment.Status.ObservedGeneration != deployment.Generation || deployment.Status.ReadyReplicas < expectedReady || deployment.Status.UpdatedReplicas < expectedReady {
				return fmt.Errorf("Deployment %s/%s is not fully rolled out: generation=%d observedGeneration=%d ready=%d updated=%d expected=%d", namespace, name, deployment.Generation, deployment.Status.ObservedGeneration, deployment.Status.ReadyReplicas, deployment.Status.UpdatedReplicas, expectedReady), nil
			}
		}
		return nil, nil
	})
}

func waitForSeedKyvernoControllersRemoved(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, namespace string) error {
	names := []string{
		commonseedresources.KyvernoAdmissionControllerDeploymentName,
		commonseedresources.KyvernoBackgroundControllerDeploymentName,
		commonseedresources.KyvernoCleanupControllerDeploymentName,
		commonseedresources.KyvernoReportsControllerDeploymentName,
	}

	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		for _, name := range names {
			err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &appsv1.Deployment{})
			if err == nil {
				return fmt.Errorf("Deployment %s/%s still exists", namespace, name), nil
			}
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to get Deployment %s/%s: %w", namespace, name, err), nil
			}
		}
		return nil, nil
	})
}

func waitForUserClusterControllerKyvernoFlag(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, namespace string, enabled bool) error {
	key := types.NamespacedName{Name: resources.UserClusterControllerDeploymentName, Namespace: namespace}
	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		deployment := &appsv1.Deployment{}
		if err := client.Get(ctx, key, deployment); err != nil {
			return fmt.Errorf("failed to get user-cluster-controller deployment: %w", err), nil
		}
		expectedReplicas := int32(1)
		if deployment.Spec.Replicas != nil {
			expectedReplicas = *deployment.Spec.Replicas
		}
		if deployment.Status.ObservedGeneration != deployment.Generation || deployment.Status.ReadyReplicas != expectedReplicas || deployment.Status.UpdatedReplicas != expectedReplicas {
			return errors.New("user-cluster-controller deployment has not finished rolling out"), nil
		}

		for _, container := range deployment.Spec.Template.Spec.Containers {
			if container.Name != resources.UserClusterControllerContainerName {
				continue
			}
			hasFlag := slices.Contains(container.Args, "-kyverno-enabled")
			if hasFlag != enabled {
				return fmt.Errorf("user-cluster-controller Kyverno flag is %t, expected %t", hasFlag, enabled), nil
			}
			return nil, nil
		}

		return errors.New("user-cluster-controller container was not found"), nil
	})
}

func waitForKyvernoUserClusterResources(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, namespace string) error {
	if err := waitForKyvernoCRDsEstablished(ctx, client, logger); err != nil {
		return err
	}

	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		if err := client.Get(ctx, types.NamespacedName{Name: namespace}, &corev1.Namespace{}); err != nil {
			return fmt.Errorf("Kyverno namespace is not ready: %w", err), nil
		}
		for _, name := range []string{commonseedresources.KyvernoConfigMapName, commonseedresources.KyvernoMetricsConfigMapName} {
			if err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &corev1.ConfigMap{}); err != nil {
				return fmt.Errorf("Kyverno ConfigMap %s/%s is not ready: %w", namespace, name, err), nil
			}
		}
		return nil, nil
	})
}

func waitForKyvernoCRDsEstablished(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger) error {
	crds, err := userclusterresources.KyvernoCRDs()
	if err != nil {
		return fmt.Errorf("failed to load expected Kyverno CRDs: %w", err)
	}

	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		for _, expected := range crds {
			crd := &apiextensionsv1.CustomResourceDefinition{}
			if err := client.Get(ctx, types.NamespacedName{Name: expected.Name}, crd); err != nil {
				return fmt.Errorf("Kyverno CRD %s is not available: %w", expected.Name, err), nil
			}
			established := false
			for _, condition := range crd.Status.Conditions {
				if condition.Type == apiextensionsv1.Established && condition.Status == apiextensionsv1.ConditionTrue {
					established = true
					break
				}
			}
			if !established {
				return fmt.Errorf("Kyverno CRD %s is not established", expected.Name), nil
			}
		}
		return nil, nil
	})
}

func waitForKyvernoCRDsRemoved(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger) error {
	crds, err := userclusterresources.KyvernoCRDs()
	if err != nil {
		return fmt.Errorf("failed to load expected Kyverno CRDs: %w", err)
	}

	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		for _, expected := range crds {
			err := client.Get(ctx, types.NamespacedName{Name: expected.Name}, &apiextensionsv1.CustomResourceDefinition{})
			if err == nil {
				return fmt.Errorf("Kyverno CRD %s still exists", expected.Name), nil
			}
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to get Kyverno CRD %s: %w", expected.Name, err), nil
			}
		}
		return nil, nil
	})
}

func waitForActivePolicyBinding(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, key types.NamespacedName, templateEnforced bool) error {
	return waitForPolicyBindingState(ctx, client, logger, key, true, metav1.ConditionTrue, kubermaticv1.PolicyBindingReasonReady, true, templateEnforced)
}

func waitForInactivePolicyBinding(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, key types.NamespacedName) error {
	return waitForPolicyBindingState(ctx, client, logger, key, false, metav1.ConditionFalse, kubermaticv1.PolicyBindingReasonKyvernoDisabled, false, false)
}

func waitForPolicyBindingState(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, key types.NamespacedName, active bool, conditionStatus metav1.ConditionStatus, reason string, expectFinalizer, templateEnforced bool) error {
	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		binding := &kubermaticv1.PolicyBinding{}
		if err := client.Get(ctx, key, binding); err != nil {
			return fmt.Errorf("failed to get PolicyBinding %s: %w", key, err), nil
		}

		if binding.Status.Active == nil || *binding.Status.Active != active {
			return fmt.Errorf("PolicyBinding %s active=%v, expected %t", key, binding.Status.Active, active), nil
		}
		if binding.Status.ObservedGeneration != binding.Generation {
			return fmt.Errorf("PolicyBinding %s observed generation %d, expected %d", key, binding.Status.ObservedGeneration, binding.Generation), nil
		}

		for _, conditionType := range []kubermaticv1.PolicyBindingConditionType{
			kubermaticv1.PolicyBindingConditionKyvernoPolicyApplied,
			kubermaticv1.PolicyBindingConditionReady,
		} {
			condition := meta.FindStatusCondition(binding.Status.Conditions, string(conditionType))
			if condition == nil || condition.Status != conditionStatus || condition.Reason != reason {
				return fmt.Errorf("PolicyBinding %s condition %s is %#v, expected status=%s reason=%s", key, conditionType, condition, conditionStatus, reason), nil
			}
		}
		if active {
			templateCondition := meta.FindStatusCondition(binding.Status.Conditions, string(kubermaticv1.PolicyBindingConditionTemplateValid))
			if templateCondition == nil || templateCondition.Status != metav1.ConditionTrue || templateCondition.Reason != kubermaticv1.PolicyBindingReasonPolicyApplied {
				return fmt.Errorf("PolicyBinding %s template condition is %#v, expected status=True reason=%s", key, templateCondition, kubermaticv1.PolicyBindingReasonPolicyApplied), nil
			}
			if binding.Status.TemplateEnforced == nil || *binding.Status.TemplateEnforced != templateEnforced {
				return fmt.Errorf("PolicyBinding %s templateEnforced=%v, expected %t", key, binding.Status.TemplateEnforced, templateEnforced), nil
			}
			if templateEnforced && binding.Annotations[kubermaticv1.AnnotationPolicyEnforced] != "true" {
				return fmt.Errorf("PolicyBinding %s was not marked as generated from an enforced template", key), nil
			}
		}

		hasFinalizer := slices.Contains(binding.Finalizers, kubermaticv1.PolicyBindingCleanupFinalizer)
		if hasFinalizer != expectFinalizer {
			return fmt.Errorf("PolicyBinding %s cleanup finalizer present=%t, expected %t", key, hasFinalizer, expectFinalizer), nil
		}

		return nil, nil
	})
}

func waitForClusterPolicyReady(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, name string) error {
	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		policy := &kyvernov1.ClusterPolicy{}
		if err := client.Get(ctx, types.NamespacedName{Name: name}, policy); err != nil {
			return fmt.Errorf("failed to get ClusterPolicy %s: %w", name, err), nil
		}
		if !policy.Status.IsReady() {
			return fmt.Errorf("ClusterPolicy %s is not ready: %#v", name, policy.Status.Conditions), nil
		}
		if policy.Labels[policybindingcontroller.LabelPolicyBinding] != name || policy.Labels[policybindingcontroller.LabelPolicyTemplate] != name {
			return fmt.Errorf("ClusterPolicy %s has unexpected KKP ownership labels: %v", name, policy.Labels), nil
		}
		return nil, nil
	})
}

func waitForPolicyReady(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, namespace, name string) error {
	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		policy := &kyvernov1.Policy{}
		if err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, policy); err != nil {
			return fmt.Errorf("failed to get Policy %s/%s: %w", namespace, name, err), nil
		}
		if !policy.Status.IsReady() {
			return fmt.Errorf("Policy %s/%s is not ready: %#v", namespace, name, policy.Status.Conditions), nil
		}
		if policy.Labels[policybindingcontroller.LabelPolicyBinding] != name || policy.Labels[policybindingcontroller.LabelPolicyTemplate] != name {
			return fmt.Errorf("Policy %s/%s has unexpected KKP ownership labels: %v", namespace, name, policy.Labels), nil
		}
		return nil, nil
	})
}

func verifyPolicyAdmission(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, namespace string) error {
	if err := wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "denied-" + rand.String(8), Namespace: namespace}}
		err := client.Create(ctx, configMap)
		if err == nil {
			if deleteErr := client.Delete(ctx, configMap); deleteErr != nil && !apierrors.IsNotFound(deleteErr) {
				return nil, fmt.Errorf("policy allowed an invalid ConfigMap and cleanup failed: %w", deleteErr)
			}
			return errors.New("policy allowed a ConfigMap without the required label"), nil
		}
		if !strings.Contains(err.Error(), policyDenyMessage) {
			return fmt.Errorf("expected Kyverno denial containing %q, got: %w", policyDenyMessage, err), nil
		}
		return nil, nil
	}); err != nil {
		return err
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allowed-" + rand.String(8),
			Namespace: namespace,
			Labels:    map[string]string{requiredLabelKey: requiredLabelValue},
		},
	}
	if err := client.Create(ctx, configMap); err != nil {
		return fmt.Errorf("policy rejected a ConfigMap with the required label: %w", err)
	}
	if err := client.Delete(ctx, configMap); err != nil {
		return fmt.Errorf("failed to delete allowed ConfigMap: %w", err)
	}
	return nil
}

func waitForPolicyTemplateFinalizers(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, name string) error {
	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		template := &kubermaticv1.PolicyTemplate{}
		if err := client.Get(ctx, types.NamespacedName{Name: name}, template); err != nil {
			return fmt.Errorf("failed to get PolicyTemplate %s: %w", name, err), nil
		}
		for _, finalizer := range []string{
			kubermaticv1.PolicyTemplatePolicyBindingCleanupFinalizer,
			kubermaticv1.PolicyTemplateSeedCleanupFinalizer,
		} {
			if !slices.Contains(template.Finalizers, finalizer) {
				return fmt.Errorf("PolicyTemplate %s does not have finalizer %s", name, finalizer), nil
			}
		}
		return nil, nil
	})
}

func waitForClusterKyvernoFinalizer(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, clusterName string, present bool) error {
	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		cluster := &kubermaticv1.Cluster{}
		if err := client.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
			return fmt.Errorf("failed to get Cluster %s: %w", clusterName, err), nil
		}
		hasFinalizer := slices.Contains(cluster.Finalizers, kyvernocontroller.CleanupFinalizer)
		if hasFinalizer != present {
			return fmt.Errorf("Cluster %s Kyverno cleanup finalizer present=%t, expected %t", clusterName, hasFinalizer, present), nil
		}
		return nil, nil
	})
}

func waitForObjectDeleted(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, object ctrlruntimeclient.Object) error {
	key := ctrlruntimeclient.ObjectKeyFromObject(object)
	return wait.PollLog(ctx, logger, waitInterval, waitTimeout, func(ctx context.Context) (error, error) {
		current, ok := object.DeepCopyObject().(ctrlruntimeclient.Object)
		if !ok {
			return nil, fmt.Errorf("object %T does not implement controller-runtime client.Object", object)
		}
		err := client.Get(ctx, key, current)
		if apierrors.IsNotFound(err) || meta.IsNoMatchError(err) {
			return nil, nil
		}
		if err != nil {
			return fmt.Errorf("failed to check whether %T %s was deleted: %w", object, key, err), nil
		}
		return fmt.Errorf("%T %s still exists", object, key), nil
	})
}
