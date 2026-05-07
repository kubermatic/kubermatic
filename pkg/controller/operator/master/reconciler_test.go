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
	"testing"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

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

	client := ctrlruntimefakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cfg, managedGateway).
		Build()

	r := &Reconciler{
		Client:            client,
		scheme:            scheme,
		gatewayAPIEnabled: true,
	}

	if err := r.reconcileGatewayAPIResources(ctx, cfg, zap.NewNop().Sugar()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var gateway gatewayapiv1.Gateway
	err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaulting.DefaultGatewayName}, &gateway)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected operator-managed Gateway to be deleted, got %v", err)
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

	if err := r.reconcileGatewayAPIResources(ctx, cfg, zap.NewNop().Sugar()); err == nil {
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

	if err := r.reconcileGatewayAPIResources(ctx, cfg, zap.NewNop().Sugar()); err != nil {
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
