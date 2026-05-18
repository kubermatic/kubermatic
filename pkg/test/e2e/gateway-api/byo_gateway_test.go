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
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	operatorcommon "k8c.io/kubermatic/v2/pkg/controller/operator/common"
	gatewayutil "k8c.io/kubermatic/v2/pkg/controller/util/gateway"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	externalGatewayNamespace = "byo-gateway-e2e"
	externalGatewayName      = "platform-gateway"
	externalGatewayClassName = "kubermatic-envoy-gateway-byo-e2e"
	externalEnvoyProxyName   = "byo-gateway-e2e-proxy"
	externalHTTPNodePort     = "30081"
	externalHTTPSNodePort    = int64(30444)
)

func TestGatewayAPIExternalGatewayMigration(t *testing.T) {
	ctx := t.Context()
	rawLogger := log.NewFromOptions(logOptions)
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))
	logger := rawLogger.Sugar()

	seedClient, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("Failed to build client: %v", err)
	}

	cfg, err := getGatewayTestKubermaticConfiguration(ctx, seedClient)
	if err != nil {
		t.Fatalf("Failed to get KubermaticConfiguration: %v", err)
	}
	originalIngress := cfg.Spec.Ingress

	dexRouteKey := types.NamespacedName{Namespace: "dex", Name: "dex"}
	dexRoute := &gatewayapiv1.HTTPRoute{}
	if err := seedClient.Get(ctx, dexRouteKey, dexRoute); err != nil {
		t.Fatalf("Failed to get Dex HTTPRoute: %v", err)
	}
	originalDexRouteSpec := dexRoute.Spec

	externalGatewayKey := types.NamespacedName{Namespace: externalGatewayNamespace, Name: externalGatewayName}
	managedGatewayKey := types.NamespacedName{Namespace: jig.KubermaticNamespace(), Name: defaulting.DefaultGatewayName}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()

		if err := updateKubermaticConfiguration(cleanupCtx, logger, seedClient, func(cfg *kubermaticv1.KubermaticConfiguration) {
			cfg.Spec.Ingress = originalIngress
		}); err != nil {
			t.Errorf("Failed to restore KubermaticConfiguration ingress settings: %v", err)
		}

		if err := updateHTTPRoute(cleanupCtx, logger, seedClient, dexRouteKey, func(route *gatewayapiv1.HTTPRoute) {
			route.Spec = originalDexRouteSpec
		}); err != nil {
			t.Errorf("Failed to restore Dex HTTPRoute parentRefs: %v", err)
		}

		if err := verifyGatewayAPIModeResources(cleanupCtx, t, seedClient, logger); err != nil {
			t.Errorf("Failed to restore managed Gateway API resources: %v", err)
		}

		if err := deleteIfExists(cleanupCtx, seedClient, externalGatewayObject(externalGatewayKey)); err != nil {
			t.Errorf("Failed to delete external Gateway: %v", err)
		}
		if err := deleteIfExists(cleanupCtx, seedClient, externalGatewayClassObject()); err != nil {
			t.Errorf("Failed to delete external GatewayClass: %v", err)
		}
		if err := deleteIfExists(cleanupCtx, seedClient, externalEnvoyProxyObject()); err != nil {
			t.Errorf("Failed to delete external EnvoyProxy: %v", err)
		}
		if err := deleteIfExists(cleanupCtx, seedClient, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: externalGatewayNamespace}}); err != nil {
			t.Errorf("Failed to delete external Gateway namespace: %v", err)
		}
	})

	logger.Info("Creating externally managed Gateway data plane for BYO Gateway e2e test...")
	if err := ensureExternalGatewayStack(ctx, seedClient); err != nil {
		t.Fatalf("Failed to create external Gateway stack: %v", err)
	}

	if err := waitForGatewayClassAccepted(ctx, logger, seedClient, externalGatewayClassName); err != nil {
		t.Fatalf("External GatewayClass was not accepted: %v", err)
	}
	if err := waitForGatewayProgrammed(ctx, logger, seedClient, externalGatewayKey); err != nil {
		t.Fatalf("External Gateway was not programmed: %v", err)
	}

	logger.Info("Switching KubermaticConfiguration to external Gateway mode...")
	if err := updateKubermaticConfiguration(ctx, logger, seedClient, func(cfg *kubermaticv1.KubermaticConfiguration) {
		cfg.Spec.Ingress.CertificateIssuer = corev1.TypedLocalObjectReference{}
		if cfg.Spec.Ingress.Gateway == nil {
			cfg.Spec.Ingress.Gateway = &kubermaticv1.KubermaticGatewayConfiguration{}
		}
		cfg.Spec.Ingress.Gateway.ExternalGateway = &kubermaticv1.KubermaticExternalGatewayReference{
			Name:      externalGatewayKey.Name,
			Namespace: externalGatewayKey.Namespace,
		}
		cfg.Spec.Ingress.Gateway.ClassName = ""
		cfg.Spec.Ingress.Gateway.InfrastructureAnnotations = nil
		cfg.Spec.Ingress.Gateway.TLS = nil
	}); err != nil {
		t.Fatalf("Failed to configure external Gateway: %v", err)
	}

	kkpRouteKey := types.NamespacedName{Namespace: jig.KubermaticNamespace(), Name: defaulting.DefaultHTTPRouteName}
	if err := waitForHTTPRouteAcceptedByGateway(ctx, logger, seedClient, kkpRouteKey, externalGatewayKey); err != nil {
		t.Fatalf("KKP HTTPRoute was not accepted by external Gateway: %v", err)
	}

	logger.Info("Moving Helm-managed Dex HTTPRoute to the external Gateway to complete the migration...")
	if err := updateHTTPRoute(ctx, logger, seedClient, dexRouteKey, func(route *gatewayapiv1.HTTPRoute) {
		route.Spec.ParentRefs = []gatewayapiv1.ParentReference{parentRefForGateway(externalGatewayKey)}
	}); err != nil {
		t.Fatalf("Failed to move Dex HTTPRoute to external Gateway: %v", err)
	}
	if err := waitForHTTPRouteAcceptedByGateway(ctx, logger, seedClient, dexRouteKey, externalGatewayKey); err != nil {
		t.Fatalf("Dex HTTPRoute was not accepted by external Gateway: %v", err)
	}

	if err := waitForGatewayDeleted(ctx, logger, seedClient, managedGatewayKey); err != nil {
		t.Fatalf("Managed Gateway was not deleted after all routes moved to external Gateway: %v", err)
	}

	logger.Info("Verifying HTTP connectivity through the external Gateway...")
	if err := verifyGatewayHTTPConnectivityThroughGateway(ctx, t, seedClient, logger, externalGatewayKey, externalHTTPNodePort); err != nil {
		t.Fatalf("External Gateway HTTP connectivity verification failed: %v", err)
	}
}

