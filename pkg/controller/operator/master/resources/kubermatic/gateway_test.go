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
	"testing"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestGatewayReconciler(t *testing.T) {
	testCases := []struct {
		name     string
		config   *kubermaticv1.KubermaticConfiguration
		validate func(t *testing.T, gw *gatewayapiv1.Gateway)
	}{
		{
			name: "Gateway created with HTTP listener only when no certificate issuer",
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Ingress: kubermaticv1.KubermaticIngressConfiguration{
						Domain: "example.com",
					},
				},
			},
			validate: func(t *testing.T, gw *gatewayapiv1.Gateway) {
				if gw.Name != gatewayName {
					t.Errorf("Expected Gateway name %q, got %q", gatewayName, gw.Name)
				}
				if len(gw.Spec.Listeners) != 1 {
					t.Fatalf("Expected 1 listener (HTTP only), got %d", len(gw.Spec.Listeners))
				}
				listener := gw.Spec.Listeners[0]
				if listener.Name != "http" {
					t.Errorf("Expected listener name 'http', got %q", listener.Name)
				}
				if listener.Protocol != gatewayapiv1.HTTPProtocolType {
					t.Errorf("Expected HTTP protocol, got %v", listener.Protocol)
				}
				if listener.Port != gatewayapiv1.PortNumber(80) {
					t.Errorf("Expected port 80, got %d", listener.Port)
				}
			},
		},
		{
			name: "Gateway created with HTTP and HTTPS listeners when Issuer configured",
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Ingress: kubermaticv1.KubermaticIngressConfiguration{
						Domain: "example.com",
						CertificateIssuer: corev1.TypedLocalObjectReference{
							Name: "letsencrypt-prod",
							Kind: certmanagerv1.ClusterIssuerKind,
						},
					},
				},
			},
			validate: func(t *testing.T, gw *gatewayapiv1.Gateway) {
				if len(gw.Spec.Listeners) != 2 {
					t.Fatalf("Expected 2 listeners (HTTP and HTTPS), got %d", len(gw.Spec.Listeners))
				}

				// Check HTTPS listener
				var httpsListener *gatewayapiv1.Listener
				for i := range gw.Spec.Listeners {
					if gw.Spec.Listeners[i].Name == "https" {
						httpsListener = &gw.Spec.Listeners[i]
						break
					}
				}
				if httpsListener == nil {
					t.Fatal("HTTPS listener not found")
				}
				if httpsListener.Protocol != gatewayapiv1.HTTPSProtocolType {
					t.Errorf("Expected HTTPS protocol, got %v", httpsListener.Protocol)
				}
				if httpsListener.TLS == nil {
					t.Fatal("Expected TLS config for HTTPS listener")
				}
				if len(httpsListener.TLS.CertificateRefs) != 1 {
					t.Errorf("Expected 1 certificate ref, got %d", len(httpsListener.TLS.CertificateRefs))
				}
				if httpsListener.TLS.CertificateRefs[0].Name != certificateSecretName {
					t.Errorf("Expected certificate ref name %q, got %q",
						certificateSecretName, httpsListener.TLS.CertificateRefs[0].Name)
				}

				// Check cert-manager annotation
				expectedAnnotation := "cert-manager.io/cluster-issuer"
				if gw.Annotations[expectedAnnotation] != "letsencrypt-prod" {
					t.Errorf("Expected annotation %q=%q, got %q",
						expectedAnnotation, "letsencrypt-prod", gw.Annotations[expectedAnnotation])
				}
			},
		},
		{
			name: "Gateway with Issuer (not ClusterIssuer) sets correct annotation",
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Ingress: kubermaticv1.KubermaticIngressConfiguration{
						Domain: "example.com",
						CertificateIssuer: corev1.TypedLocalObjectReference{
							Name: "my-issuer",
							Kind: certmanagerv1.IssuerKind,
						},
					},
				},
			},
			validate: func(t *testing.T, gw *gatewayapiv1.Gateway) {
				expectedAnnotation := "cert-manager.io/issuer"
				if gw.Annotations[expectedAnnotation] != "my-issuer" {
					t.Errorf("Expected annotation %q=%q, got %q",
						expectedAnnotation, "my-issuer", gw.Annotations[expectedAnnotation])
				}
				// Ensure ClusterIssuer annotation is NOT present
				unexpectedAnnotation := "cert-manager.io/cluster-issuer"
				if _, exists := gw.Annotations[unexpectedAnnotation]; exists {
					t.Errorf("Annotation %q should not exist when using Issuer kind", unexpectedAnnotation)
				}
			},
		},
		{
			name: "Gateway has correct labels and GatewayClassName",
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Ingress: kubermaticv1.KubermaticIngressConfiguration{
						Domain: "example.com",
						Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
							ClassName: defaulting.DefaultGatewayClassName,
						},
					},
				},
			},
			validate: func(t *testing.T, gw *gatewayapiv1.Gateway) {
				expectedLabel := "kubermatic"
				if gw.Labels["app.kubernetes.io/name"] != expectedLabel {
					t.Errorf("Expected label app.kubernetes.io/name=%q, got %q",
						expectedLabel, gw.Labels["app.kubernetes.io/name"])
				}
				expectedClassName := gatewayapiv1.ObjectName(defaulting.DefaultGatewayClassName)
				if gw.Spec.GatewayClassName != expectedClassName {
					t.Errorf("Expected GatewayClassName %q, got %q",
						expectedClassName, gw.Spec.GatewayClassName)
				}
			},
		},
		{
			name: "Gateway AllowedRoutes has correct namespace selector",
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Ingress: kubermaticv1.KubermaticIngressConfiguration{
						Domain: "example.com",
					},
				},
			},
			validate: func(t *testing.T, gw *gatewayapiv1.Gateway) {
				listener := gw.Spec.Listeners[0]
				if listener.AllowedRoutes == nil {
					t.Fatal("Expected AllowedRoutes to be set")
				}
				if listener.AllowedRoutes.Namespaces == nil {
					t.Fatal("Expected AllowedRoutes.Namespaces to be set")
				}
				if *listener.AllowedRoutes.Namespaces.From != gatewayapiv1.NamespacesFromSelector {
					t.Errorf("Expected NamespacesFromSelector, got %v", *listener.AllowedRoutes.Namespaces.From)
				}
				if listener.AllowedRoutes.Namespaces.Selector == nil {
					t.Fatal("Expected namespace selector to be set")
				}
				expectedLabel := common.GatewayAccessLabelKey
				if listener.AllowedRoutes.Namespaces.Selector.MatchLabels[expectedLabel] != "true" {
					t.Errorf("Expected label selector %q=true, got %q",
						expectedLabel, listener.AllowedRoutes.Namespaces.Selector.MatchLabels[expectedLabel])
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			creatorGetter := GatewayReconciler(tc.config, "kubermatic")
			_, creator := creatorGetter()

			reconciled, err := creator(&gatewayapiv1.Gateway{})
			if err != nil {
				t.Fatalf("GatewayReconciler failed: %v", err)
			}

			if tc.validate != nil {
				tc.validate(t, reconciled)
			}
		})
	}
}

