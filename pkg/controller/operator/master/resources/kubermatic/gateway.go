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

package kubermatic

import (
	"context"
	"fmt"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	gatewayName   = defaulting.DefaultGatewayName
	httpRouteName = defaulting.DefaultHTTPRouteName
)

// GatewayReconciler returns a reconciler for the main Gateway resource.
func GatewayReconciler(cfg *kubermaticv1.KubermaticConfiguration, namespace string) kkpreconciling.NamedGatewayAPIGatewayReconcilerFactory {
	return func() (string, kkpreconciling.GatewayAPIGatewayReconciler) {
		return gatewayName, func(g *gatewayapiv1.Gateway) (*gatewayapiv1.Gateway, error) {
			g.Name = gatewayName
			g.Namespace = namespace

			if g.Labels == nil {
				g.Labels = make(map[string]string)
			}
			g.Labels[common.NameLabel] = "kubermatic"

			if g.Annotations == nil {
				g.Annotations = make(map[string]string)
			}

			gatewayClassName := defaulting.DefaultGatewayClassName
			if cfg.Spec.Ingress.Gateway != nil && cfg.Spec.Ingress.Gateway.ClassName != "" {
				gatewayClassName = cfg.Spec.Ingress.Gateway.ClassName
			}
			g.Spec.GatewayClassName = gatewayapiv1.ObjectName(gatewayClassName)

			// Build the listeners slice. HTTP is always present.
			// HTTPs is only added when a CertificateIssuer is configured.
			listeners := []gatewayapiv1.Listener{
				{
					Name:     "http",
					Protocol: gatewayapiv1.HTTPProtocolType,
					Port:     gatewayapiv1.PortNumber(80),
					AllowedRoutes: &gatewayapiv1.AllowedRoutes{
						Namespaces: &gatewayapiv1.RouteNamespaces{
							From: ptr.To(gatewayapiv1.NamespacesFromSelector),
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									common.GatewayAccessLabelKey: "true",
								},
							},
						},
					},
				},
			}

			issuer := cfg.Spec.Ingress.CertificateIssuer
			if issuer.Name != "" {
				if issuer.Kind != certmanagerv1.IssuerKind && issuer.Kind != certmanagerv1.ClusterIssuerKind {
					return nil, fmt.Errorf("unknown Certificate Issuer Kind %q configured", issuer.Kind)
				}

				delete(g.Annotations, certmanagerv1.IngressIssuerNameAnnotationKey)
				delete(g.Annotations, certmanagerv1.IngressClusterIssuerNameAnnotationKey)

				switch issuer.Kind {
				case certmanagerv1.IssuerKind:
					g.Annotations[certmanagerv1.IngressIssuerNameAnnotationKey] = issuer.Name
				case certmanagerv1.ClusterIssuerKind:
					g.Annotations[certmanagerv1.IngressClusterIssuerNameAnnotationKey] = issuer.Name
				}
				listeners = append(listeners, gatewayapiv1.Listener{
					Name:     "https",
					Protocol: gatewayapiv1.HTTPSProtocolType,
					Port:     gatewayapiv1.PortNumber(443),
					AllowedRoutes: &gatewayapiv1.AllowedRoutes{
						Namespaces: &gatewayapiv1.RouteNamespaces{
							From: ptr.To(gatewayapiv1.NamespacesFromSelector),
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									common.GatewayAccessLabelKey: "true",
								},
							},
						},
					},
					TLS: &gatewayapiv1.GatewayTLSConfig{
						Mode: ptr.To(gatewayapiv1.TLSModeTerminate),
						CertificateRefs: []gatewayapiv1.SecretObjectReference{
							{
								Name: certificateSecretName,
							},
						},
					},
				})
			}

			g.Spec.Listeners = listeners

			return g, nil
		}
	}
}

// HTTPRouteReconciler returns a reconciler for the HTTPRoute resource that routes to KKP services.
func HTTPRouteReconciler(cfg *kubermaticv1.KubermaticConfiguration, namespace string) kkpreconciling.NamedGatewayAPIHTTPRouteReconcilerFactory {
	return func() (string, kkpreconciling.GatewayAPIHTTPRouteReconciler) {
		return httpRouteName, func(r *gatewayapiv1.HTTPRoute) (*gatewayapiv1.HTTPRoute, error) {
			r.Name = httpRouteName
			r.Namespace = namespace

			if r.Labels == nil {
				r.Labels = make(map[string]string)
			}
			r.Labels[common.NameLabel] = "kubermatic"

			if r.Annotations == nil {
				r.Annotations = make(map[string]string)
			}

			parentNs := gatewayapiv1.Namespace(namespace)
			r.Spec.ParentRefs = []gatewayapiv1.ParentReference{
				{
					Name:      gatewayName,
					Namespace: &parentNs,
				},
			}

			r.Spec.Hostnames = []gatewayapiv1.Hostname{
				gatewayapiv1.Hostname(cfg.Spec.Ingress.Domain),
			}

			oneHour := gatewayapiv1.Duration("3600s")

			pathPrefix := gatewayapiv1.PathMatchPathPrefix

			r.Spec.Rules = []gatewayapiv1.HTTPRouteRule{
				{
					Matches: []gatewayapiv1.HTTPRouteMatch{
						{
							Path: &gatewayapiv1.HTTPPathMatch{
								Type:  &pathPrefix,
								Value: ptr.To("/api"),
							},
						},
					},
					BackendRefs: []gatewayapiv1.HTTPBackendRef{
						{
							BackendRef: gatewayapiv1.BackendRef{
								BackendObjectReference: gatewayapiv1.BackendObjectReference{
									Name: APIDeploymentName,
									Port: ptr.To(gatewayapiv1.PortNumber(80)),
								},
							},
						},
					},
					Timeouts: &gatewayapiv1.HTTPRouteTimeouts{
						// nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
						// nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
						Request:        &oneHour,
						BackendRequest: &oneHour,
					},
				},
				{
					Matches: []gatewayapiv1.HTTPRouteMatch{
						{
							Path: &gatewayapiv1.HTTPPathMatch{
								Type:  &pathPrefix,
								Value: ptr.To("/"),
							},
						},
					},
					BackendRefs: []gatewayapiv1.HTTPBackendRef{
						{
							BackendRef: gatewayapiv1.BackendRef{
								BackendObjectReference: gatewayapiv1.BackendObjectReference{
									Name: UIDeploymentName,
									Port: ptr.To(gatewayapiv1.PortNumber(80)),
								},
							},
						},
					},
					Timeouts: &gatewayapiv1.HTTPRouteTimeouts{
						// nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
						// nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
						Request:        &oneHour,
						BackendRequest: &oneHour,
					},
				},
			}

			return r, nil
		}
	}
}

