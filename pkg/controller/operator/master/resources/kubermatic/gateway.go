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
	"errors"
	"fmt"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	gatewayutil "k8c.io/kubermatic/v2/pkg/controller/util/gateway"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling/modifier"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	gatewayName   = defaulting.DefaultGatewayName
	httpRouteName = defaulting.DefaultHTTPRouteName
)

// GatewayReconciler returns a reconciler for the main Gateway resource.
// existingListeners contains the current listeners from the Gateway (empty for new Gateways).
// Non-core listeners (not http/https) from existingListeners are preserved.
func GatewayReconciler(
	cfg *kubermaticv1.KubermaticConfiguration,
	namespace string,
	existingListeners []gatewayapiv1.Listener,
) kkpreconciling.NamedGatewayAPIGatewayReconcilerFactory {
	return func() (string, kkpreconciling.GatewayAPIGatewayReconciler) {
		return gatewayName, func(g *gatewayapiv1.Gateway) (*gatewayapiv1.Gateway, error) {
			g.Name = gatewayName
			g.Namespace = namespace

			if g.Labels == nil {
				g.Labels = make(map[string]string)
			}
			g.Labels[common.NameLabel] = defaulting.DefaultGatewayName

			if g.Annotations == nil {
				g.Annotations = make(map[string]string)
			}

			gatewayClassName := defaulting.DefaultGatewayClassName
			gatewayConfig := cfg.Spec.Ingress.Gateway
			if gatewayConfig != nil {
				if gatewayConfig.ClassName != "" {
					gatewayClassName = gatewayConfig.ClassName
				}
			}
			g.Spec.Infrastructure = reconcileGatewayInfrastructure(g.Spec.Infrastructure, gatewayConfig)
			g.Spec.GatewayClassName = gatewayapiv1.ObjectName(gatewayClassName)

			// Build core listeners. HTTP is always present.
			// HTTPS is added when either a CertificateIssuer or a Gateway TLS Secret is configured.
			coreListeners := []gatewayapiv1.Listener{
				{
					Name:     "http",
					Protocol: gatewayapiv1.HTTPProtocolType,
					Port:     gatewayapiv1.PortNumber(80),
					AllowedRoutes: &gatewayapiv1.AllowedRoutes{
						Namespaces: &gatewayapiv1.RouteNamespaces{
							From: ptr.To(gatewayapiv1.NamespacesFromSelector),
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									common.GatewayAccessLabelKey: common.GatewayAccessLabelValue,
								},
							},
						},
					},
				},
			}

			issuer := cfg.Spec.Ingress.CertificateIssuer
			var tlsSecretRef *gatewayapiv1.SecretObjectReference
			if cfg.Spec.Ingress.Gateway != nil && cfg.Spec.Ingress.Gateway.TLS != nil &&
				cfg.Spec.Ingress.Gateway.TLS.SecretRef != nil &&
				cfg.Spec.Ingress.Gateway.TLS.SecretRef.Name != "" {
				tlsSecretRef = &gatewayapiv1.SecretObjectReference{
					Name: gatewayapiv1.ObjectName(cfg.Spec.Ingress.Gateway.TLS.SecretRef.Name),
				}
				if cfg.Spec.Ingress.Gateway.TLS.SecretRef.Namespace != "" {
					tlsNamespace := gatewayapiv1.Namespace(cfg.Spec.Ingress.Gateway.TLS.SecretRef.Namespace)
					tlsSecretRef.Namespace = &tlsNamespace
				}
			}

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
				if tlsSecretRef == nil {
					tlsSecretRef = &gatewayapiv1.SecretObjectReference{
						Name: certificateSecretName,
					}
				}
			}

			if tlsSecretRef != nil {
				coreListeners = append(coreListeners, gatewayapiv1.Listener{
					Name:     "https",
					Hostname: ptr.To(gatewayapiv1.Hostname(cfg.Spec.Ingress.Domain)),
					Protocol: gatewayapiv1.HTTPSProtocolType,
					Port:     gatewayapiv1.PortNumber(443),
					AllowedRoutes: &gatewayapiv1.AllowedRoutes{
						Namespaces: &gatewayapiv1.RouteNamespaces{
							From: ptr.To(gatewayapiv1.NamespacesFromSelector),
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									common.GatewayAccessLabelKey: common.GatewayAccessLabelValue,
								},
							},
						},
					},
					TLS: &gatewayapiv1.ListenerTLSConfig{
						Mode: ptr.To(gatewayapiv1.TLSModeTerminate),
						CertificateRefs: []gatewayapiv1.SecretObjectReference{
							*tlsSecretRef,
						},
					},
				})
			}

			// Merge core listeners with preserved non-core from existing (sorted)
			g.Spec.Listeners = gatewayutil.MergeListeners(coreListeners, existingListeners)

			return g, nil
		}
	}
}

