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
	"fmt"
	"strings"

	"k8c.io/kubermatic/v2/pkg/resources"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling/modifier"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GatewayAPIAdmissionPolicyName is the name of both the policy and its binding.
const GatewayAPIAdmissionPolicyName = "kubermatic-kubelb-gateway-api-crds"

// UpstreamSafeUpgradesPolicyName is the name of both the safe-upgrades policy and its binding that ship
// with every upstream Gateway API release.
const UpstreamSafeUpgradesPolicyName = "safe-upgrades.gateway.networking.k8s.io"

// managedByLabelValue marks KKP as the owner of the admission policy objects.
const managedByLabelValue = "kkp"

// gatewayAPIGroups are the API groups whose CRDs the kubeLB CCM installs. Matching is by group only,
// not channel: the channel annotation can be set by whoever submits the CRD.
var gatewayAPIGroups = []string{
	"gateway.networking.k8s.io",
	"gateway.networking.x-k8s.io",
}

func managedByLabels() map[string]string {
	return map[string]string{modifier.ManagedByLabel: managedByLabelValue}
}

// GatewayAPIValidatingAdmissionPolicyReconciler returns the policy that rejects Gateway API CRD writes
// from everyone except the kubeLB CCM, so no other bundle can replace the CCM's CRDs. The upstream
// safe-upgrades policy such a bundle brings along is removed separately by the controller.
func GatewayAPIValidatingAdmissionPolicyReconciler() kkpreconciling.NamedValidatingAdmissionPolicyReconcilerFactory {
	return func() (string, kkpreconciling.ValidatingAdmissionPolicyReconciler) {
		return GatewayAPIAdmissionPolicyName, func(policy *admissionregistrationv1.ValidatingAdmissionPolicy) (*admissionregistrationv1.ValidatingAdmissionPolicy, error) {
			policy.Labels = resources.BaseAppLabels(resources.KubeLBAppName, managedByLabels())
			policy.Spec = admissionregistrationv1.ValidatingAdmissionPolicySpec{
				FailurePolicy: ptr.To(admissionregistrationv1.Fail),
				MatchConstraints: &admissionregistrationv1.MatchResources{
					// Set apiserver defaults explicitly, or the reconciler updates the policy
					// in an endless loop.
					MatchPolicy:       ptr.To(admissionregistrationv1.Equivalent),
					NamespaceSelector: &metav1.LabelSelector{},
					ObjectSelector:    &metav1.LabelSelector{},
					ResourceRules: []admissionregistrationv1.NamedRuleWithOperations{
						{
							RuleWithOperations: admissionregistrationv1.RuleWithOperations{
								// Deleting a CRD would also delete every Gateway and Route of that kind.
								Operations: []admissionregistrationv1.OperationType{
									admissionregistrationv1.Create,
									admissionregistrationv1.Update,
									admissionregistrationv1.Delete,
								},
								Rule: admissionregistrationv1.Rule{
									APIGroups:   []string{apiextensionsv1.GroupName},
									APIVersions: []string{apiextensionsv1.SchemeGroupVersion.Version},
									Resources:   []string{"customresourcedefinitions"},
									Scope:       ptr.To(admissionregistrationv1.AllScopes),
								},
							},
						},
					},
				},
				MatchConditions: []admissionregistrationv1.MatchCondition{
					{
						// Only guard the Gateway API CRDs.
						Name:       "gateway-api-crds-only",
						Expression: gatewayAPIGroupsMatchExpression(),
					},
					{
						// The CCM owns these CRDs and must keep write access.
						Name:       "exclude-kubelb-ccm",
						Expression: fmt.Sprintf("request.userInfo.username != %q", resources.KubeLBCCMCertUsername),
					},
				},
				Validations: []admissionregistrationv1.Validation{
					{
						Expression: "false",
						Reason:     ptr.To(metav1.StatusReasonForbidden),
						// Field paths use the JSON spelling "kubelb"; the CRD schema silently prunes
						// "kubeLB", so a copied path must be exact.
						Message: "Gateway API CRDs in this cluster are installed and managed by KubeLB, because " +
							"Gateway API support is enabled for this cluster (spec.kubelb.enableGatewayAPI). " +
							"Installing or modifying them breaks the KubeLB CCM. Set " +
							"spec.kubelb.disableGatewayAPIProtection to true to manage the Gateway API CRDs " +
							"yourself, or disable spec.kubelb.enableGatewayAPI to stop KubeLB from using them.",
					},
				},
			}

			return policy, nil
		}
	}
}

// GatewayAPIValidatingAdmissionPolicyBindingReconciler returns the binding that enforces the policy.
func GatewayAPIValidatingAdmissionPolicyBindingReconciler() kkpreconciling.NamedValidatingAdmissionPolicyBindingReconcilerFactory {
	return func() (string, kkpreconciling.ValidatingAdmissionPolicyBindingReconciler) {
		return GatewayAPIAdmissionPolicyName, func(binding *admissionregistrationv1.ValidatingAdmissionPolicyBinding) (*admissionregistrationv1.ValidatingAdmissionPolicyBinding, error) {
			binding.Labels = resources.BaseAppLabels(resources.KubeLBAppName, managedByLabels())
			binding.Spec = admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
				PolicyName:        GatewayAPIAdmissionPolicyName,
				ValidationActions: []admissionregistrationv1.ValidationAction{admissionregistrationv1.Deny},
			}

			return binding, nil
		}
	}
}

// GatewayAPIAdmissionPolicyResourcesForDeletion returns the objects to delete, binding first: that
// stops enforcement at once and never leaves a binding pointing at a deleted policy.
func GatewayAPIAdmissionPolicyResourcesForDeletion() []ctrlruntimeclient.Object {
	return []ctrlruntimeclient.Object{
		&admissionregistrationv1.ValidatingAdmissionPolicyBinding{
			ObjectMeta: metav1.ObjectMeta{Name: GatewayAPIAdmissionPolicyName},
		},
		&admissionregistrationv1.ValidatingAdmissionPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: GatewayAPIAdmissionPolicyName},
		},
	}
}

// UpstreamSafeUpgradesResourcesForDeletion returns the upstream safe-upgrades objects, binding first.
func UpstreamSafeUpgradesResourcesForDeletion() []ctrlruntimeclient.Object {
	return []ctrlruntimeclient.Object{
		&admissionregistrationv1.ValidatingAdmissionPolicyBinding{
			ObjectMeta: metav1.ObjectMeta{Name: UpstreamSafeUpgradesPolicyName},
		},
		&admissionregistrationv1.ValidatingAdmissionPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: UpstreamSafeUpgradesPolicyName},
		},
	}
}

// gatewayAPIGroupsMatchExpression matches CRDs of the Gateway API groups. DELETE has no `object`, only
// `oldObject`; reading `object` directly would fail and, with failurePolicy Fail, block every CRD delete.
func gatewayAPIGroupsMatchExpression() string {
	const crd = "(object == null ? oldObject : object)"

	clauses := make([]string, 0, len(gatewayAPIGroups))
	for _, group := range gatewayAPIGroups {
		clauses = append(clauses, fmt.Sprintf("%s.spec.group == %q", crd, group))
	}

	return strings.Join(clauses, " || ")
}
