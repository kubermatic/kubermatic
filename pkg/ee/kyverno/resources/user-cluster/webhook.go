//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2025 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
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
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Kyverno Webhook configuration names.
const (
	// PolicyValidatingWebhookConfigurationName default policy validating webhook configuration name.
	PolicyValidatingWebhookConfigurationName = "kyverno-policy-validating-webhook-cfg"
	// ValidatingWebhookConfigurationName Kyverno resource validating webhook configuration name.
	ValidatingWebhookConfigurationName = "kyverno-resource-validating-webhook-cfg"
	// ExceptionValidatingWebhookConfigurationName Kyverno exception validating webhook configuration name.
	ExceptionValidatingWebhookConfigurationName = "kyverno-exception-validating-webhook-cfg"
	// CELExceptionValidatingWebhookConfigurationName Kyverno CEL exception validating webhook configuration name.
	CELExceptionValidatingWebhookConfigurationName = "kyverno-cel-exception-validating-webhook-cfg"
	// GlobalContextValidatingWebhookConfigurationName Kyverno global context validating webhook configuration name.
	GlobalContextValidatingWebhookConfigurationName = "kyverno-global-context-validating-webhook-cfg"
	// CleanupValidatingWebhookConfigurationName Kyverno cleanup validating webhook configuration name.
	CleanupValidatingWebhookConfigurationName = "kyverno-cleanup-validating-webhook-cfg"
	// PolicyMutatingWebhookConfigurationName default policy mutating webhook configuration name.
	PolicyMutatingWebhookConfigurationName = "kyverno-policy-mutating-webhook-cfg"
	// MutatingWebhookConfigurationName Kyverno resource mutating webhook configuration name.
	MutatingWebhookConfigurationName = "kyverno-resource-mutating-webhook-cfg"
	// VerifyMutatingWebhookConfigurationName Kyverno verify mutating webhook configuration name.
	VerifyMutatingWebhookConfigurationName = "kyverno-verify-mutating-webhook-cfg"
	// TTLValidatingWebhookConfigurationName Kyverno ttl label validating webhook configuration name.
	TTLValidatingWebhookConfigurationName = "kyverno-ttl-validating-webhook-cfg"
)

// WebhooksForDeletion returns a list of webhooks that should be deleted when the Kyverno reports controller is removed.
func WebhooksForDeletion() []ctrlruntimeclient.Object {
	return []ctrlruntimeclient.Object{
		&admissionregistrationv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: PolicyValidatingWebhookConfigurationName,
			},
		},
		&admissionregistrationv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: ValidatingWebhookConfigurationName,
			},
		},
		&admissionregistrationv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: ExceptionValidatingWebhookConfigurationName,
			},
		},
		&admissionregistrationv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: CELExceptionValidatingWebhookConfigurationName,
			},
		},
		&admissionregistrationv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: GlobalContextValidatingWebhookConfigurationName,
			},
		},
		&admissionregistrationv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: CleanupValidatingWebhookConfigurationName,
			},
		},
		&admissionregistrationv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: TTLValidatingWebhookConfigurationName,
			},
		},
		&admissionregistrationv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: PolicyMutatingWebhookConfigurationName,
			},
		},
		&admissionregistrationv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: MutatingWebhookConfigurationName,
			},
		},
		&admissionregistrationv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: VerifyMutatingWebhookConfigurationName,
			},
		},
	}
}
