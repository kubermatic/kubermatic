/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package kubermaticmaster

import (
	"context"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/util/yamled"

	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestValidateKubermaticConfigurationRejectsExternalGatewayWithoutName(t *testing.T) {
	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: KubermaticOperatorNamespace,
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "example.com",
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{},
				},
			},
			FeatureGates: map[string]bool{
				"HeadlessInstallation": true,
			},
		},
	}

	failures := validateKubermaticConfiguration(cfg)
	if len(failures) == 0 {
		t.Fatal("expected validation failure")
	}

	if !strings.Contains(failures[0].Error(), "externalGateway.name") {
		t.Fatalf("expected externalGateway.name validation failure, got %v", failures)
	}
}

func TestDefaultDexGatewayHTTPRouteValuesUsesExternalGateway(t *testing.T) {
	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
						Name:      "platform-gateway",
						Namespace: "networking",
					},
				},
			},
		},
	}

	doc, err := yamled.Load(strings.NewReader(`
migrateGatewayAPI: true
httpRoute:
  gatewayName: kubermatic
  gatewayNamespace: kubermatic
`))
	if err != nil {
		t.Fatalf("failed to load Helm values: %v", err)
	}

	defaultDexGatewayHTTPRouteValues(cfg, doc, logrus.New())

	gatewayName, _ := doc.GetString(yamled.Path{"httpRoute", "gatewayName"})
	if gatewayName != "platform-gateway" {
		t.Errorf("expected Dex HTTPRoute Gateway name platform-gateway, got %s", gatewayName)
	}

	gatewayNamespace, _ := doc.GetString(yamled.Path{"httpRoute", "gatewayNamespace"})
	if gatewayNamespace != "networking" {
		t.Errorf("expected Dex HTTPRoute Gateway namespace networking, got %s", gatewayNamespace)
	}
}

func TestDexGatewayObjectKeyUsesConfiguredHTTPRouteGateway(t *testing.T) {
	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
						Name:      "platform-gateway",
						Namespace: "networking",
					},
				},
			},
		},
	}

	doc, err := yamled.Load(strings.NewReader(`
httpRoute:
  gatewayName: dex-gateway
  gatewayNamespace: dex-networking
`))
	if err != nil {
		t.Fatalf("failed to load Helm values: %v", err)
	}

	got := dexGatewayObjectKey(cfg, stack.DeployOptions{HelmValues: doc})
	if got.Name != "dex-gateway" {
		t.Errorf("expected Dex Gateway name dex-gateway, got %s", got.Name)
	}
	if got.Namespace != "dex-networking" {
		t.Errorf("expected Dex Gateway namespace dex-networking, got %s", got.Namespace)
	}
}

func TestGatewayObjectKeyUsesExternalGateway(t *testing.T) {
	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
						Name:      "platform-gateway",
						Namespace: "networking",
					},
				},
			},
		},
	}

	got := gatewayObjectKey(cfg)
	if got.Name != "platform-gateway" {
		t.Errorf("expected Gateway name platform-gateway, got %s", got.Name)
	}
	if got.Namespace != "networking" {
		t.Errorf("expected Gateway namespace networking, got %s", got.Namespace)
	}
}

func TestWaitForGatewayAllowsExternalGatewayWithoutAddress(t *testing.T) {
	ctx := context.Background()
	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{Namespace: "kubermatic"},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
						Name:      "platform-gateway",
						Namespace: "networking",
					},
				},
			},
		},
	}

	gw := &gatewayapiv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "platform-gateway",
			Namespace: "networking",
		},
		Status: gatewayapiv1.GatewayStatus{
			Conditions: []metav1.Condition{
				{
					Type:   string(gatewayapiv1.GatewayConditionProgrammed),
					Status: metav1.ConditionTrue,
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithObjects(gw).Build()
	if _, err := waitForGateway(ctx, logrus.NewEntry(logrus.New()), client, cfg); err != nil {
		t.Fatalf("expected external Gateway without addresses to be accepted, got %v", err)
	}
}

func TestIsGatewayOwnedByKubermaticConfiguration(t *testing.T) {
	controller := true

	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			UID: types.UID("test-uid"),
		},
	}

	gw := &gatewayapiv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       "KubermaticConfiguration",
					Name:       "kubermatic",
					UID:        types.UID("test-uid"),
					Controller: &controller,
				},
			},
		},
	}

	if !isGatewayOwnedByKubermaticConfiguration(gw, cfg) {
		t.Fatal("expected Gateway to be recognized as operator-owned")
	}

	gw.OwnerReferences[0].UID = types.UID("other-uid")
	if isGatewayOwnedByKubermaticConfiguration(gw, cfg) {
		t.Fatal("expected Gateway with another owner UID not to be recognized as operator-owned")
	}

	gw.OwnerReferences = nil
	gw.Labels = map[string]string{
		"app.kubernetes.io/name":       "kubermatic",
		"app.kubernetes.io/managed-by": "kubermatic-operator",
	}
	if isGatewayOwnedByKubermaticConfiguration(gw, cfg) {
		t.Fatal("expected Gateway without KubermaticConfiguration controller owner reference not to be recognized as operator-owned")
	}
}

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

	if !httpRouteReferencesGateway(route, gatewayName) {
		t.Fatal("expected HTTPRoute to reference the active Gateway")
	}
	if !httpRouteAcceptedByGateway(route, gatewayName) {
		t.Fatal("expected HTTPRoute to be accepted by the active Gateway")
	}

	route.Status.Parents[0].Conditions[0].ObservedGeneration = 1
	if httpRouteAcceptedByGateway(route, gatewayName) {
		t.Fatal("expected stale HTTPRoute Accepted condition not to be ready")
	}

	route.Status.Parents[0].Conditions[0].ObservedGeneration = 2
	route.Status.Parents[0].ParentRef.Name = "other-gateway"
	if httpRouteAcceptedByGateway(route, gatewayName) {
		t.Fatal("expected HTTPRoute status for another Gateway not to be ready")
	}
}

