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

package kubermaticmaster

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var fastPoll = gatewayAPIReadinessPollConfig{interval: time.Millisecond, timeout: 50 * time.Millisecond}

func defaultKubermaticConfiguration(namespace string) *kubermaticv1.KubermaticConfiguration {
	return &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
	}
}

func TestWaitForGatewayAndHTTPRoutesReadyAllAccepted(t *testing.T) {
	cfg := defaultKubermaticConfiguration("kubermatic")
	gatewayKey := gatewayObjectKey(cfg)
	kubermaticRoute := acceptedHTTPRoute(
		types.NamespacedName{Namespace: cfg.Namespace, Name: defaulting.DefaultHTTPRouteName},
		gatewayKey,
	)
	dexRoute := acceptedHTTPRoute(
		types.NamespacedName{Namespace: DexNamespace, Name: DexChartName},
		gatewayKey,
	)
	gw := programmedGateway(gatewayKey)

	client := fake.NewClientBuilder().WithObjects(gw, kubermaticRoute, dexRoute).Build()

	opt := stack.DeployOptions{KubermaticConfiguration: cfg}
	if err := waitForGatewayAndHTTPRoutesReadyWithPollConfig(context.Background(), logrus.NewEntry(logrus.New()), client, opt, fastPoll); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWaitForGatewayAndHTTPRoutesReadyHeadlessSkips(t *testing.T) {
	cfg := defaultKubermaticConfiguration("kubermatic")
	cfg.Spec.FeatureGates = map[string]bool{"HeadlessInstallation": true}

	// no Gateway/HTTPRoute objects exist; headless skip path must short-circuit.
	client := fake.NewClientBuilder().Build()

	opt := stack.DeployOptions{KubermaticConfiguration: cfg}
	if err := waitForGatewayAndHTTPRoutesReadyWithPollConfig(context.Background(), logrus.NewEntry(logrus.New()), client, opt, fastPoll); err != nil {
		t.Fatalf("expected headless installation to skip waits, got %v", err)
	}
}

func TestWaitForGatewayAndHTTPRoutesReadySkipsDexWhenSkipped(t *testing.T) {
	cfg := defaultKubermaticConfiguration("kubermatic")
	gatewayKey := gatewayObjectKey(cfg)
	kubermaticRoute := acceptedHTTPRoute(
		types.NamespacedName{Namespace: cfg.Namespace, Name: defaulting.DefaultHTTPRouteName},
		gatewayKey,
	)
	gw := programmedGateway(gatewayKey)

	// no Dex HTTPRoute exists; if Dex is in SkipCharts, the wait must not require it.
	client := fake.NewClientBuilder().WithObjects(gw, kubermaticRoute).Build()

	opt := stack.DeployOptions{KubermaticConfiguration: cfg, SkipCharts: []string{DexChartName}}
	if err := waitForGatewayAndHTTPRoutesReadyWithPollConfig(context.Background(), logrus.NewEntry(logrus.New()), client, opt, fastPoll); err != nil {
		t.Fatalf("expected SkipCharts=dex to skip Dex HTTPRoute wait, got %v", err)
	}
}

func TestWaitForGatewayAndHTTPRoutesReadyFailsOnUnacceptedKubermaticRoute(t *testing.T) {
	cfg := defaultKubermaticConfiguration("kubermatic")
	gatewayKey := gatewayObjectKey(cfg)
	rejected := rejectedHTTPRoute(
		types.NamespacedName{Namespace: cfg.Namespace, Name: defaulting.DefaultHTTPRouteName},
		gatewayKey,
		string(gatewayapiv1.RouteReasonNotAllowedByListeners),
	)
	gw := programmedGateway(gatewayKey)

	client := fake.NewClientBuilder().WithObjects(gw, rejected).Build()

	opt := stack.DeployOptions{KubermaticConfiguration: cfg}
	err := waitForGatewayAndHTTPRoutesReadyWithPollConfig(context.Background(), logrus.NewEntry(logrus.New()), client, opt, fastPoll)
	if err == nil {
		t.Fatal("expected error for unaccepted kubermatic HTTPRoute, got nil")
	}
	if !strings.Contains(err.Error(), "kubermatic") {
		t.Fatalf("expected error to mention kubermatic route, got %v", err)
	}
}

func TestCleanupLegacyIngressesDeletesKubermaticAndDex(t *testing.T) {
	cfg := defaultKubermaticConfiguration("kubermatic")
	kubermaticIngress := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Namespace: cfg.Namespace, Name: defaulting.DefaultIngressName}}
	dexIngress := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Namespace: DexNamespace, Name: DexChartName}}

	client := fake.NewClientBuilder().WithObjects(kubermaticIngress, dexIngress).Build()

	opt := stack.DeployOptions{KubermaticConfiguration: cfg}
	if err := cleanupLegacyIngresses(context.Background(), logrus.NewEntry(logrus.New()), client, opt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertIngressGone(t, client, types.NamespacedName{Namespace: cfg.Namespace, Name: defaulting.DefaultIngressName})
	assertIngressGone(t, client, types.NamespacedName{Namespace: DexNamespace, Name: DexChartName})
}

