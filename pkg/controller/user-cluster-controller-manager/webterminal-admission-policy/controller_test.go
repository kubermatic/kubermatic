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

package webterminaladmissionpolicy

import (
	"context"
	"testing"

	"go.uber.org/zap"

	webterminalresources "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/webterminal-admission-policy/resources"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const testSession = resources.WebTerminalSessionNamePrefix + "5d41402abc4b2a76b9719d911017c592"

func newTestReconciler(enabled, paused bool, existingObjects ...ctrlruntimeclient.Object) (*reconciler, ctrlruntimeclient.Client) {
	client := fake.NewClientBuilder().WithObjects(existingObjects...).Build()

	return &reconciler{
		log:             zap.NewNop().Sugar(),
		client:          client,
		clusterIsPaused: func(context.Context) (bool, error) { return paused, nil },
		enabled:         enabled,
	}, client
}

func serviceAccount(namespace, name string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name}}
}

// existingPolicies returns the policies and bindings as the controller creates them.
func existingPolicies(t *testing.T) []ctrlruntimeclient.Object {
	t.Helper()

	r, client := newTestReconciler(true, false)
	if _, err := r.Reconcile(context.Background(), reconcile.Request{}); err != nil {
		t.Fatalf("failed to create the policies: %v", err)
	}

	objects := []ctrlruntimeclient.Object{}
	for _, name := range webterminalresources.PolicyNames() {
		policy := &admissionregistrationv1.ValidatingAdmissionPolicy{}
		binding := &admissionregistrationv1.ValidatingAdmissionPolicyBinding{}
		for _, object := range []ctrlruntimeclient.Object{policy, binding} {
			if err := client.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: name}, object); err != nil {
				t.Fatalf("expected %T %s: %v", object, name, err)
			}
			object.SetResourceVersion("")
			objects = append(objects, object)
		}
	}

	return objects
}

// countPolicies returns how many of the policies and of their bindings are in the cluster.
func countPolicies(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client) (policies, bindings int) {
	t.Helper()

	for _, name := range webterminalresources.PolicyNames() {
		err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: name}, &admissionregistrationv1.ValidatingAdmissionPolicy{})
		if err == nil {
			policies++
		} else if !apierrors.IsNotFound(err) {
			t.Fatalf("failed to get policy %s: %v", name, err)
		}

		err = client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: name}, &admissionregistrationv1.ValidatingAdmissionPolicyBinding{})
		if err == nil {
			bindings++
		} else if !apierrors.IsNotFound(err) {
			t.Fatalf("failed to get binding %s: %v", name, err)
		}
	}

	return policies, bindings
}

