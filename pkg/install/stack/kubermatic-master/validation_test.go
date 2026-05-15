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
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/util/yamled"

	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
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

func TestValidateKubermaticConfigurationAllowsExternalGatewayDefaultKey(t *testing.T) {
	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: KubermaticOperatorNamespace,
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "example.com",
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
						Name: defaulting.DefaultGatewayName,
					},
				},
			},
			FeatureGates: map[string]bool{
				"HeadlessInstallation": true,
			},
		},
	}

	if failures := validateKubermaticConfiguration(cfg); len(failures) > 0 {
		t.Fatalf("expected default Gateway key to be statically valid, got %v", failures)
	}
}

func TestDexGatewayObjectKeyUsesExternalGatewayWithoutMutatedValues(t *testing.T) {
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

	got := dexGatewayObjectKey(cfg, stack.DeployOptions{HelmValues: doc})
	if got.Name != "platform-gateway" {
		t.Errorf("expected Dex Gateway name platform-gateway, got %s", got.Name)
	}
	if got.Namespace != "networking" {
		t.Errorf("expected Dex Gateway namespace networking, got %s", got.Namespace)
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

func TestWaitForGatewayRejectsDeletingExternalGateway(t *testing.T) {
	ctx := context.Background()
	pollConfig := gatewayAPIReadinessPollConfig{interval: time.Millisecond, timeout: 10 * time.Millisecond}

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

	deletionTime := metav1.Now()
	gw := &gatewayapiv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "platform-gateway",
			Namespace:         "networking",
			DeletionTimestamp: &deletionTime,
			Finalizers:        []string{"test/finalizer"},
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
	if _, err := waitForGatewayWithPollConfig(ctx, logrus.NewEntry(logrus.New()), client, cfg, pollConfig); err == nil {
		t.Fatal("expected deleting external Gateway not to be accepted")
	}
}

func TestWaitForGatewayRejectsOperatorOwnedExternalGateway(t *testing.T) {
	ctx := context.Background()
	pollConfig := gatewayAPIReadinessPollConfig{interval: time.Millisecond, timeout: 10 * time.Millisecond}

	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "kubermatic",
			UID:       types.UID("test-uid"),
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

	tests := []struct {
		name     string
		ownerUID types.UID
	}{
		{
			name:     "current owner",
			ownerUID: cfg.UID,
		},
		{
			name:     "stale owner",
			ownerUID: types.UID("other-config-uid"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gw := &gatewayapiv1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "platform-gateway",
					Namespace: "networking",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: kubermaticv1.SchemeGroupVersion.String(),
							Kind:       "KubermaticConfiguration",
							Name:       "kubermatic",
							UID:        tt.ownerUID,
							Controller: ptr.To(true),
						},
					},
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
			_, err := waitForGatewayWithPollConfig(ctx, logrus.NewEntry(logrus.New()), client, cfg, pollConfig)
			if err == nil {
				t.Fatal("expected operator-owned external Gateway not to be accepted")
			}
			if !errors.Is(err, errOperatorOwnedExternalGateway) {
				t.Fatalf("expected errOperatorOwnedExternalGateway sentinel, got %v", err)
			}
			if !strings.Contains(err.Error(), "remove KubermaticConfiguration controller ownerReferences") {
				t.Fatalf("expected error to include ownerReference recovery hint, got %v", err)
			}
			if strings.Contains(err.Error(), "failed to become ready within") {
				t.Fatalf("operator-owned Gateway error must not be wrapped with the readiness-timeout message, got %v", err)
			}
		})
	}
}

