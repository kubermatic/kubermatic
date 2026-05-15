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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	gatewayutil "k8c.io/kubermatic/v2/pkg/controller/util/gateway"
	"k8c.io/kubermatic/v2/pkg/defaulting"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	// maxListenerNameLength is the maximum length for a DNS-1035 label.
	maxListenerNameLength = 63
	// listenerNamePrefixLen is the prefix length before hash suffix.
	listenerNamePrefixLen = maxListenerNameLength - 1 - 8 // 54 = 63 - 1 (dash) - 8 (hash)
	// listenerNameHashLen is the length of the hash suffix.
	listenerNameHashLen = 8
	// maxGatewayListeners is the Gateway API maximum number of listeners per Gateway.
	maxGatewayListeners = 64
)

type listenerTLSMode int

const (
	listenerTLSModeDisabled listenerTLSMode = iota
	listenerTLSModeCertManager
	listenerTLSModeStaticCertificateRefs
)

type listenerSyncConfig struct {
	tlsMode         listenerTLSMode
	certificateRefs []gatewayapiv1.SecretObjectReference
}

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
	if !r.managesGateway(gtw) {
		l.Debug("Gateway is not the operator-managed default Gateway, skipping")
		return nil
	}

	syncConfig := r.listenerSyncConfig(gtw)
	if syncConfig.tlsMode == listenerTLSModeDisabled && !r.hasNonCoreListeners(gtw) {
		l.Debug("Gateway has no TLS listener sync configuration, skipping")
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
	desiredListeners := r.desiredListeners(l, gtw, httpRoutes, syncConfig)
	if len(desiredListeners) > maxGatewayListeners {
		return fmt.Errorf("listener limit reached: %d listeners (max %d)", len(desiredListeners), maxGatewayListeners)
	}

	// patch Gateway if listeners changed
	return r.patchGatewayListeners(ctx, gtw, desiredListeners)
}

func (r *Reconciler) managesGateway(gtw *gatewayapiv1.Gateway) bool {
	if gtw.Name != defaulting.DefaultGatewayName || gtw.Namespace != r.namespace {
		return false
	}

	return common.HasAnyKubermaticConfigurationControllerOwnerReference(gtw.OwnerReferences)
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

func (r *Reconciler) listenerSyncConfig(gtw *gatewayapiv1.Gateway) listenerSyncConfig {
	if r.usesCertManager(gtw) {
		return listenerSyncConfig{tlsMode: listenerTLSModeCertManager}
	}

	httpsListener := gatewayutil.CoreListener(gtw.Spec.Listeners, gatewayutil.CoreListenerHTTPS)
	if httpsListener == nil || httpsListener.TLS == nil || len(httpsListener.TLS.CertificateRefs) == 0 {
		return listenerSyncConfig{tlsMode: listenerTLSModeDisabled}
	}

	return listenerSyncConfig{
		tlsMode:         listenerTLSModeStaticCertificateRefs,
		certificateRefs: slices.Clone(httpsListener.TLS.CertificateRefs),
	}
}

func (r *Reconciler) hasNonCoreListeners(gtw *gatewayapiv1.Gateway) bool {
	for _, listener := range gtw.Spec.Listeners {
		if _, isCore := gatewayutil.CoreListenerNames[listener.Name]; !isCore {
			return true
		}
	}

	return false
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
	syncConfig listenerSyncConfig,
) []gatewayapiv1.Listener {
	if syncConfig.tlsMode == listenerTLSModeDisabled {
		return slices.Clone(gateway.Spec.Listeners)
	}

	// preserve the core HTTP and HTTPS listeners.
	listeners := make([]gatewayapiv1.Listener, 0)
	for _, l := range gateway.Spec.Listeners {
		if _, isCore := gatewayutil.CoreListenerNames[l.Name]; isCore {
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
			if h == "" {
				log.Debug(
					"Skipping empty hostname in HTTPRoute %s/%s - not valid for cert-manager TLS",
					route.Namespace, route.Name,
				)
				continue
			}

			if _, exists := hostnameToCertName[h]; !exists {
				// first HTTPRoute with this hostname determines cert name
				hostnameToCertName[h] = fmt.Sprintf("%s-%s", route.Namespace, route.Name)
			}
		}
	}

	hostnames := slices.Collect(maps.Keys(hostnameToCertName))
	slices.Sort(hostnames)

	for _, hostname := range hostnames {
		certName := hostnameToCertName[hostname]
		certificateRefs := slices.Clone(syncConfig.certificateRefs)
		if syncConfig.tlsMode == listenerTLSModeCertManager {
			certificateRefs = []gatewayapiv1.SecretObjectReference{
				{
					Name:  gatewayapiv1.ObjectName(certName),
					Group: (*gatewayapiv1.Group)(ptr.To("")),
					Kind:  (*gatewayapiv1.Kind)(ptr.To("Secret")),
				},
			}
		}

		listener := gatewayapiv1.Listener{
			Name:     gatewayapiv1.SectionName(sanitizeListenerName(hostname)),
			Hostname: ptr.To(gatewayapiv1.Hostname(hostname)),
			Port:     gatewayapiv1.PortNumber(443),
			Protocol: gatewayapiv1.HTTPSProtocolType,
			TLS: &gatewayapiv1.ListenerTLSConfig{
				Mode:            ptr.To(gatewayapiv1.TLSModeTerminate),
				CertificateRefs: certificateRefs,
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

	gatewayutil.SortListenersByName(listeners)
	return listeners
}

// sanitizeListenerName converts a hostname to a valid DNS-1035 label for a listener name.
// Wildcard hostnames (*.example.com) are prefixed with "w-" and the asterisk is removed.
// For names exceeding 63 characters, a hash suffix ensures uniqueness.
func sanitizeListenerName(hostname string) string {
	name := strings.ToLower(hostname)

	if rest, found := strings.CutPrefix(name, "*."); found {
		name = "w-" + rest
	}

	name = strings.ReplaceAll(name, ".", "-")

	// if name fits within limit, return as-is after trimming trailing dashes
	if len(name) <= maxListenerNameLength {
		return strings.TrimRight(name, "-")
	}

	// for long names, use hash suffix to ensure uniqueness
	// format: <prefix>-<hash> = maxListenerNameLength chars total
	hashBytes := sha256.Sum256([]byte(hostname))
	hash := hex.EncodeToString(hashBytes[:])[:listenerNameHashLen]

	if len(name) > listenerNamePrefixLen {
		name = name[:listenerNamePrefixLen]
	}
	name = strings.TrimRight(name, "-")

	// ensure name doesn't start with dash (invalid DNS label)
	if name == "" || name[0] == '-' {
		name = "x"
	}

	return name + "-" + hash
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
