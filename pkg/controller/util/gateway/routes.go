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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	// kubeLBMaxNameLength and kubeLBNameSuffixLength mirror the naming constants
	// KubeLB uses when syncing tenant objects into the management cluster.
	kubeLBMaxNameLength    = 60
	kubeLBNameSuffixLength = 4
)

// ParentReferenceMatchesGateway returns true when parentRef points at gatewayName.
func ParentReferenceMatchesGateway(routeNamespace string, parentRef gatewayapiv1.ParentReference, gatewayName types.NamespacedName) bool {
	if !parentReferenceIsGateway(parentRef) {
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

// ParentStatusMatchesGateway returns true when the parentRef of a route status
// entry belongs to gatewayName. Besides an exact match, this accepts the status
// KubeLB copies back into tenant clusters: KubeLB serves the tenant's Gateway
// through a renamed copy ("<namespace>-<name>") in the tenant namespace on the
// management cluster, so the status parentRef carries that name and namespace
// instead. The renamed form is only accepted when the route's spec references
// gatewayName, so status for unrelated Gateways is never mistaken for it.
func ParentStatusMatchesGateway(route *gatewayapiv1.HTTPRoute, parentRef gatewayapiv1.ParentReference, gatewayName types.NamespacedName) bool {
	if ParentReferenceMatchesGateway(route.Namespace, parentRef, gatewayName) {
		return true
	}

	return parentReferenceIsGateway(parentRef) &&
		string(parentRef.Name) == kubeLBGatewayName(gatewayName) &&
		HTTPRouteReferencesGateway(route, gatewayName)
}

func parentReferenceIsGateway(parentRef gatewayapiv1.ParentReference) bool {
	if parentRef.Group != nil && string(*parentRef.Group) != gatewayapiv1.GroupName {
		return false
	}
	if parentRef.Kind != nil && string(*parentRef.Kind) != "Gateway" {
		return false
	}

	return true
}

// kubeLBGatewayName returns the name KubeLB gives gatewayName on the management
// cluster (see GenerateName in k8c.io/kubelb/internal/kubelb).
func kubeLBGatewayName(gatewayName types.NamespacedName) string {
	name := fmt.Sprintf("%s-%s", gatewayName.Namespace, gatewayName.Name)
	if len(name) < kubeLBMaxNameLength {
		return name
	}

	hash := sha256.Sum256([]byte(name))
	suffix := hex.EncodeToString(hash[:])[:kubeLBNameSuffixLength]
	prefix := strings.TrimRight(name[:kubeLBMaxNameLength-(kubeLBNameSuffixLength+1)], "-")

	return fmt.Sprintf("%s-%s", prefix, suffix)
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
		if !ParentStatusMatchesGateway(route, parentStatus.ParentRef, gatewayName) {
			continue
		}

		accepted := meta.FindStatusCondition(parentStatus.Conditions, string(gatewayapiv1.RouteConditionAccepted))
		if accepted != nil && accepted.Status == metav1.ConditionTrue && accepted.ObservedGeneration >= route.Generation {
			return true
		}
	}

	return false
}
