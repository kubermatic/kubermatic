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
	"k8c.io/kubermatic/v2/pkg/resources"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling/modifier"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// The names of the ValidatingAdmissionPolicies of the web terminal. The binding of a policy
	// has the name of the policy.
	AccessPolicyName                = "kubermatic-webterminal-access"
	ServiceAccountPodsPolicyName    = "kubermatic-webterminal-serviceaccount-pods"
	ServiceAccountSecretsPolicyName = "kubermatic-webterminal-serviceaccount-secrets"

	// managedByLabelValue marks KKP as the owner of the admission policy objects.
	managedByLabelValue = "kkp"

	// exceptKKP makes a policy skip the requests of KKP itself. The KKP API and the KKP
	// controllers reach a user cluster with the internal admin kubeconfig, whose certificate has
	// this username. The admin kubeconfig that users download has another one.
	exceptKKP = "request.userInfo.username != '" + resources.InternalUserClusterAdminKubeconfigCertUsername + "'"
)

// PolicyNames returns the names of the policies, which are the names of their bindings as well.
func PolicyNames() []string {
	names := []string{}
	for _, policy := range policies() {
		names = append(names, policy.name)
	}

	return names
}

type policy struct {
	name        string
	rules       []admissionregistrationv1.NamedRuleWithOperations
	validations []admissionregistrationv1.Validation
}

// usesSessionServiceAccount returns the expression that is true if the given pod runs as the
// ServiceAccount of a session.
func usesSessionServiceAccount(pod string) string {
	return "(has(" + pod + ".spec.serviceAccountName) && " + pod + ".spec.serviceAccountName.startsWith('" + resources.WebTerminalSessionNamePrefix + "'))"
}

// policies returns the policies that keep everybody but KKP away from the identity of a web
// terminal session that runs as a ServiceAccount (feature gate WebTerminalServiceAccount).
//
// The kubectl proxy sidecar of the web terminal pod acts as the session user for whoever reaches
// it, and the ServiceAccount of the session may impersonate that user. RBAC can't take a
// permission away from somebody who has it in kube-system, so these ways to the identity are
// closed at admission, for everybody but KKP:
//
//   - a session in the pod: exec, attach, port-forward and ephemeral containers,
//   - a token of the ServiceAccount: requested from the API or written into a Secret,
//   - another program with the token: a new pod that runs as the ServiceAccount, or another
//     image in the web terminal pod.
//
// There is one policy per kind of object, because the kube-apiserver checks the expressions of a
// policy against every kind the policy matches.
func policies() []policy {
	sessionName := "startsWith('" + resources.WebTerminalSessionNamePrefix + "')"

	return []policy{
		{
			name: AccessPolicyName,
			rules: []admissionregistrationv1.NamedRuleWithOperations{
				rule([]admissionregistrationv1.OperationType{admissionregistrationv1.Connect}, "pods/exec", "pods/attach", "pods/portforward"),
				rule([]admissionregistrationv1.OperationType{admissionregistrationv1.Update}, "pods/ephemeralcontainers"),
				rule([]admissionregistrationv1.OperationType{admissionregistrationv1.Create}, "serviceaccounts/token"),
			},
			validations: []admissionregistrationv1.Validation{
				validation(
					"!(request.resource.resource == 'pods' && request.name."+sessionName+")",
					"only KKP may open a session in a web terminal pod",
				),
				// The kubelet requests the token the web terminal pod mounts. A node only gets
				// tokens for the pods that run on it.
				validation(
					"!(request.resource.resource == 'serviceaccounts' && request.name."+sessionName+") || 'system:nodes' in request.userInfo.groups",
					"only KKP may request tokens for the ServiceAccount of a web terminal session",
				),
			},
		},
		{
			name: ServiceAccountPodsPolicyName,
			rules: []admissionregistrationv1.NamedRuleWithOperations{
				rule([]admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update}, "pods"),
			},
			validations: []admissionregistrationv1.Validation{
				// A request has no old object unless it is an update, and a policy that fails
				// refuses the request: every validation names the operation it is for.
				validation(
					"request.operation != 'CREATE' || !"+usesSessionServiceAccount("object"),
					"only KKP may create pods that use the ServiceAccount of a web terminal session",
				),
				// The image is what an update can change in a container of a running pod, and the
				// web terminal pod has no init containers. Other updates stay possible, like the
				// one that removes a finalizer when the pod is deleted.
				validation(
					"request.operation != 'UPDATE' || !"+usesSessionServiceAccount("oldObject")+
						" || object.spec.containers.all(c, oldObject.spec.containers.exists(o, o.name == c.name && o.image == c.image))",
					"only KKP may change the images of a web terminal pod",
				),
			},
		},
		{
			name: ServiceAccountSecretsPolicyName,
			rules: []admissionregistrationv1.NamedRuleWithOperations{
				rule([]admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update}, "secrets"),
			},
			validations: []admissionregistrationv1.Validation{
				// Kubernetes writes a token that does not expire into a Secret of this type.
				validation(
					"!(object.type == 'kubernetes.io/service-account-token' && has(object.metadata.annotations)"+
						" && 'kubernetes.io/service-account.name' in object.metadata.annotations"+
						" && object.metadata.annotations['kubernetes.io/service-account.name']."+sessionName+")",
					"only KKP may create token Secrets for the ServiceAccount of a web terminal session",
				),
			},
		},
	}
}

