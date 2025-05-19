//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0")
                     Copyright © 2025 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package userclusterresources

import (
	"crypto/x509"
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/reconciler/pkg/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	// These names follow upstream Kyverno defaults (see kyverno/charts/templates/*webhook-cfg.yaml).
	mutatingWebhookConfigName   = "kyverno-resource-mutating-webhook-cfg"
	validatingWebhookConfigName = "kyverno-resource-validating-webhook-cfg"

	kyvernoServiceName = "kyverno-svc"
)

// MutatingWebhookConfigurationReconciler returns a reconciler that ensures the Kyverno
// resource-mutation webhook configuration exists in the user cluster.
// Kyverno will later update this object with its full rule set; our reconciler only needs
// to guarantee its presence along with the correct CA bundle and service reference.
func MutatingWebhookConfigurationReconciler(caCert *x509.Certificate, namespace string) reconciling.NamedMutatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.MutatingWebhookConfigurationReconciler) {
		return mutatingWebhookConfigName, func(cfg *admissionregistrationv1.MutatingWebhookConfiguration) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
			sideEffects := admissionregistrationv1.SideEffectClassNoneOnDryRun
			timeout := int32(10)
			reinvocation := admissionregistrationv1.IfNeededReinvocationPolicy
			matchPolicy := admissionregistrationv1.Equivalent

			endpointURL := fmt.Sprintf("https://%s.%s.svc.cluster.local./mutate", kyvernoServiceName, namespace)

			// Ensure we have exactly 2 webhooks (ignore + fail)
			if len(cfg.Webhooks) < 2 {
				cfg.Webhooks = make([]admissionregistrationv1.MutatingWebhook, 2)
			}

			baseWebhook := func(name string, failure admissionregistrationv1.FailurePolicyType) admissionregistrationv1.MutatingWebhook {
				return admissionregistrationv1.MutatingWebhook{
					Name:                    name,
					FailurePolicy:           &failure,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          &timeout,
					AdmissionReviewVersions: []string{"v1", "v1beta1"},
					ReinvocationPolicy:      &reinvocation,
					MatchPolicy:             ptr.To(matchPolicy),
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						URL:      ptr.To[string](endpointURL),
						CABundle: triple.EncodeCertPEM(caCert),
					},
					Rules: []admissionregistrationv1.RuleWithOperations{{
						Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update, admissionregistrationv1.Delete},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{"*"},
							APIVersions: []string{"*"},
							Resources:   []string{"*/*"},
							Scope:       ptr.To(admissionregistrationv1.AllScopes),
						},
					}},
				}
			}

			cfg.Webhooks[0] = baseWebhook("mutate.kyverno.svc-ignore", admissionregistrationv1.Ignore)
			cfg.Webhooks[1] = baseWebhook("mutate.kyverno.svc-fail", admissionregistrationv1.Fail)

			return cfg, nil
		}
	}
}

// ValidatingWebhookConfigurationReconciler ensures Kyverno's validating webhook configuration exists.
func ValidatingWebhookConfigurationReconciler(caCert *x509.Certificate, namespace string) reconciling.NamedValidatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.ValidatingWebhookConfigurationReconciler) {
		return validatingWebhookConfigName, func(cfg *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
			sideEffects := admissionregistrationv1.SideEffectClassNoneOnDryRun
			timeout := int32(10)
			matchPolicy := admissionregistrationv1.Equivalent

			endpointURL := fmt.Sprintf("https://%s.%s.svc.cluster.local./validate", kyvernoServiceName, namespace)

			// Ensure we have exactly 2 webhooks (ignore + fail)
			if len(cfg.Webhooks) < 2 {
				cfg.Webhooks = make([]admissionregistrationv1.ValidatingWebhook, 2)
			}

			baseWebhook := func(name string, failure admissionregistrationv1.FailurePolicyType) admissionregistrationv1.ValidatingWebhook {
				return admissionregistrationv1.ValidatingWebhook{
					Name:                    name,
					FailurePolicy:           &failure,
					SideEffects:             &sideEffects,
					TimeoutSeconds:          &timeout,
					AdmissionReviewVersions: []string{"v1", "v1beta1"},
					MatchPolicy:             &matchPolicy,
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						URL:      ptr.To[string](endpointURL),
						CABundle: triple.EncodeCertPEM(caCert),
					},
					NamespaceSelector: &metav1.LabelSelector{},
					ObjectSelector:    &metav1.LabelSelector{},
					Rules: []admissionregistrationv1.RuleWithOperations{{
						Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update, admissionregistrationv1.Delete},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{"*"},
							APIVersions: []string{"*"},
							Resources:   []string{"*/*"},
							Scope:       ptr.To(admissionregistrationv1.AllScopes),
						},
					}},
				}
			}

			cfg.Webhooks[0] = baseWebhook("validate.kyverno.svc-ignore", admissionregistrationv1.Ignore)
			cfg.Webhooks[1] = baseWebhook("validate.kyverno.svc-fail", admissionregistrationv1.Fail)

			return cfg, nil
		}
	}
}
