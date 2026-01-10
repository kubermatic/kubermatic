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

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var (
	credentials jig.AWSCredentials
	logOptions  = utils.DefaultLogOptions
)

const (
	defaultInterval         = 5 * time.Second
	defaultTimeout          = 10 * time.Minute
	controlPlaneHealthCheck = 10 * time.Minute
)

func init() {
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

func verifyGatewayAPIResourcesInstalled(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, namespace string, logger *zap.SugaredLogger) error {
	t.Helper()

	logger.Info("Checking GatewayClass...")
	gcName := types.NamespacedName{Name: defaulting.DefaultGatewayClassName}
	gc := &gatewayapiv1.GatewayClass{}

	err := wait.PollImmediateLog(ctx, logger, defaultInterval, 2*time.Minute, func(ctx context.Context) (transient error, terminal error) {
		if err := client.Get(ctx, gcName, gc); err != nil {
			return fmt.Errorf("GatewayClass not found: %w", err), nil
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("GatewayClass %q not found: %w", defaulting.DefaultGatewayClassName, err)
	}

	logger.Infof("GatewayClass %q exists", defaulting.DefaultGatewayClassName)

	logger.Info("Checking main Gateway...")
	gtwName := types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultGatewayName}
	gtw := &gatewayapiv1.Gateway{}

	err = wait.PollImmediateLog(ctx, logger, defaultInterval, defaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		if err := client.Get(ctx, gtwName, gtw); err != nil {
			return fmt.Errorf("Gateway not found: %w", err), nil
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("Gateway not found: %w", err)
	}

	logger.Infof("Gateway %q exists", ctrlruntimeclient.ObjectKeyFromObject(gtw).String())

	logger.Info("Checking main HTTPRoute...")
	hrName := types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultHTTPRouteName}
	hr := &gatewayapiv1.HTTPRoute{}

	err = wait.PollImmediateLog(ctx, logger, defaultInterval, defaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		if err := client.Get(ctx, hrName, hr); err != nil {
			return fmt.Errorf("HTTPRoute not found: %w", err), nil
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("HTTPRoute not found: %w", err)
	}

	logger.Infof("HTTPRoute %q exists", ctrlruntimeclient.ObjectKeyFromObject(hr).String())

	return nil
}

func verifyNoIngressResources(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, namespace string, logger *zap.SugaredLogger) error {
	t.Helper()

	ingressName := types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultIngressName}
	err := wait.PollImmediateLog(ctx, logger, defaultInterval, 2*time.Minute, func(ctx context.Context) (transient error, terminal error) {
		ingress := &networkingv1.Ingress{}
		err := client.Get(ctx, ingressName, ingress)
		if err == nil {
			return fmt.Errorf("Ingress should not exist in Gateway API mode"), nil
		}
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to check for Ingress: %w", err)
		}
		return nil, nil
	})

	if err != nil {
		return err
	}

	logger.Infof("No Ingress resources found (expected for Gateway API mode)")
	return nil
}

func verifyNoGatewayAPIResources(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, namespace string, logger *zap.SugaredLogger) error {
	t.Helper()
	gwName := types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultGatewayName}
	err := wait.PollImmediateLog(ctx, logger, defaultInterval, 2*time.Minute, func(ctx context.Context) (transient error, terminal error) {
		gw := &gatewayapiv1.Gateway{}
		err := client.Get(ctx, gwName, gw)
		if err == nil {
			return fmt.Errorf("Gateway should not exist in Ingress mode"), nil
		}
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to check for Gateway: %w", err)
		}
		return nil, nil
	})

	if err != nil {
		return err
	}

	hrName := types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultHTTPRouteName}
	err = wait.PollImmediateLog(ctx, logger, defaultInterval, 2*time.Minute, func(ctx context.Context) (transient error, terminal error) {
		hr := &gatewayapiv1.HTTPRoute{}
		err := client.Get(ctx, hrName, hr)
		if err == nil {
			return fmt.Errorf("HTTPRoute should not exist in Ingress mode"), nil
		}
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to check for HTTPRoute: %w", err)
		}
		return nil, nil
	})

	if err != nil {
		return err
	}

	logger.Infof("No Gateway API resources found (expected for Ingress mode)")
	return nil
}

func verifyUserClusterGatewayResources(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, namespace string, logger *zap.SugaredLogger) error {
	t.Helper()

	logger.Info("Checking user cluster Gateway...")
	gwName := types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultGatewayName}
	gw := &gatewayapiv1.Gateway{}

	err := wait.PollImmediateLog(ctx, logger, defaultInterval, defaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		if err := client.Get(ctx, gwName, gw); err != nil {
			return fmt.Errorf("user cluster Gateway not found: %w", err), nil
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("user cluster Gateway not found: %w", err)
	}
	logger.Infof("User cluster Gateway %s/%s exists", namespace, defaulting.DefaultGatewayName)

	logger.Info("Checking user cluster HTTPRoute...")
	hrName := types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultHTTPRouteName}
	hr := &gatewayapiv1.HTTPRoute{}

	err = wait.PollImmediateLog(ctx, logger, defaultInterval, defaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		if err := client.Get(ctx, hrName, hr); err != nil {
			return fmt.Errorf("user cluster HTTPRoute not found: %w", err), nil
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("user cluster HTTPRoute not found: %w", err)
	}
	logger.Infof("User cluster HTTPRoute %s/%s exists", namespace, defaulting.DefaultHTTPRouteName)

	ingressName := types.NamespacedName{Namespace: namespace, Name: "kubermatic"}
	ingress := &networkingv1.Ingress{}

	err = wait.PollImmediateLog(ctx, logger, defaultInterval, 2*time.Minute, func(ctx context.Context) (transient error, terminal error) {
		err := client.Get(ctx, ingressName, ingress)
		if err == nil {
			return fmt.Errorf("Ingress should not exist in user cluster namespace in Gateway API mode"), nil
		}
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to check for Ingress: %w", err)
		}
		return nil, nil
	})

	if err != nil {
		return err
	}

	logger.Infof("No Ingress in user cluster namespace (expected)")
	return nil
}

func verifyNamespaceGatewayLabel(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, namespace string, logger *zap.SugaredLogger) error {
	t.Helper()

	logger.Info("Waiting for namespace Gateway access label...")

	err := wait.PollImmediateLog(ctx, logger, defaultInterval, defaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		ns := &corev1.Namespace{}
		if err := client.Get(ctx, types.NamespacedName{Name: namespace}, ns); err != nil {
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
		return fmt.Errorf("namespace label verification failed: %w", err)
	}

	logger.Infof("Namespace %s has Gateway access label", namespace)
	return nil
}