type gatewayComparable struct {
	Spec        gatewayapiv1.GatewaySpec
	Labels      map[string]string
	Annotations map[string]string
}

func comparableGateway(gw *gatewayapiv1.Gateway) gatewayComparable {
	return gatewayComparable{
		Spec:        gw.Spec,
		Labels:      gw.Labels,
		Annotations: gw.Annotations,
	}
}

type httpRouteComparable struct {
	Spec        gatewayapiv1.HTTPRouteSpec
	Labels      map[string]string
	Annotations map[string]string
}

func comparableHTTPRoute(hr *gatewayapiv1.HTTPRoute) httpRouteComparable {
	return httpRouteComparable{
		Spec:        hr.Spec,
		Labels:      hr.Labels,
		Annotations: hr.Annotations,
	}
}

// EnsureGateway creates or updates the Gateway. Uses direct client operations instead of the standard reconciling
// helpers to avoid cache-wait timeouts. Envoy Gateway continuously updates the Gateway Status, which would cause
// the cache-wait logic to time out.
func EnsureGateway(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	log *zap.SugaredLogger,
	cfg *kubermaticv1.KubermaticConfiguration,
	namespace string,
) error {
	factory := GatewayReconciler(cfg, namespace)
	gatewayName, reconciler := factory()

	desired := &gatewayapiv1.Gateway{}
	if _, err := reconciler(desired); err != nil {
		return fmt.Errorf("failed to build desired Gateway: %w", err)
	}

	key := types.NamespacedName{Namespace: namespace, Name: gatewayName}

	var existing gatewayapiv1.Gateway
	if err := client.Get(ctx, key, &existing); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debugw("Creating Gateway", "name", gatewayName, "namespace", namespace)
			return client.Create(ctx, desired)
		}

		return fmt.Errorf("failed to get Gateway %s/%s: %w", namespace, gatewayName, err)
	}

	// compare only Spec/Labels/Annotations (ignore Status to avoid update loops)
	if equality.Semantic.DeepEqual(comparableGateway(&existing), comparableGateway(desired)) {
		log.Debugw("Gateway unchanged, skipping update", "name", gatewayName)

		return nil
	}

	updated := existing.DeepCopy()
	updated.Spec = desired.Spec
	updated.Labels = desired.Labels
	if updated.Annotations == nil {
		updated.Annotations = make(map[string]string)
	}

	for k, v := range desired.Annotations {
		updated.Annotations[k] = v
	}

	log.Debugw("Updating Gateway", "name", gatewayName)
	return client.Update(ctx, updated)
}

// EnsureHTTPRoute creates or updates the HTTPRoute. Uses direct client operations instead of the standard reconciling
// helpers to avoid cache-wait timeouts.Envoy Gateway continuously updates the HTTPRoute Status, which would cause
// the cache-wait logic to time out.
func EnsureHTTPRoute(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	log *zap.SugaredLogger,
	cfg *kubermaticv1.KubermaticConfiguration,
	namespace string,
) error {
	factory := HTTPRouteReconciler(cfg, namespace)
	routeName, reconciler := factory()

	desired := &gatewayapiv1.HTTPRoute{}
	if _, err := reconciler(desired); err != nil {
		return fmt.Errorf("failed to build desired HTTPRoute: %w", err)
	}

	key := types.NamespacedName{Namespace: namespace, Name: routeName}

	var existing gatewayapiv1.HTTPRoute
	if err := client.Get(ctx, key, &existing); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debugw("Creating HTTPRoute", "name", routeName, "namespace", namespace)

			return client.Create(ctx, desired)
		}

		return fmt.Errorf("failed to get HTTPRoute %s/%s: %w", namespace, routeName, err)
	}

	// compare only Spec/Labels/Annotations (ignore Status to avoid update loops)
	if equality.Semantic.DeepEqual(comparableHTTPRoute(&existing), comparableHTTPRoute(desired)) {
		log.Debugw("HTTPRoute unchanged, skipping update", "name", routeName)

		return nil
	}

	updated := existing.DeepCopy()
	updated.Spec = desired.Spec
	updated.Labels = desired.Labels
	if updated.Annotations == nil {
		updated.Annotations = make(map[string]string)
	}
	for k, v := range desired.Annotations {
		updated.Annotations[k] = v
	}

	log.Debugw("Updating HTTPRoute", "name", routeName)
	return client.Update(ctx, updated)
}
