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

				// This only gets set when the APIServer supports it, so carry it over
				var scope *admissionregistrationv1beta1.ScopeType
				if len(vwc.Webhooks) != 1 {
					vwc.Webhooks = []admissionregistrationv1beta1.ValidatingWebhook{{}}
				} else if len(vwc.Webhooks[0].Rules) > 0 {
					scope = vwc.Webhooks[0].Rules[0].Scope
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
							Scope:       scope,
						},
					},
				}
				vwc.Webhooks[0].FailurePolicy = &failurePolicy
				return vwc, nil
			}
	}
}
