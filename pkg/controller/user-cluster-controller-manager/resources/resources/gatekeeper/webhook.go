package gatekeeper

import (
	"crypto/x509"
	"fmt"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
)

// ValidatingwebhookConfigurationCreator returns the ValidatingwebhookConfiguration for gatekeeper
func ValidatingWebhookConfigurationCreator(caCert *x509.Certificate, namespace string) reconciling.NamedValidatingWebhookConfigurationCreatorGetter {
	return func() (string, reconciling.ValidatingWebhookConfigurationCreator) {
		return resources.GatekeeperValidatingWebhookConfigurationName, func(validatingWebhookConfigurationWebhookConfiguration *admissionregistrationv1beta1.ValidatingWebhookConfiguration) (*admissionregistrationv1beta1.ValidatingWebhookConfiguration, error) {
			failurePolicyIgnore := admissionregistrationv1beta1.Ignore
			failurePolicyFail := admissionregistrationv1beta1.Fail
			sideEffectsNone := admissionregistrationv1beta1.SideEffectClassNone
			matchPolicyExact := admissionregistrationv1beta1.Exact
			allScopes := admissionregistrationv1beta1.AllScopes

			validatingWebhookConfigurationWebhookConfiguration.Labels = map[string]string{"gatekeeper.sh/system": "yes"}
			validatingWebhookConfigurationWebhookConfiguration.Webhooks = []admissionregistrationv1beta1.ValidatingWebhook{
				{
					Name:                    "validation.gatekeeper.sh",
					AdmissionReviewVersions: []string{admissionregistrationv1beta1.SchemeGroupVersion.Version},
					FailurePolicy:           &failurePolicyIgnore,
					SideEffects:             &sideEffectsNone,
					TimeoutSeconds:          pointer.Int32Ptr(5),
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
					ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
						URL: pointer.StringPtr(fmt.Sprintf(
							"https://%s.%s.svc.cluster.local/v1/admit", resources.GatekeeperWebhookServiceName, namespace)),
						CABundle: triple.EncodeCertPEM(caCert),
					},
					Rules: []admissionregistrationv1beta1.RuleWithOperations{
						{
							Operations: []admissionregistrationv1beta1.OperationType{
								admissionregistrationv1beta1.Create,
								admissionregistrationv1beta1.Update,
							},
							Rule: admissionregistrationv1beta1.Rule{
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
					AdmissionReviewVersions: []string{admissionregistrationv1beta1.SchemeGroupVersion.Version},
					FailurePolicy:           &failurePolicyFail,
					SideEffects:             &sideEffectsNone,
					TimeoutSeconds:          pointer.Int32Ptr(5),
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
					ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
						URL: pointer.StringPtr(fmt.Sprintf(
							"https://%s.%s.svc.cluster.local/v1/admitlabel", resources.GatekeeperWebhookServiceName, namespace)),
						CABundle: triple.EncodeCertPEM(caCert),
					},
					Rules: []admissionregistrationv1beta1.RuleWithOperations{
						{
							Operations: []admissionregistrationv1beta1.OperationType{
								admissionregistrationv1beta1.Create,
								admissionregistrationv1beta1.Update,
							},
							Rule: admissionregistrationv1beta1.Rule{
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
