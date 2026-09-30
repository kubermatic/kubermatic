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

package resources

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/kubelb"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const testClusterNamespace = "cluster-test"

func newTestCluster(kubeLB kubermaticv1.KubeLB, status *kubermaticv1.KubeLBStatus) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: kubermaticv1.ClusterSpec{
			KubeLB: &kubeLB,
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: testClusterNamespace,
			KubeLB:        status,
		},
	}
}

func newPolicy(name string) *admissionregistrationv1.ValidatingAdmissionPolicy {
	return &admissionregistrationv1.ValidatingAdmissionPolicy{ObjectMeta: metav1.ObjectMeta{Name: name}}
}

func newPolicyBinding(name string) *admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	return &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{PolicyName: name},
	}
}

// policyAndBindingExist reports whether the policy and the binding with the given name are present.
func policyAndBindingExist(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, name string) (policyFound, bindingFound bool) {
	t.Helper()

	found := func(obj ctrlruntimeclient.Object) bool {
		err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: name}, obj)
		if err != nil && !apierrors.IsNotFound(err) {
			t.Fatalf("failed to get %T %q: %v", obj, name, err)
		}
		return err == nil
	}

	return found(&admissionregistrationv1.ValidatingAdmissionPolicy{}), found(&admissionregistrationv1.ValidatingAdmissionPolicyBinding{})
}

func TestReconcileValidatingAdmissionPolicies(t *testing.T) {
	kkpPolicy := kubelb.GatewayAPIAdmissionPolicyName
	upstreamPolicy := kubelb.UpstreamSafeUpgradesPolicyName

	protected := kubermaticv1.KubeLB{Enabled: true, EnableGatewayAPI: ptr.To(true)}
	optedOut := kubermaticv1.KubeLB{Enabled: true, EnableGatewayAPI: ptr.To(true), DisableGatewayAPIProtection: true}
	gatewayAPIDisabled := kubermaticv1.KubeLB{Enabled: true, EnableGatewayAPI: ptr.To(false)}
	kubeLBDisabled := kubermaticv1.KubeLB{Enabled: false, EnableGatewayAPI: ptr.To(true)}

	testCases := []struct {
		name            string
		cluster         *kubermaticv1.Cluster
		adminFlag       bool
		existingObjects []ctrlruntimeclient.Object

		expectedKKPPolicy      bool
		expectedUpstreamPolicy bool
		expectedStatus         *kubermaticv1.KubeLBStatus
	}{
		{
			name:              "protection on",
			cluster:           newTestCluster(protected, nil),
			expectedKKPPolicy: true,
			expectedStatus:    &kubermaticv1.KubeLBStatus{GatewayAPIProtected: true},
		},
		{
			name:            "the cluster opts out",
			cluster:         newTestCluster(optedOut, nil),
			existingObjects: []ctrlruntimeclient.Object{newPolicy(kkpPolicy), newPolicyBinding(kkpPolicy)},
			expectedStatus:  &kubermaticv1.KubeLBStatus{GatewayAPIProtected: false},
		},
		{
			name:            "an admin disables it",
			cluster:         newTestCluster(protected, nil),
			adminFlag:       true,
			existingObjects: []ctrlruntimeclient.Object{newPolicy(kkpPolicy), newPolicyBinding(kkpPolicy)},
			expectedStatus:  &kubermaticv1.KubeLBStatus{GatewayAPIProtected: false},
		},
		{
			name:            "Gateway API disabled clears a stale status",
			cluster:         newTestCluster(gatewayAPIDisabled, &kubermaticv1.KubeLBStatus{GatewayAPIProtected: true}),
			existingObjects: []ctrlruntimeclient.Object{newPolicy(kkpPolicy), newPolicyBinding(kkpPolicy)},
		},
		{
			// Disabling kubeLB keeps enableGatewayAPI but removes the CCM.
			name:            "kubeLB disabled with Gateway API still set",
			cluster:         newTestCluster(kubeLBDisabled, nil),
			existingObjects: []ctrlruntimeclient.Object{newPolicy(kkpPolicy), newPolicyBinding(kkpPolicy)},
		},
		{
			// A previous removal deleted the binding but failed before the policy.
			name:            "a leftover policy without its binding is removed",
			cluster:         newTestCluster(gatewayAPIDisabled, nil),
			existingObjects: []ctrlruntimeclient.Object{newPolicy(kkpPolicy)},
		},
		{
			name:              "protection on removes the upstream safe-upgrades policy",
			cluster:           newTestCluster(protected, nil),
			existingObjects:   []ctrlruntimeclient.Object{newPolicy(upstreamPolicy), newPolicyBinding(upstreamPolicy)},
			expectedKKPPolicy: true,
			expectedStatus:    &kubermaticv1.KubeLBStatus{GatewayAPIProtected: true},
		},
		{
			// The user manages the CRDs then, and may rely on the upstream policy.
			name:                   "an opted-out cluster keeps the upstream safe-upgrades policy",
			cluster:                newTestCluster(optedOut, nil),
			existingObjects:        []ctrlruntimeclient.Object{newPolicy(upstreamPolicy), newPolicyBinding(upstreamPolicy)},
			expectedUpstreamPolicy: true,
			expectedStatus:         &kubermaticv1.KubeLBStatus{GatewayAPIProtected: false},
		},
		{
			name:                   "Gateway API disabled keeps the upstream safe-upgrades policy",
			cluster:                newTestCluster(gatewayAPIDisabled, nil),
			existingObjects:        []ctrlruntimeclient.Object{newPolicy(upstreamPolicy), newPolicyBinding(upstreamPolicy)},
			expectedUpstreamPolicy: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			seedClient := fake.NewClientBuilder().WithObjects(tc.cluster).Build()
			userClient := fake.NewClientBuilder().WithObjects(tc.existingObjects...).Build()

			r := &reconciler{
				Client:                            userClient,
				seedClient:                        seedClient,
				namespace:                         testClusterNamespace,
				kubeLBDisableGatewayAPIProtection: tc.adminFlag,
			}

			if err := r.reconcileValidatingAdmissionPolicies(ctx, reconcileData{cluster: tc.cluster}); err != nil {
				t.Fatalf("reconcile failed: %v", err)
			}

			policyFound, bindingFound := policyAndBindingExist(ctx, t, userClient, kkpPolicy)
			if policyFound != tc.expectedKKPPolicy || bindingFound != tc.expectedKKPPolicy {
				t.Errorf("expected KKP policy and binding present=%v, got policy=%v binding=%v", tc.expectedKKPPolicy, policyFound, bindingFound)
			}

			policyFound, bindingFound = policyAndBindingExist(ctx, t, userClient, upstreamPolicy)
			if policyFound != tc.expectedUpstreamPolicy || bindingFound != tc.expectedUpstreamPolicy {
				t.Errorf("expected upstream policy and binding present=%v, got policy=%v binding=%v", tc.expectedUpstreamPolicy, policyFound, bindingFound)
			}

			cluster := &kubermaticv1.Cluster{}
			if err := seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(tc.cluster), cluster); err != nil {
				t.Fatalf("failed to get cluster: %v", err)
			}

			switch {
			case tc.expectedStatus == nil && cluster.Status.KubeLB != nil:
				t.Errorf("expected no kubeLB status, got %+v", *cluster.Status.KubeLB)
			case tc.expectedStatus != nil && cluster.Status.KubeLB == nil:
				t.Errorf("expected kubeLB status %+v, got none", *tc.expectedStatus)
			case tc.expectedStatus != nil && *cluster.Status.KubeLB != *tc.expectedStatus:
				t.Errorf("expected kubeLB status %+v, got %+v", *tc.expectedStatus, *cluster.Status.KubeLB)
			}
		})
	}
}

