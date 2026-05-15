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

package master

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestMergeReconcileResults(t *testing.T) {
	tests := []struct {
		name string
		a    reconcile.Result
		b    reconcile.Result
		want reconcile.Result
	}{
		{
			name: "keeps delayed requeue",
			a:    reconcile.Result{RequeueAfter: time.Minute},
			want: reconcile.Result{RequeueAfter: time.Minute},
		},
		{
			name: "uses next delayed requeue",
			b:    reconcile.Result{RequeueAfter: time.Minute},
			want: reconcile.Result{RequeueAfter: time.Minute},
		},
		{
			name: "uses sooner delayed requeue",
			a:    reconcile.Result{RequeueAfter: time.Minute},
			b:    reconcile.Result{RequeueAfter: 30 * time.Second},
			want: reconcile.Result{RequeueAfter: 30 * time.Second},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergeReconcileResults(tt.a, tt.b); got != tt.want {
				t.Fatalf("expected %#v, got %#v", tt.want, got)
			}
		})
	}
}

func TestReconcileGatewayAPIResourcesSwitchesToExternalGateway(t *testing.T) {
	ctx := context.Background()
	namespace := "kubermatic"
	scheme := fake.NewScheme()

	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubermatic",
			Namespace: namespace,
			UID:       types.UID("test-uid"),
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "kkp.example.com",
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
						Name:      "platform-gateway",
						Namespace: "networking",
					},
				},
			},
		},
	}

	managedGateway := &gatewayapiv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaulting.DefaultGatewayName,
			Namespace: namespace,
		},
	}
	if err := controllerutil.SetControllerReference(cfg, managedGateway, scheme); err != nil {
		t.Fatalf("failed to set controller reference: %v", err)
	}

	externalGateway := &gatewayapiv1.Gateway{
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

	client := ctrlruntimefakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cfg, managedGateway, externalGateway).
		Build()

	r := &Reconciler{
		Client:            client,
		scheme:            scheme,
		gatewayAPIEnabled: true,
	}

	result, err := r.reconcileGatewayAPIResources(ctx, cfg, zap.NewNop().Sugar())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.RequeueAfter != externalGatewayReadinessRequeueAfter {
		t.Fatalf("expected requeue after %s while waiting for external Gateway acceptance, got %s", externalGatewayReadinessRequeueAfter, result.RequeueAfter)
	}

	var gateway gatewayapiv1.Gateway
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultGatewayName}, &gateway); err != nil {
		t.Fatalf("expected operator-managed Gateway to be preserved until external route is accepted, got %v", err)
	}

	var route gatewayapiv1.HTTPRoute
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultHTTPRouteName}, &route); err != nil {
		t.Fatalf("expected HTTPRoute to be reconciled, got %v", err)
	}

	if len(route.Spec.ParentRefs) != 2 {
		t.Fatalf("expected external and old managed parentRefs, got %d", len(route.Spec.ParentRefs))
	}

	parentRef := route.Spec.ParentRefs[0]
	if string(parentRef.Name) != "platform-gateway" {
		t.Fatalf("expected HTTPRoute parent Gateway platform-gateway, got %s", parentRef.Name)
	}
	if parentRef.Namespace == nil || string(*parentRef.Namespace) != "networking" {
		t.Fatalf("expected HTTPRoute parent Gateway namespace networking, got %v", parentRef.Namespace)
	}

	oldParentRef := route.Spec.ParentRefs[1]
	if oldParentRef.Name != gatewayapiv1.ObjectName(defaulting.DefaultGatewayName) {
		t.Fatalf("expected old managed parentRef %s, got %s", defaulting.DefaultGatewayName, oldParentRef.Name)
	}
	if oldParentRef.Namespace == nil || string(*oldParentRef.Namespace) != namespace {
		t.Fatalf("expected old managed parentRef namespace %s, got %v", namespace, oldParentRef.Namespace)
	}

	route.Status.Parents = []gatewayapiv1.RouteParentStatus{
		{
			ParentRef: parentRef,
			Conditions: []metav1.Condition{
				{
					Type:               string(gatewayapiv1.RouteConditionAccepted),
					Status:             metav1.ConditionTrue,
					ObservedGeneration: route.Generation,
				},
			},
		},
	}
	if err := client.Update(ctx, &route); err != nil {
		t.Fatalf("failed to update HTTPRoute status: %v", err)
	}

	result, err = r.reconcileGatewayAPIResources(ctx, cfg, zap.NewNop().Sugar())
	if err != nil {
		t.Fatalf("expected no error after external route acceptance, got %v", err)
	}
	if result.RequeueAfter != 0 {
		t.Fatalf("expected no requeue after external route acceptance, got %+v", result)
	}

	err = client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultGatewayName}, &gateway)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected operator-managed Gateway to be deleted after external route acceptance, got %v", err)
	}

	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultHTTPRouteName}, &route); err != nil {
		t.Fatalf("expected HTTPRoute to remain reconciled, got %v", err)
	}
	if len(route.Spec.ParentRefs) != 1 {
		t.Fatalf("expected one parentRef after migration, got %d", len(route.Spec.ParentRefs))
	}
	if route.Spec.ParentRefs[0].Name != parentRef.Name || route.Spec.ParentRefs[0].Namespace == nil || *route.Spec.ParentRefs[0].Namespace != *parentRef.Namespace {
		t.Fatalf("expected HTTPRoute to reference external Gateway only, got %v", route.Spec.ParentRefs)
	}
	if len(route.Status.Parents) != 1 || route.Status.Parents[0].ParentRef.Name != parentRef.Name {
		t.Fatalf("expected HTTPRoute status to contain only external parent status, got %v", route.Status.Parents)
	}

	var ns corev1.Namespace
	if err := client.Get(ctx, types.NamespacedName{Name: namespace}, &ns); err != nil {
		t.Fatalf("expected namespace to be reconciled, got %v", err)
	}
	if ns.Labels[common.GatewayAccessLabelKey] != "true" {
		t.Fatalf("expected namespace label %q=true, got %q", common.GatewayAccessLabelKey, ns.Labels[common.GatewayAccessLabelKey])
	}
}

