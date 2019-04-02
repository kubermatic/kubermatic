package machinecontroller

import (
	"crypto/x509"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	"k8s.io/api/admissionregistration/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	certutil "k8s.io/client-go/util/cert"
)

// MutatingwebhookConfigurationCreator returns the MutatingwebhookConfiguration for the machine controler
func MutatingwebhookConfigurationCreator(caCert *x509.Certificate, namespace string) reconciling.NamedMutatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.MutatingWebhookConfigurationCreator) {
		return resources.MachineControllerMutatingWebhookConfigurationName, func(mutatingWebhookConfiguration *v1beta1.MutatingWebhookConfiguration) (*v1beta1.MutatingWebhookConfiguration, error) {
			failurePolicy := admissionregistrationv1beta1.Fail
			mdURL := fmt.Sprintf("https://%s.%s.svc.cluster.local./machinedeployments", resources.MachineControllerWebhookServiceName, namespace)
			mURL := fmt.Sprintf("https://%s.%s.svc.cluster.local./machines", resources.MachineControllerWebhookServiceName, namespace)
			sideEffectClass := admissionregistrationv1beta1.SideEffectClassNone

			mutatingWebhookConfiguration.Webhooks = []admissionregistrationv1beta1.Webhook{
				{
					Name:              fmt.Sprintf("%s-machinedeployments", resources.MachineControllerMutatingWebhookConfigurationName),
					NamespaceSelector: &metav1.LabelSelector{},
					FailurePolicy:     &failurePolicy,
					SideEffects:       &sideEffectClass,
					Rules: []admissionregistrationv1beta1.RuleWithOperations{
						{
							Operations: []admissionregistrationv1beta1.OperationType{admissionregistrationv1beta1.Create, admissionregistrationv1beta1.Update},
							Rule: admissionregistrationv1beta1.Rule{
								APIGroups:   []string{clusterAPIGroup},
								APIVersions: []string{clusterAPIVersion},
								Resources:   []string{"machinedeployments"},
							},
						},
					},
					ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
						URL:      &mdURL,
						CABundle: certutil.EncodeCertPEM(caCert),
					},
				},
				{
					Name:              fmt.Sprintf("%s-machines", resources.MachineControllerMutatingWebhookConfigurationName),
					NamespaceSelector: &metav1.LabelSelector{},
					FailurePolicy:     &failurePolicy,
					SideEffects:       &sideEffectClass,
					Rules: []admissionregistrationv1beta1.RuleWithOperations{
						{
							Operations: []admissionregistrationv1beta1.OperationType{admissionregistrationv1beta1.Create, admissionregistrationv1beta1.Update},
							Rule: admissionregistrationv1beta1.Rule{
								APIGroups:   []string{clusterAPIGroup},
								APIVersions: []string{clusterAPIVersion},
								Resources:   []string{"machines"},
							},
						},
					},
					ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
						URL:      &mURL,
						CABundle: certutil.EncodeCertPEM(caCert),
					},
				},
			}

			return mutatingWebhookConfiguration, nil
		}
	}
}
