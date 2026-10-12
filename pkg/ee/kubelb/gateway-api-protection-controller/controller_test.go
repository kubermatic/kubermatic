//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2026 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package gatewayapiprotectioncontroller

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	gatewayapiresources "k8c.io/kubermatic/v2/pkg/ee/kubelb/gateway-api-protection-controller/resources"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func newTestCluster(kubeLB kubermaticv1.KubeLB, status *kubermaticv1.KubeLBStatus) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: kubermaticv1.ClusterSpec{
			KubeLB: &kubeLB,
		},
		Status: kubermaticv1.ClusterStatus{
			KubeLB: status,
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

// newTestReconciler returns a reconciler backed by fake seed and user-cluster clients, with the cluster
// in the seed and existingObjects already present in the user cluster.
func newTestReconciler(cluster *kubermaticv1.Cluster, adminDisabled bool, existingObjects ...ctrlruntimeclient.Object) (r *reconciler, seedClient, userClient ctrlruntimeclient.Client) {
	seedClient = fake.NewClientBuilder().WithObjects(cluster).Build()
	userClient = fake.NewClientBuilder().WithObjects(existingObjects...).Build()

	return &reconciler{seedClient: seedClient, userClient: userClient, adminDisabled: adminDisabled}, seedClient, userClient
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

// expectKubeLBStatus fails the test unless the stored cluster has the given kubeLB status; nil means none.
func expectKubeLBStatus(ctx context.Context, t *testing.T, seedClient ctrlruntimeclient.Client, key *kubermaticv1.Cluster, expected *kubermaticv1.KubeLBStatus) {
	t.Helper()

	cluster := &kubermaticv1.Cluster{}
	if err := seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(key), cluster); err != nil {
		t.Fatalf("failed to get cluster: %v", err)
	}

	switch {
	case expected == nil && cluster.Status.KubeLB != nil:
		t.Errorf("expected no kubeLB status, got %+v", *cluster.Status.KubeLB)
	case expected != nil && cluster.Status.KubeLB == nil:
		t.Errorf("expected kubeLB status %+v, got none", *expected)
	case expected != nil && *cluster.Status.KubeLB != *expected:
		t.Errorf("expected kubeLB status %+v, got %+v", *expected, *cluster.Status.KubeLB)
	}
}

func TestReconcile(t *testing.T) {
	kkpPolicy := gatewayapiresources.GatewayAPIAdmissionPolicyName
	upstreamPolicy := gatewayapiresources.UpstreamSafeUpgradesPolicyName

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
			r, seedClient, userClient := newTestReconciler(tc.cluster, tc.adminFlag, tc.existingObjects...)

			if err := r.reconcile(ctx, tc.cluster); err != nil {
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

			expectKubeLBStatus(ctx, t, seedClient, tc.cluster, tc.expectedStatus)
		})
	}
}

// TestReconcileIsIdempotent checks that a second sync leaves the policy unchanged. The fake client does
// not apply apiserver defaults, so it cannot catch a diff against defaulted fields; that is covered by
// setting them explicitly in the policy reconciler.
func TestReconcileIsIdempotent(t *testing.T) {
	ctx := context.Background()
	cluster := newTestCluster(kubermaticv1.KubeLB{Enabled: true, EnableGatewayAPI: ptr.To(true)}, nil)
	r, _, userClient := newTestReconciler(cluster, false)

	if err := r.reconcile(ctx, cluster); err != nil {
		t.Fatalf("first reconcile failed: %v", err)
	}

	firstPolicy := &admissionregistrationv1.ValidatingAdmissionPolicy{}
	if err := userClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: gatewayapiresources.GatewayAPIAdmissionPolicyName}, firstPolicy); err != nil {
		t.Fatalf("failed to get policy: %v", err)
	}

	if err := r.reconcile(ctx, cluster); err != nil {
		t.Fatalf("second reconcile failed: %v", err)
	}

	secondPolicy := &admissionregistrationv1.ValidatingAdmissionPolicy{}
	if err := userClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: gatewayapiresources.GatewayAPIAdmissionPolicyName}, secondPolicy); err != nil {
		t.Fatalf("failed to get policy: %v", err)
	}

	if firstPolicy.ResourceVersion != secondPolicy.ResourceVersion {
		t.Errorf("expected the second reconcile to leave the policy unchanged, resourceVersion went from %s to %s", firstPolicy.ResourceVersion, secondPolicy.ResourceVersion)
	}
}