// reconcileGatewayInfrastructure keeps unmanaged infrastructure fields from the
// existing Gateway, while making KubermaticConfiguration the source of truth for
// infrastructure annotations.
func reconcileGatewayInfrastructure(
	existing *gatewayapiv1.GatewayInfrastructure,
	gatewayConfig *kubermaticv1.KubermaticGatewayConfiguration,
) *gatewayapiv1.GatewayInfrastructure {
	var infra *gatewayapiv1.GatewayInfrastructure
	if existing != nil {
		infra = existing.DeepCopy()
		infra.Annotations = nil
	}

	if gatewayConfig != nil && len(gatewayConfig.InfrastructureAnnotations) > 0 {
		if infra == nil {
			infra = &gatewayapiv1.GatewayInfrastructure{}
		}

		infra.Annotations = make(map[gatewayapiv1.AnnotationKey]gatewayapiv1.AnnotationValue, len(gatewayConfig.InfrastructureAnnotations))
		for key, value := range gatewayConfig.InfrastructureAnnotations {
			infra.Annotations[gatewayapiv1.AnnotationKey(key)] = gatewayapiv1.AnnotationValue(value)
		}
	}

	if infra != nil && len(infra.Labels) == 0 && len(infra.Annotations) == 0 && infra.ParametersRef == nil {
		return nil
	}

	return infra
}

// HTTPRouteReconciler returns a reconciler for the HTTPRoute resource that routes to KKP services.
func HTTPRouteReconciler(cfg *kubermaticv1.KubermaticConfiguration, namespace string) kkpreconciling.NamedGatewayAPIHTTPRouteReconcilerFactory {
	return HTTPRouteReconcilerWithParentRefs(cfg, namespace, []gatewayapiv1.ParentReference{gatewayParentReference(cfg, namespace)})
}

