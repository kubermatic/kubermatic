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

package gateway

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// ParentReferenceMatchesGateway returns true when parentRef points at gatewayName.
func ParentReferenceMatchesGateway(routeNamespace string, parentRef gatewayapiv1.ParentReference, gatewayName types.NamespacedName) bool {
	if parentRef.Group != nil && string(*parentRef.Group) != gatewayapiv1.GroupName {
		return false
	}
	if parentRef.Kind != nil && string(*parentRef.Kind) != "Gateway" {
		return false
	}
	if string(parentRef.Name) != gatewayName.Name {
		return false
	}

	parentNamespace := routeNamespace
	if parentRef.Namespace != nil {
		parentNamespace = string(*parentRef.Namespace)
	}

	return parentNamespace == gatewayName.Namespace
}

// HTTPRouteReferencesGateway returns true when route references gatewayName.
func HTTPRouteReferencesGateway(route *gatewayapiv1.HTTPRoute, gatewayName types.NamespacedName) bool {
	for _, parentRef := range route.Spec.ParentRefs {
		if ParentReferenceMatchesGateway(route.Namespace, parentRef, gatewayName) {
			return true
		}
	}

	return false
}

// HTTPRouteAcceptedByGateway returns true when route has been accepted by gatewayName
// for its current generation.
func HTTPRouteAcceptedByGateway(route *gatewayapiv1.HTTPRoute, gatewayName types.NamespacedName) bool {
	for _, parentStatus := range route.Status.Parents {
		if !ParentReferenceMatchesGateway(route.Namespace, parentStatus.ParentRef, gatewayName) {
			continue
		}

		accepted := meta.FindStatusCondition(parentStatus.Conditions, string(gatewayapiv1.RouteConditionAccepted))
		if accepted != nil && accepted.Status == metav1.ConditionTrue && accepted.ObservedGeneration >= route.Generation {
			return true
		}
	}

	return false
}