func TestDeleteIngressAfterHTTPRouteReady(t *testing.T) {
	ctx := context.Background()
	gatewayName := types.NamespacedName{Namespace: "networking", Name: "platform-gateway"}
	gatewayNamespace := gatewayapiv1.Namespace(gatewayName.Namespace)
	routeName := types.NamespacedName{Namespace: "dex", Name: "dex"}

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeName.Name,
			Namespace: routeName.Namespace,
		},
	}
	route := &gatewayapiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:       routeName.Name,
			Namespace:  routeName.Namespace,
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
							Name:      gatewayapiv1.ObjectName(gatewayName.Name),
							Namespace: &gatewayNamespace,
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

	client := fake.NewClientBuilder().WithObjects(ingress, route).Build()
	if err := deleteIngressAfterHTTPRouteReady(ctx, logrus.NewEntry(logrus.New()), client, routeName, routeName, gatewayName); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var fetched networkingv1.Ingress
	if err := client.Get(ctx, routeName, &fetched); !apierrors.IsNotFound(err) {
		t.Fatalf("expected Ingress to be deleted, got %v", err)
	}
}

func TestClusterVersionIsConfigured(t *testing.T) {
	testcases := []struct {
		name       string
		version    semver.Semver
		versions   kubermaticv1.KubermaticVersioningConfiguration
		configured bool
	}{
		{
			name:    "version is directly supported",
			version: *semver.NewSemverOrDie("1.0.0"),
			versions: kubermaticv1.KubermaticVersioningConfiguration{
				Versions: []semver.Semver{
					*semver.NewSemverOrDie("1.0.0"),
				},
			},
			configured: true,
		},
		{
			name:    "version is not configured",
			version: *semver.NewSemverOrDie("1.0.0"),
			versions: kubermaticv1.KubermaticVersioningConfiguration{
				Versions: []semver.Semver{
					*semver.NewSemverOrDie("2.0.0"),
				},
			},
			configured: false,
		},
		{
			name:    "update constraint matches because it's auto update",
			version: *semver.NewSemverOrDie("1.0.0"),
			versions: kubermaticv1.KubermaticVersioningConfiguration{
				Updates: []kubermaticv1.Update{
					{
						From:      "1.0.0",
						To:        "2.0.0",
						Automatic: ptr.To(true),
					},
				},
			},
			configured: true,
		},
		{
			name:    "constraint expression matches",
			version: *semver.NewSemverOrDie("1.2.3"),
			versions: kubermaticv1.KubermaticVersioningConfiguration{
				Updates: []kubermaticv1.Update{
					{
						From:      "1.2.*",
						To:        "2.0.0",
						Automatic: ptr.To(true),
					},
				},
			},
			configured: true,
		},
		{
			name:    "update constraint does not match because it's no auto update",
			version: *semver.NewSemverOrDie("1.0.0"),
			versions: kubermaticv1.KubermaticVersioningConfiguration{
				Updates: []kubermaticv1.Update{
					{
						From:      "1.0.0",
						To:        "2.0.0",
						Automatic: ptr.To(false),
					},
				},
			},
			configured: false,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			config := kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Versions: testcase.versions,
				},
			}

			constraints, err := getAutoUpdateConstraints(&config)
			if err != nil {
				t.Fatalf("Failed to determine update constraints: %v", err)
			}

			configured := clusterVersionIsConfigured(testcase.version, &config, constraints)
			if configured != testcase.configured {
				if testcase.configured {
					t.Fatalf("Expected %q to be supported, but it is not.", testcase.version)
				} else {
					t.Fatalf("Expected %q to not be supported, but it is.", testcase.version)
				}
			}
		})
	}
}

func Test_isPublicIp(t *testing.T) {
	testCases := map[string]bool{
		"10.100.197.9":   false,
		"172.16.1.9":     false,
		"192.168.1.1":    false,
		"167.233.10.245": true,
	}

	for ip, want := range testCases {
		if got := isPublicIP(ip); got != want {
			t.Errorf("isPublicIp(%s) = %v , want: %v", ip, got, want)
		}
	}
}
