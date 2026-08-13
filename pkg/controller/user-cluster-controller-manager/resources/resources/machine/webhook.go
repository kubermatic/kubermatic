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

	"k8c.io/kubermatic/v2/pkg/machine/accelerator"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	"k8c.io/reconciler/pkg/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	machineValidatingWebhookConfigurationName = "kubermatic-machine-validation"
	// AcceleratorMutatingWebhookConfigurationName is the KKP-owned Machine footprint webhook configuration.
	AcceleratorMutatingWebhookConfigurationName = "kubermatic-machine-accelerator-footprint"
	// AcceleratorValidatingWebhookConfigurationName protects the KKP-owned Machine footprint contract.
	AcceleratorValidatingWebhookConfigurationName = "kubermatic-machine-accelerator-footprint-validation"
)

// ValidatingWebhookConfigurationReconciler returns the ValidatingWebhookConfiguration for the machine CRD.
func ValidatingWebhookConfigurationReconciler(caCert *x509.Certificate, namespace string) reconciling.NamedValidatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.ValidatingWebhookConfigurationReconciler) {
		return machineValidatingWebhookConfigurationName, func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
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
					TimeoutSeconds:          ptr.To[int32](3),
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
							Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
						},
					},
				},
			}
			return hook, nil
		}
	}
}

// AcceleratorValidatingWebhookConfigurationReconciler returns the Machine footprint validation webhook configuration.
func AcceleratorValidatingWebhookConfigurationReconciler(caCert *x509.Certificate, namespace string) reconciling.NamedValidatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.ValidatingWebhookConfigurationReconciler) {
		return AcceleratorValidatingWebhookConfigurationName, func(hook *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.NamespacedScope

			url := fmt.Sprintf("https://%s.%s.svc.cluster.local.:%d%s",
				resources.UserClusterWebhookServiceName,
				namespace,
				resources.UserClusterWebhookUserListenPort,
				accelerator.ValidatingWebhookPath,
			)

			hook.Webhooks = []admissionregistrationv1.ValidatingWebhook{{
				Name:                    "accelerator-footprints.machines.cluster.k8c.io",
				AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
				MatchPolicy:             &matchPolicy,
				FailurePolicy:           &failurePolicy,
				SideEffects:             &sideEffects,
				TimeoutSeconds:          ptr.To[int32](3),
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: triple.EncodeCertPEM(caCert),
					URL:      &url,
				},
				ObjectSelector:    &metav1.LabelSelector{},
				NamespaceSelector: &metav1.LabelSelector{},
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{clusterv1alpha1.SchemeGroupVersion.Group},
						APIVersions: []string{clusterv1alpha1.SchemeGroupVersion.Version},
						Resources:   []string{"machines"},
						Scope:       &scope,
					},
					Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update},
				}},
			}}

			return hook, nil
		}
	}
}

// AcceleratorMutatingWebhookConfigurationReconciler returns the feature-gated Machine footprint webhook configuration.
func AcceleratorMutatingWebhookConfigurationReconciler(caCert *x509.Certificate, namespace string) reconciling.NamedMutatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.MutatingWebhookConfigurationReconciler) {
		return AcceleratorMutatingWebhookConfigurationName, func(hook *admissionregistrationv1.MutatingWebhookConfiguration) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
			matchPolicy := admissionregistrationv1.Exact
			failurePolicy := admissionregistrationv1.Fail
			reinvocationPolicy := admissionregistrationv1.IfNeededReinvocationPolicy
			sideEffects := admissionregistrationv1.SideEffectClassNone
			scope := admissionregistrationv1.NamespacedScope

			url := fmt.Sprintf("https://%s.%s.svc.cluster.local.:%d%s",
				resources.UserClusterWebhookServiceName,
				namespace,
				resources.UserClusterWebhookUserListenPort,
				accelerator.MutatingWebhookPath,
			)

			hook.Webhooks = []admissionregistrationv1.MutatingWebhook{{
				Name:                    "accelerator-footprints.machines.cluster.k8c.io",
				AdmissionReviewVersions: []string{admissionregistrationv1.SchemeGroupVersion.Version, admissionregistrationv1beta1.SchemeGroupVersion.Version},
				MatchPolicy:             &matchPolicy,
				FailurePolicy:           &failurePolicy,
				ReinvocationPolicy:      &reinvocationPolicy,
				SideEffects:             &sideEffects,
				TimeoutSeconds:          ptr.To[int32](3),
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: triple.EncodeCertPEM(caCert),
					URL:      &url,
				},
				ObjectSelector:    &metav1.LabelSelector{},
				NamespaceSelector: &metav1.LabelSelector{},
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{clusterv1alpha1.SchemeGroupVersion.Group},
						APIVersions: []string{clusterv1alpha1.SchemeGroupVersion.Version},
						Resources:   []string{"machines"},
						Scope:       &scope,
					},
					Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
				}},
			}}

			return hook, nil
		}
	}
}