func TestReconcileGatewayAPIResourcesRecordsMissingExternalGateway(t *testing.T) {
	ctx := context.Background()
	namespace := "kubermatic"
	scheme := fake.NewScheme()

	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubermatic",
			Namespace: namespace,
			UID:       types.UID("test-uid"),
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "kkp.example.com",
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
						Name:      "platform-gateway",
						Namespace: "networking",
					},
				},
			},
		},
	}

	client := ctrlruntimefakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cfg).
		Build()
	recorder := events.NewFakeRecorder(10)
	r := &Reconciler{
		Client:            client,
		recorder:          recorder,
		scheme:            scheme,
		gatewayAPIEnabled: true,
	}

	if _, err := r.reconcileGatewayAPIResources(ctx, cfg, zap.NewNop().Sugar()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	select {
	case event := <-recorder.Events:
		if !strings.Contains(event, "Warning ExternalGatewayMissing") {
			t.Fatalf("expected ExternalGatewayMissing warning event, got %q", event)
		}
		if !strings.Contains(event, "networking/platform-gateway") {
			t.Fatalf("expected event to mention external Gateway, got %q", event)
		}
	default:
		t.Fatal("expected ExternalGatewayMissing warning event")
	}
}

func TestReconcileGatewayAPIResourcesKeepsManagedGatewayWhileOtherHTTPRoutesReferenceIt(t *testing.T) {
	ctx := context.Background()
	namespace := "kubermatic"
	scheme := fake.NewScheme()
	externalNamespace := gatewayapiv1.Namespace("networking")
	managedGatewayNamespace := gatewayapiv1.Namespace(namespace)

	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubermatic",
			Namespace: namespace,
			UID:       types.UID("test-uid"),
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "kkp.example.com",
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
						Name:      "platform-gateway",
						Namespace: "networking",
					},
				},
			},
		},
	}

	managedGateway := &gatewayapiv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaulting.DefaultGatewayName,
			Namespace: namespace,
		},
	}
	if err := controllerutil.SetControllerReference(cfg, managedGateway, scheme); err != nil {
		t.Fatalf("failed to set controller reference: %v", err)
	}

	externalGateway := &gatewayapiv1.Gateway{
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

	externalParentRef := gatewayapiv1.ParentReference{
		Name:      "platform-gateway",
		Namespace: &externalNamespace,
	}
	existingKKPRoute := &gatewayapiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:       defaulting.DefaultHTTPRouteName,
			Namespace:  namespace,
			Generation: 2,
		},
		Spec: gatewayapiv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayapiv1.CommonRouteSpec{
				ParentRefs: []gatewayapiv1.ParentReference{externalParentRef},
			},
		},
		Status: gatewayapiv1.HTTPRouteStatus{
			RouteStatus: gatewayapiv1.RouteStatus{
				Parents: []gatewayapiv1.RouteParentStatus{
					{
						ParentRef: externalParentRef,
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

	dexRoute := &gatewayapiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dex",
			Namespace: "dex",
		},
		Spec: gatewayapiv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayapiv1.CommonRouteSpec{
				ParentRefs: []gatewayapiv1.ParentReference{
					{
						Name:      gatewayapiv1.ObjectName(defaulting.DefaultGatewayName),
						Namespace: &managedGatewayNamespace,
					},
				},
			},
		},
	}

	client := ctrlruntimefakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cfg, managedGateway, externalGateway, existingKKPRoute, dexRoute).
		Build()

	r := &Reconciler{
		Client:            client,
		scheme:            scheme,
		gatewayAPIEnabled: true,
	}

	result, err := r.reconcileGatewayAPIResources(ctx, cfg, zap.NewNop().Sugar())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.RequeueAfter != externalGatewayReadinessRequeueAfter {
		t.Fatalf("expected requeue after %s while another HTTPRoute still references the managed Gateway, got %s", externalGatewayReadinessRequeueAfter, result.RequeueAfter)
	}

	var gateway gatewayapiv1.Gateway
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultGatewayName}, &gateway); err != nil {
		t.Fatalf("expected operator-managed Gateway to remain while Dex route references it, got %v", err)
	}

	var route gatewayapiv1.HTTPRoute
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultHTTPRouteName}, &route); err != nil {
		t.Fatalf("expected KKP HTTPRoute to remain reconciled, got %v", err)
	}
	if len(route.Spec.ParentRefs) != 1 || route.Spec.ParentRefs[0].Name != externalParentRef.Name {
		t.Fatalf("expected KKP HTTPRoute to reference external Gateway only, got %v", route.Spec.ParentRefs)
	}

	if err := client.Get(ctx, types.NamespacedName{Namespace: "dex", Name: "dex"}, &route); err != nil {
		t.Fatalf("expected Dex HTTPRoute to remain, got %v", err)
	}
	if len(route.Spec.ParentRefs) != 1 || route.Spec.ParentRefs[0].Name != gatewayapiv1.ObjectName(defaulting.DefaultGatewayName) {
		t.Fatalf("expected Dex HTTPRoute to keep referencing managed Gateway, got %v", route.Spec.ParentRefs)
	}
}

