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

// GatewayAPIAdmissionPolicyName is the name of both the ValidatingAdmissionPolicy and its binding that
// reserve the Gateway API CRDs for the kubeLB CCM.
const GatewayAPIAdmissionPolicyName = "kubermatic-kubelb-gateway-api-crds"

// managedByLabelValue identifies KKP as the owner of the admission policy objects, so that it is
// obvious where they come from when they show up in a user cluster.
const managedByLabelValue = "kkp"

// gatewayAPIGroups are the API groups whose CRDs the kubeLB CCM installs and manages while Gateway API
// support is enabled for a cluster. Both groups are covered because the CCM's bundle spans both.
//
// The release channel is deliberately not part of this: which channel the CCM installs is its own
// configuration, the annotation recording it is writable by whoever submits the CRD, and both channels
// ship the same upstream safe-upgrades policy. Anything that keys off the channel would be guessing.
var gatewayAPIGroups = []string{
	"gateway.networking.k8s.io",
	"gateway.networking.x-k8s.io",
}

func managedByLabels() map[string]string {
	return map[string]string{modifier.ManagedByLabel: managedByLabelValue}
}

// GatewayAPIValidatingAdmissionPolicyReconciler returns the ValidatingAdmissionPolicy that rejects
// Gateway API CRD writes from everyone except the kubeLB CCM.
//
// While Gateway API support is enabled, the CCM owns these CRDs and installs them from the experimental
// channel. Installing a different set on top of them, in particular the standard channel, breaks the
// CCM: the Gateway API ships its own safe-upgrades ValidatingAdmissionPolicy alongside those manifests,
// which then rejects the CCM's experimental writes with "Installing experimental CRDs on top of
// standard channel CRDs is prohibited by default", leaving the CCM in a crash loop.
//
// This is a guard rail rather than a guarantee, because a cluster-admin of the user cluster can delete
// the policy. It is reconciled continuously, so it comes back.
func GatewayAPIValidatingAdmissionPolicyReconciler() kkpreconciling.NamedValidatingAdmissionPolicyReconcilerFactory {
	return func() (string, kkpreconciling.ValidatingAdmissionPolicyReconciler) {
		return GatewayAPIAdmissionPolicyName, func(policy *admissionregistrationv1.ValidatingAdmissionPolicy) (*admissionregistrationv1.ValidatingAdmissionPolicy, error) {
			policy.Labels = resources.BaseAppLabels(resources.KubeLBAppName, managedByLabels())
			policy.Spec = admissionregistrationv1.ValidatingAdmissionPolicySpec{
				FailurePolicy: ptr.To(admissionregistrationv1.Fail),
				MatchConstraints: &admissionregistrationv1.MatchResources{
					// These fields are defaulted by the apiserver, so they have to be set to those
					// defaults explicitly. Leaving them empty makes the reconciler detect a diff on
					// every sync and update the policy in an endless loop.
					MatchPolicy:       ptr.To(admissionregistrationv1.Equivalent),
					NamespaceSelector: &metav1.LabelSelector{},
					ObjectSelector:    &metav1.LabelSelector{},
					ResourceRules: []admissionregistrationv1.NamedRuleWithOperations{
						{
							RuleWithOperations: admissionregistrationv1.RuleWithOperations{
								// DELETE is guarded as well: dropping one of these CRDs takes every
								// Gateway and Route in the cluster with it and makes the CCM try to
								// reinstall it.
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
						// Only guard the Gateway API CRDs, every other CRD is none of our business.
						Name:       "gateway-api-crds-only",
						Expression: gatewayAPIGroupsMatchExpression(),
					},
					{
						// The CCM owns these CRDs and has to stay able to write them. It reaches the
						// user cluster with a client certificate issued for this username.
						Name:       "exclude-kubelb-ccm",
						Expression: fmt.Sprintf("request.userInfo.username != %q", resources.KubeLBCCMCertUsername),
					},
				},
				Validations: []admissionregistrationv1.Validation{
					{
						Expression: "false",
						Reason:     ptr.To(metav1.StatusReasonForbidden),
						// Both paths out are spelled exactly as they serialize: the JSON tag on the field is
						// "kubelb", not "kubeLB", and the cluster CRD schema silently prunes the wrong
						// spelling - so anyone copying it out of this message would see no effect at all.
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

// GatewayAPIValidatingAdmissionPolicyBindingReconciler returns the binding that activates
// GatewayAPIValidatingAdmissionPolicyReconciler's policy in Deny mode.
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

// GatewayAPIAdmissionPolicyResourcesForDeletion returns the admission policy objects to remove once
// Gateway API support is no longer enabled, at which point kubeLB stops owning the Gateway API CRDs and
// the user is free to manage them again. The binding comes first so the policy is never left active
// without it.
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

// gatewayAPIGroupsMatchExpression builds the CEL expression restricting the policy to CRDs of the
// Gateway API groups.
//
// On DELETE the admission request carries no `object`, only `oldObject`, so the CRD has to be picked
// from whichever of the two is set. Reading `object.spec.group` directly would make the expression fail
// on every DELETE, and because the policy uses failurePolicy Fail, a failing match condition rejects
// the request. That would block deleting any CustomResourceDefinition in the cluster, not just the
// Gateway API ones.
func gatewayAPIGroupsMatchExpression() string {
	const crd = "(object == null ? oldObject : object)"

	clauses := make([]string, 0, len(gatewayAPIGroups))
	for _, group := range gatewayAPIGroups {
		clauses = append(clauses, fmt.Sprintf("%s.spec.group == %q", crd, group))
	}

	return strings.Join(clauses, " || ")
}