func TestGatewayReconcilerKeepsAnnotations(t *testing.T) {
	// Test that custom annotations are preserved (similar to IngressReconcilerKeepsAnnotations)
	cfg := &kubermaticv1.KubermaticConfiguration{
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "example.com",
			},
		},
	}
	creatorGetter := GatewayReconciler(cfg, "kubermatic")
	_, creator := creatorGetter()

	testCases := []struct {
		name string
		gw   *gatewayapiv1.Gateway
	}{
		{
			name: "do not fail on nil map",
			gw:   &gatewayapiv1.Gateway{},
		},
		{
			name: "keep existing annotations",
			gw: &gatewayapiv1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"custom-annotation": "custom-value",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			existingAnnotations := map[string]string{}
			if tc.gw.Annotations != nil {
				for k, v := range tc.gw.Annotations {
					existingAnnotations[k] = v
				}
			}

			reconciled, err := creator(tc.gw)
			if err != nil {
				t.Fatalf("GatewayReconciler failed: %v", err)
			}

			for k, v := range existingAnnotations {
				if reconciledValue := reconciled.Annotations[k]; reconciledValue != v {
					t.Errorf("Expected annotation %q with value %q, but got %q.", k, v, reconciledValue)
				}
			}
		})
	}
}

func TestHTTPRouteReconciler(t *testing.T) {
	testCases := []struct {
		name     string
		config   *kubermaticv1.KubermaticConfiguration
		validate func(t *testing.T, route *gatewayapiv1.HTTPRoute)
	}{
		{
			name: "HTTPRoute has correct parent reference to Gateway",
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Ingress: kubermaticv1.KubermaticIngressConfiguration{
						Domain: "example.com",
					},
				},
			},
			validate: func(t *testing.T, route *gatewayapiv1.HTTPRoute) {
				if route.Name != httpRouteName {
					t.Errorf("Expected HTTPRoute name %q, got %q", httpRouteName, route.Name)
				}
				if len(route.Spec.ParentRefs) != 1 {
					t.Fatalf("Expected 1 parent reference, got %d", len(route.Spec.ParentRefs))
				}
				parentRef := route.Spec.ParentRefs[0]
				if parentRef.Name != gatewayName {
					t.Errorf("Expected parent ref name %q, got %q", gatewayName, parentRef.Name)
				}
				if parentRef.Namespace == nil || *parentRef.Namespace != "kubermatic" {
					t.Errorf("Expected parent ref namespace 'kubermatic', got %v", parentRef.Namespace)
				}
			},
		},
		{
			name: "HTTPRoute has correct hostname",
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Ingress: kubermaticv1.KubermaticIngressConfiguration{
						Domain: "example.com",
					},
				},
			},
			validate: func(t *testing.T, route *gatewayapiv1.HTTPRoute) {
				if len(route.Spec.Hostnames) != 1 {
					t.Fatalf("Expected 1 hostname, got %d", len(route.Spec.Hostnames))
				}
				if string(route.Spec.Hostnames[0]) != "example.com" {
					t.Errorf("Expected hostname 'example.com', got %q", route.Spec.Hostnames[0])
				}
			},
		},
		{
			name: "HTTPRoute has two rules for /api and /",
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Ingress: kubermaticv1.KubermaticIngressConfiguration{
						Domain: "example.com",
					},
				},
			},
			validate: func(t *testing.T, route *gatewayapiv1.HTTPRoute) {
				if len(route.Spec.Rules) != 2 {
					t.Fatalf("Expected 2 rules, got %d", len(route.Spec.Rules))
				}
			},
		},
		{
			name: "HTTPRoute /api rule targets API service with correct backend",
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Ingress: kubermaticv1.KubermaticIngressConfiguration{
						Domain: "example.com",
					},
				},
			},
			validate: func(t *testing.T, route *gatewayapiv1.HTTPRoute) {
				apiRule := route.Spec.Rules[0]
				if len(apiRule.Matches) != 1 {
					t.Fatalf("Expected 1 match for /api rule, got %d", len(apiRule.Matches))
				}
				if apiRule.Matches[0].Path.Value == nil || *apiRule.Matches[0].Path.Value != "/api" {
					t.Errorf("Expected path match '/api', got %v", apiRule.Matches[0].Path.Value)
				}
				if len(apiRule.BackendRefs) != 1 {
					t.Fatalf("Expected 1 backend ref, got %d", len(apiRule.BackendRefs))
				}
				if apiRule.BackendRefs[0].Name != APIDeploymentName {
					t.Errorf("Expected backend ref name %q, got %q", APIDeploymentName, apiRule.BackendRefs[0].Name)
				}
				if apiRule.BackendRefs[0].Port == nil || *apiRule.BackendRefs[0].Port != 80 {
					t.Errorf("Expected backend port 80, got %v", apiRule.BackendRefs[0].Port)
				}
			},
		},
		{
			name: "HTTPRoute / rule targets UI service with correct backend",
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Ingress: kubermaticv1.KubermaticIngressConfiguration{
						Domain: "example.com",
					},
				},
			},
			validate: func(t *testing.T, route *gatewayapiv1.HTTPRoute) {
				uiRule := route.Spec.Rules[1]
				if len(uiRule.BackendRefs) != 1 {
					t.Fatalf("Expected 1 backend ref, got %d", len(uiRule.BackendRefs))
				}
				if uiRule.BackendRefs[0].Name != UIDeploymentName {
					t.Errorf("Expected backend ref name %q, got %q", UIDeploymentName, uiRule.BackendRefs[0].Name)
				}
			},
		},
		{
			name: "HTTPRoute has 3600s timeout on both rules (matches nginx proxy-timeout)",
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Ingress: kubermaticv1.KubermaticIngressConfiguration{
						Domain: "example.com",
					},
				},
			},
			validate: func(t *testing.T, route *gatewayapiv1.HTTPRoute) {
				expectedTimeout := gatewayapiv1.Duration("3600s")
				for i, rule := range route.Spec.Rules {
					if rule.Timeouts == nil {
						t.Errorf("Rule %d: Expected Timeouts to be set", i)
						continue
					}
					if rule.Timeouts.Request == nil || *rule.Timeouts.Request != expectedTimeout {
						t.Errorf("Rule %d: Expected Request timeout 3600s, got %v", i, rule.Timeouts.Request)
					}
					if rule.Timeouts.BackendRequest == nil || *rule.Timeouts.BackendRequest != expectedTimeout {
						t.Errorf("Rule %d: Expected BackendRequest timeout 3600s, got %v", i, rule.Timeouts.BackendRequest)
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			creatorGetter := HTTPRouteReconciler(tc.config, "kubermatic")
			_, creator := creatorGetter()

			reconciled, err := creator(&gatewayapiv1.HTTPRoute{})
			if err != nil {
				t.Fatalf("HTTPRouteReconciler failed: %v", err)
			}

			if tc.validate != nil {
				tc.validate(t, reconciled)
			}
		})
	}
}