func TestReconcile(t *testing.T) {
	all := len(webterminalresources.PolicyNames())

	testcases := []struct {
		name string
		// enabled is the -web-terminal-serviceaccount flag
		enabled bool
		paused  bool
		// policiesExist puts the policies and the bindings into the cluster before the reconciliation
		policiesExist   bool
		serviceAccounts []*corev1.ServiceAccount
		expected        int
	}{
		{
			name:     "enabled: the policies are created before any session exists",
			enabled:  true,
			expected: all,
		},
		{
			name:          "enabled: existing policies are kept",
			enabled:       true,
			policiesExist: true,
			expected:      all,
		},
		{
			name:     "not enabled, no session: nothing is created",
			expected: 0,
		},
		{
			name:          "not enabled, no session: the policies are removed",
			policiesExist: true,
			expected:      0,
		},
		{
			name:            "not enabled, but a session exists: its policies are kept",
			policiesExist:   true,
			serviceAccounts: []*corev1.ServiceAccount{serviceAccount(metav1.NamespaceSystem, testSession)},
			expected:        all,
		},
		{
			// the KKP API can have the feature gate before this controller is restarted with the flag
			name:            "not enabled, but a session exists: the policies are created for it",
			serviceAccounts: []*corev1.ServiceAccount{serviceAccount(metav1.NamespaceSystem, testSession)},
			expected:        all,
		},
		{
			name:          "not enabled: other ServiceAccounts are no sessions",
			policiesExist: true,
			serviceAccounts: []*corev1.ServiceAccount{
				serviceAccount(metav1.NamespaceSystem, "default"),
				serviceAccount(metav1.NamespaceSystem, "coredns"),
				serviceAccount(metav1.NamespaceDefault, testSession),
			},
			expected: 0,
		},
		{
			name:     "paused: nothing is created",
			enabled:  true,
			paused:   true,
			expected: 0,
		},
		{
			name:          "paused: nothing is removed",
			paused:        true,
			policiesExist: true,
			expected:      all,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			objects := []ctrlruntimeclient.Object{}
			if tc.policiesExist {
				objects = append(objects, existingPolicies(t)...)
			}
			for _, sa := range tc.serviceAccounts {
				objects = append(objects, sa)
			}
			r, client := newTestReconciler(tc.enabled, tc.paused, objects...)

			if _, err := r.Reconcile(ctx, reconcile.Request{}); err != nil {
				t.Fatalf("reconcile failed: %v", err)
			}

			policies, bindings := countPolicies(ctx, t, client)
			if policies != tc.expected || bindings != tc.expected {
				t.Errorf("expected %d policies and bindings, got %d policies and %d bindings", tc.expected, policies, bindings)
			}
		})
	}
}

// TestReconcileRestoresPolicies checks that a policy that no longer refuses anything, a binding that
// no longer enforces its policy and a deleted binding are put back.
func TestReconcileRestoresPolicies(t *testing.T) {
	ctx := context.Background()
	objects := existingPolicies(t)
	r, client := newTestReconciler(true, false, objects...)

	expected := map[string]admissionregistrationv1.ValidatingAdmissionPolicySpec{}
	for _, object := range objects {
		if policy, ok := object.(*admissionregistrationv1.ValidatingAdmissionPolicy); ok {
			expected[policy.Name] = *policy.Spec.DeepCopy()
		}
	}

	policy := &admissionregistrationv1.ValidatingAdmissionPolicy{}
	if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: webterminalresources.AccessPolicyName}, policy); err != nil {
		t.Fatal(err)
	}
	policy.Spec.Validations = []admissionregistrationv1.Validation{{Expression: "true"}}
	if err := client.Update(ctx, policy); err != nil {
		t.Fatal(err)
	}

	binding := &admissionregistrationv1.ValidatingAdmissionPolicyBinding{}
	if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: webterminalresources.ServiceAccountPodsPolicyName}, binding); err != nil {
		t.Fatal(err)
	}
	binding.Spec.ValidationActions = []admissionregistrationv1.ValidationAction{admissionregistrationv1.Warn}
	if err := client.Update(ctx, binding); err != nil {
		t.Fatal(err)
	}

	if err := client.Delete(ctx, &admissionregistrationv1.ValidatingAdmissionPolicyBinding{ObjectMeta: metav1.ObjectMeta{Name: webterminalresources.ServiceAccountSecretsPolicyName}}); err != nil {
		t.Fatal(err)
	}

	if _, err := r.Reconcile(ctx, reconcile.Request{}); err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	for name, spec := range expected {
		policy := &admissionregistrationv1.ValidatingAdmissionPolicy{}
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: name}, policy); err != nil {
			t.Fatalf("expected policy %s: %v", name, err)
		}
		if len(policy.Spec.Validations) != len(spec.Validations) || policy.Spec.Validations[0].Expression != spec.Validations[0].Expression {
			t.Errorf("policy %s: expected the validations to be put back, got %+v", name, policy.Spec.Validations)
		}

		binding := &admissionregistrationv1.ValidatingAdmissionPolicyBinding{}
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: name}, binding); err != nil {
			t.Fatalf("expected binding %s: %v", name, err)
		}
		if binding.Spec.PolicyName != name || len(binding.Spec.ValidationActions) != 1 || binding.Spec.ValidationActions[0] != admissionregistrationv1.Deny {
			t.Errorf("binding %s: expected it to deny for its policy, got %+v", name, binding.Spec)
		}
	}
}