func TestReconcileGatewayAPIResourcesKeepsManagedGatewayWhenExternalRouteRejected(t *testing.T) {
	ctx := context.Background()
	namespace := "kubermatic"
	scheme := fake.NewScheme()
	externalNamespace := gatewayapiv1.Namespace("networking")

	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubermatic",
			Namespace: namespace,
			UID:       types.UID("test-uid"),
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "kkp.example.com",
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
						Name:      "platform-gateway",
						Namespace: "networking",
					},
				},
			},
		},
	}

	managedGateway := &gatewayapiv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaulting.DefaultGatewayName,
			Namespace: namespace,
		},
	}
	if err := controllerutil.SetControllerReference(cfg, managedGateway, scheme); err != nil {
		t.Fatalf("failed to set controller reference: %v", err)
	}

	externalParentRef := gatewayapiv1.ParentReference{
		Name:      "platform-gateway",
		Namespace: &externalNamespace,
	}
	existingRoute := &gatewayapiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:       defaulting.DefaultHTTPRouteName,
			Namespace:  namespace,
			Generation: 2,
		},
		Spec: gatewayapiv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayapiv1.CommonRouteSpec{
				ParentRefs: []gatewayapiv1.ParentReference{externalParentRef},
			},
		},
		Status: gatewayapiv1.HTTPRouteStatus{
			RouteStatus: gatewayapiv1.RouteStatus{
				Parents: []gatewayapiv1.RouteParentStatus{
					{
						ParentRef: externalParentRef,
						Conditions: []metav1.Condition{
							{
								Type:               string(gatewayapiv1.RouteConditionAccepted),
								Status:             metav1.ConditionFalse,
								Reason:             "NotAllowedByListeners",
								ObservedGeneration: 2,
							},
						},
					},
				},
			},
		},
	}
	externalGateway := &gatewayapiv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "platform-gateway",
			Namespace: "networking",
		},
	}

	client := ctrlruntimefakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cfg, managedGateway, externalGateway, existingRoute).
		Build()

	r := &Reconciler{
		Client:            client,
		scheme:            scheme,
		gatewayAPIEnabled: true,
	}

	result, err := r.reconcileGatewayAPIResources(ctx, cfg, zap.NewNop().Sugar())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.RequeueAfter != externalGatewayReadinessRequeueAfter {
		t.Fatalf("expected requeue after %s while external route is rejected, got %s", externalGatewayReadinessRequeueAfter, result.RequeueAfter)
	}

	var gateway gatewayapiv1.Gateway
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultGatewayName}, &gateway); err != nil {
		t.Fatalf("expected operator-managed Gateway to remain while external route is rejected, got %v", err)
	}

	var route gatewayapiv1.HTTPRoute
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultHTTPRouteName}, &route); err != nil {
		t.Fatalf("expected HTTPRoute to exist, got %v", err)
	}
	if len(route.Spec.ParentRefs) != 2 {
		t.Fatalf("expected HTTPRoute to keep external and old managed parentRefs, got %v", route.Spec.ParentRefs)
	}
}