func rule(operations []admissionregistrationv1.OperationType, names ...string) admissionregistrationv1.NamedRuleWithOperations {
	return admissionregistrationv1.NamedRuleWithOperations{
		RuleWithOperations: admissionregistrationv1.RuleWithOperations{
			Operations: operations,
			Rule: admissionregistrationv1.Rule{
				APIGroups:   []string{corev1.GroupName},
				APIVersions: []string{corev1.SchemeGroupVersion.Version},
				Resources:   names,
				Scope:       ptr.To(admissionregistrationv1.AllScopes),
			},
		},
	}
}

func validation(expression, message string) admissionregistrationv1.Validation {
	return admissionregistrationv1.Validation{
		Expression: expression,
		Message:    message,
		Reason:     ptr.To(metav1.StatusReasonForbidden),
	}
}

func labels() map[string]string {
	return resources.BaseAppLabels(resources.WebTerminalAppLabelValue, map[string]string{modifier.ManagedByLabel: managedByLabelValue})
}

// ValidatingAdmissionPolicyReconcilers returns the reconcilers of the policies. Every policy
// applies in kube-system, the namespace of the web terminal.
func ValidatingAdmissionPolicyReconcilers() []kkpreconciling.NamedValidatingAdmissionPolicyReconcilerFactory {
	factories := []kkpreconciling.NamedValidatingAdmissionPolicyReconcilerFactory{}
	for _, p := range policies() {
		factories = append(factories, func() (string, kkpreconciling.ValidatingAdmissionPolicyReconciler) {
			return p.name, func(existing *admissionregistrationv1.ValidatingAdmissionPolicy) (*admissionregistrationv1.ValidatingAdmissionPolicy, error) {
				existing.Labels = labels()
				existing.Spec = admissionregistrationv1.ValidatingAdmissionPolicySpec{
					// a request the policy can't evaluate is refused
					FailurePolicy: ptr.To(admissionregistrationv1.Fail),
					MatchConstraints: &admissionregistrationv1.MatchResources{
						// Set apiserver defaults explicitly, or the reconciler updates the policy
						// in an endless loop.
						MatchPolicy: ptr.To(admissionregistrationv1.Equivalent),
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{corev1.LabelMetadataName: metav1.NamespaceSystem},
						},
						ObjectSelector: &metav1.LabelSelector{},
						ResourceRules:  p.rules,
					},
					MatchConditions: []admissionregistrationv1.MatchCondition{
						{Name: "except-kkp", Expression: exceptKKP},
					},
					Validations: p.validations,
				}

				return existing, nil
			}
		})
	}

	return factories
}

// ValidatingAdmissionPolicyBindingReconcilers returns the reconcilers of the bindings that
// enforce the policies.
func ValidatingAdmissionPolicyBindingReconcilers() []kkpreconciling.NamedValidatingAdmissionPolicyBindingReconcilerFactory {
	factories := []kkpreconciling.NamedValidatingAdmissionPolicyBindingReconcilerFactory{}
	for _, name := range PolicyNames() {
		factories = append(factories, func() (string, kkpreconciling.ValidatingAdmissionPolicyBindingReconciler) {
			return name, func(existing *admissionregistrationv1.ValidatingAdmissionPolicyBinding) (*admissionregistrationv1.ValidatingAdmissionPolicyBinding, error) {
				existing.Labels = labels()
				existing.Spec = admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
					PolicyName:        name,
					ValidationActions: []admissionregistrationv1.ValidationAction{admissionregistrationv1.Deny},
				}

				return existing, nil
			}
		})
	}

	return factories
}

// ResourcesForDeletion returns the objects to delete, bindings first: that stops enforcement at
// once and never leaves a binding pointing at a deleted policy.
func ResourcesForDeletion() []ctrlruntimeclient.Object {
	objects := []ctrlruntimeclient.Object{}
	for _, name := range PolicyNames() {
		objects = append(objects, &admissionregistrationv1.ValidatingAdmissionPolicyBinding{ObjectMeta: metav1.ObjectMeta{Name: name}})
	}
	for _, name := range PolicyNames() {
		objects = append(objects, &admissionregistrationv1.ValidatingAdmissionPolicy{ObjectMeta: metav1.ObjectMeta{Name: name}})
	}

	return objects
}