func TestKubeLBGatewayAPIEnabled(t *testing.T) {
	clusterWith := func(kubeLBEnabled bool, gatewayAPI *bool) *kubermaticv1.Cluster {
		return &kubermaticv1.Cluster{
			Spec: kubermaticv1.ClusterSpec{
				KubeLB: &kubermaticv1.KubeLB{Enabled: kubeLBEnabled, EnableGatewayAPI: gatewayAPI},
			},
		}
	}

	testCases := []struct {
		name     string
		cluster  *kubermaticv1.Cluster
		expected bool
	}{
		{
			name:     "kubeLB with Gateway API",
			cluster:  clusterWith(true, ptr.To(true)),
			expected: true,
		},
		{
			name:     "kubeLB without Gateway API",
			cluster:  clusterWith(true, ptr.To(false)),
			expected: false,
		},
		{
			name:     "kubeLB with Gateway API unset",
			cluster:  clusterWith(true, nil),
			expected: false,
		},
		{
			// The CCM is gone, so the policy must go too.
			name:     "kubeLB disabled with Gateway API still set",
			cluster:  clusterWith(false, ptr.To(true)),
			expected: false,
		},
		{
			name:     "a cluster without kubeLB settings",
			cluster:  &kubermaticv1.Cluster{},
			expected: false,
		},
		{
			name:     "a nil cluster",
			cluster:  nil,
			expected: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := kubeLBGatewayAPIEnabled(test.cluster); got != test.expected {
				t.Errorf("expected %v, got %v", test.expected, got)
			}
		})
	}
}

func TestGatewayAPIProtectionDisabled(t *testing.T) {
	clusterWith := func(disabled bool) *kubermaticv1.Cluster {
		return &kubermaticv1.Cluster{
			Spec: kubermaticv1.ClusterSpec{
				KubeLB: &kubermaticv1.KubeLB{DisableGatewayAPIProtection: disabled},
			},
		}
	}

	testCases := []struct {
		name string
		// adminFlag is the -kubelb-disable-gateway-api-protection flag.
		adminFlag bool
		cluster   *kubermaticv1.Cluster
		expected  bool
	}{
		{
			name:     "nothing configured keeps the guard",
			cluster:  clusterWith(false),
			expected: false,
		},
		{
			name:      "an admin disables it for every cluster",
			adminFlag: true,
			cluster:   clusterWith(false),
			expected:  true,
		},
		{
			name:     "a cluster can opt out on its own",
			cluster:  clusterWith(true),
			expected: true,
		},
		{
			// Both disabling it must not cancel out.
			name:      "an admin and the cluster both disable it",
			adminFlag: true,
			cluster:   clusterWith(true),
			expected:  true,
		},
		{
			// The common case; must not panic.
			name:     "a cluster without kubeLB settings keeps the guard",
			cluster:  &kubermaticv1.Cluster{},
			expected: false,
		},
		{
			name:     "a nil cluster keeps the guard",
			cluster:  nil,
			expected: false,
		},
		{
			name:      "a nil cluster still honours the admin setting",
			adminFlag: true,
			cluster:   nil,
			expected:  true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			r := &reconciler{adminDisabled: test.adminFlag}

			if got := r.gatewayAPIProtectionDisabled(test.cluster); got != test.expected {
				t.Errorf("expected %v, got %v", test.expected, got)
			}
		})
	}
}
