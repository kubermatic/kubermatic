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

package kubelb

import (
	"strings"
	"testing"

	"k8c.io/kubermatic/v2/pkg/resources"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
)

func TestGatewayAPIValidatingAdmissionPolicyReconciler(t *testing.T) {
	name, reconciler := GatewayAPIValidatingAdmissionPolicyReconciler()()
	if name != GatewayAPIAdmissionPolicyName {
		t.Errorf("expected policy name %q, got %q", GatewayAPIAdmissionPolicyName, name)
	}

	policy, err := reconciler(&admissionregistrationv1.ValidatingAdmissionPolicy{})
	if err != nil {
		t.Fatalf("failed to reconcile policy: %v", err)
	}

	// A cluster must not silently lose the guard rail if the policy cannot be evaluated.
	if policy.Spec.FailurePolicy == nil || *policy.Spec.FailurePolicy != admissionregistrationv1.Fail {
		t.Error("expected failurePolicy Fail")
	}

	// The policy has to reject writes rather than allow them, so the only validation must be a
	// constant denial.
	if len(policy.Spec.Validations) != 1 {
		t.Fatalf("expected exactly 1 validation, got %d", len(policy.Spec.Validations))
	}
	if expr := policy.Spec.Validations[0].Expression; expr != "false" {
		t.Errorf("expected validation expression %q, got %q", "false", expr)
	}

	// Matching anything other than these CRD operations would either miss the case we care about or
	// interfere with unrelated requests.
	if len(policy.Spec.MatchConstraints.ResourceRules) != 1 {
		t.Fatalf("expected exactly 1 resource rule, got %d", len(policy.Spec.MatchConstraints.ResourceRules))
	}
	rule := policy.Spec.MatchConstraints.ResourceRules[0]
	if got, want := rule.Resources, []string{"customresourcedefinitions"}; !equalStrings(got, want) {
		t.Errorf("expected resources %v, got %v", want, got)
	}
	wantOps := []admissionregistrationv1.OperationType{
		admissionregistrationv1.Create,
		admissionregistrationv1.Update,
		admissionregistrationv1.Delete,
	}
	if len(rule.Operations) != len(wantOps) {
		t.Fatalf("expected operations %v, got %v", wantOps, rule.Operations)
	}
	for i, op := range wantOps {
		if rule.Operations[i] != op {
			t.Errorf("expected operation %d to be %q, got %q", i, op, rule.Operations[i])
		}
	}

	// Both match conditions are load bearing: without the group check the policy would block every
	// CRD in the cluster, and without the username check it would block the kubeLB CCM itself.
	conditions := map[string]string{}
	for _, condition := range policy.Spec.MatchConditions {
		conditions[condition.Name] = condition.Expression
	}

	groupExpr, ok := conditions["gateway-api-crds-only"]
	if !ok {
		t.Fatal("expected a gateway-api-crds-only match condition")
	}
	// The policy also matches DELETE, where the request carries no object. Reading object.spec.group
	// unconditionally would make the expression fail on every DELETE, and since failurePolicy is Fail,
	// that would reject deleting any CRD in the cluster.
	if !strings.Contains(groupExpr, "object == null ? oldObject : object") {
		t.Errorf("expected group match condition to fall back to oldObject for DELETE, got %q", groupExpr)
	}
	for _, group := range gatewayAPIGroups {
		if !strings.Contains(groupExpr, group) {
			t.Errorf("expected group match condition to cover %q, got %q", group, groupExpr)
		}
	}

	ccmExpr, ok := conditions["exclude-kubelb-ccm"]
	if !ok {
		t.Fatal("expected an exclude-kubelb-ccm match condition")
	}
	if !strings.Contains(ccmExpr, resources.KubeLBCCMCertUsername) {
		t.Errorf("expected CCM match condition to reference %q, got %q", resources.KubeLBCCMCertUsername, ccmExpr)
	}
	if !strings.Contains(ccmExpr, "!=") {
		t.Errorf("expected CCM match condition to exempt the CCM, got %q", ccmExpr)
	}
}

func TestGatewayAPIValidatingAdmissionPolicyBindingReconciler(t *testing.T) {
	name, reconciler := GatewayAPIValidatingAdmissionPolicyBindingReconciler()()
	if name != GatewayAPIAdmissionPolicyName {
		t.Errorf("expected binding name %q, got %q", GatewayAPIAdmissionPolicyName, name)
	}

	binding, err := reconciler(&admissionregistrationv1.ValidatingAdmissionPolicyBinding{})
	if err != nil {
		t.Fatalf("failed to reconcile binding: %v", err)
	}

	// A binding pointing at another policy, or one that only audits, would leave the CRDs unguarded.
	if binding.Spec.PolicyName != GatewayAPIAdmissionPolicyName {
		t.Errorf("expected binding to reference policy %q, got %q", GatewayAPIAdmissionPolicyName, binding.Spec.PolicyName)
	}
	if len(binding.Spec.ValidationActions) != 1 || binding.Spec.ValidationActions[0] != admissionregistrationv1.Deny {
		t.Errorf("expected validationActions [Deny], got %v", binding.Spec.ValidationActions)
	}
}

func TestGatewayAPIAdmissionPolicyResourcesForDeletion(t *testing.T) {
	objects := GatewayAPIAdmissionPolicyResourcesForDeletion()
	if len(objects) != 2 {
		t.Fatalf("expected 2 objects to delete, got %d", len(objects))
	}

	// The binding has to go first, otherwise the policy would briefly stay in effect without it.
	if _, ok := objects[0].(*admissionregistrationv1.ValidatingAdmissionPolicyBinding); !ok {
		t.Errorf("expected the binding to be deleted first, got %T", objects[0])
	}
	if _, ok := objects[1].(*admissionregistrationv1.ValidatingAdmissionPolicy); !ok {
		t.Errorf("expected the policy to be deleted second, got %T", objects[1])
	}

	for _, object := range objects {
		if object.GetName() != GatewayAPIAdmissionPolicyName {
			t.Errorf("expected name %q, got %q", GatewayAPIAdmissionPolicyName, object.GetName())
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
