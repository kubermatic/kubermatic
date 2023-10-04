/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package csisnapshotter

import (
	"crypto/x509"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/reconciler/pkg/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// ValidatingSnapshotWebhookConfigurationReconciler returns the ValidatingWebhookConfiguration for the CSI external snapshotter.
// Sourced from: https://github.com/kubernetes-csi/external-snapshotter/blob/v6.2.2/deploy/kubernetes/webhook-example/admission-configuration-template
func ValidatingSnapshotWebhookConfigurationReconciler(caCert *x509.Certificate, namespace, name string) reconciling.NamedValidatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.ValidatingWebhookConfigurationReconciler) {
		return name, func(validatingWebhookConfiguration *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			sideEffect := admissionregistrationv1.SideEffectClassNone
			matchPolicy := admissionregistrationv1.Equivalent
			failurePolicy := admissionregistrationv1.Fail
			scope := admissionregistrationv1.AllScopes

			validatingWebhookConfiguration.Webhooks = []admissionregistrationv1.ValidatingWebhook{
				{
					Name:                    name,
					AdmissionReviewVersions: []string{"v1"},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffect,
					TimeoutSeconds:          ptr.To[int32](2),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Namespace: namespace,
							Name:      resources.CSISnapshotValidationWebhookName,
							Path:      ptr.To("/volumesnapshot"),
							Port:      ptr.To[int32](443),
						},
						CABundle: triple.EncodeCertPEM(caCert),
					},
					NamespaceSelector: &metav1.LabelSelector{},
					ObjectSelector:    &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups: []string{
									"snapshot.storage.k8s.io",
								},
								APIVersions: []string{
									"v1",
								},
								Resources: []string{
									"volumesnapshots",
									"volumesnapshotcontents",
									"volumesnapshotclasses",
								},
								Scope: &scope,
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