func getGatewayTestKubermaticConfiguration(ctx context.Context, c ctrlruntimeclient.Client) (*kubermaticv1.KubermaticConfiguration, error) {
	cfg := &kubermaticv1.KubermaticConfiguration{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: jig.KubermaticNamespace(), Name: kubermaticConfigurationName}, cfg); err != nil {
		return nil, fmt.Errorf("failed to get KubermaticConfiguration: %w", err)
	}
	return cfg, nil
}

func updateKubermaticConfiguration(ctx context.Context, logger *zap.SugaredLogger, c ctrlruntimeclient.Client, mutate func(*kubermaticv1.KubermaticConfiguration)) error {
	return wait.PollImmediateLog(ctx, logger, time.Second, time.Minute, func(ctx context.Context) (transient error, terminal error) {
		cfg, err := getGatewayTestKubermaticConfiguration(ctx, c)
		if err != nil {
			return err, nil
		}

		mutate(cfg)

		if err := c.Update(ctx, cfg); err != nil {
			if apierrors.IsConflict(err) {
				return fmt.Errorf("failed to update KubermaticConfiguration due to conflict: %w", err), nil
			}
			return nil, fmt.Errorf("failed to update KubermaticConfiguration: %w", err)
		}

		return nil, nil
	})
}

func updateHTTPRoute(ctx context.Context, logger *zap.SugaredLogger, c ctrlruntimeclient.Client, key types.NamespacedName, mutate func(*gatewayapiv1.HTTPRoute)) error {
	return wait.PollImmediateLog(ctx, logger, time.Second, time.Minute, func(ctx context.Context) (transient error, terminal error) {
		route := &gatewayapiv1.HTTPRoute{}
		if err := c.Get(ctx, key, route); err != nil {
			return err, nil
		}

		mutate(route)

		if err := c.Update(ctx, route); err != nil {
			if apierrors.IsConflict(err) {
				return fmt.Errorf("failed to update HTTPRoute %q due to conflict: %w", key.String(), err), nil
			}
			return nil, fmt.Errorf("failed to update HTTPRoute %q: %w", key.String(), err)
		}

		return nil, nil
	})
}

func ensureExternalGatewayStack(ctx context.Context, c ctrlruntimeclient.Client) error {
	if err := ensureNamespace(ctx, c, externalGatewayNamespace); err != nil {
		return err
	}
	if err := ensureUnstructured(ctx, c, externalEnvoyProxyObject()); err != nil {
		return err
	}
	if err := ensureGatewayClass(ctx, c, externalGatewayClassObject()); err != nil {
		return err
	}
	return ensureGateway(ctx, c, externalGatewayObject(types.NamespacedName{Namespace: externalGatewayNamespace, Name: externalGatewayName}))
}

