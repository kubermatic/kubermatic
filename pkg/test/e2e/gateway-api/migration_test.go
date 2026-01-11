//go:build e2e

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

package gatewayapi

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestGatewayAPIPreMigration(t *testing.T) {
	ctx := context.Background()
	rawLogger := log.NewFromOptions(logOptions)
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))
	logger := rawLogger.Sugar()

	if err := credentials.Parse(); err != nil {
		t.Fatalf("Failed to get credentials: %v", err)
	}

	seedClient, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("Failed to build client: %v", err)
	}

	kubermaticNamespace := jig.KubermaticNamespace()

	if err := verifyIngressModeActive(ctx, t, seedClient, kubermaticNamespace, logger); err != nil {
		t.Fatalf("Ingress mode verification failed: %v", err)
	}

	if err := verifyNoGatewayAPIResources(ctx, t, seedClient, kubermaticNamespace, logger); err != nil {
		t.Fatalf("Gateway API resources should not exist in Ingress mode: %v", err)
	}

	testJig := jig.NewAWSCluster(seedClient, logger, credentials, 1, nil)
	testJig.ClusterJig.WithTestName("gateway-pre-migration")

	_, cluster, err := testJig.Setup(ctx, jig.WaitForReadyPods)
	if err != nil {
		t.Fatalf("Failed to setup test cluster: %v", err)
	}

	clusterNamespace := cluster.Status.NamespaceName
	logger.Infof("Test cluster created: %s (namespace: %s)", cluster.Name, clusterNamespace)

	if err := verifyClusterIngressResources(ctx, t, seedClient, clusterNamespace, logger); err != nil {
		t.Fatalf("Cluster Ingress verification failed: %v", err)
	}

	if err := testJig.WaitForHealthyControlPlane(ctx, controlPlaneHealthCheck); err != nil {
		t.Fatalf("Cluster did not become healthy: %v", err)
	}
}

func TestGatewayAPIPostMigration(t *testing.T) {
	ctx := context.Background()
	rawLogger := log.NewFromOptions(logOptions)
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))
	logger := rawLogger.Sugar()

	if err := credentials.Parse(); err != nil {
		t.Fatalf("Failed to get credentials: %v", err)
	}

	seedClient, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("Failed to build client: %v", err)
	}

	kubermaticNamespace := jig.KubermaticNamespace()

	if err := verifyGatewayAPIResourcesInstalled(ctx, t, seedClient, kubermaticNamespace, logger); err != nil {
		t.Fatalf("Gateway API mode verification failed: %v", err)
	}

	if err := verifyNoIngressResources(ctx, t, seedClient, kubermaticNamespace, logger); err != nil {
		t.Fatalf("Old Ingress should have been removed: %v", err)
	}

	cluster, err := findTestCluster(ctx, seedClient, logger)
	if err != nil {
		t.Fatalf("Failed to find pre-migration cluster: %v", err)
	}
	logger.Infof("Found pre-migration cluster: %s", cluster.Name)

	clusterNamespace := cluster.Status.NamespaceName

	testJig := jig.NewClusterJig(seedClient, logger)
	testJig.WithExistingCluster(cluster.Name)
	testJig.WithProjectName(cluster.Labels[kubermaticv1.ProjectIDLabelKey])

	defer func() {
		if err := testJig.Delete(ctx, true); err != nil {
			t.Logf("Warning: failed to delete cluster: %v", err)
		}
	}()

	if err := verifyUserClusterGatewayResources(ctx, t, seedClient, clusterNamespace, logger); err != nil {
		t.Fatalf("Cluster Gateway API resources verification failed: %v", err)
	}

	if err := verifyNamespaceGatewayLabel(ctx, t, seedClient, clusterNamespace, logger); err != nil {
		t.Fatalf("Namespace label verification failed: %v", err)
	}

	if err := testJig.WaitForHealthyControlPlane(ctx, controlPlaneHealthCheck); err != nil {
		t.Fatalf("Cluster did not become healthy after migration: %v", err)
	}
}

func verifyIngressModeActive(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, namespace string, logger *zap.SugaredLogger) error {
	t.Helper()

	logger.Info("Checking main Ingress...")
	ingressName := types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultIngressName}
	ingress := &networkingv1.Ingress{}

	err := wait.PollImmediateLog(ctx, logger, defaultInterval, defaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		if err := client.Get(ctx, ingressName, ingress); err != nil {
			return fmt.Errorf("Ingress not found: %w", err), nil
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("Ingress not found: %w", err)
	}
	logger.Infof("Ingress %s/%s exists", namespace, defaulting.DefaultIngressName)

	if ingress.Annotations != nil {
		logger.Infof("Ingress annotations: %v", ingress.Annotations)
	}

	return nil
}

func verifyClusterIngressResources(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, namespace string, logger *zap.SugaredLogger) error {
	t.Helper()
	ingressName := types.NamespacedName{Namespace: namespace, Name: "kubermatic"}
	ingress := &networkingv1.Ingress{}

	err := wait.PollImmediateLog(ctx, logger, defaultInterval, defaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		if err := client.Get(ctx, ingressName, ingress); err != nil {
			return fmt.Errorf("cluster Ingress not found: %w", err), nil
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("cluster Ingress not found: %w", err)
	}
	logger.Infof("Cluster Ingress %s/%s exists", namespace, "kubermatic")
	gwName := types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultGatewayName}
	gw := &gatewayapiv1.Gateway{}
	err = client.Get(ctx, gwName, gw)
	if err == nil {
		return fmt.Errorf("Gateway should not exist in user cluster namespace in Ingress mode")
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check for Gateway: %w", err)
	}

	logger.Infof("No Gateway in cluster namespace (expected for Ingress mode)")
	return nil
}

func findTestCluster(ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger) (*kubermaticv1.Cluster, error) {
	clusterList := &kubermaticv1.ClusterList{}

	if err := client.List(ctx, clusterList); err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	for _, cluster := range clusterList.Items {
		if cluster.Labels != nil {
			if testName, ok := cluster.Labels["kubermatic-test"]; ok && testName == "gateway-pre-migration" {
				return &cluster, nil
			}
		}
	}

	for _, cluster := range clusterList.Items {
		if cluster.Labels != nil {
			if testName, ok := cluster.Labels["kubermatic-test"]; ok {
				logger.Infof("Found cluster with test label: %s", testName)
				return &cluster, nil
			}
		}
	}

	return nil, fmt.Errorf("no test cluster found")
}