// HTTPRouteReconcilerWithParentRefs returns a reconciler for the HTTPRoute resource with explicit parent references.
func HTTPRouteReconcilerWithParentRefs(cfg *kubermaticv1.KubermaticConfiguration, namespace string, parentRefs []gatewayapiv1.ParentReference) kkpreconciling.NamedGatewayAPIHTTPRouteReconcilerFactory {
	return func() (string, kkpreconciling.GatewayAPIHTTPRouteReconciler) {
		return httpRouteName, func(r *gatewayapiv1.HTTPRoute) (*gatewayapiv1.HTTPRoute, error) {
			r.Name = httpRouteName
			r.Namespace = namespace

			if r.Labels == nil {
				r.Labels = make(map[string]string)
			}
			r.Labels[common.NameLabel] = common.GatewayName

			if r.Annotations == nil {
				r.Annotations = make(map[string]string)
			}

			r.Spec.ParentRefs = parentRefs

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

func gatewayParentReference(cfg *kubermaticv1.KubermaticConfiguration, namespace string) gatewayapiv1.ParentReference {
	parentName := gatewayapiv1.ObjectName(gatewayName)
	parentNamespace := gatewayapiv1.Namespace(namespace)

	if cfg != nil {
		gatewayConfig := cfg.Spec.Ingress.Gateway
		if gatewayConfig != nil && gatewayConfig.UsesExternalGateway() {
			parentName = gatewayapiv1.ObjectName(gatewayConfig.ExternalGateway.Name)
			parentNamespace = gatewayapiv1.Namespace(gatewayConfig.ExternalGatewayNamespace(namespace))
		}
	}

	return gatewayapiv1.ParentReference{
		Name:      parentName,
		Namespace: &parentNamespace,
	}
}

// DefaultGatewayParentReference returns a parentRef to the operator-managed default Gateway.
func DefaultGatewayParentReference(namespace string) gatewayapiv1.ParentReference {
	parentNamespace := gatewayapiv1.Namespace(namespace)
	return gatewayapiv1.ParentReference{
		Name:      gatewayName,
		Namespace: &parentNamespace,
	}
}

// appendParentReferenceIfMissing deduplicates whole-Gateway parentRefs; SectionName
// and Port are intentionally ignored because the operator only owns Gateway-wide refs.
func appendParentReferenceIfMissing(routeNamespace string, parentRefs []gatewayapiv1.ParentReference, parentRef gatewayapiv1.ParentReference) []gatewayapiv1.ParentReference {
	gatewayKey := types.NamespacedName{Name: string(parentRef.Name), Namespace: routeNamespace}
	if parentRef.Namespace != nil {
		gatewayKey.Namespace = string(*parentRef.Namespace)
	}

	for _, existing := range parentRefs {
		if gatewayutil.ParentReferenceMatchesGateway(routeNamespace, existing, gatewayKey) {
			return parentRefs
		}
	}

	return append(parentRefs, parentRef)
}

// ExternalGatewayKey returns the configured external Gateway key and whether
// an external Gateway reference is configured.
func ExternalGatewayKey(cfg *kubermaticv1.KubermaticConfiguration, namespace string) (types.NamespacedName, bool) {
	if cfg == nil || !cfg.Spec.Ingress.Gateway.UsesExternalGateway() {
		return types.NamespacedName{}, false
	}

	gatewayConfig := cfg.Spec.Ingress.Gateway
	return types.NamespacedName{
		Name:      gatewayConfig.ExternalGateway.Name,
		Namespace: gatewayConfig.ExternalGatewayNamespace(namespace),
	}, true
}

// HTTPRouteAcceptedByExternalGateway returns true when the configured external
// Gateway is ready and has accepted the managed HTTPRoute for its current generation.
func HTTPRouteAcceptedByExternalGateway(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	cfg *kubermaticv1.KubermaticConfiguration,
	namespace string,
) (bool, error) {
	var route gatewayapiv1.HTTPRoute
	key := types.NamespacedName{Namespace: namespace, Name: httpRouteName}
	if err := client.Get(ctx, key, &route); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get HTTPRoute %q: %w", key.String(), err)
	}

	gatewayKey, ok := ExternalGatewayKey(cfg, namespace)
	if !ok {
		return false, nil
	}

	if !gatewayutil.HTTPRouteAcceptedByGateway(&route, gatewayKey) {
		return false, nil
	}

	var gateway gatewayapiv1.Gateway
	if err := client.Get(ctx, gatewayKey, &gateway); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get external Gateway %q: %w", gatewayKey.String(), err)
	}

	if gateway.DeletionTimestamp != nil {
		return false, nil
	}

	return gatewayProgrammedForCurrentGeneration(&gateway), nil
}

func gatewayProgrammedForCurrentGeneration(gateway *gatewayapiv1.Gateway) bool {
	programmed := meta.FindStatusCondition(gateway.Status.Conditions, string(gatewayapiv1.GatewayConditionProgrammed))
	return programmed != nil &&
		programmed.Status == metav1.ConditionTrue &&
		programmed.ObservedGeneration >= gateway.Generation
}

// HTTPRoutesReferencingManagedGateway returns all HTTPRoutes that still point at
// the operator-managed default Gateway. This relies on a cluster-scoped cache:
// migration blockers can live outside the operator namespace, for example Dex
// and IAP HTTPRoutes. The operator-managed KKP HTTPRoute is excluded because
// EnsureHTTPRoute rewrites its parentRefs in the same reconcile cycle and a
// stale controller cache could otherwise count it as its own blocker.
func HTTPRoutesReferencingManagedGateway(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	namespace string,
) ([]types.NamespacedName, error) {
	var routeList gatewayapiv1.HTTPRouteList
	if err := client.List(ctx, &routeList); err != nil {
		return nil, fmt.Errorf("failed to list HTTPRoutes: %w", err)
	}

	gatewayKey := types.NamespacedName{Namespace: namespace, Name: gatewayName}
	references := []types.NamespacedName{}
	for i := range routeList.Items {
		route := &routeList.Items[i]
		if route.Namespace == namespace && route.Name == httpRouteName {
			continue
		}
		if gatewayutil.HTTPRouteReferencesGateway(route, gatewayKey) {
			references = append(references, types.NamespacedName{Namespace: route.Namespace, Name: route.Name})
		}
	}

	return references, nil
}

// gatewayComparable holds the fields used to detect meaningful changes between
// existing and desired Gateway state.
type gatewayComparable struct {
	Spec            gatewayapiv1.GatewaySpec
	Labels          map[string]string
	Annotations     map[string]string
	OwnerReferences []metav1.OwnerReference
}

func comparableGateway(gw *gatewayapiv1.Gateway) gatewayComparable {
	return gatewayComparable{
		Spec:            gw.Spec,
		Labels:          gw.Labels,
		Annotations:     gw.Annotations,
		OwnerReferences: gw.OwnerReferences,
	}
}

// httpRouteComparable holds the fields used to detect meaningful changes between
// existing and desired HTTPRoute state.
type httpRouteComparable struct {
	Spec            gatewayapiv1.HTTPRouteSpec
	Labels          map[string]string
	Annotations     map[string]string
	OwnerReferences []metav1.OwnerReference
}

func comparableHTTPRoute(hr *gatewayapiv1.HTTPRoute) httpRouteComparable {
	return httpRouteComparable{
		Spec:            hr.Spec,
		Labels:          hr.Labels,
		Annotations:     hr.Annotations,
		OwnerReferences: hr.OwnerReferences,
	}
}

// setControllerReference sets a controller owner reference on the object,
// ignoring AlreadyOwnedError when the object already has a different
// controller owner. Matches the pattern in modifier.Ownership().
func setControllerReference(owner metav1.Object, controlled ctrlruntimeclient.Object, scheme *runtime.Scheme) error {
	if err := controllerutil.SetControllerReference(owner, controlled, scheme); err != nil {
		var alreadyOwned *controllerutil.AlreadyOwnedError
		if !errors.As(err, &alreadyOwned) {
			return fmt.Errorf("failed to set owner reference: %w", err)
		}
	}
	return nil
}

// IsExternalGatewayNotOperatorOwned rejects externalGateway references that point at an operator-managed Gateway.
// Unowned Gateways, including a Gateway named kubermatic/kubermatic, are left to the user and are valid BYO targets.
// The returned bool reports whether the referenced external Gateway currently exists.
func IsExternalGatewayNotOperatorOwned(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	cfg *kubermaticv1.KubermaticConfiguration,
	namespace string,
) (bool, error) {
	key, ok := ExternalGatewayKey(cfg, namespace)
	if !ok {
		return false, nil
	}

	var existing gatewayapiv1.Gateway
	if err := client.Get(ctx, key, &existing); err != nil {
		if apierrors.IsNotFound(err) {
			// The Gateway can be created independently/asynchronously by the platform team;
			// the installer performs readiness checks when it needs the Gateway to exist.
			return false, nil
		}
		return false, fmt.Errorf("failed to get external Gateway %q: %w", key.String(), err)
	}

	// A Gateway with a non-nil DeletionTimestamp will never serve new routes, so
	// treat it as missing. The reconciler then records the "missing external
	// Gateway" event and retries until the deletion finishes and the user (or
	// platform team) recreates the Gateway.
	if existing.DeletionTimestamp != nil {
		return false, nil
	}

	// Reject any KubermaticConfiguration controller ownership, current or stale.
	if common.HasAnyKubermaticConfigurationControllerOwnerReference(existing.OwnerReferences) {
		return true, fmt.Errorf("external Gateway %q is operator-managed and cannot be used as spec.ingress.gateway.externalGateway; remove KubermaticConfiguration controller ownerReferences before reusing it as an external Gateway", key.String())
	}

	return true, nil
}

// ManagedGatewayExists returns true when the operator-managed default Gateway exists.
func ManagedGatewayExists(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	namespace string,
) (bool, error) {
	key := types.NamespacedName{Namespace: namespace, Name: gatewayName}

	var existing gatewayapiv1.Gateway
	if err := client.Get(ctx, key, &existing); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get Gateway %q: %w", key.String(), err)
	}

	return common.HasAnyKubermaticConfigurationControllerOwnerReference(existing.OwnerReferences), nil
}

