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

func TestHTTPRouteAcceptedByKubeLBGateway(t *testing.T) {
	gatewayName := types.NamespacedName{Namespace: "kubermatic", Name: "kubermatic"}
	gatewayNamespace := gatewayapiv1.Namespace(gatewayName.Namespace)
	tenantNamespace := gatewayapiv1.Namespace("tenant-abc")

	newRoute := func(statusParentName string) *gatewayapiv1.HTTPRoute {
		return &gatewayapiv1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "kubermatic",
				Namespace:  "kubermatic",
				Generation: 1,
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
								Name:      gatewayapiv1.ObjectName(statusParentName),
								Namespace: &tenantNamespace,
							},
							Conditions: []metav1.Condition{
								{
									Type:               string(gatewayapiv1.RouteConditionAccepted),
									Status:             metav1.ConditionTrue,
									ObservedGeneration: 1,
								},
							},
						},
					},
				},
			},
		}
	}

	if !HTTPRouteAcceptedByGateway(newRoute("kubermatic-kubermatic"), gatewayName) {
		t.Fatal("expected HTTPRoute accepted by the KubeLB copy of the Gateway to be ready")
	}

	if HTTPRouteAcceptedByGateway(newRoute("other-gateway"), gatewayName) {
		t.Fatal("expected HTTPRoute status for another Gateway not to be ready")
	}

	route := newRoute("kubermatic-kubermatic")
	route.Spec.ParentRefs[0].Name = "other-gateway"
	if HTTPRouteAcceptedByGateway(route, gatewayName) {
		t.Fatal("expected KubeLB status not to count when the HTTPRoute does not reference the Gateway")
	}

	route = newRoute("kubermatic-kubermatic")
	route.Status.Parents[0].Conditions[0].Status = metav1.ConditionFalse
	if HTTPRouteAcceptedByGateway(route, gatewayName) {
		t.Fatal("expected HTTPRoute rejected by the KubeLB copy of the Gateway not to be ready")
	}
}

func TestKubeLBGatewayName(t *testing.T) {
	testCases := []struct {
		gateway  types.NamespacedName
		expected string
	}{
		{
			gateway:  types.NamespacedName{Namespace: "kubermatic", Name: "kubermatic"},
			expected: "kubermatic-kubermatic",
		},
		{
			gateway:  types.NamespacedName{Namespace: "a-very-long-gateway-namespace-name", Name: "a-very-long-gateway-object-name"},
			expected: "a-very-long-gateway-namespace-name-a-very-long-gateway-31aa",
		},
	}

	for _, tc := range testCases {
		if got := kubeLBGatewayName(tc.gateway); got != tc.expected {
			t.Errorf("kubeLBGatewayName(%s) = %q, expected %q", tc.gateway, got, tc.expected)
		}
	}
}
