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

package machinecontroller

import (
	"crypto/x509"
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MutatingwebhookConfigurationCreator returns the MutatingwebhookConfiguration for the machine controller
func MutatingwebhookConfigurationCreator(caCert *x509.Certificate, namespace string) reconciling.NamedMutatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.MutatingWebhookConfigurationCreator) {
		return resources.MachineControllerMutatingWebhookConfigurationName, func(mutatingWebhookConfiguration *admissionregistrationv1beta1.MutatingWebhookConfiguration) (*admissionregistrationv1beta1.MutatingWebhookConfiguration, error) {
			failurePolicy := admissionregistrationv1beta1.Fail
			mdURL := fmt.Sprintf("https://%s.%s.svc.cluster.local./machinedeployments", resources.MachineControllerWebhookServiceName, namespace)
			mURL := fmt.Sprintf("https://%s.%s.svc.cluster.local./machines", resources.MachineControllerWebhookServiceName, namespace)

			// This only gets set when the APIServer supports it, so carry it over
			var scope *admissionregistrationv1beta1.ScopeType
			if len(mutatingWebhookConfiguration.Webhooks) != 2 {
				mutatingWebhookConfiguration.Webhooks = []admissionregistrationv1beta1.MutatingWebhook{{}, {}}
			} else if len(mutatingWebhookConfiguration.Webhooks[0].Rules) > 0 {
				scope = mutatingWebhookConfiguration.Webhooks[0].Rules[0].Scope
			}

			mutatingWebhookConfiguration.Webhooks[0].Name = fmt.Sprintf("%s-machinedeployments", resources.MachineControllerMutatingWebhookConfigurationName)
			mutatingWebhookConfiguration.Webhooks[0].NamespaceSelector = &metav1.LabelSelector{}
			mutatingWebhookConfiguration.Webhooks[0].FailurePolicy = &failurePolicy
			mutatingWebhookConfiguration.Webhooks[0].Rules = []admissionregistrationv1beta1.RuleWithOperations{{
				Operations: []admissionregistrationv1beta1.OperationType{admissionregistrationv1beta1.Create, admissionregistrationv1beta1.Update},
				Rule: admissionregistrationv1beta1.Rule{
					APIGroups:   []string{clusterAPIGroup},
					APIVersions: []string{clusterAPIVersion},
					Resources:   []string{"machinedeployments"},
					Scope:       scope,
				},
			}}
			mutatingWebhookConfiguration.Webhooks[0].ClientConfig = admissionregistrationv1beta1.WebhookClientConfig{
				URL:      &mdURL,
				CABundle: triple.EncodeCertPEM(caCert),
			}

			mutatingWebhookConfiguration.Webhooks[1].Name = fmt.Sprintf("%s-machines", resources.MachineControllerMutatingWebhookConfigurationName)
			mutatingWebhookConfiguration.Webhooks[1].NamespaceSelector = &metav1.LabelSelector{}
			mutatingWebhookConfiguration.Webhooks[1].FailurePolicy = &failurePolicy
			mutatingWebhookConfiguration.Webhooks[1].Rules = []admissionregistrationv1beta1.RuleWithOperations{{
				Operations: []admissionregistrationv1beta1.OperationType{admissionregistrationv1beta1.Create, admissionregistrationv1beta1.Update},
				Rule: admissionregistrationv1beta1.Rule{
					APIGroups:   []string{clusterAPIGroup},
					APIVersions: []string{clusterAPIVersion},
					Resources:   []string{"machines"},
					Scope:       scope,
				},
			}}
			mutatingWebhookConfiguration.Webhooks[1].ClientConfig = admissionregistrationv1beta1.WebhookClientConfig{
				URL:      &mURL,
				CABundle: triple.EncodeCertPEM(caCert),
			}

			return mutatingWebhookConfiguration, nil
		}
	}
}