func TestCleanupLegacyIngressesSkipsDexWhenSkipped(t *testing.T) {
	cfg := defaultKubermaticConfiguration("kubermatic")
	kubermaticIngress := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Namespace: cfg.Namespace, Name: defaulting.DefaultIngressName}}
	dexIngress := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Namespace: DexNamespace, Name: DexChartName}}

	client := fake.NewClientBuilder().WithObjects(kubermaticIngress, dexIngress).Build()

	opt := stack.DeployOptions{KubermaticConfiguration: cfg, SkipCharts: []string{DexChartName}}
	if err := cleanupLegacyIngresses(context.Background(), logrus.NewEntry(logrus.New()), client, opt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertIngressGone(t, client, types.NamespacedName{Namespace: cfg.Namespace, Name: defaulting.DefaultIngressName})

	dex := &networkingv1.Ingress{}
	if err := client.Get(context.Background(), types.NamespacedName{Namespace: DexNamespace, Name: DexChartName}, dex); err != nil {
		t.Fatalf("expected Dex Ingress to remain when Dex chart is skipped, got error: %v", err)
	}
}

func TestCleanupLegacyIngressesIdempotentWhenNothingPresent(t *testing.T) {
	cfg := defaultKubermaticConfiguration("kubermatic")
	client := fake.NewClientBuilder().Build()

	opt := stack.DeployOptions{KubermaticConfiguration: cfg}
	if err := cleanupLegacyIngresses(context.Background(), logrus.NewEntry(logrus.New()), client, opt); err != nil {
		t.Fatalf("expected idempotent no-op, got %v", err)
	}
}

// TestCleanupLegacyIngressesSkipsWhenSkipIngressCleanupSet verifies the flag path:
// when opt.SkipIngressCleanup is true, the legacy Ingress objects must remain in
// place so nginx can keep serving traffic while DNS is flipped to Envoy Gateway.
func TestCleanupLegacyIngressesSkipsWhenSkipIngressCleanupSet(t *testing.T) {
	cfg := defaultKubermaticConfiguration("kubermatic")
	kubermaticIngress := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{
		Namespace: cfg.Namespace,
		Name:      defaulting.DefaultIngressName,
	}}
	dexIngress := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{
		Namespace: DexNamespace,
		Name:      DexChartName,
	}}
	client := fake.NewClientBuilder().WithObjects(kubermaticIngress, dexIngress).Build()

	opt := stack.DeployOptions{KubermaticConfiguration: cfg, SkipIngressCleanup: true}
	if err := cleanupLegacyIngresses(context.Background(), logrus.NewEntry(logrus.New()), client, opt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both Ingress objects must still exist.
	remaining := &networkingv1.Ingress{}
	if err := client.Get(context.Background(), types.NamespacedName{Namespace: cfg.Namespace, Name: defaulting.DefaultIngressName}, remaining); err != nil {
		t.Fatalf("expected kubermatic Ingress to remain, got: %v", err)
	}
	if err := client.Get(context.Background(), types.NamespacedName{Namespace: DexNamespace, Name: DexChartName}, remaining); err != nil {
		t.Fatalf("expected Dex Ingress to remain, got: %v", err)
	}
}

func TestWaitForGatewayAndHTTPRoutesReadyFailsOnUnacceptedDexRoute(t *testing.T) {
	cfg := defaultKubermaticConfiguration("kubermatic")
	gatewayKey := gatewayObjectKey(cfg)
	kubermaticRoute := acceptedHTTPRoute(
		types.NamespacedName{Namespace: cfg.Namespace, Name: defaulting.DefaultHTTPRouteName},
		gatewayKey,
	)
	rejectedDex := rejectedHTTPRoute(
		types.NamespacedName{Namespace: DexNamespace, Name: DexChartName},
		gatewayKey,
		string(gatewayapiv1.RouteReasonNotAllowedByListeners),
	)
	gw := programmedGateway(gatewayKey)

	client := fake.NewClientBuilder().WithObjects(gw, kubermaticRoute, rejectedDex).Build()

	opt := stack.DeployOptions{KubermaticConfiguration: cfg}
	err := waitForGatewayAndHTTPRoutesReadyWithPollConfig(context.Background(), logrus.NewEntry(logrus.New()), client, opt, fastPoll)
	if err == nil {
		t.Fatal("expected error for unaccepted Dex HTTPRoute, got nil")
	}
	if !strings.Contains(err.Error(), DexChartName) {
		t.Fatalf("expected error to mention dex route, got %v", err)
	}
}

func assertIngressGone(t *testing.T, c ctrlruntimeclient.Client, key types.NamespacedName) {
	t.Helper()
	ing := &networkingv1.Ingress{}
	err := c.Get(context.Background(), key, ing)
	if err == nil {
		t.Fatalf("expected Ingress %s to be deleted, but it still exists", key.String())
	}
	if !apierrors.IsNotFound(err) {
		t.Fatalf("unexpected error checking %s: %v", key.String(), err)
	}
}
