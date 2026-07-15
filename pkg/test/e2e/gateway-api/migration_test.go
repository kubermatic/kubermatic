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
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

const nginxIngressControllerNamespace = "nginx-ingress-controller"

// syntheticLegacyIngresses lists the Ingress objects that the migration test
// pre-creates to simulate the leftover state from a 2.30 install. The 2.31
// installer must delete all of them when running `cleanupLegacyIngresses`.
var syntheticLegacyIngresses = []types.NamespacedName{
	{Namespace: "kubermatic", Name: "kubermatic"},
	{Namespace: "dex", Name: "dex"},
}

// TestGatewayAPIPreMigration asserts that, before the migration runs, the seed
// is in legacy Ingress mode: the nginx-ingress Deployment serves traffic and
// no Gateway API resources have been created yet. Mirrors the pre-migration
// check from main's run-gateway-api-migration-e2e.sh so the upgrade-path E2E
// has matching coverage.
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
// The namespace deletion implies that every resource in it (Deployment, LoadBalancer
// Service, etc.) has been torn down; the cloud LB is removed by cloud-controller-manager
// in reaction to the Service going away.
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
}

// TestLegacyIngressesCleanedUp verifies that the synthetic legacy Ingress objects
// pre-created by the migration test fixture (kubermatic/kubermatic, dex/dex) have
// been deleted by the installer's cleanupLegacyIngresses step.
func TestLegacyIngressesCleanedUp(t *testing.T) {
	ctx := context.Background()
	rawLogger := log.NewFromOptions(logOptions)
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))
	logger := rawLogger.Sugar()

	seedClient, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("Failed to build client: %v", err)
	}

	for _, key := range syntheticLegacyIngresses {
		ing := &networkingv1.Ingress{}
		err := seedClient.Get(ctx, key, ing)
		if err == nil {
			t.Fatalf("expected legacy Ingress %s to be deleted after --clean-nginx-lb, but it still exists", key.String())
		}
		if !apierrors.IsNotFound(err) {
			t.Fatalf("unexpected error checking legacy Ingress %s: %v", key.String(), err)
		}
		logger.Infof("legacy Ingress %s is gone, as expected", key.String())
	}
}
