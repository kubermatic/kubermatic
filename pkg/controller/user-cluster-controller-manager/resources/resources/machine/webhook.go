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

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	machineValidatingWebhookConfigurationName = "kubermatic-machine-validation"
)

// ValidatingWebhookConfigurationCreator returns the ValidatingWebhookConfiguration for the machine CRD.
func ValidatingWebhookConfigurationCreator(caCert *x509.Certificate, namespace string) reconciling.NamedValidatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.ValidatingWebhookConfigurationCreator) {
		return machineValidatingWebhookConfigurationName, func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			// TODO - change to Fail when the resource quotas are fully implemented and tested
			failurePolicy := admissionregistrationv1.Ignore
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.NamespacedScope

			url := fmt.Sprintf("https://%s.%s.svc.cluster.local.:%d/validate-cluster-k8s-io-v1alpha1-machine",
				resources.UserClusterWebhookServiceName,
				namespace,
				resources.UserClusterWebhookUserListenPort,
			)

			hook.Webhooks = []admissionregistrationv1.ValidatingWebhook{
				{
					Name:                    "machines.cluster.k8c.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          pointer.Int32Ptr(3),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: triple.EncodeCertPEM(caCert),
						URL:      &url,
					},
					ObjectSelector:    &metav1.LabelSelector{},
					NamespaceSelector: &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{clusterv1alpha1.SchemeGroupVersion.Group},
								APIVersions: []string{clusterv1alpha1.SchemeGroupVersion.Version},
								Resources:   []string{"machines"},
								Scope:       &scope,
							},
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
							},
						},
					},
				},
			}
			return hook, nil
		}
	}
}