func TestReconcileGatewayAPIResourcesRejectsExternalDefaultManagedGateway(t *testing.T) {
	ctx := context.Background()
	namespace := "kubermatic"
	scheme := fake.NewScheme()

	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubermatic",
			Namespace: namespace,
			UID:       types.UID("test-uid"),
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "kkp.example.com",
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
						Name: defaulting.DefaultGatewayName,
					},
				},
			},
		},
	}

	managedGateway := &gatewayapiv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaulting.DefaultGatewayName,
			Namespace: namespace,
		},
	}
	if err := controllerutil.SetControllerReference(cfg, managedGateway, scheme); err != nil {
		t.Fatalf("failed to set controller reference: %v", err)
	}

	client := ctrlruntimefakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cfg, managedGateway).
		Build()

	r := &Reconciler{
		Client:            client,
		scheme:            scheme,
		gatewayAPIEnabled: true,
	}

	if _, err := r.reconcileGatewayAPIResources(ctx, cfg, zap.NewNop().Sugar()); err == nil {
		t.Fatal("expected externalGateway pointing at the managed default Gateway to be rejected")
	}

	var gateway gatewayapiv1.Gateway
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultGatewayName}, &gateway); err != nil {
		t.Fatalf("expected operator-managed Gateway to remain, got %v", err)
	}

	var route gatewayapiv1.HTTPRoute
	err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultHTTPRouteName}, &route)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected HTTPRoute not to be reconciled, got %v", err)
	}
}

func TestReconcileGatewayAPIResourcesAllowsUnownedExternalDefaultGateway(t *testing.T) {
	ctx := context.Background()
	namespace := "kubermatic"
	scheme := fake.NewScheme()

	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubermatic",
			Namespace: namespace,
			UID:       types.UID("test-uid"),
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "kkp.example.com",
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
						Name: defaulting.DefaultGatewayName,
					},
				},
			},
		},
	}

	externalGateway := &gatewayapiv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaulting.DefaultGatewayName,
			Namespace: namespace,
		},
	}

	client := ctrlruntimefakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cfg, externalGateway).
		Build()

	r := &Reconciler{
		Client:            client,
		scheme:            scheme,
		gatewayAPIEnabled: true,
	}

	if _, err := r.reconcileGatewayAPIResources(ctx, cfg, zap.NewNop().Sugar()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var gateway gatewayapiv1.Gateway
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultGatewayName}, &gateway); err != nil {
		t.Fatalf("expected unowned external Gateway to remain, got %v", err)
	}
	if len(gateway.OwnerReferences) > 0 {
		t.Fatalf("expected external Gateway ownership to remain untouched, got %v", gateway.OwnerReferences)
	}

	var route gatewayapiv1.HTTPRoute
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultHTTPRouteName}, &route); err != nil {
		t.Fatalf("expected HTTPRoute to be reconciled, got %v", err)
	}
	if len(route.Spec.ParentRefs) != 1 || route.Spec.ParentRefs[0].Name != gatewayapiv1.ObjectName(defaulting.DefaultGatewayName) {
		t.Fatalf("expected HTTPRoute to reference default-named external Gateway, got %v", route.Spec.ParentRefs)
	}
}

