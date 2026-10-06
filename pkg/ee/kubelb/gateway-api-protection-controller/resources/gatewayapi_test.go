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

package resources

import (
	"slices"
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

	// Fail closed if the policy cannot be evaluated.
	if policy.Spec.FailurePolicy == nil || *policy.Spec.FailurePolicy != admissionregistrationv1.Fail {
		t.Error("expected failurePolicy Fail")
	}

	// The only validation must be a constant denial.
	if len(policy.Spec.Validations) != 1 {
		t.Fatalf("expected exactly 1 validation, got %d", len(policy.Spec.Validations))
	}
	if expr := policy.Spec.Validations[0].Expression; expr != "false" {
		t.Errorf("expected validation expression %q, got %q", "false", expr)
	}

	// Match exactly CRD CREATE, UPDATE and DELETE.
	if len(policy.Spec.MatchConstraints.ResourceRules) != 1 {
		t.Fatalf("expected exactly 1 resource rule, got %d", len(policy.Spec.MatchConstraints.ResourceRules))
	}
	rule := policy.Spec.MatchConstraints.ResourceRules[0]
	if got, want := rule.Resources, []string{"customresourcedefinitions"}; !slices.Equal(got, want) {
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

	// Without the group check every CRD is blocked; without the username check the CCM is.
	conditions := map[string]string{}
	for _, condition := range policy.Spec.MatchConditions {
		conditions[condition.Name] = condition.Expression
	}

	groupExpr, ok := conditions["gateway-api-crds-only"]
	if !ok {
		t.Fatal("expected a gateway-api-crds-only match condition")
	}
	// DELETE has no object; reading it directly would block every CRD delete.
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

	// Must enforce our policy in Deny mode.
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

	// Binding first, so no binding is left pointing at a deleted policy.
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

func TestUpstreamSafeUpgradesResourcesForDeletion(t *testing.T) {
	objects := UpstreamSafeUpgradesResourcesForDeletion()
	if len(objects) != 2 {
		t.Fatalf("expected 2 objects to delete, got %d", len(objects))
	}

	// Binding first, so no binding is left pointing at a deleted policy.
	if _, ok := objects[0].(*admissionregistrationv1.ValidatingAdmissionPolicyBinding); !ok {
		t.Errorf("expected the binding to be deleted first, got %T", objects[0])
	}
	if _, ok := objects[1].(*admissionregistrationv1.ValidatingAdmissionPolicy); !ok {
		t.Errorf("expected the policy to be deleted second, got %T", objects[1])
	}

	for _, object := range objects {
		if object.GetName() != UpstreamSafeUpgradesPolicyName {
			t.Errorf("expected name %q, got %q", UpstreamSafeUpgradesPolicyName, object.GetName())
		}
	}
}