func TestValidateExternalGatewayNotOperatorOwned(t *testing.T) {
	ctx := context.Background()
	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "kubermatic",
			UID:       types.UID("test-uid"),
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

	operatorOwnerRef := metav1.OwnerReference{
		APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		Kind:       "KubermaticConfiguration",
		Name:       "kubermatic",
		UID:        cfg.UID,
		Controller: ptr.To(true),
	}
	deletionTime := metav1.Now()

	tests := []struct {
		name    string
		objects []ctrlruntimeclient.Object
		wantErr bool
	}{
		{
			name: "missing Gateway is allowed",
		},
		{
			name: "unowned Gateway is allowed",
			objects: []ctrlruntimeclient.Object{
				&gatewayapiv1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "platform-gateway",
						Namespace: "networking",
					},
				},
			},
		},
		{
			name: "deleting operator-owned Gateway is treated as not present",
			objects: []ctrlruntimeclient.Object{
				&gatewayapiv1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "platform-gateway",
						Namespace:         "networking",
						OwnerReferences:   []metav1.OwnerReference{operatorOwnerRef},
						DeletionTimestamp: &deletionTime,
						Finalizers:        []string{"test/finalizer"},
					},
				},
			},
		},
		{
			name: "operator-owned Gateway is rejected",
			objects: []ctrlruntimeclient.Object{
				&gatewayapiv1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "platform-gateway",
						Namespace:       "networking",
						OwnerReferences: []metav1.OwnerReference{operatorOwnerRef},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithObjects(tt.objects...).Build()
			err := validateExternalGatewayNotOperatorOwned(ctx, client, cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected validation error")
				}
				if !errors.Is(err, errOperatorOwnedExternalGateway) {
					t.Fatalf("expected errOperatorOwnedExternalGateway sentinel, got %v", err)
				}
				if !strings.Contains(err.Error(), "remove KubermaticConfiguration controller ownerReferences") {
					t.Fatalf("expected error to include ownerReference recovery hint, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no validation error, got %v", err)
			}
		})
	}
}

func TestIsGatewayOwnedByKubermaticConfiguration(t *testing.T) {
	tests := []struct {
		name      string
		ownerRefs []metav1.OwnerReference
		labels    map[string]string
		wantOwned bool
	}{
		{
			name: "current KubermaticConfiguration controller owner is owned",
			ownerRefs: []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       "KubermaticConfiguration",
					Name:       "kubermatic",
					UID:        types.UID("test-uid"),
					Controller: ptr.To(true),
				},
			},
			wantOwned: true,
		},
		{
			name: "stale KubermaticConfiguration controller owner is owned",
			ownerRefs: []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       "KubermaticConfiguration",
					Name:       "kubermatic",
					UID:        types.UID("other-uid"),
					Controller: ptr.To(true),
				},
			},
			wantOwned: true,
		},
		{
			name: "non-controller KubermaticConfiguration owner is not owned",
			ownerRefs: []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       "KubermaticConfiguration",
					Name:       "kubermatic",
					UID:        types.UID("test-uid"),
					Controller: ptr.To(false),
				},
			},
			wantOwned: false,
		},
		{
			name: "operator-style labels without owner reference are not owned",
			labels: map[string]string{
				"app.kubernetes.io/name":       "kubermatic",
				"app.kubernetes.io/managed-by": "kubermatic-operator",
			},
			wantOwned: false,
		},
		{
			name:      "no owner references at all are not owned",
			wantOwned: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gw := &gatewayapiv1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: tt.ownerRefs,
					Labels:          tt.labels,
				},
			}
			if got := isGatewayOwnedByKubermaticConfiguration(gw); got != tt.wantOwned {
				t.Fatalf("isGatewayOwnedByKubermaticConfiguration() = %v, want %v", got, tt.wantOwned)
			}
		})
	}
}

