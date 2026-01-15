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
	"net/http"
	"net/url"
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
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var (
	logOptions = utils.DefaultLogOptions
)

const (
	defaultInterval = 5 * time.Second
	defaultTimeout  = 10 * time.Minute
)

func init() {
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

// verifyGatewayAPIModeResources verifies that the Gateway API resources are properly installed in Gateway API mode
// and no `kubermatic/kubermatic` ingress resource exists in the cluster.
func verifyGatewayAPIModeResources(ctx context.Context, t *testing.T, c ctrlruntimeclient.Client, l *zap.SugaredLogger) error {
	t.Helper()

	ns := jig.KubermaticNamespace()

	l.Info("Verifying GatewayClass...")
	gcName := types.NamespacedName{Name: defaulting.DefaultGatewayClassName}
	gc := &gatewayapiv1.GatewayClass{}

	err := wait.PollImmediateLog(ctx, l, defaultInterval, 2*time.Minute, func(ctx context.Context) (transient error, terminal error) {
		err := c.Get(ctx, gcName, gc)
		if err != nil {
			return err, nil
		}

		gcAccepted := meta.IsStatusConditionTrue(
			gc.Status.Conditions,
			string(gatewayapiv1.GatewayClassConditionStatusAccepted),
		)
		if gcAccepted {
			return nil, nil
		}

		return fmt.Errorf("GatewayClass %q is not accepted yet, status: %+v", defaulting.DefaultGatewayClassName, gc.Status), nil
	})
	if err != nil {
		return fmt.Errorf("GatewayClass %q not found: %w", defaulting.DefaultGatewayClassName, err)
	}

	l.Infof("GatewayClass %q exists", defaulting.DefaultGatewayClassName)

	gtwName := types.NamespacedName{Namespace: ns, Name: defaulting.DefaultGatewayName}
	l.Infof("verifying Gateway %q", gtwName.String())

	gtw := &gatewayapiv1.Gateway{}
	err = wait.PollImmediateLog(ctx, l, defaultInterval, defaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		err := c.Get(ctx, gtwName, gtw)
		if err != nil {
			return fmt.Errorf("Gateway not found: %w", err), nil
		}

		gtwProgrammed := meta.IsStatusConditionTrue(
			gtw.Status.Conditions,
			string(gatewayapiv1.GatewayConditionProgrammed),
		)
		if !gtwProgrammed {
			l.Infof("%+v", gtw.Status.Conditions)
			return fmt.Errorf("Gateway %q is not programmed yet", gtwName.String()), nil
		}

		listeners := gtw.Status.Listeners
		if len(listeners) == 0 {
			return fmt.Errorf("Gateway %q has no listeners yet, status: %+v", gtwName.String(), gtw.Status), nil
		}

		attachedRoutes := listeners[0].AttachedRoutes
		if attachedRoutes != 2 {
			return fmt.Errorf("Gateway %q has no expected attached routes, status: %+v", gtwName.String(), gtw.Status), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("user cluster Gateway not found: %w", err)
	}

	l.Infof("Gateway %q exists", gtwName.String())

	hrNn := types.NamespacedName{Namespace: ns, Name: defaulting.DefaultHTTPRouteName}
	l.Infof("verifying HTTPRoute %q", hrNn.String())
	hr := &gatewayapiv1.HTTPRoute{}

	err = wait.PollImmediateLog(ctx, l, defaultInterval, defaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		if err := c.Get(ctx, hrNn, hr); err != nil {
			return fmt.Errorf("HTTPRoute not found: %w", err), nil
		}

		if len(hr.Status.Parents) == 0 {
			return fmt.Errorf("HTTPRoute %q has no parents yet, status: %+v", defaulting.DefaultHTTPRouteName, hr.Status), nil
		}

		if len(hr.Status.Parents) != 1 {
			return fmt.Errorf("HTTPRoute %q has unexpected number of parents: %d", defaulting.DefaultHTTPRouteName, len(hr.Status.Parents)), nil
		}

		controllerName := hr.Status.Parents[0].ControllerName
		if controllerName != "gateway.k8c.io/envoy-gateway" {
			return fmt.Errorf("HTTPRoute %q has unexpected parent Gateway %q", hrNn.String(), controllerName), nil
		}

		parentRef := hr.Status.Parents[0].ParentRef
		if parentRef.Name != defaulting.DefaultGatewayName ||
			parentRef.Namespace == nil || *parentRef.Namespace != gatewayapiv1.Namespace(jig.KubermaticNamespace()) {
			return fmt.Errorf("HTTPRoute %q has unexpected parent Gateway name %q", hrNn.String(), parentRef.Name), nil
		}

		routeAccepted := meta.IsStatusConditionTrue(
			hr.Status.Parents[0].Conditions,
			string(gatewayapiv1.RouteConditionAccepted),
		)
		if routeAccepted {
			return nil, nil
		}

		return fmt.Errorf("Route %q is not accepted yet, status: %+v", hrNn.String(), hr.Status), nil
	})
	if err != nil {
		return fmt.Errorf("user cluster HTTPRoute not found: %w", err)
	}

	l.Infof("HTTPRoute %q exists", hrNn.String())

	ingNn := types.NamespacedName{Namespace: ns, Name: defaulting.DefaultIngressName}
	l.Infof("verifying that Ingress %q does not exist since we use Gatewy API", ingNn.String())

	ingress := &networkingv1.Ingress{}
	err = wait.PollImmediateLog(ctx, l, defaultInterval, 2*time.Minute, func(ctx context.Context) (transient error, terminal error) {
		err := c.Get(ctx, ingNn, ingress)
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

	l.Infof("No kubermatic ingress in the namespace as expected since we run Gateway API mode")

	return nil
}

func verifyNoGatewayAPIResources(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, logger *zap.SugaredLogger) error {
	t.Helper()
	ns := jig.KubermaticNamespace()

	logger.Info("verify that no gateway api resources exist in ingress mode")
	gtwName := types.NamespacedName{Namespace: ns, Name: defaulting.DefaultGatewayName}
	err := wait.PollImmediateLog(ctx, logger, defaultInterval, 2*time.Minute, func(ctx context.Context) (transient error, terminal error) {
		gw := &gatewayapiv1.Gateway{}

		err := client.Get(ctx, gtwName, gw)
		switch {
		case err == nil:
			return nil, fmt.Errorf("Gateway found but should not exist")
		case apierrors.IsNotFound(err), meta.IsNoMatchError(err):
			return nil, nil
		default:
			return fmt.Errorf("unexpected error checking Gateway: %w", err), nil
		}
	})
	if err != nil {
		return err
	}

	routeName := types.NamespacedName{Namespace: ns, Name: defaulting.DefaultHTTPRouteName}
	err = wait.PollImmediateLog(ctx, logger, defaultInterval, 2*time.Minute, func(ctx context.Context) (transient error, terminal error) {
		route := &gatewayapiv1.HTTPRoute{}

		err := client.Get(ctx, routeName, route)
		switch {
		case err == nil:
			return nil, fmt.Errorf("HTTPRoute found but should not exist")
		case apierrors.IsNotFound(err), meta.IsNoMatchError(err):
			return nil, nil
		default:
			return fmt.Errorf("unexpected error checking HTTPRoute: %w", err), nil
		}
	})
	if err != nil {
		return err
	}

	logger.Infof("No Gateway API resources found (expected for Ingress mode)")
	return nil
}

func verifyNamespaceGatewayLabel(ctx context.Context, t *testing.T, c ctrlruntimeclient.Client, l *zap.SugaredLogger) error {
	t.Helper()

	namespaces := []string{jig.KubermaticNamespace(), "dex"}

	errs := make([]error, 0)
	for _, ns := range namespaces {
		l.Infof("Waiting for namespace %q Gateway access label...", ns)

		err := wait.PollImmediateLog(ctx, l, defaultInterval, defaultTimeout, func(ctx context.Context) (transient error, terminal error) {
			namespace := &corev1.Namespace{}

			err := c.Get(ctx, types.NamespacedName{Name: ns}, namespace)
			if err != nil {
				return fmt.Errorf("failed to get namespace: %w", err), nil
			}

			if namespace.Labels == nil {
				return fmt.Errorf("namespace has no labels"), nil
			}

			expectedLabel := common.GatewayAccessLabelKey
			if namespace.Labels[expectedLabel] != "true" {
				return fmt.Errorf("namespace missing %q label (has: %+v)", expectedLabel, namespace.Labels), nil
			}

			return nil, nil
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("namespace %q label verification failed: %w", ns, err))
		}
	}
	if len(errs) > 0 {
		return kerrors.NewAggregate(errs)
	}

	return nil
}

func verifyIngressMode(ctx context.Context, t *testing.T, c ctrlruntimeclient.Client, l *zap.SugaredLogger) error {
	t.Helper()

	ns := jig.KubermaticNamespace()

	ingressName := types.NamespacedName{Namespace: ns, Name: defaulting.DefaultIngressName}
	l.Infof("Checking main %q Ingress...", ingressName.String())

	ingress := &networkingv1.Ingress{}
	err := wait.PollImmediateLog(ctx, l, defaultInterval, defaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		if err := c.Get(ctx, ingressName, ingress); err != nil {
			return fmt.Errorf("Ingress not found: %w", err), nil
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("Ingress not found: %w", err)
	}

	l.Infof("Ingress %q exists", ingressName.String())

	return nil
}

func verifyGatewayHTTPConnectivity(ctx context.Context, t *testing.T, c ctrlruntimeclient.Client, l *zap.SugaredLogger) error {
	t.Helper()

	l.Info("Testing HTTP connectivity through Gateway...")

	httprouteName := types.NamespacedName{
		Namespace: jig.KubermaticNamespace(),
		Name:      defaulting.DefaultHTTPRouteName,
	}

	route := &gatewayapiv1.HTTPRoute{}
	err := wait.PollImmediateLog(ctx, l, defaultInterval, defaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		if err := c.Get(ctx, httprouteName, route); err != nil {
			return fmt.Errorf("HTTPRoute not found: %w", err), nil
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for HTTPRoute: %w", err)
	}

	if len(route.Spec.Hostnames) == 0 {
		return fmt.Errorf("HTTPRoute has no hostnames configured")
	}

	hostname := string(route.Spec.Hostnames[0])
	l.Infof("Using HTTPRoute hostname: %s", hostname)

	gtwName := types.NamespacedName{
		Name:      defaulting.DefaultGatewayName,
		Namespace: jig.KubermaticNamespace(),
	}

	var gatewayIP string
	err = wait.PollImmediateLog(ctx, l, defaultInterval, 2*time.Minute, func(ctx context.Context) (transient error, terminal error) {
		gw := &gatewayapiv1.Gateway{}
		if err := c.Get(ctx, gtwName, gw); err != nil {
			return fmt.Errorf("Gateway not found: %w", err), nil
		}

		if len(gw.Status.Addresses) == 0 {
			return fmt.Errorf("Gateway has no addresses yet, status: %+v", gw.Status), nil
		}

		for _, addr := range gw.Status.Addresses {
			if addr.Type != nil && *addr.Type == gatewayapiv1.IPAddressType {
				gatewayIP = string(addr.Value)
				return nil, nil
			}
		}

		return fmt.Errorf("Gateway has no IPAddress addresses"), nil
	})
	if err != nil {
		return fmt.Errorf("failed to get Gateway address: %w", err)
	}

	const envoyNodePort = "30080"

	address := fmt.Sprintf("%s:%s", gatewayIP, envoyNodePort)

	httpClient := &http.Client{}

	baseURL := (&url.URL{
		Scheme: "http",
		Host:   address,
	}).String()
	l.Infof("Using Gateway address from Gateway.status.addresses, baseUrl %v", baseURL)

	k8cAPIHealthz, err := url.JoinPath(baseURL, "api", "v1", "healthz")
	if err != nil {
		return fmt.Errorf("failed to construct healthz URL: %w", err)
	}

	l.Infof("Testing /api/v1/healthz endpoint (k8c API), sending request %s", k8cAPIHealthz)
	err = wait.PollImmediateLog(ctx, l, defaultInterval, 2*time.Minute, func(ctx context.Context) (transient error, terminal error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, k8cAPIHealthz, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err), nil
		}

		req.Host = hostname

		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("health check request failed: %w", err), nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code for /api/v1/healthz: got %d, expected %d", resp.StatusCode, http.StatusOK), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("healthz check failed: %w", err)
	}

	l.Infof("Health check endpoint /api/v1/healthz returned 200 OK")

	dexHealthzURL, err := url.JoinPath(baseURL, "dex", "healthz")
	if err != nil {
		return fmt.Errorf("failed to construct dex healthz URL: %w", err)
	}

	l.Info("Testing /dex/healthz endpoint (Dex), sending request to %v", dexHealthzURL)
	err = wait.PollImmediateLog(ctx, l, defaultInterval, 2*time.Minute, func(ctx context.Context) (transient error, terminal error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, dexHealthzURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err), nil
		}

		req.Host = hostname

		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("dex health check request failed: %w", err), nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code for /dex/healthz: got %d, expected %d", resp.StatusCode, http.StatusOK), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("dex healthz check failed: %w", err)
	}

	l.Infof("Dex health check endpoint /dex/healthz returned 200 OK")
	return nil
}
