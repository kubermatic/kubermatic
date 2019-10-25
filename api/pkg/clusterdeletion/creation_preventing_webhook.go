package clusterdeletion

import (
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	utilpointer "k8s.io/utils/pointer"
)

// creationPreventingWebhook returns a ValidatingWebhookConfiguration that is intentionally defunct
// and will prevent all creation requests from succeeding.
func creationPreventingWebhook(apiGroup string, resources []string) reconciling.NamedValidatingWebhookConfigurationCreatorGetter {
	failurePolicy := admissionregistrationv1beta1.Fail
	return func() (string, reconciling.ValidatingWebhookConfigurationCreator) {
		return "kubernetes-cluster-cleanup-" + strings.Join(resources, "-"),
			func(vwc *admissionregistrationv1beta1.ValidatingWebhookConfiguration) (*admissionregistrationv1beta1.ValidatingWebhookConfiguration, error) {
				if vwc.Annotations == nil {
					vwc.Annotations = map[string]string{}
				}
				vwc.Annotations[annotationKeyDescription] = "This webhook configuration exists to prevent creation of any new stateful resources in a cluster that is currently being terminated"
				if len(vwc.Webhooks) != 1 {
					vwc.Webhooks = []admissionregistrationv1beta1.ValidatingWebhook{{}}
				}
				// Must be a domain with at least three segments separated by dots
				vwc.Webhooks[0].Name = "kubernetes.cluster.cleanup"
				vwc.Webhooks[0].ClientConfig = admissionregistrationv1beta1.WebhookClientConfig{
					URL: utilpointer.StringPtr("https://127.0.0.1:1"),
				}
				vwc.Webhooks[0].Rules = []admissionregistrationv1beta1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1beta1.OperationType{admissionregistrationv1beta1.Create},
						Rule: admissionregistrationv1beta1.Rule{
							APIGroups:   []string{apiGroup},
							APIVersions: []string{"*"},
							Resources:   resources,
						},
					},
				}
				vwc.Webhooks[0].FailurePolicy = &failurePolicy
				return vwc, nil
			}
	}
}
