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
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/utils/pointer"
)

// ValidatingwebhookConfigurationCreator returns the ValidatingwebhookConfiguration for the machine controller
func ValidatingwebhookConfigurationCreator(caCert *x509.Certificate, namespace, name string) reconciling.NamedValidatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.ValidatingWebhookConfigurationCreator) {
		return name, func(validatingWebhookConfiguration *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			sideEffect := admissionregistrationv1.SideEffectClassNone
			failurePolicy := admissionregistrationv1.Fail
			validatingWebhookConfiguration.Name = name
			validatingWebhookConfiguration.Namespace = namespace
			validatingWebhookConfiguration.Webhooks = []admissionregistrationv1.ValidatingWebhook{
				{
					Name: name,
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Namespace: namespace,
							Name:      resources.CSIMigrationWebhookName,
							Path:      pointer.String("/validate"),
						},
						CABundle: triple.EncodeCertPEM(caCert),
					},
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
							},
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
						},
					},
					SideEffects:             &sideEffect,
					AdmissionReviewVersions: []string{"v1"},
					FailurePolicy:           &failurePolicy,
				},
			}

			return validatingWebhookConfiguration, nil
		}
	}
}