func TestEnsureGatewayCreatesNew(t *testing.T) {
	ctx := context.Background()
	cfg := &kubermaticv1.KubermaticConfiguration{
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "kubermatic.example.com",
			},
		},
	}
	namespace := "kubermatic"

	client := fake.NewClientBuilder().WithScheme(fake.NewScheme()).Build()

	err := EnsureGateway(ctx, client, zap.NewNop().Sugar(), cfg, namespace)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var created gatewayapiv1.Gateway
	err = client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: gatewayName}, &created)
	if err != nil {
		t.Fatalf("Gateway should exist after EnsureGateway: %v", err)
	}

	if created.Name != gatewayName {
		t.Errorf("expected name %s, got %s", gatewayName, created.Name)
	}

	if created.Namespace != namespace {
		t.Errorf("expected namespace %s, got %s", namespace, created.Namespace)
	}

	if len(created.Spec.Listeners) != 1 {
		t.Fatalf("expected 1 listener (HTTP only, no issuer), got %d", len(created.Spec.Listeners))
	}

	httpListener := created.Spec.Listeners[0]
	if httpListener.Port != gatewayapiv1.PortNumber(80) {
		t.Errorf("expected port 80, got %d", httpListener.Port)
	}

	if httpListener.Protocol != gatewayapiv1.HTTPProtocolType {
		t.Errorf("expected protocol HTTP, got %s", httpListener.Protocol)
	}
}