// EnsureManagedGatewayAbsent deletes the operator-owned default Gateway when BYO Gateway is configured.
// It intentionally leaves unowned or externally owned Gateways untouched.
func EnsureManagedGatewayAbsent(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	log *zap.SugaredLogger,
	namespace string,
) error {
	key := types.NamespacedName{Namespace: namespace, Name: gatewayName}

	var existing gatewayapiv1.Gateway
	if err := client.Get(ctx, key, &existing); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get Gateway %q: %w", key.String(), err)
	}

	if !common.HasAnyKubermaticConfigurationControllerOwnerReference(existing.OwnerReferences) {
		log.Debugw("Leaving non-operator-owned Gateway untouched", "name", gatewayName, "namespace", namespace)
		return nil
	}

	log.Debugw("Deleting operator-managed Gateway because externalGateway is configured", "name", gatewayName, "namespace", namespace)
	return client.Delete(ctx, &existing)
}

// EnsureGateway creates or updates the Gateway. It still uses GatewayReconciler
// to build desired state, but applies it with direct client operations instead
// of the standard reconciling helpers to avoid cache-wait timeouts. Envoy
// Gateway continuously updates the Gateway Status, which would cause the
// cache-wait logic to time out.
func EnsureGateway(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	log *zap.SugaredLogger,
	cfg *kubermaticv1.KubermaticConfiguration,
	namespace string,
	scheme *runtime.Scheme,
) error {
	key := types.NamespacedName{Namespace: namespace, Name: gatewayName}

	var existing gatewayapiv1.Gateway
	existingListeners := []gatewayapiv1.Listener{}

	err := client.Get(ctx, key, &existing)
	if err == nil {
		existingListeners = existing.Spec.Listeners
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to get Gateway %q: %w", key.String(), err)
	}

	_, reconciler := GatewayReconciler(cfg, namespace, existingListeners)()

	desired := &gatewayapiv1.Gateway{}
	// Carry over existing infrastructure so the reconciler can preserve unmanaged
	// fields while reconciling annotation ownership from config.
	if err == nil && existing.Spec.Infrastructure != nil {
		desired.Spec.Infrastructure = existing.Spec.Infrastructure.DeepCopy()
	}
	if _, err := reconciler(desired); err != nil {
		return fmt.Errorf("failed to build desired Gateway: %w", err)
	}

	if err := setControllerReference(cfg, desired, scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on Gateway: %w", err)
	}
	kubernetes.EnsureLabels(desired, map[string]string{
		modifier.ManagedByLabel: common.OperatorName,
	})

	if apierrors.IsNotFound(err) {
		log.Debugw("Creating Gateway", "name", gatewayName, "namespace", namespace)
		return client.Create(ctx, desired)
	}

	updated := existing.DeepCopy()
	updated.Spec = desired.Spec
	kubernetes.EnsureLabels(updated, desired.Labels)

	// Remove stale cert-manager ownership markers before merging desired annotations.
	// EnsureAnnotations preserves unrelated keys, but it does not delete managed keys
	// that are no longer part of the desired state (for example after switching to
	// manual Gateway TLS).
	annotations := updated.GetAnnotations()
	delete(annotations, certmanagerv1.IngressIssuerNameAnnotationKey)
	delete(annotations, certmanagerv1.IngressClusterIssuerNameAnnotationKey)
	updated.SetAnnotations(annotations)
	kubernetes.EnsureAnnotations(updated, desired.Annotations)

	if err := setControllerReference(cfg, updated, scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on Gateway: %w", err)
	}

	// compare only Spec/Labels/Annotations (ignore Status to avoid update loops)
	if equality.Semantic.DeepEqual(comparableGateway(&existing), comparableGateway(updated)) {
		log.Debugw("Gateway unchanged, skipping update", "name", gatewayName)

		return nil
	}

	log.Debugw("Updating Gateway", "name", gatewayName)
	return client.Update(ctx, updated)
}

