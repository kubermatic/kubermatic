package machinecontroller

import (
	"crypto/x509"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MutatingwebhookConfigurationCreator returns the MutatingwebhookConfiguration for the machine controler
func MutatingwebhookConfigurationCreator(caCert *x509.Certificate, namespace string) reconciling.NamedMutatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.MutatingWebhookConfigurationCreator) {
		return resources.MachineControllerMutatingWebhookConfigurationName, func(mutatingWebhookConfiguration *admissionregistrationv1beta1.MutatingWebhookConfiguration) (*admissionregistrationv1beta1.MutatingWebhookConfiguration, error) {
			failurePolicy := admissionregistrationv1beta1.Fail
			mdURL := fmt.Sprintf("https://%s.%s.svc.cluster.local./machinedeployments", resources.MachineControllerWebhookServiceName, namespace)
			mURL := fmt.Sprintf("https://%s.%s.svc.cluster.local./machines", resources.MachineControllerWebhookServiceName, namespace)

			if len(mutatingWebhookConfiguration.Webhooks) != 2 {
				mutatingWebhookConfiguration.Webhooks = []admissionregistrationv1beta1.MutatingWebhook{{}, {}}
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