func TestReconcileGatewayAPIResourcesSwitchesBackToManagedGateway(t *testing.T) {
	ctx := context.Background()
	namespace := "kubermatic"
	scheme := fake.NewScheme()
	externalNamespace := gatewayapiv1.Namespace("networking")

	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubermatic",
			Namespace: namespace,
			UID:       types.UID("test-uid"),
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "kkp.example.com",
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ClassName: defaulting.DefaultGatewayClassName,
				},
			},
		},
	}

	existingRoute := &gatewayapiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaulting.DefaultHTTPRouteName,
			Namespace: namespace,
		},
		Spec: gatewayapiv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayapiv1.CommonRouteSpec{
				ParentRefs: []gatewayapiv1.ParentReference{
					{
						Name:      gatewayapiv1.ObjectName("platform-gateway"),
						Namespace: &externalNamespace,
					},
				},
			},
		},
	}

	client := ctrlruntimefakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cfg, existingRoute).
		Build()

	r := &Reconciler{
		Client:            client,
		scheme:            scheme,
		gatewayAPIEnabled: true,
	}

	if _, err := r.reconcileGatewayAPIResources(ctx, cfg, zap.NewNop().Sugar()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var gateway gatewayapiv1.Gateway
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultGatewayName}, &gateway); err != nil {
		t.Fatalf("expected managed Gateway to be recreated, got %v", err)
	}
	if !common.HasKubermaticConfigurationControllerOwnerReference(gateway.OwnerReferences, cfg) {
		t.Fatalf("expected managed Gateway to have KubermaticConfiguration owner reference, got %v", gateway.OwnerReferences)
	}

	var route gatewayapiv1.HTTPRoute
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultHTTPRouteName}, &route); err != nil {
		t.Fatalf("expected HTTPRoute to exist, got %v", err)
	}
	if len(route.Spec.ParentRefs) != 1 {
		t.Fatalf("expected one parentRef, got %d", len(route.Spec.ParentRefs))
	}
	parentRef := route.Spec.ParentRefs[0]
	if parentRef.Name != gatewayapiv1.ObjectName(defaulting.DefaultGatewayName) {
		t.Fatalf("expected HTTPRoute to reference managed Gateway %s, got %s", defaulting.DefaultGatewayName, parentRef.Name)
	}
	if parentRef.Namespace == nil || *parentRef.Namespace != gatewayapiv1.Namespace(namespace) {
		t.Fatalf("expected HTTPRoute parent namespace %s, got %v", namespace, parentRef.Namespace)
	}
}

func TestReconcileGatewayAPIResourcesCreatesExternalHTTPRouteWithoutManagedGateway(t *testing.T) {
	ctx := context.Background()
	namespace := "kubermatic"
	scheme := fake.NewScheme()

	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubermatic",
			Namespace: namespace,
			UID:       types.UID("test-uid"),
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "kkp.example.com",
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
						Name:      "platform-gateway",
						Namespace: "networking",
					},
				},
			},
		},
	}

	client := ctrlruntimefakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cfg).
		Build()

	r := &Reconciler{
		Client:            client,
		scheme:            scheme,
		gatewayAPIEnabled: true,
	}

	if _, err := r.reconcileGatewayAPIResources(ctx, cfg, zap.NewNop().Sugar()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var gateway gatewayapiv1.Gateway
	err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultGatewayName}, &gateway)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected no operator-managed Gateway to be created, got %v", err)
	}

	var route gatewayapiv1.HTTPRoute
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultHTTPRouteName}, &route); err != nil {
		t.Fatalf("expected HTTPRoute to be reconciled, got %v", err)
	}

	if len(route.Spec.ParentRefs) != 1 {
		t.Fatalf("expected one parentRef, got %d", len(route.Spec.ParentRefs))
	}

	parentRef := route.Spec.ParentRefs[0]
	if string(parentRef.Name) != "platform-gateway" {
		t.Fatalf("expected HTTPRoute parent Gateway platform-gateway, got %s", parentRef.Name)
	}
	if parentRef.Namespace == nil || string(*parentRef.Namespace) != "networking" {
		t.Fatalf("expected HTTPRoute parent Gateway namespace networking, got %v", parentRef.Namespace)
	}

	var ns corev1.Namespace
	if err := client.Get(ctx, types.NamespacedName{Name: namespace}, &ns); err != nil {
		t.Fatalf("expected namespace to be reconciled, got %v", err)
	}
	if ns.Labels[common.GatewayAccessLabelKey] != "true" {
		t.Fatalf("expected namespace label %q=true, got %q", common.GatewayAccessLabelKey, ns.Labels[common.GatewayAccessLabelKey])
	}
}