func ensureNamespace(ctx context.Context, c ctrlruntimeclient.Client, name string) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	if err := c.Create(ctx, namespace); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create namespace %q: %w", name, err)
	}
	return nil
}

func ensureGatewayClass(ctx context.Context, c ctrlruntimeclient.Client, desired *gatewayapiv1.GatewayClass) error {
	existing := &gatewayapiv1.GatewayClass{}
	key := types.NamespacedName{Name: desired.Name}
	if err := c.Get(ctx, key, existing); err != nil {
		if apierrors.IsNotFound(err) {
			if err := c.Create(ctx, desired); err != nil {
				return fmt.Errorf("failed to create GatewayClass %q: %w", desired.Name, err)
			}
			return nil
		}
		return fmt.Errorf("failed to get GatewayClass %q: %w", desired.Name, err)
	}

	existing.Spec = desired.Spec
	if err := c.Update(ctx, existing); err != nil {
		return fmt.Errorf("failed to update GatewayClass %q: %w", desired.Name, err)
	}
	return nil
}

func ensureGateway(ctx context.Context, c ctrlruntimeclient.Client, desired *gatewayapiv1.Gateway) error {
	existing := &gatewayapiv1.Gateway{}
	key := types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}
	if err := c.Get(ctx, key, existing); err != nil {
		if apierrors.IsNotFound(err) {
			if err := c.Create(ctx, desired); err != nil {
				return fmt.Errorf("failed to create Gateway %q: %w", key.String(), err)
			}
			return nil
		}
		return fmt.Errorf("failed to get Gateway %q: %w", key.String(), err)
	}

	existing.Spec = desired.Spec
	if err := c.Update(ctx, existing); err != nil {
		return fmt.Errorf("failed to update Gateway %q: %w", key.String(), err)
	}
	return nil
}

func ensureUnstructured(ctx context.Context, c ctrlruntimeclient.Client, desired *unstructured.Unstructured) error {
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(desired.GroupVersionKind())

	key := types.NamespacedName{Namespace: desired.GetNamespace(), Name: desired.GetName()}
	if err := c.Get(ctx, key, existing); err != nil {
		if apierrors.IsNotFound(err) {
			if err := c.Create(ctx, desired); err != nil {
				return fmt.Errorf("failed to create %s %q: %w", desired.GetKind(), key.String(), err)
			}
			return nil
		}
		return fmt.Errorf("failed to get %s %q: %w", desired.GetKind(), key.String(), err)
	}

	existing.Object["spec"] = desired.Object["spec"]
	if err := c.Update(ctx, existing); err != nil {
		return fmt.Errorf("failed to update %s %q: %w", desired.GetKind(), key.String(), err)
	}
	return nil
}

func externalGatewayClassObject() *gatewayapiv1.GatewayClass {
	proxyNamespace := gatewayapiv1.Namespace(externalGatewayNamespace)
	return &gatewayapiv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{Name: externalGatewayClassName},
		Spec: gatewayapiv1.GatewayClassSpec{
			ControllerName: gatewayapiv1.GatewayController("gateway.k8c.io/envoy-gateway"),
			ParametersRef: &gatewayapiv1.ParametersReference{
				Group:     gatewayapiv1.Group("gateway.envoyproxy.io"),
				Kind:      gatewayapiv1.Kind("EnvoyProxy"),
				Name:      externalEnvoyProxyName,
				Namespace: &proxyNamespace,
			},
		},
	}
}

func externalGatewayObject(key types.NamespacedName) *gatewayapiv1.Gateway {
	return &gatewayapiv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: gatewayapiv1.GatewaySpec{
			GatewayClassName: gatewayapiv1.ObjectName(externalGatewayClassName),
			Listeners: []gatewayapiv1.Listener{
				{
					Name:     "http",
					Protocol: gatewayapiv1.HTTPProtocolType,
					Port:     gatewayapiv1.PortNumber(80),
					AllowedRoutes: &gatewayapiv1.AllowedRoutes{
						Namespaces: &gatewayapiv1.RouteNamespaces{
							From: ptr.To(gatewayapiv1.NamespacesFromSelector),
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									operatorcommon.GatewayAccessLabelKey: operatorcommon.GatewayAccessLabelValue,
								},
							},
						},
					},
				},
			},
		},
	}
}

