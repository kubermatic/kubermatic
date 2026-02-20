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

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	// ControllerName is the name of this controller.
	ControllerName = "kkp-httproute-gateway-sync"
)

// Reconciler watches HTTPRoutes and syncs Gateway listeners for cert-manager.
type Reconciler struct {
	ctrlruntimeclient.Client

	log                 *zap.SugaredLogger
	recorder            events.EventRecorder
	namespace           string           // kubermatic namespace where Gateway lives
	watchedNamespaceSet sets.Set[string] // namespaces to watch HTTPRoutes in
}

// Add creates a new HTTPRoute-Gateway sync controller.
func Add(
	ctx context.Context,
	mgr manager.Manager,
	log *zap.SugaredLogger,
	namespace string,
	watchedNamespaces []string,
) error {
	r := &Reconciler{
		Client:              mgr.GetClient(),
		recorder:            mgr.GetEventRecorder(ControllerName),
		log:                 log.Named(ControllerName),
		namespace:           namespace,
		watchedNamespaceSet: sets.New(watchedNamespaces...),
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
		}).
		For(&gatewayapiv1.Gateway{}, builder.WithPredicates(predicate.ByNamespace(namespace))).
		Watches(&gatewayapiv1.HTTPRoute{}, enqueueGatewayForHTTPRoute(r)).
		Build(r)

	return err
}

// enqueueGatewayForHTTPRoute maps HTTPRoute changes to Gateway reconcile requests.
func enqueueGatewayForHTTPRoute(r *Reconciler) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
		httpRoute, ok := obj.(*gatewayapiv1.HTTPRoute)
		if !ok {
			return nil
		}

		// filter by watched namespaces
		if !r.watchedNamespaceSet.Has(httpRoute.Namespace) {
			return nil
		}

		var requests []reconcile.Request
		for _, parentRef := range httpRoute.Spec.ParentRefs {
			if parentRef.Kind != nil && *parentRef.Kind != "Gateway" {
				continue
			}

			// default to reconciler's namespace (kubermatic)
			ns := r.namespace
			if parentRef.Namespace != nil {
				ns = string(*parentRef.Namespace)
			}

			// only reconcile Gateways in target namespace
			if ns != r.namespace {
				continue
			}

			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      string(parentRef.Name),
					Namespace: ns,
				},
			})
		}

		return requests
	})
}