func TestEnsureGatewayUpdatesExisting(t *testing.T) {
	ctx := context.Background()
	namespace := "kubermatic"

	existing := &gatewayapiv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gatewayName,
			Namespace: namespace,
			Labels: map[string]string{
				"old-label": "old-value",
			},
		},
		Spec: gatewayapiv1.GatewaySpec{
			GatewayClassName: "old-class-name",
			Listeners: []gatewayapiv1.Listener{
				{
					Name:     "http",
					Protocol: gatewayapiv1.HTTPProtocolType,
					Port:     gatewayapiv1.PortNumber(80),
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(fake.NewScheme()).WithObjects(existing).Build()

	cfg := &kubermaticv1.KubermaticConfiguration{
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "kubermatic.example.com",
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ClassName: defaulting.DefaultGatewayClassName,
				},
			},
		},
	}

	err := EnsureGateway(ctx, client, zap.NewNop().Sugar(), cfg, namespace)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var updated gatewayapiv1.Gateway
	err = client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: gatewayName}, &updated)
	if err != nil {
		t.Fatalf("Gateway should exist: %v", err)
	}

	if updated.Spec.GatewayClassName != gatewayapiv1.ObjectName(defaulting.DefaultGatewayClassName) {
		t.Errorf("expected GatewayClassName %s, got %s", defaulting.DefaultGatewayClassName, updated.Spec.GatewayClassName)
	}
}