// TestReconcileIsIdempotent checks that a second reconciliation writes nothing.
func TestReconcileIsIdempotent(t *testing.T) {
	ctx := context.Background()
	r, client := newTestReconciler(true, false)

	resourceVersions := func() map[string]string {
		versions := map[string]string{}
		for _, name := range webterminalresources.PolicyNames() {
			policy := &admissionregistrationv1.ValidatingAdmissionPolicy{}
			binding := &admissionregistrationv1.ValidatingAdmissionPolicyBinding{}
			for _, object := range []ctrlruntimeclient.Object{policy, binding} {
				if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: name}, object); err != nil {
					t.Fatalf("expected %T %s: %v", object, name, err)
				}
			}
			versions["policy/"+name] = policy.ResourceVersion
			versions["binding/"+name] = binding.ResourceVersion
		}

		return versions
	}

	if _, err := r.Reconcile(ctx, reconcile.Request{}); err != nil {
		t.Fatalf("first reconcile failed: %v", err)
	}
	before := resourceVersions()

	if _, err := r.Reconcile(ctx, reconcile.Request{}); err != nil {
		t.Fatalf("second reconcile failed: %v", err)
	}
	for key, version := range resourceVersions() {
		if before[key] != version {
			t.Errorf("%s was written by the second reconciliation", key)
		}
	}
}

// TestReconcileRemovesPoliciesAfterLastSession checks that, once the mode is switched off, the
// policies stay for the sessions that still exist and go with the last of them.
func TestReconcileRemovesPoliciesAfterLastSession(t *testing.T) {
	ctx := context.Background()
	all := len(webterminalresources.PolicyNames())
	other := serviceAccount(metav1.NamespaceSystem, resources.WebTerminalSessionNamePrefix+"another")
	objects := append(existingPolicies(t), serviceAccount(metav1.NamespaceSystem, testSession), other)
	r, client := newTestReconciler(false, false, objects...)

	expect := func(expected int, when string) {
		t.Helper()

		if _, err := r.Reconcile(ctx, reconcile.Request{}); err != nil {
			t.Fatalf("reconcile failed: %v", err)
		}
		if policies, bindings := countPolicies(ctx, t, client); policies != expected || bindings != expected {
			t.Errorf("%s: expected %d policies and bindings, got %d and %d", when, expected, policies, bindings)
		}
	}

	expect(all, "two sessions")

	if err := client.Delete(ctx, serviceAccount(metav1.NamespaceSystem, testSession)); err != nil {
		t.Fatal(err)
	}
	expect(all, "one session")

	if err := client.Delete(ctx, other); err != nil {
		t.Fatal(err)
	}
	expect(0, "no session")
}

func TestIsSessionServiceAccount(t *testing.T) {
	testcases := []struct {
		name           string
		serviceAccount *corev1.ServiceAccount
		expected       bool
	}{
		{"the ServiceAccount of a session", serviceAccount(metav1.NamespaceSystem, testSession), true},
		{"another ServiceAccount in kube-system", serviceAccount(metav1.NamespaceSystem, "webterminal"), false},
		{"the name of a session in another namespace", serviceAccount(metav1.NamespaceDefault, testSession), false},
		{
			// the name decides: a label can be removed from the ServiceAccount of a session and put on any other
			name: "another ServiceAccount with the label of the web terminal",
			serviceAccount: &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Namespace: metav1.NamespaceSystem,
				Name:      "default",
				Labels:    map[string]string{resources.AppLabelKey: resources.WebTerminalAppLabelValue},
			}},
			expected: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isSessionServiceAccount(tc.serviceAccount); got != tc.expected {
				t.Errorf("expected %t, got %t", tc.expected, got)
			}
		})
	}
}
