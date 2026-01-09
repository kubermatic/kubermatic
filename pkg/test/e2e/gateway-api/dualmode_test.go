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
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
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

const (
	defaultTimeout     = 10 * time.Minute
	defaultInterval    = 5 * time.Second
	healthCheckTimeout = 10 * time.Minute
)

func TestGatewayAPIDualModeSwitching(t *testing.T) {
	ctx := context.Background()
	rawLogger := log.NewFromOptions(logOptions)
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))
	logger := rawLogger.Sugar()

	if err := credentials.Parse(); err != nil {
		t.Fatalf("Failed to get credentials: %v", err)
	}

	seedClient, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("Failed to get seed client: %v", err)
	}

	kubermaticNamespace := jig.KubermaticNamespace()
	cfgName := types.NamespacedName{
		Namespace: kubermaticNamespace,
		Name:      "kubermatic",
	}

	originalCfg, err := getKubermaticConfiguration(ctx, seedClient, cfgName)
	if err != nil {
		t.Fatalf("Failed to get KubermaticConfiguration: %v", err)
	}

	originalGatewayEnabled := originalCfg.GatewayAPIEnabled()

	t.Cleanup(func() {
		logger.Info("Restoring original Gateway.enable setting")
		currentCfg, getErr := getKubermaticConfiguration(ctx, seedClient, cfgName)
		if getErr != nil {
			logger.Errorf("Failed to get current configuration for restoration: %v", getErr)
			return
		}
		if currentCfg.GatewayAPIEnabled() != originalGatewayEnabled {
			if err := setGatewayEnable(ctx, seedClient, cfgName, originalGatewayEnabled); err != nil {
				logger.Errorf("Failed to restore original configuration: %v", err)
			}
		}
	})

	testJig := jig.NewAWSCluster(seedClient, logger, credentials, 1, nil)
	testJig.ClusterJig.WithTestName("gateway-dualmode")

	_, cluster, err := testJig.Setup(ctx, jig.WaitForReadyPods)
	defer testJig.Cleanup(ctx, t, true)
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}

	clusterNamespace := cluster.Status.NamespaceName
	logger.Infof("Using test cluster namespace for testing: %s", clusterNamespace)
	verifyMode(t, ctx, seedClient, logger, clusterNamespace, originalGatewayEnabled)

	if !originalGatewayEnabled {
		logger.Info("Switching to Gateway API mode...")
		if err := setGatewayEnable(ctx, seedClient, cfgName, true); err != nil {
			t.Fatalf("Failed to enable Gateway API: %v", err)
		}

		logger.Info("Waiting for cluster to become healthy after enabling Gateway API...")
		if err := testJig.WaitForHealthyControlPlane(ctx, healthCheckTimeout); err != nil {
			t.Fatalf("Cluster did not become healthy after enabling Gateway API: %v", err)
		}

		logger.Info("Verifying Gateway API mode...")
		verifyMode(t, ctx, seedClient, logger, clusterNamespace, true)

		logger.Info("Switching back to Ingress mode...")
		if err := setGatewayEnable(ctx, seedClient, cfgName, false); err != nil {
			t.Fatalf("Failed to disable Gateway API: %v", err)
		}

		logger.Info("Waiting for cluster to become healthy after disabling Gateway API...")
		if err := testJig.WaitForHealthyControlPlane(ctx, healthCheckTimeout); err != nil {
			t.Fatalf("Cluster did not become healthy after disabling Gateway API: %v", err)
		}

		logger.Info("Verifying Ingress mode restored...")
		verifyMode(t, ctx, seedClient, logger, clusterNamespace, false)
	} else {
		logger.Info("Switching to Ingress mode...")
		if err := setGatewayEnable(ctx, seedClient, cfgName, false); err != nil {
			t.Fatalf("Failed to disable Gateway API: %v", err)
		}

		logger.Info("Waiting for cluster to become healthy after disabling Gateway API...")
		if err := testJig.WaitForHealthyControlPlane(ctx, healthCheckTimeout); err != nil {
			t.Fatalf("Cluster did not become healthy after disabling Gateway API: %v", err)
		}

		logger.Info("Verifying Ingress mode...")
		verifyMode(t, ctx, seedClient, logger, clusterNamespace, false)

		logger.Info("Switching back to Gateway API mode...")
		if err := setGatewayEnable(ctx, seedClient, cfgName, true); err != nil {
			t.Fatalf("Failed to enable Gateway API: %v", err)
		}

		logger.Info("Waiting for cluster to become healthy after enabling Gateway API...")
		if err := testJig.WaitForHealthyControlPlane(ctx, healthCheckTimeout); err != nil {
			t.Fatalf("Cluster did not become healthy after enabling Gateway API: %v", err)
		}

		logger.Info("Verifying Gateway API mode restored...")
		verifyMode(t, ctx, seedClient, logger, clusterNamespace, true)
	}

	logger.Info("Gateway API dual-mode switching test completed successfully")
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
		t.Fatalf("Failed to get seed client: %v", err)
	}

	kubermaticNamespace := jig.KubermaticNamespace()
	cfgName := types.NamespacedName{
		Namespace: kubermaticNamespace,
		Name:      "kubermatic",
	}

	originalCfg, err := getKubermaticConfiguration(ctx, seedClient, cfgName)
	if err != nil {
		t.Fatalf("Failed to get KubermaticConfiguration: %v", err)
	}

	originalGatewayEnabled := originalCfg.GatewayAPIEnabled()

	t.Cleanup(func() {
		logger.Info("Restoring original Gateway.enable setting")
		currentCfg, getErr := getKubermaticConfiguration(ctx, seedClient, cfgName)
		if getErr != nil {
			logger.Errorf("Failed to get current configuration for restoration: %v", getErr)
			return
		}
		if currentCfg.GatewayAPIEnabled() != originalGatewayEnabled {
			if err := setGatewayEnable(ctx, seedClient, cfgName, originalGatewayEnabled); err != nil {
				logger.Errorf("Failed to restore original configuration: %v", err)
			}
		}
	})

	testJig := jig.NewAWSCluster(seedClient, logger, credentials, 1, nil)
	testJig.ClusterJig.WithTestName("gateway-namespace-label")

	_, cluster, err := testJig.Setup(ctx, jig.WaitForReadyPods)
	defer testJig.Cleanup(ctx, t, true)
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}

	clusterNamespace := cluster.Status.NamespaceName

	if !originalGatewayEnabled {
		logger.Info("Enabling Gateway API...")
		if err := setGatewayEnable(ctx, seedClient, cfgName, true); err != nil {
			t.Fatalf("Failed to enable Gateway API: %v", err)
		}

		logger.Info("Waiting for cluster to become healthy after enabling Gateway API...")
		if err := testJig.WaitForHealthyControlPlane(ctx, healthCheckTimeout); err != nil {
			t.Fatalf("Cluster did not become healthy: %v", err)
		}
	}

	logger.Info("Waiting for namespace label to be applied...")

	err = wait.PollImmediateLog(ctx, logger, defaultInterval, defaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		ns := &corev1.Namespace{}
		if err := seedClient.Get(ctx, types.NamespacedName{Name: clusterNamespace}, ns); err != nil {
			return fmt.Errorf("failed to get namespace: %w", err), nil
		}

		expectedLabel := common.GatewayAccessLabelKey
		if ns.Labels == nil {
			return fmt.Errorf("namespace has no labels"), nil
		}

		if ns.Labels[expectedLabel] != "true" {
			return fmt.Errorf("namespace missing %q label (has: %v)", expectedLabel, ns.Labels), nil
		}

		return nil, nil
	})

	if err != nil {
		t.Fatalf("Failed to wait for namespace label: %v", err)
	}

	t.Log("Namespace label verified successfully")
}

