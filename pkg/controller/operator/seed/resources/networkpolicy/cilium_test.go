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

package networkpolicy

import (
	"context"
	"testing"

	"k8c.io/reconciler/pkg/reconciling"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSeedApiserverAllowReconciler(t *testing.T) {
	ctx := context.Background()

	gvk := schema.GroupVersionKind{
		Group:   "cilium.io",
		Version: "v2",
		Kind:    ciliumClusterwideNetworkPolicyKind,
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(gvk.GroupVersion().WithKind(gvk.Kind+"List"), &unstructured.UnstructuredList{})

	client := ctrlruntimefakeclient.NewClientBuilder().WithScheme(scheme).Build()

	creators := []reconciling.NamedUnstructuredReconcilerFactory{
		SeedApiserverAllowReconciler(),
	}

	if err := reconciling.ReconcileUnstructureds(ctx, creators, "", client); err != nil {
		t.Fatalf("failed to create policy: %v", err)
	}

	policy := &unstructured.Unstructured{}
	policy.SetGroupVersionKind(gvk)
	if err := client.Get(ctx, types.NamespacedName{Name: CiliumSeedApiserverAllow}, policy); err != nil {
		t.Fatalf("failed to get created policy: %v", err)
	}

	egress, found, err := unstructured.NestedSlice(policy.Object, "spec", "egress")
	if err != nil || !found {
		t.Fatalf("policy has no egress rules: %v", err)
	}
	if len(egress) != 1 {
		t.Fatalf("expected exactly one egress rule, got %d", len(egress))
	}
	entities, found, err := unstructured.NestedStringSlice(egress[0].(map[string]interface{}), "toEntities")
	if err != nil || !found || len(entities) != 1 || entities[0] != "kube-apiserver" {
		t.Fatalf("expected toEntities [kube-apiserver], got %v (err: %v)", entities, err)
	}

	matchLabels, found, err := unstructured.NestedStringMap(policy.Object, "spec", "endpointSelector", "matchLabels")
	if err != nil || !found {
		t.Fatalf("policy has no endpointSelector.matchLabels: %v", err)
	}
	if matchLabels["app"] != "apiserver" {
		t.Fatalf("expected matchLabels app=apiserver, got %v", matchLabels)
	}

	resourceVersion := policy.GetResourceVersion()

	// reconciling again must be a no-op; with the previous typed implementation this
	// step either panicked (cilium >= 1.17) or issued a spurious update (cilium 1.16)
	if err := reconciling.ReconcileUnstructureds(ctx, creators, "", client); err != nil {
		t.Fatalf("failed to reconcile existing policy: %v", err)
	}

	if err := client.Get(ctx, types.NamespacedName{Name: CiliumSeedApiserverAllow}, policy); err != nil {
		t.Fatalf("failed to get reconciled policy: %v", err)
	}

	if rv := policy.GetResourceVersion(); rv != resourceVersion {
		t.Fatalf("expected reconciliation to be a no-op, but resourceVersion changed from %s to %s", resourceVersion, rv)
	}
}

// TestSeedApiserverAllowReconcilerMigratesLegacyPolicy simulates upgrading a seed where the
// policy was created by the previous typed implementation, which stored the matchLabels key
// in Cilium's source-prefixed form ("any:app"). The reconciler is expected to rewrite it once
// to the canonical form ("app") and be a no-op afterwards; both forms are normalized by the
// Cilium agent to the same internal key ("any.app"), so enforcement is unaffected.
func TestSeedApiserverAllowReconcilerMigratesLegacyPolicy(t *testing.T) {
	ctx := context.Background()

	gvk := schema.GroupVersionKind{
		Group:   "cilium.io",
		Version: "v2",
		Kind:    ciliumClusterwideNetworkPolicyKind,
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(gvk.GroupVersion().WithKind(gvk.Kind+"List"), &unstructured.UnstructuredList{})

	legacy := &unstructured.Unstructured{}
	legacy.SetGroupVersionKind(gvk)
	legacy.SetName(CiliumSeedApiserverAllow)
	legacy.Object["spec"] = map[string]interface{}{
		"endpointSelector": map[string]interface{}{
			"matchLabels": map[string]interface{}{
				"any:app": "apiserver",
			},
		},
		"egress": []interface{}{
			map[string]interface{}{
				"toEntities": []interface{}{
					"kube-apiserver",
				},
			},
		},
	}

	client := ctrlruntimefakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(legacy).Build()

	creators := []reconciling.NamedUnstructuredReconcilerFactory{
		SeedApiserverAllowReconciler(),
	}

	if err := reconciling.ReconcileUnstructureds(ctx, creators, "", client); err != nil {
		t.Fatalf("failed to reconcile legacy policy: %v", err)
	}

	policy := &unstructured.Unstructured{}
	policy.SetGroupVersionKind(gvk)
	if err := client.Get(ctx, types.NamespacedName{Name: CiliumSeedApiserverAllow}, policy); err != nil {
		t.Fatalf("failed to get migrated policy: %v", err)
	}

	matchLabels, found, err := unstructured.NestedStringMap(policy.Object, "spec", "endpointSelector", "matchLabels")
	if err != nil || !found {
		t.Fatalf("policy has no endpointSelector.matchLabels: %v", err)
	}
	if matchLabels["app"] != "apiserver" || len(matchLabels) != 1 {
		t.Fatalf("expected matchLabels to be rewritten to {app: apiserver}, got %v", matchLabels)
	}

	resourceVersion := policy.GetResourceVersion()

	if err := reconciling.ReconcileUnstructureds(ctx, creators, "", client); err != nil {
		t.Fatalf("failed to reconcile migrated policy: %v", err)
	}

	if err := client.Get(ctx, types.NamespacedName{Name: CiliumSeedApiserverAllow}, policy); err != nil {
		t.Fatalf("failed to get reconciled policy: %v", err)
	}

	if rv := policy.GetResourceVersion(); rv != resourceVersion {
		t.Fatalf("expected reconciliation after migration to be a no-op, but resourceVersion changed from %s to %s", resourceVersion, rv)
	}
}
