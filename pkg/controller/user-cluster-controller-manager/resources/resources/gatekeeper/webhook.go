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
	"crypto/x509"
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// ValidatingWebhookConfigurationCreator returns the ValidatingwebhookConfiguration for gatekeeper
func ValidatingWebhookConfigurationCreator(caCert *x509.Certificate, namespace string) reconciling.NamedValidatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.ValidatingWebhookConfigurationCreator) {
		return resources.GatekeeperValidatingWebhookConfigurationName, func(validatingWebhookConfigurationWebhookConfiguration *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			failurePolicyIgnore := admissionregistrationv1.Ignore
			sideEffectsNone := admissionregistrationv1.SideEffectClassNone
			matchPolicyExact := admissionregistrationv1.Exact
			allScopes := admissionregistrationv1.AllScopes

			validatingWebhookConfigurationWebhookConfiguration.Labels = map[string]string{"gatekeeper.sh/system": "yes"}
			validatingWebhookConfigurationWebhookConfiguration.Webhooks = []admissionregistrationv1.ValidatingWebhook{
				{
					Name:                    "validation.gatekeeper.sh",
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version},
					FailurePolicy:           &failurePolicyIgnore,
					SideEffects:             &sideEffectsNone,
					TimeoutSeconds:          pointer.Int32Ptr(2),
					MatchPolicy:             &matchPolicyExact,
					NamespaceSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "control-plane",
								Operator: metav1.LabelSelectorOpDoesNotExist,
							},
							{
								Key:      "admission.gatekeeper.sh/ignore",
								Operator: metav1.LabelSelectorOpDoesNotExist,
							},
						},
					},
					ObjectSelector: &metav1.LabelSelector{},
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						URL: pointer.StringPtr(fmt.Sprintf(
							"https://%s.%s.svc.cluster.local/v1/admit", resources.GatekeeperWebhookServiceName, namespace)),
						CABundle: triple.EncodeCertPEM(caCert),
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
				{
					Name:                    "check-ignore-label.gatekeeper.sh",
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version},
					FailurePolicy:           &failurePolicyIgnore,
					SideEffects:             &sideEffectsNone,
					TimeoutSeconds:          pointer.Int32Ptr(2),
					MatchPolicy:             &matchPolicyExact,
					NamespaceSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "control-plane",
								Operator: metav1.LabelSelectorOpDoesNotExist,
							},
							{
								Key:      "admission.gatekeeper.sh/ignore",
								Operator: metav1.LabelSelectorOpDoesNotExist,
							},
						},
					},
					ObjectSelector: &metav1.LabelSelector{},
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						URL: pointer.StringPtr(fmt.Sprintf(
							"https://%s.%s.svc.cluster.local/v1/admitlabel", resources.GatekeeperWebhookServiceName, namespace)),
						CABundle: triple.EncodeCertPEM(caCert),
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
