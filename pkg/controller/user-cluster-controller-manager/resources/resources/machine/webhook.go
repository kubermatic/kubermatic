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

package machine

import (
	"crypto/x509"
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	clusterAPIGroup   = "cluster.k8s.io"
	clusterAPIVersion = "v1alpha1"
)

// ValidatingWebhookConfigurationCreator returns the ValidatingWebhookConfiguration for the machine CRD.
func ValidatingWebhookConfigurationCreator(caCert *x509.Certificate, namespace string) reconciling.NamedValidatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.ValidatingWebhookConfigurationCreator) {
		return resources.MachineValidatingWebhookConfigurationName, func(validatingWebhookConfiguration *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			// TODO - change to Fail when the resource quotas are fully implemented and tested
			failurePolicy := admissionregistrationv1.Ignore
			sideEffects := admissionregistrationv1.SideEffectClassNone
			mURL := fmt.Sprintf("https://%s.%s.svc.cluster.local./validate-cluster-k8s-io-v1-machine", resources.UserClusterWebhookServiceName, namespace)
			reviewVersions := []string{"v1", "v1beta1"}

			// This only gets set when the APIServer supports it, so carry it over
			var scope *admissionregistrationv1.ScopeType
			if len(validatingWebhookConfiguration.Webhooks) != 1 {
				validatingWebhookConfiguration.Webhooks = []admissionregistrationv1.ValidatingWebhook{{}, {}}
			} else if len(validatingWebhookConfiguration.Webhooks[0].Rules) > 0 {
				scope = validatingWebhookConfiguration.Webhooks[0].Rules[0].Scope
			}

			validatingWebhookConfiguration.Webhooks[0].Name = fmt.Sprintf("%s-machines", resources.MachineValidatingWebhookConfigurationName)
			validatingWebhookConfiguration.Webhooks[0].NamespaceSelector = &metav1.LabelSelector{}
			validatingWebhookConfiguration.Webhooks[0].SideEffects = &sideEffects
			validatingWebhookConfiguration.Webhooks[0].FailurePolicy = &failurePolicy
			validatingWebhookConfiguration.Webhooks[0].AdmissionReviewVersions = reviewVersions
			validatingWebhookConfiguration.Webhooks[0].Rules = []admissionregistrationv1.RuleWithOperations{{
				Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update},
				Rule: admissionregistrationv1.Rule{
					APIGroups:   []string{clusterAPIGroup},
					APIVersions: []string{clusterAPIVersion},
					Resources:   []string{"machines"},
					Scope:       scope,
				},
			}}
			validatingWebhookConfiguration.Webhooks[0].ClientConfig = admissionregistrationv1.WebhookClientConfig{
				URL:      &mURL,
				CABundle: triple.EncodeCertPEM(caCert),
			}

			return validatingWebhookConfiguration, nil
		}
	}
}