func TestCleanupGatewayAPIResourcesRespectsHTTPRouteOwnership(t *testing.T) {
	ctx := context.Background()
	controller := true

	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubermatic",
			Namespace: KubermaticOperatorNamespace,
			UID:       types.UID("test-uid"),
		},
	}

	testCases := []struct {
		name       string
		ownerRefs  []metav1.OwnerReference
		wantExists bool
	}{
		{
			name:       "leaves unowned HTTPRoute",
			wantExists: true,
		},
		{
			name: "deletes operator-owned HTTPRoute",
			ownerRefs: []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       "KubermaticConfiguration",
					Name:       cfg.Name,
					UID:        cfg.UID,
					Controller: &controller,
				},
			},
			wantExists: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			routeName := types.NamespacedName{Namespace: cfg.Namespace, Name: defaulting.DefaultHTTPRouteName}
			route := &gatewayapiv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:            routeName.Name,
					Namespace:       routeName.Namespace,
					OwnerReferences: tc.ownerRefs,
				},
			}

			client := fake.NewClientBuilder().WithObjects(route).Build()
			if err := cleanupGatewayAPIResources(ctx, logrus.NewEntry(logrus.New()), client, cfg); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			var fetched gatewayapiv1.HTTPRoute
			err := client.Get(ctx, routeName, &fetched)
			if tc.wantExists {
				if err != nil {
					t.Fatalf("expected HTTPRoute to remain, got %v", err)
				}
				return
			}

			if !apierrors.IsNotFound(err) {
				t.Fatalf("expected HTTPRoute to be deleted, got %v", err)
			}
		})
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

func TestCleanupIngressSkipsDexWhenDexChartIsSkipped(t *testing.T) {
	ctx := context.Background()
	gatewayName := types.NamespacedName{Namespace: "networking", Name: "platform-gateway"}
	gatewayNamespace := gatewayapiv1.Namespace(gatewayName.Namespace)

	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: KubermaticOperatorNamespace,
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
						Name:      gatewayName.Name,
						Namespace: gatewayName.Namespace,
					},
				},
			},
		},
	}

	gateway := &gatewayapiv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gatewayName.Name,
			Namespace: gatewayName.Namespace,
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

	kubermaticIngressName := types.NamespacedName{Namespace: cfg.Namespace, Name: defaulting.DefaultIngressName}
	kubermaticRouteName := types.NamespacedName{Namespace: cfg.Namespace, Name: defaulting.DefaultHTTPRouteName}
	dexIngressName := types.NamespacedName{Namespace: DexNamespace, Name: DexChartName}

	kubermaticIngress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubermaticIngressName.Name,
			Namespace: kubermaticIngressName.Namespace,
		},
	}
	kubermaticRoute := &gatewayapiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:       kubermaticRouteName.Name,
			Namespace:  kubermaticRouteName.Namespace,
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
	dexIngress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dexIngressName.Name,
			Namespace: dexIngressName.Namespace,
		},
	}

	client := fake.NewClientBuilder().WithObjects(gateway, kubermaticIngress, kubermaticRoute, dexIngress).Build()
	if err := cleanupIngress(ctx, logrus.NewEntry(logrus.New()), client, stack.DeployOptions{
		KubermaticConfiguration: cfg,
		SkipCharts:              []string{DexChartName},
	}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var fetchedKubermaticIngress networkingv1.Ingress
	if err := client.Get(ctx, kubermaticIngressName, &fetchedKubermaticIngress); !apierrors.IsNotFound(err) {
		t.Fatalf("expected Kubermatic Ingress to be deleted, got %v", err)
	}

	var fetchedDexIngress networkingv1.Ingress
	if err := client.Get(ctx, dexIngressName, &fetchedDexIngress); err != nil {
		t.Fatalf("expected skipped Dex Ingress to remain, got %v", err)
	}
}

func TestCleanupIngressSkipsGatewayWaitWhenNoLegacyIngressExists(t *testing.T) {
	ctx := context.Background()
	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: KubermaticOperatorNamespace,
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

	client := fake.NewClientBuilder().Build()
	if err := cleanupIngress(ctx, logrus.NewEntry(logrus.New()), client, stack.DeployOptions{
		KubermaticConfiguration: cfg,
	}); err != nil {
		t.Fatalf("expected cleanup without legacy Ingress to skip Gateway wait, got %v", err)
	}
}

