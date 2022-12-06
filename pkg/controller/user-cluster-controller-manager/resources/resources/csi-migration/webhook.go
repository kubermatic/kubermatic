/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package csimigration

import (
	"crypto/x509"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/reconciler/pkg/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// ValidatingwebhookConfigurationReconciler returns the ValidatingwebhookConfiguration for the machine controller.
func ValidatingwebhookConfigurationReconciler(caCert *x509.Certificate, namespace, name string) reconciling.NamedValidatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.ValidatingWebhookConfigurationReconciler) {
		return name, func(validatingWebhookConfiguration *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			sideEffect := admissionregistrationv1.SideEffectClassNone
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			scopeAll := admissionregistrationv1.AllScopes
			scopeNamespace := admissionregistrationv1.NamespacedScope

			validatingWebhookConfiguration.Webhooks = []admissionregistrationv1.ValidatingWebhook{
				{
					Name:                    name,
					AdmissionReviewVersions: []string{"v1"},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffect,
					TimeoutSeconds:          pointer.Int32(30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Namespace: namespace,
							Name:      resources.VSphereCSIValidatingWebhookSVCName,
							Path:      pointer.String("/validate"),
							Port:      pointer.Int32(443),
						},
						CABundle: triple.EncodeCertPEM(caCert),
					},
					NamespaceSelector: &metav1.LabelSelector{},
					ObjectSelector:    &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups: []string{
									"storage.k8s.io",
								},
								APIVersions: []string{
									"v1",
									"v1beta1",
								},
								Resources: []string{
									"storageclasses",
								},
								Scope: &scopeAll,
							},
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
						},
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups: []string{
									"",
								},
								APIVersions: []string{
									"v1",
									"v1beta1",
								},
								Resources: []string{
									"persistentvolumeclaims",
								},
								Scope: &scopeNamespace,
							},
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
						},
					},
				},
			}

			return validatingWebhookConfiguration, nil
		}
	}
}
