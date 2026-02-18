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

package httproutegatewaysync

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("gateway", request.NamespacedName)
	log.Debug("Reconciling")

	gtw := &gatewayapiv1.Gateway{}

	err := r.Get(ctx, request.NamespacedName, gtw)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get gateway: %w", err)
	}

	err = r.reconcile(ctx, log, gtw)
	if err != nil {
		r.recorder.Eventf(gtw, nil, corev1.EventTypeWarning, "ReconcilingFailed", "Reconciling", err.Error())
		return reconcile.Result{}, fmt.Errorf("failed to reconcile gateway %s: %w", request.Name, err)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcile(ctx context.Context, l *zap.SugaredLogger, gtw *gatewayapiv1.Gateway) error {
	if !r.usesCertManager(gtw) {
		l.Debug("Gateway does not use cert-manager, skipping")
		return nil
	}

	if gtw.DeletionTimestamp != nil {
		// nothing special to do, listeners will be removed with Gateway
		return nil
	}

	// list all HTTPRoutes that reference this Gateway
	httpRoutes, err := r.listHTTPRoutesForGateway(ctx, gtw)
	if err != nil {
		return fmt.Errorf("failed to list HTTPRoutes: %w", err)
	}

	// extract desired listeners from HTTPRoutes
	desiredListeners := r.desiredListeners(l, gtw, httpRoutes)
	if len(desiredListeners) > 64 {
		return fmt.Errorf("listener limit reached: %d listeners (max 64)", len(desiredListeners))
	}

	// patch Gateway if listeners changed
	return r.patchGatewayListeners(ctx, gtw, desiredListeners)
}

// usesCertManager checks if Gateway has cert-manager annotations.
func (r *Reconciler) usesCertManager(gtw *gatewayapiv1.Gateway) bool {
	annotations := gtw.GetAnnotations()
	if annotations == nil {
		return false
	}

	_, hasIssuer := annotations[certmanagerv1.IngressIssuerNameAnnotationKey]
	_, hasClusterIssuer := annotations[certmanagerv1.IngressClusterIssuerNameAnnotationKey]
	return hasIssuer || hasClusterIssuer
}

func (r *Reconciler) listHTTPRoutesForGateway(ctx context.Context, gtw *gatewayapiv1.Gateway) ([]gatewayapiv1.HTTPRoute, error) {
	var routes []gatewayapiv1.HTTPRoute

	for _, ns := range r.watchedNamespaceSet.UnsortedList() {
		routeList := &gatewayapiv1.HTTPRouteList{}
		if err := r.List(ctx, routeList, ctrlruntimeclient.InNamespace(ns)); err != nil {
			return nil, fmt.Errorf("failed to list HTTPRoutes in namespace %s: %w", ns, err)
		}

		for _, route := range routeList.Items {
			if r.referencesGateway(route, gtw) {
				routes = append(routes, route)
			}
		}
	}

	return routes, nil
}

func (r *Reconciler) referencesGateway(route gatewayapiv1.HTTPRoute, gtw *gatewayapiv1.Gateway) bool {
	for _, parentRef := range route.Spec.ParentRefs {
		if parentRef.Kind != nil && *parentRef.Kind != "Gateway" {
			continue
		}
		if string(parentRef.Name) != gtw.Name {
			continue
		}

		ns := route.Namespace
		if parentRef.Namespace != nil {
			ns = string(*parentRef.Namespace)
		}

		if ns == gtw.Namespace {
			return true
		}
	}
	return false
}

func (r *Reconciler) desiredListeners(
	log *zap.SugaredLogger,
	gateway *gatewayapiv1.Gateway,
	httpRoutes []gatewayapiv1.HTTPRoute,
) []gatewayapiv1.Listener {
	// preserve the core HTTP and HTTPs listeners.
	listeners := make([]gatewayapiv1.Listener, 0)
	for _, l := range gateway.Spec.Listeners {
		if l.Protocol == gatewayapiv1.HTTPProtocolType && l.Name == "http" {
			listeners = append(listeners, l)
		} else if l.Protocol == gatewayapiv1.HTTPSProtocolType && l.Name == "https" {
			listeners = append(listeners, l)
		}
	}

	// sort routes for deterministic certificate naming: the first route with a hostname
	// determines the certificate name.
	slices.SortFunc(httpRoutes, func(a, b gatewayapiv1.HTTPRoute) int {
		if a.Namespace != b.Namespace {
			return strings.Compare(a.Namespace, b.Namespace)
		}
		return strings.Compare(a.Name, b.Name)
	})

	// track hostnames and their certificate names
	// this controller acts as a bridge and does not filter hostnames.
	// certificate issuance (including wildcard support) is cert-manager's responsibility.
	hostnameToCertName := make(map[string]string)
	for _, route := range httpRoutes {
		for _, hostname := range route.Spec.Hostnames {
			h := string(hostname)
			if _, exists := hostnameToCertName[h]; !exists {
				// first HTTPRoute with this hostname determines cert name
				hostnameToCertName[h] = fmt.Sprintf("%s-%s", route.Namespace, route.Name)
			}
		}
	}

	for hostname, certName := range hostnameToCertName {
		listener := gatewayapiv1.Listener{
			Name:     gatewayapiv1.SectionName(sanitizeListenerName(hostname)),
			Hostname: ptr.To(gatewayapiv1.Hostname(hostname)),
			Port:     gatewayapiv1.PortNumber(443),
			Protocol: gatewayapiv1.HTTPSProtocolType,
			TLS: &gatewayapiv1.ListenerTLSConfig{
				Mode: ptr.To(gatewayapiv1.TLSModeTerminate),
				CertificateRefs: []gatewayapiv1.SecretObjectReference{
					{
						Name:  gatewayapiv1.ObjectName(certName),
						Group: (*gatewayapiv1.Group)(ptr.To("")),
						Kind:  (*gatewayapiv1.Kind)(ptr.To("Secret")),
					},
				},
			},
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
		}
		listeners = append(listeners, listener)
	}

	return listeners
}

// sanitizeListenerName converts a hostname to a valid listener name.
// Wildcard hostnames (*.example.com) are prefixed with "w-" and the asterisk is removed.
func sanitizeListenerName(hostname string) string {
	name := strings.ToLower(hostname)

	if rest, found := strings.CutPrefix(name, "*."); found {
		name = "w-" + rest
	}

	name = strings.ReplaceAll(name, ".", "-")
	if len(name) > 63 {
		name = name[:63]
	}

	return strings.TrimRight(name, "-")
}

func (r *Reconciler) patchGatewayListeners(
	ctx context.Context,
	gtw *gatewayapiv1.Gateway,
	desiredListeners []gatewayapiv1.Listener,
) error {
	if reflect.DeepEqual(gtw.Spec.Listeners, desiredListeners) {
		return nil
	}

	oldGateway := gtw.DeepCopy()
	gtw.Spec.Listeners = desiredListeners

	return r.Patch(ctx, gtw, ctrlruntimeclient.MergeFrom(oldGateway))
}
