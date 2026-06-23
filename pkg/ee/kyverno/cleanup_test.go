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

package kyverno

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestRemovePolicyBindingCleanupFinalizers(t *testing.T) {
	ctx := context.Background()
	const clusterNamespace = "cluster-test"

	cluster := &kubermaticv1.Cluster{
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: clusterNamespace,
		},
	}
	bindingWithFinalizer := &kubermaticv1.PolicyBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "with-finalizer",
			Namespace:  clusterNamespace,
			Finalizers: []string{kubermaticv1.PolicyBindingCleanupFinalizer},
		},
		Spec: kubermaticv1.PolicyBindingSpec{
			PolicyTemplateRef: corev1.ObjectReference{Name: "policy-template"},
		},
	}
	bindingWithoutFinalizer := &kubermaticv1.PolicyBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "without-finalizer",
			Namespace: clusterNamespace,
		},
		Spec: kubermaticv1.PolicyBindingSpec{
			PolicyTemplateRef: corev1.ObjectReference{Name: "policy-template"},
		},
	}
	otherNamespaceBinding := &kubermaticv1.PolicyBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "other-namespace",
			Namespace:  "cluster-other",
			Finalizers: []string{kubermaticv1.PolicyBindingCleanupFinalizer},
		},
		Spec: kubermaticv1.PolicyBindingSpec{
			PolicyTemplateRef: corev1.ObjectReference{Name: "policy-template"},
		},
	}

	seedClient := fake.NewClientBuilder().
		WithScheme(fake.NewScheme()).
		WithObjects(bindingWithFinalizer, bindingWithoutFinalizer, otherNamespaceBinding).
		Build()

	r := &reconciler{Client: seedClient}
	if err := r.removePolicyBindingCleanupFinalizers(ctx, cluster); err != nil {
		t.Fatalf("removePolicyBindingCleanupFinalizers failed: %v", err)
	}

	updatedBinding := &kubermaticv1.PolicyBinding{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: bindingWithFinalizer.Name, Namespace: bindingWithFinalizer.Namespace}, updatedBinding); err != nil {
		t.Fatalf("failed to get updated binding: %v", err)
	}
	for _, finalizer := range updatedBinding.Finalizers {
		if finalizer == kubermaticv1.PolicyBindingCleanupFinalizer {
			t.Fatalf("expected cleanup finalizer to be removed, got %v", updatedBinding.Finalizers)
		}
	}

	otherBinding := &kubermaticv1.PolicyBinding{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: otherNamespaceBinding.Name, Namespace: otherNamespaceBinding.Namespace}, otherBinding); err != nil {
		t.Fatalf("failed to get other namespace binding: %v", err)
	}
	if len(otherBinding.Finalizers) != 1 || otherBinding.Finalizers[0] != kubermaticv1.PolicyBindingCleanupFinalizer {
		t.Fatalf("expected other namespace binding to keep cleanup finalizer, got %v", otherBinding.Finalizers)
	}
}