func externalEnvoyProxyObject() *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      externalEnvoyProxyName,
				"namespace": externalGatewayNamespace,
			},
			"spec": map[string]interface{}{
				"provider": map[string]interface{}{
					"type": "Kubernetes",
					"kubernetes": map[string]interface{}{
						"envoyDeployment": map[string]interface{}{
							"replicas": int64(1),
							"container": map[string]interface{}{
								"image": "envoyproxy/envoy:distroless-v1.36.3",
							},
						},
						"envoyService": map[string]interface{}{
							"type":                  "NodePort",
							"externalTrafficPolicy": "Cluster",
							"patch": map[string]interface{}{
								"type": "JSONMerge",
								"value": map[string]interface{}{
									"spec": map[string]interface{}{
										"type": "NodePort",
										"ports": []interface{}{
											map[string]interface{}{
												"name":       "http",
												"port":       int64(80),
												"nodePort":   int64(30081),
												"targetPort": int64(10080),
											},
											map[string]interface{}{
												"name":       "https",
												"port":       int64(443),
												"nodePort":   externalHTTPSNodePort,
												"targetPort": int64(10443),
											},
										},
									},
								},
							},
						},
					},
				},
				"ipFamily": "IPv4",
			},
		},
	}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.envoyproxy.io",
		Version: "v1alpha1",
		Kind:    "EnvoyProxy",
	})
	return obj
}

func waitForGatewayClassAccepted(ctx context.Context, logger *zap.SugaredLogger, c ctrlruntimeclient.Client, name string) error {
	return wait.PollImmediateLog(ctx, logger, defaultInterval, defaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		gatewayClass := &gatewayapiv1.GatewayClass{}
		if err := c.Get(ctx, types.NamespacedName{Name: name}, gatewayClass); err != nil {
			return fmt.Errorf("GatewayClass %q not found: %w", name, err), nil
		}

		if meta.IsStatusConditionTrue(gatewayClass.Status.Conditions, string(gatewayapiv1.GatewayClassConditionStatusAccepted)) {
			return nil, nil
		}

		return fmt.Errorf("GatewayClass %q is not accepted yet, status: %+v", name, gatewayClass.Status), nil
	})
}

func waitForGatewayProgrammed(ctx context.Context, logger *zap.SugaredLogger, c ctrlruntimeclient.Client, key types.NamespacedName) error {
	return wait.PollImmediateLog(ctx, logger, defaultInterval, defaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		gateway := &gatewayapiv1.Gateway{}
		if err := c.Get(ctx, key, gateway); err != nil {
			return fmt.Errorf("Gateway %q not found: %w", key.String(), err), nil
		}

		programmed := meta.FindStatusCondition(gateway.Status.Conditions, string(gatewayapiv1.GatewayConditionProgrammed))
		if programmed != nil && programmed.Status == metav1.ConditionTrue && programmed.ObservedGeneration >= gateway.Generation {
			return nil, nil
		}

		return fmt.Errorf("Gateway %q is not programmed yet, status: %+v", key.String(), gateway.Status), nil
	})
}

func waitForHTTPRouteAcceptedByGateway(ctx context.Context, logger *zap.SugaredLogger, c ctrlruntimeclient.Client, routeKey, gatewayKey types.NamespacedName) error {
	return wait.PollImmediateLog(ctx, logger, defaultInterval, defaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		route := &gatewayapiv1.HTTPRoute{}
		if err := c.Get(ctx, routeKey, route); err != nil {
			return fmt.Errorf("HTTPRoute %q not found: %w", routeKey.String(), err), nil
		}

		if !gatewayutil.HTTPRouteReferencesGateway(route, gatewayKey) {
			return fmt.Errorf("HTTPRoute %q does not reference Gateway %q, parentRefs: %+v", routeKey.String(), gatewayKey.String(), route.Spec.ParentRefs), nil
		}

		if gatewayutil.HTTPRouteAcceptedByGateway(route, gatewayKey) {
			return nil, nil
		}

		return fmt.Errorf("HTTPRoute %q is not accepted by Gateway %q yet, status: %+v", routeKey.String(), gatewayKey.String(), route.Status), nil
	})
}

func waitForGatewayDeleted(ctx context.Context, logger *zap.SugaredLogger, c ctrlruntimeclient.Client, key types.NamespacedName) error {
	return wait.PollImmediateLog(ctx, logger, defaultInterval, defaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		gateway := &gatewayapiv1.Gateway{}
		err := c.Get(ctx, key, gateway)
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get Gateway %q: %w", key.String(), err)
		}
		return fmt.Errorf("Gateway %q still exists", key.String()), nil
	})
}

func parentRefForGateway(key types.NamespacedName) gatewayapiv1.ParentReference {
	namespace := gatewayapiv1.Namespace(key.Namespace)
	return gatewayapiv1.ParentReference{
		Name:      gatewayapiv1.ObjectName(key.Name),
		Namespace: &namespace,
	}
}

func deleteIfExists(ctx context.Context, c ctrlruntimeclient.Client, obj ctrlruntimeclient.Object) error {
	err := c.Delete(ctx, obj)
	if err == nil || apierrors.IsNotFound(err) {
		return nil
	}
	return err
}
