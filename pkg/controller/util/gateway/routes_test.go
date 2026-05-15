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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestHTTPRouteReadinessForGateway(t *testing.T) {
	gatewayName := types.NamespacedName{Namespace: "networking", Name: "platform-gateway"}
	gatewayNamespace := gatewayapiv1.Namespace(gatewayName.Namespace)

	route := &gatewayapiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "kubermatic",
			Namespace:  "kubermatic",
			Generation: 2,
		},
		Spec: gatewayapiv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayapiv1.CommonRouteSpec{
				ParentRefs: []gatewayapiv1.ParentReference{
					{
						Name:      gatewayapiv1.ObjectName(gatewayName.Name),
						Namespace: &gatewayNamespace,
					},
				},
			},
		},
		Status: gatewayapiv1.HTTPRouteStatus{
			RouteStatus: gatewayapiv1.RouteStatus{
				Parents: []gatewayapiv1.RouteParentStatus{
					{
						ParentRef: gatewayapiv1.ParentReference{
							Name:      gatewayapiv1.ObjectName(gatewayName.Name),
							Namespace: &gatewayNamespace,
						},
						Conditions: []metav1.Condition{
							{
								Type:               string(gatewayapiv1.RouteConditionAccepted),
								Status:             metav1.ConditionTrue,
								ObservedGeneration: 2,
							},
						},
					},
				},
			},
		},
	}

	if !HTTPRouteReferencesGateway(route, gatewayName) {
		t.Fatal("expected HTTPRoute to reference the active Gateway")
	}
	if !HTTPRouteAcceptedByGateway(route, gatewayName) {
		t.Fatal("expected HTTPRoute to be accepted by the active Gateway")
	}

	route.Status.Parents[0].Conditions[0].ObservedGeneration = 1
	if HTTPRouteAcceptedByGateway(route, gatewayName) {
		t.Fatal("expected stale HTTPRoute Accepted condition not to be ready")
	}

	route.Status.Parents[0].Conditions[0].ObservedGeneration = 2
	route.Status.Parents[0].ParentRef.Name = "other-gateway"
	if HTTPRouteAcceptedByGateway(route, gatewayName) {
		t.Fatal("expected HTTPRoute status for another Gateway not to be ready")
	}
}