func TestCleanupIngressWaitsForDexOverrideGateway(t *testing.T) {
	ctx := context.Background()
	pollConfig := gatewayAPIReadinessPollConfig{interval: time.Millisecond, timeout: 10 * time.Millisecond}

	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: KubermaticOperatorNamespace,
		},
	}

	doc, err := yamled.Load(strings.NewReader(`
migrateGatewayAPI: true
httpRoute:
  gatewayName: dex-gateway
  gatewayNamespace: dex-networking
`))
	if err != nil {
		t.Fatalf("failed to load Helm values: %v", err)
	}

	dexGatewayName := types.NamespacedName{Namespace: "dex-networking", Name: "dex-gateway"}
	dexGatewayNamespace := gatewayapiv1.Namespace(dexGatewayName.Namespace)
	dexIngressName := types.NamespacedName{Namespace: DexNamespace, Name: DexChartName}
	dexIngress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dexIngressName.Name,
			Namespace: dexIngressName.Namespace,
		},
	}
	dexRoute := &gatewayapiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:       dexIngressName.Name,
			Namespace:  dexIngressName.Namespace,
			Generation: 1,
		},
		Spec: gatewayapiv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayapiv1.CommonRouteSpec{
				ParentRefs: []gatewayapiv1.ParentReference{
					{
						Name:      gatewayapiv1.ObjectName(dexGatewayName.Name),
						Namespace: &dexGatewayNamespace,
					},
				},
			},
		},
		Status: gatewayapiv1.HTTPRouteStatus{
			RouteStatus: gatewayapiv1.RouteStatus{
				Parents: []gatewayapiv1.RouteParentStatus{
					{
						ParentRef: gatewayapiv1.ParentReference{
							Name:      gatewayapiv1.ObjectName(dexGatewayName.Name),
							Namespace: &dexGatewayNamespace,
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

	client := fake.NewClientBuilder().WithObjects(dexIngress, dexRoute).Build()
	err = cleanupIngressWithPollConfig(ctx, logrus.NewEntry(logrus.New()), client, stack.DeployOptions{
		KubermaticConfiguration: cfg,
		HelmValues:              doc,
	}, pollConfig)
	if err == nil {
		t.Fatal("expected cleanup to wait for the Dex Gateway")
	}
	if !strings.Contains(err.Error(), dexGatewayName.String()) {
		t.Fatalf("expected error to mention Dex Gateway %s, got %v", dexGatewayName.String(), err)
	}

	var fetched networkingv1.Ingress
	if err := client.Get(ctx, dexIngressName, &fetched); err != nil {
		t.Fatalf("expected Dex Ingress to remain while Dex Gateway is not ready, got %v", err)
	}
}

func TestDeleteIngressWaitsForFreshHTTPRouteAcceptedGeneration(t *testing.T) {
	ctx := context.Background()
	gatewayName := types.NamespacedName{Namespace: "networking", Name: "platform-gateway"}
	gatewayNamespace := gatewayapiv1.Namespace(gatewayName.Namespace)
	routeName := types.NamespacedName{Namespace: "dex", Name: "dex"}
	pollConfig := gatewayAPIReadinessPollConfig{interval: time.Millisecond, timeout: time.Second}

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
								ObservedGeneration: 1,
							},
						},
					},
				},
			},
		},
	}

	httpRouteGets := 0
	client := fake.NewClientBuilder().
		WithObjects(ingress, route).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, client ctrlruntimeclient.WithWatch, key ctrlruntimeclient.ObjectKey, obj ctrlruntimeclient.Object, opts ...ctrlruntimeclient.GetOption) error {
				if err := client.Get(ctx, key, obj, opts...); err != nil {
					return err
				}

				fetchedRoute, ok := obj.(*gatewayapiv1.HTTPRoute)
				if !ok || key != routeName {
					return nil
				}

				httpRouteGets++
				if httpRouteGets > 1 {
					fetchedRoute.Status.Parents[0].Conditions[0].ObservedGeneration = fetchedRoute.Generation
				}

				return nil
			},
		}).
		Build()

	if err := deleteIngressAfterHTTPRouteReadyWithPollConfig(ctx, logrus.NewEntry(logrus.New()), client, routeName, routeName, gatewayName, pollConfig); err != nil {
		t.Fatalf("expected no error after fresh HTTPRoute status, got %v", err)
	}

	if httpRouteGets < 2 {
		t.Fatalf("expected stale HTTPRoute status to require another poll, got %d get(s)", httpRouteGets)
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
