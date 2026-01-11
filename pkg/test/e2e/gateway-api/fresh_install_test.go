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
	"testing"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestGatewayAPIFreshInstall(t *testing.T) {
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

	logger.Info("Verifying Gateway API resources installation in the seed cluster...")
	if err := verifyGatewayAPIResourcesInstalled(ctx, t, seedClient, kubermaticNamespace, logger); err != nil {
		t.Fatalf("Gateway API resources not properly installed: %v", err)
	}

	if err := verifyNoIngressResources(ctx, t, seedClient, kubermaticNamespace, logger); err != nil {
		t.Fatalf("Ingress resources should not exist in Gateway API mode: %v", err)
	}

	testJig := jig.NewAWSCluster(seedClient, logger, credentials, 1, nil)
	testJig.ClusterJig.WithTestName("gateway-fresh-install")

	_, cluster, err := testJig.Setup(ctx, jig.WaitForReadyPods)
	defer testJig.Cleanup(ctx, t, true)

	if err != nil {
		t.Fatalf("Failed to setup test cluster: %v", err)
	}

	clusterNamespace := cluster.Status.NamespaceName
	logger.Infof("Test cluster created: %s (namespace: %s)", cluster.Name, clusterNamespace)

	if err := verifyUserClusterGatewayResources(ctx, t, seedClient, clusterNamespace, logger); err != nil {
		t.Fatalf("User cluster Gateway API resources verification failed: %v", err)
	}

	if err := verifyNamespaceGatewayLabel(ctx, t, seedClient, clusterNamespace, logger); err != nil {
		t.Fatalf("Namespace label verification failed: %v", err)
	}

	if err := testJig.WaitForHealthyControlPlane(ctx, controlPlaneHealthCheck); err != nil {
		t.Fatalf("Cluster did not become healthy: %v", err)
	}
}

func TestGatewayAPINamespaceLabel(t *testing.T) {
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

	testJig := jig.NewAWSCluster(seedClient, logger, credentials, 1, nil)
	testJig.ClusterJig.WithTestName("gateway-namespace-label")

	_, cluster, err := testJig.Setup(ctx, jig.WaitForReadyPods)
	defer testJig.Cleanup(ctx, t, true)

	if err != nil {
		t.Fatalf("Failed to setup test cluster: %v", err)
	}

	clusterNamespace := cluster.Status.NamespaceName

	if err := testJig.WaitForHealthyControlPlane(ctx, controlPlaneHealthCheck); err != nil {
		t.Fatalf("Cluster did not become healthy: %v", err)
	}

	if err := verifyNamespaceGatewayLabel(ctx, t, seedClient, clusterNamespace, logger); err != nil {
		t.Fatalf("Namespace label verification failed: %v", err)
	}
}