func getKubermaticConfiguration(ctx context.Context, client ctrlruntimeclient.Client, name types.NamespacedName) (*kubermaticv1.KubermaticConfiguration, error) {
	cfg := &kubermaticv1.KubermaticConfiguration{}
	if err := client.Get(ctx, name, cfg); err != nil {
		return nil, fmt.Errorf("failed to get KubermaticConfiguration: %w", err)
	}
	return cfg, nil
}

func setGatewayEnable(ctx context.Context, client ctrlruntimeclient.Client, name types.NamespacedName, enable bool) error {
	currentCfg, err := getKubermaticConfiguration(ctx, client, name)
	if err != nil {
		return err
	}

	patch := ctrlruntimeclient.MergeFrom(currentCfg.DeepCopy())
	if currentCfg.Spec.Ingress.Gateway == nil {
		currentCfg.Spec.Ingress.Gateway = &kubermaticv1.KubermaticGatewayConfiguration{}
	}
	currentCfg.Spec.Ingress.Gateway.Enable = enable

	if err := client.Patch(ctx, currentCfg, patch); err != nil {
		return fmt.Errorf("failed to patch KubermaticConfiguration: %w", err)
	}

	return nil
}

func verifyMode(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, logger *zap.SugaredLogger, clusterNamespace string, gatewayEnabled bool) {
	logger.Infof("Verifying current mode (gatewayEnabled=%v)...", gatewayEnabled)
	logger.Infof("Waiting for resources to be in expected state (gatewayEnabled=%v)...", gatewayEnabled)

	err := wait.PollImmediateLog(ctx, logger, defaultInterval, defaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		if gatewayEnabled {
			gateway := &gatewayapiv1.Gateway{}
			gwName := types.NamespacedName{Namespace: clusterNamespace, Name: defaulting.DefaultGatewayName}
			if err := client.Get(ctx, gwName, gateway); err != nil {
				return fmt.Errorf("Gateway not found yet: %w", err), nil
			}

			route := &gatewayapiv1.HTTPRoute{}
			routeName := types.NamespacedName{Namespace: clusterNamespace, Name: defaulting.DefaultHTTPRouteName}
			if err := client.Get(ctx, routeName, route); err != nil {
				return fmt.Errorf("HTTPRoute not found yet: %w", err), nil
			}

			ingress := &networkingv1.Ingress{}
			ingressName := types.NamespacedName{Namespace: clusterNamespace, Name: "kubermatic"}
			if err := client.Get(ctx, ingressName, ingress); err == nil {
				return fmt.Errorf("Ingress still exists (should be cleaned up)"), nil
			} else if !apierrors.IsNotFound(err) {
				return nil, fmt.Errorf("unexpected error checking for Ingress: %w", err)
			}

			return nil, nil
		} else {
			ingress := &networkingv1.Ingress{}
			ingressName := types.NamespacedName{Namespace: clusterNamespace, Name: "kubermatic"}
			if err := client.Get(ctx, ingressName, ingress); err != nil {
				return fmt.Errorf("Ingress not found yet: %w", err), nil
			}

			gateway := &gatewayapiv1.Gateway{}
			gwName := types.NamespacedName{Namespace: clusterNamespace, Name: defaulting.DefaultGatewayName}
			if err := client.Get(ctx, gwName, gateway); err == nil {
				return fmt.Errorf("Gateway still exists (should be cleaned up)"), nil
			} else if !apierrors.IsNotFound(err) {
				return nil, fmt.Errorf("unexpected error checking for Gateway: %w", err)
			}

			route := &gatewayapiv1.HTTPRoute{}
			routeName := types.NamespacedName{Namespace: clusterNamespace, Name: defaulting.DefaultHTTPRouteName}
			if err := client.Get(ctx, routeName, route); err == nil {
				return fmt.Errorf("HTTPRoute still exists (should be cleaned up)"), nil
			} else if !apierrors.IsNotFound(err) {
				return nil, fmt.Errorf("unexpected error checking for HTTPRoute: %w", err)
			}

			return nil, nil
		}
	})

	if err != nil {
		t.Fatalf("Failed to verify mode: %v", err)
	}
}
