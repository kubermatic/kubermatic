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

package applications

import (
	"crypto/x509"
	"fmt"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/reconciler/pkg/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const ApplicationInstallationAdmissionWebhookName = "kubermatic-application-installations"

func ApplicationInstallationValidatingWebhookConfigurationReconciler(caCert *x509.Certificate, namespace string) reconciling.NamedValidatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.ValidatingWebhookConfigurationReconciler) {
		return ApplicationInstallationAdmissionWebhookName, func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.AllScopes

			url := fmt.Sprintf("https://%s.%s.svc.cluster.local./validate-application-installation", resources.UserClusterWebhookServiceName, namespace)

			hook.Webhooks = []admissionregistrationv1.ValidatingWebhook{
				{
					Name:                    "applicationinstallations.apps.kubermatic.k8c.io", // this should be a FQDN
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          ptr.To[int32](30),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: triple.EncodeCertPEM(caCert),
						URL:      &url,
					},
					ObjectSelector:    &metav1.LabelSelector{},
					NamespaceSelector: &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{appskubermaticv1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{"applicationinstallations"},
								Scope:       &scope,
							},
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
								admissionregistrationv1.Delete,
							},
						},
					},
				},
			}

			return hook, nil
		}
	}
}

func ApplicationInstallationMutatingWebhookConfigurationReconciler(caCert *x509.Certificate, namespace string) reconciling.NamedMutatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.MutatingWebhookConfigurationReconciler) {
		return ApplicationInstallationAdmissionWebhookName, func(hook *admissionregistrationv1.MutatingWebhookConfiguration) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			reinvocationPolicy := admissionregistrationv1.NeverReinvocationPolicy
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.AllScopes

			url := fmt.Sprintf("https://%s.%s.svc.cluster.local./mutate-application-installation", resources.UserClusterWebhookServiceName, namespace)

			hook.Webhooks = []admissionregistrationv1.MutatingWebhook{
				{
					Name:                    appskubermaticv1.ApplicationInstallationResourceName + "." + appskubermaticv1.GroupName, // this should be a FQDN,
					AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
					MatchPolicy:             &matchPolicy,
					FailurePolicy:           &failurePolicy,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          ptr.To[int32](30),
					ReinvocationPolicy:      &reinvocationPolicy,
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						CABundle: triple.EncodeCertPEM(caCert),
						URL:      &url,
					},
					Rules: []admissionregistrationv1.RuleWithOperations{
						{
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{appskubermaticv1.GroupName},
								APIVersions: []string{"*"},
								Resources:   []string{appskubermaticv1.ApplicationInstallationResourceName},
								Scope:       &scope,
							},
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
						},
					},
					NamespaceSelector: &metav1.LabelSelector{},
					ObjectSelector:    &metav1.LabelSelector{},
				},
			}
			return hook, nil
		}
	}
}
