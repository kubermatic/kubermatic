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

	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
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