func TestEnsureGatewaySkipsWhenUnchanged(t *testing.T) {
	ctx := context.Background()
	namespace := "kubermatic"

	cfg := &kubermaticv1.KubermaticConfiguration{
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "kubermatic.example.com",
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ClassName: defaulting.DefaultGatewayClassName,
				},
			},
		},
	}
	factory := GatewayReconciler(cfg, namespace)
	_, reconciler := factory()

	desired := &gatewayapiv1.Gateway{}
	if _, err := reconciler(desired); err != nil {
		t.Fatalf("failed to build desired Gateway: %v", err)
	}

	existing := desired.DeepCopy()
	existing.Status = gatewayapiv1.GatewayStatus{
		Conditions: []metav1.Condition{
			{
				Type:               "Accepted",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             "Accepted",
				Message:            "Gateway is accepted",
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(fake.NewScheme()).WithObjects(existing).Build()

	err := EnsureGateway(ctx, client, zap.NewNop().Sugar(), cfg, namespace)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var fetched gatewayapiv1.Gateway
	err = client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: gatewayName}, &fetched)
	if err != nil {
		t.Fatalf("Gateway should exist: %v", err)
	}

	if len(fetched.Status.Conditions) == 0 {
		t.Error("Expected Status conditions to be preserved, but they were cleared (Update was called when it shouldn't have been)")
	}
}

func TestEnsureHTTPRouteCreatesNew(t *testing.T) {
	ctx := context.Background()
	cfg := &kubermaticv1.KubermaticConfiguration{
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "kubermatic.example.com",
			},
		},
	}
	namespace := "kubermatic"

	client := fake.NewClientBuilder().WithScheme(fake.NewScheme()).Build()

	err := EnsureHTTPRoute(ctx, client, zap.NewNop().Sugar(), cfg, namespace)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var created gatewayapiv1.HTTPRoute
	err = client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: httpRouteName}, &created)
	if err != nil {
		t.Fatalf("HTTPRoute should exist after EnsureHTTPRoute: %v", err)
	}

	if created.Name != httpRouteName {
		t.Errorf("expected name %s, got %s", httpRouteName, created.Name)
	}

	if len(created.Spec.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(created.Spec.Rules))
	}

	apiRule := created.Spec.Rules[0]
	if len(apiRule.Matches) != 1 {
		t.Errorf("expected 1 match in /api rule, got %d", len(apiRule.Matches))
	}
	if len(apiRule.BackendRefs) != 1 {
		t.Errorf("expected 1 backend in /api rule, got %d", len(apiRule.BackendRefs))
	}
	if apiRule.BackendRefs[0].Name != APIDeploymentName {
		t.Errorf("expected backend %s, got %s", APIDeploymentName, apiRule.BackendRefs[0].Name)
	}
}

func TestEnsureHTTPRouteSkipsWhenUnchanged(t *testing.T) {
	ctx := context.Background()
	namespace := "kubermatic"

	cfg := &kubermaticv1.KubermaticConfiguration{
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "kubermatic.example.com",
			},
		},
	}

	factory := HTTPRouteReconciler(cfg, namespace)
	_, reconciler := factory()

	desired := &gatewayapiv1.HTTPRoute{}
	if _, err := reconciler(desired); err != nil {
		t.Fatalf("failed to build desired HTTPRoute: %v", err)
	}

	existing := desired.DeepCopy()
	existing.Status.RouteStatus = gatewayapiv1.RouteStatus{
		Parents: []gatewayapiv1.RouteParentStatus{
			{
				Conditions: []metav1.Condition{
					{
						Type:               "Accepted",
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(fake.NewScheme()).WithObjects(existing).Build()

	err := EnsureHTTPRoute(ctx, client, zap.NewNop().Sugar(), cfg, namespace)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var fetched gatewayapiv1.HTTPRoute

	err = client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: httpRouteName}, &fetched)
	if err != nil {
		t.Fatalf("HTTPRoute should exist: %v", err)
	}

	if len(fetched.Status.Parents) == 0 {
		t.Error("Expected Status Parents to be preserved, but they were cleared (Update was called when it shouldn't have been)")
	}
}
