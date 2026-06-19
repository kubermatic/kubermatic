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
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	nginxIngressControllerNamespace   = "nginx-ingress-controller"
	nginxIngressControllerServiceName = "nginx-ingress-controller-controller"
)

func TestGatewayAPIPreMigration(t *testing.T) {
	ctx := context.Background()
	rawLogger := log.NewFromOptions(logOptions)
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))
	logger := rawLogger.Sugar()

	seedClient, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("Failed to build client: %v", err)
	}

	if err := verifyIngressMode(ctx, t, seedClient, logger); err != nil {
		t.Fatalf("Ingress mode verification failed: %v", err)
	}

	if err := verifyNoGatewayAPIResources(ctx, t, seedClient, logger); err != nil {
		t.Fatalf("Gateway API resources should not exist in Ingress mode: %v", err)
	}
}

func TestGatewayAPIPostMigration(t *testing.T) {
	ctx := context.Background()
	rawLogger := log.NewFromOptions(logOptions)
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))
	logger := rawLogger.Sugar()

	seedClient, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("Failed to build client: %v", err)
	}

	if err := verifyGatewayAPIModeResources(ctx, t, seedClient, logger); err != nil {
		t.Fatalf("Gateway API mode verification failed: %v", err)
	}

	logger.Info("Verifying HTTP connectivity through Gateway...")
	if err := verifyGatewayHTTPConnectivity(ctx, t, seedClient, logger); err != nil {
		t.Fatalf("Gateway HTTP connectivity verification failed: %v", err)
	}
}

// TestNginxIngressControllerCleanedUp verifies that the legacy nginx-ingress-controller
// Helm release has been removed after the installer was re-run with --clean-nginx-lb.
// The namespace and the LoadBalancer Service must both be gone (Service deletion
// triggers cloud LB teardown).
func TestNginxIngressControllerCleanedUp(t *testing.T) {
	ctx := context.Background()
	rawLogger := log.NewFromOptions(logOptions)
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))
	logger := rawLogger.Sugar()

	seedClient, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("Failed to build client: %v", err)
	}

	ns := &corev1.Namespace{}
	err = seedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: nginxIngressControllerNamespace}, ns)
	switch {
	case apierrors.IsNotFound(err):
		logger.Infof("nginx-ingress-controller namespace %q is gone, as expected", nginxIngressControllerNamespace)
	case err != nil:
		t.Fatalf("failed to probe for %s namespace: %v", nginxIngressControllerNamespace, err)
	case ns.DeletionTimestamp == nil:
		t.Fatalf("expected nginx-ingress-controller namespace to be deleted or terminating after --clean-nginx-lb, but it is still active: %+v", ns)
	default:
		logger.Infof("nginx-ingress-controller namespace is terminating, as expected")
	}

	svc := &corev1.Service{}
	err = seedClient.Get(ctx, types.NamespacedName{Namespace: nginxIngressControllerNamespace, Name: nginxIngressControllerServiceName}, svc)
	if err == nil {
		t.Fatalf("expected LoadBalancer Service %s/%s to be deleted after --clean-nginx-lb, but it still exists", nginxIngressControllerNamespace, nginxIngressControllerServiceName)
	}
	if !apierrors.IsNotFound(err) {
		t.Fatalf("unexpected error checking LoadBalancer Service: %v", err)
	}
	logger.Info("LoadBalancer Service has been removed; cloud LB should be torn down by cloud-controller-manager")
}
