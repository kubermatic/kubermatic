/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package gatekeeper

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// ValidatingWebhookConfigurationReconciler returns the ValidatingwebhookConfiguration for gatekeeper.
func ValidatingWebhookConfigurationReconciler(timeout int) reconciling.NamedValidatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.ValidatingWebhookConfigurationReconciler) {
		return resources.GatekeeperValidatingWebhookConfigurationName, func(validatingWebhookConfigurationWebhookConfiguration *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			failurePolicyIgnore := admissionregistrationv1.Ignore
			sideEffectsNone := admissionregistrationv1.SideEffectClassNone
			matchPolicyExact := admissionregistrationv1.Exact
			allScopes := admissionregistrationv1.AllScopes

			validatingWebhookConfigurationWebhookConfiguration.Labels = map[string]string{"gatekeeper.sh/system": "yes"}
			// Get cabundle if set
			var caBundle []byte
			if len(validatingWebhookConfigurationWebhookConfiguration.Webhooks) > 0 {
				caBundle = validatingWebhookConfigurationWebhookConfiguration.Webhooks[0].ClientConfig.CABundle
			}
			validatingWebhookConfigurationWebhookConfiguration.Webhooks = []admissionregistrationv1.ValidatingWebhook{
				{
					Name:                    "validation.gatekeeper.sh",
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					FailurePolicy:           &failurePolicyIgnore,
					SideEffects:             &sideEffectsNone,
					TimeoutSeconds:          ptr.To[int32](int32(timeout)),
					MatchPolicy:             &matchPolicyExact,
					NamespaceSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "control-plane",
								Operator: metav1.LabelSelectorOpDoesNotExist,
							},
							{
								Key:      resources.GatekeeperExemptNamespaceLabel,
								Operator: metav1.LabelSelectorOpDoesNotExist,
							},
							{
								Key:      "kubernetes.io/metadata.name",
								Operator: metav1.LabelSelectorOpNotIn,
								Values:   []string{resources.GatekeeperNamespace},
							},
						},
					},
					ObjectSelector: &metav1.LabelSelector{},
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: caBundle,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      resources.GatekeeperWebhookServiceName,
							Namespace: resources.GatekeeperNamespace,
							Path:      ptr.To("/v1/admit"),
							Port:      ptr.To[int32](443),
						},
					},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{"*"},
								APIVersions: []string{"*"},
								Resources: []string{
									"*",
									// Explicitly list all known subresources except "status" (to avoid destabilizing the cluster and increasing load on gatekeeper).
									// You can find a rough list of subresources by doing a case-sensitive search in the Kubernetes codebase for 'Subresource("'
									"pods/ephemeralcontainers",
									"pods/exec",
									"pods/log",
									"pods/eviction",
									"pods/portforward",
									"pods/proxy",
									"pods/attach",
									"pods/binding",
									"deployments/scale",
									"replicasets/scale",
									"statefulsets/scale",
									"replicationcontrollers/scale",
									"services/proxy",
									"nodes/proxy",
									// For constraints that mitigate CVE-2020-8554
									"services/status",
								},
								Scope: &allScopes,
							},
						},
					},
				},
				{
					Name:                    "check-ignore-label.gatekeeper.sh",
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					FailurePolicy:           &failurePolicyIgnore,
					SideEffects:             &sideEffectsNone,
					TimeoutSeconds:          ptr.To[int32](int32(timeout)),
					MatchPolicy:             &matchPolicyExact,
					NamespaceSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "control-plane",
								Operator: metav1.LabelSelectorOpDoesNotExist,
							},
							{
								Key:      resources.GatekeeperExemptNamespaceLabel,
								Operator: metav1.LabelSelectorOpDoesNotExist,
							},
							{
								Key:      "kubernetes.io/metadata.name",
								Operator: metav1.LabelSelectorOpNotIn,
								Values:   []string{resources.GatekeeperNamespace},
							},
						},
					},
					ObjectSelector: &metav1.LabelSelector{},
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: caBundle,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      resources.GatekeeperWebhookServiceName,
							Namespace: resources.GatekeeperNamespace,
							Path:      ptr.To("/v1/admitlabel"),
							Port:      ptr.To[int32](443),
						},
					},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{""},
								APIVersions: []string{"*"},
								Resources:   []string{"namespaces"},
								Scope:       &allScopes,
							},
						},
					},
				},
			}

			return validatingWebhookConfigurationWebhookConfiguration, nil
		}
	}
}

func MutatingWebhookConfigurationReconciler(timeout int) reconciling.NamedMutatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.MutatingWebhookConfigurationReconciler) {
		return resources.GatekeeperMutatingWebhookConfigurationName, func(mutatingWebhookConfigurationWebhookConfiguration *admissionregistrationv1.MutatingWebhookConfiguration) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
			failurePolicyIgnore := admissionregistrationv1.Ignore
			sideEffectsNone := admissionregistrationv1.SideEffectClassNone
			matchPolicyExact := admissionregistrationv1.Exact
			allScopes := admissionregistrationv1.AllScopes
			reinvocationPolicy := admissionregistrationv1.NeverReinvocationPolicy

			mutatingWebhookConfigurationWebhookConfiguration.Labels = map[string]string{"gatekeeper.sh/system": "yes"}
			// Get cabundle if set
			var caBundle []byte
			if len(mutatingWebhookConfigurationWebhookConfiguration.Webhooks) > 0 {
				caBundle = mutatingWebhookConfigurationWebhookConfiguration.Webhooks[0].ClientConfig.CABundle
			}
			mutatingWebhookConfigurationWebhookConfiguration.Webhooks = []admissionregistrationv1.MutatingWebhook{

				{
					Name:                    "mutation.gatekeeper.sh",
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					FailurePolicy:           &failurePolicyIgnore,
					ReinvocationPolicy:      &reinvocationPolicy,
					SideEffects:             &sideEffectsNone,
					TimeoutSeconds:          ptr.To[int32](int32(timeout)),
					MatchPolicy:             &matchPolicyExact,
					NamespaceSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "control-plane",
								Operator: metav1.LabelSelectorOpDoesNotExist,
							},
							{
								Key:      resources.GatekeeperExemptNamespaceLabel,
								Operator: metav1.LabelSelectorOpDoesNotExist,
							},
							{
								Key:      "kubernetes.io/metadata.name",
								Operator: metav1.LabelSelectorOpNotIn,
								Values:   []string{resources.GatekeeperNamespace},
							},
						},
					},
					ObjectSelector: &metav1.LabelSelector{},
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: caBundle,
						Service: &admissionregistrationv1.ServiceReference{
							Name:      resources.GatekeeperWebhookServiceName,
							Namespace: resources.GatekeeperNamespace,
							Path:      ptr.To("/v1/mutate"),
							Port:      ptr.To[int32](443),
						},
					},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{"*"},
								APIVersions: []string{"*"},
								Resources:   []string{"*"},
								Scope:       &allScopes,
							},
						},
					},
				},
			}
			return mutatingWebhookConfigurationWebhookConfiguration, nil
		}
	}
}