// TestReconcileValidatingAdmissionPoliciesIsIdempotent checks that a second sync leaves the policy
// unchanged. The fake client does not apply apiserver defaults, so it cannot catch a diff against
// defaulted fields; that is covered by setting them explicitly in the policy reconciler.
func TestReconcileValidatingAdmissionPoliciesIsIdempotent(t *testing.T) {
	ctx := context.Background()
	cluster := newTestCluster(kubermaticv1.KubeLB{Enabled: true, EnableGatewayAPI: ptr.To(true)}, nil)

	seedClient := fake.NewClientBuilder().WithObjects(cluster).Build()
	userClient := fake.NewClientBuilder().Build()
	r := &reconciler{Client: userClient, seedClient: seedClient, namespace: testClusterNamespace}

	if err := r.reconcileValidatingAdmissionPolicies(ctx, reconcileData{cluster: cluster}); err != nil {
		t.Fatalf("first reconcile failed: %v", err)
	}

	firstPolicy := &admissionregistrationv1.ValidatingAdmissionPolicy{}
	if err := userClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: kubelb.GatewayAPIAdmissionPolicyName}, firstPolicy); err != nil {
		t.Fatalf("failed to get policy: %v", err)
	}

	if err := r.reconcileValidatingAdmissionPolicies(ctx, reconcileData{cluster: cluster}); err != nil {
		t.Fatalf("second reconcile failed: %v", err)
	}

	secondPolicy := &admissionregistrationv1.ValidatingAdmissionPolicy{}
	if err := userClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: kubelb.GatewayAPIAdmissionPolicyName}, secondPolicy); err != nil {
		t.Fatalf("failed to get policy: %v", err)
	}

	if firstPolicy.ResourceVersion != secondPolicy.ResourceVersion {
		t.Errorf("expected the second reconcile to leave the policy unchanged, resourceVersion went from %s to %s", firstPolicy.ResourceVersion, secondPolicy.ResourceVersion)
	}
}
