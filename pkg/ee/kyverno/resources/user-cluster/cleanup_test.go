//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2026 Kubermatic GmbH

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
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestResourcesForDeletionDeletesWebhooksBeforeNamespace(t *testing.T) {
	const clusterNamespace = "cluster-test"

	resources := ResourcesForDeletion(&kubermaticv1.Cluster{
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: clusterNamespace,
		},
	})

	webhooks := WebhooksForDeletion()
	if len(resources) <= len(webhooks) {
		t.Fatalf("expected webhook and non-webhook resources, got %d", len(resources))
	}

	for i := range webhooks {
		switch resources[i].(type) {
		case *admissionregistrationv1.ValidatingWebhookConfiguration, *admissionregistrationv1.MutatingWebhookConfiguration:
		default:
			t.Fatalf("expected resource %d to be a webhook, got %T", i, resources[i])
		}
	}

	last := resources[len(resources)-1]
	namespace, ok := last.(*corev1.Namespace)
	if !ok {
		t.Fatalf("expected namespace to be deleted last, got %T", last)
	}
	if namespace.Name != clusterNamespace {
		t.Fatalf("expected namespace %q, got %q", clusterNamespace, namespace.Name)
	}
}