// EnsureHTTPRoute creates or updates the HTTPRoute. EnsureHTTPRouteWithAdditionalParentRefs
// still uses HTTPRouteReconcilerWithParentRefs to build desired state, but
// applies it with direct client operations instead of the standard reconciling
// helpers to avoid cache-wait timeouts. Envoy Gateway continuously updates the
// HTTPRoute Status, which would cause the cache-wait logic to time out.
func EnsureHTTPRoute(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	log *zap.SugaredLogger,
	cfg *kubermaticv1.KubermaticConfiguration,
	namespace string,
	scheme *runtime.Scheme,
) error {
	return EnsureHTTPRouteWithAdditionalParentRefs(ctx, client, log, cfg, namespace, scheme, nil)
}

// EnsureHTTPRouteWithAdditionalParentRefs creates or updates the HTTPRoute while preserving additional parentRefs.
func EnsureHTTPRouteWithAdditionalParentRefs(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	log *zap.SugaredLogger,
	cfg *kubermaticv1.KubermaticConfiguration,
	namespace string,
	scheme *runtime.Scheme,
	additionalParentRefs []gatewayapiv1.ParentReference,
) error {
	parentRefs := []gatewayapiv1.ParentReference{gatewayParentReference(cfg, namespace)}
	for _, parentRef := range additionalParentRefs {
		parentRefs = appendParentReferenceIfMissing(namespace, parentRefs, parentRef)
	}

	factory := HTTPRouteReconcilerWithParentRefs(cfg, namespace, parentRefs)
	routeName, reconciler := factory()

	desired := &gatewayapiv1.HTTPRoute{}
	if _, err := reconciler(desired); err != nil {
		return fmt.Errorf("failed to build desired HTTPRoute: %w", err)
	}

	if err := setControllerReference(cfg, desired, scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on HTTPRoute: %w", err)
	}
	kubernetes.EnsureLabels(desired, map[string]string{
		modifier.ManagedByLabel: common.OperatorName,
	})

	key := types.NamespacedName{Namespace: namespace, Name: routeName}

	var existing gatewayapiv1.HTTPRoute
	if err := client.Get(ctx, key, &existing); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debugw("Creating HTTPRoute", "name", routeName, "namespace", namespace)

			return client.Create(ctx, desired)
		}

		return fmt.Errorf("failed to get HTTPRoute %s/%s: %w", namespace, routeName, err)
	}

	// Build the merged state: desired labels/annotations on top of existing ones.
	// This preserves user-added metadata (e.g. external-dns annotations) while
	// ensuring operator-managed fields are up to date.
	updated := existing.DeepCopy()
	updated.Spec = desired.Spec
	kubernetes.EnsureLabels(updated, desired.Labels)
	kubernetes.EnsureAnnotations(updated, desired.Annotations)

	if err := setControllerReference(cfg, updated, scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on HTTPRoute: %w", err)
	}

	// compare only Spec/Labels/Annotations (ignore Status to avoid update loops)
	if equality.Semantic.DeepEqual(comparableHTTPRoute(&existing), comparableHTTPRoute(updated)) {
		log.Debugw("HTTPRoute unchanged, skipping update", "name", routeName)

		return nil
	}

	log.Debugw("Updating HTTPRoute", "name", routeName)
	return client.Update(ctx, updated)
}
