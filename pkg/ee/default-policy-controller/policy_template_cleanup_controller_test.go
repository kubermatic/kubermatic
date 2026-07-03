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

package defaultpolicycontroller

import (
	"context"
	"testing"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestPolicyTemplateCleanupReconcilerAddsFinalizer(t *testing.T) {
	ctx := context.Background()

	policyTemplate := genPolicyTemplate(policyName, false, false, kubermaticv1.PolicyTemplateVisibilityGlobal, "", nil, nil)
	seedClient := fake.
		NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(policyTemplate).
		Build()

	r := &policyTemplateCleanupReconciler{
		Client: seedClient,
		log:    zap.NewNop().Sugar(),
	}

	if _, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: policyName}}); err != nil {
		t.Fatalf("expected cleanup reconcile to succeed: %v", err)
	}

	updatedPolicyTemplate := &kubermaticv1.PolicyTemplate{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: policyName}, updatedPolicyTemplate); err != nil {
		t.Fatalf("failed to get PolicyTemplate: %v", err)
	}

	if !kuberneteshelper.HasFinalizer(updatedPolicyTemplate, kubermaticv1.PolicyTemplatePolicyBindingCleanupFinalizer) {
		t.Fatalf("expected PolicyTemplate cleanup finalizer to be added, got %v", updatedPolicyTemplate.Finalizers)
	}
}

func TestPolicyTemplateCleanupReconcilerDeletesPolicyBindingsWhenTemplateIsGone(t *testing.T) {
	ctx := context.Background()

	matchingBinding := genPolicyBindingForTemplate("matching-binding", clusterNamespace, policyName)
	otherBinding := genPolicyBindingForTemplate("other-binding", clusterNamespace, "other-policy")
	seedClient := fake.
		NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(matchingBinding, otherBinding).
		Build()

	r := &policyTemplateCleanupReconciler{
		Client: seedClient,
		log:    zap.NewNop().Sugar(),
	}

	if _, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: policyName}}); err != nil {
		t.Fatalf("expected cleanup reconcile to succeed: %v", err)
	}

	err := seedClient.Get(ctx, types.NamespacedName{Name: matchingBinding.Name, Namespace: matchingBinding.Namespace}, &kubermaticv1.PolicyBinding{})
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected matching PolicyBinding to be deleted, got err=%v", err)
	}

	if err := seedClient.Get(ctx, types.NamespacedName{Name: otherBinding.Name, Namespace: otherBinding.Namespace}, &kubermaticv1.PolicyBinding{}); err != nil {
		t.Fatalf("expected unrelated PolicyBinding to remain: %v", err)
	}
}

func TestPolicyTemplateCleanupReconcilerDeletesPolicyBindingsWhenTemplateIsTerminating(t *testing.T) {
	ctx := context.Background()

	policyTemplate := genPolicyTemplate(policyName, false, false, kubermaticv1.PolicyTemplateVisibilityGlobal, "", nil, nil)
	now := metav1.Now()
	policyTemplate.DeletionTimestamp = &now
	policyTemplate.Finalizers = []string{
		kubermaticv1.PolicyTemplateSeedCleanupFinalizer,
		kubermaticv1.PolicyTemplatePolicyBindingCleanupFinalizer,
	}

	matchingBinding := genPolicyBindingForTemplate("matching-binding", clusterNamespace, policyName)

	seedClient := fake.
		NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(policyTemplate, matchingBinding).
		Build()

	r := &policyTemplateCleanupReconciler{
		Client: seedClient,
		log:    zap.NewNop().Sugar(),
	}

	if _, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: policyName}}); err != nil {
		t.Fatalf("expected cleanup reconcile to succeed: %v", err)
	}

	err := seedClient.Get(ctx, types.NamespacedName{Name: matchingBinding.Name, Namespace: matchingBinding.Namespace}, &kubermaticv1.PolicyBinding{})
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected matching PolicyBinding to be deleted, got err=%v", err)
	}

	updatedPolicyTemplate := &kubermaticv1.PolicyTemplate{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: policyName}, updatedPolicyTemplate); err != nil {
		t.Fatalf("failed to get terminating PolicyTemplate: %v", err)
	}
	if kuberneteshelper.HasFinalizer(updatedPolicyTemplate, kubermaticv1.PolicyTemplatePolicyBindingCleanupFinalizer) {
		t.Fatalf("expected PolicyTemplate cleanup finalizer to be removed, got %v", updatedPolicyTemplate.Finalizers)
	}
	if !kuberneteshelper.HasFinalizer(updatedPolicyTemplate, kubermaticv1.PolicyTemplateSeedCleanupFinalizer) {
		t.Fatalf("expected unrelated PolicyTemplate finalizer to remain, got %v", updatedPolicyTemplate.Finalizers)
	}
}

func TestPolicyTemplateCleanupReconcilerSkipsActiveTemplate(t *testing.T) {
	ctx := context.Background()

	policyTemplate := genPolicyTemplate(policyName, false, false, kubermaticv1.PolicyTemplateVisibilityGlobal, "", nil, nil)
	matchingBinding := genPolicyBindingForTemplate("matching-binding", clusterNamespace, policyName)

	seedClient := fake.
		NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(policyTemplate, matchingBinding).
		Build()

	r := &policyTemplateCleanupReconciler{
		Client: seedClient,
		log:    zap.NewNop().Sugar(),
	}

	if _, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: policyName}}); err != nil {
		t.Fatalf("expected cleanup reconcile to succeed: %v", err)
	}

	if err := seedClient.Get(ctx, types.NamespacedName{Name: matchingBinding.Name, Namespace: matchingBinding.Namespace}, &kubermaticv1.PolicyBinding{}); err != nil {
		t.Fatalf("expected matching PolicyBinding to remain for active PolicyTemplate: %v", err)
	}

	updatedPolicyTemplate := &kubermaticv1.PolicyTemplate{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: policyName}, updatedPolicyTemplate); err != nil {
		t.Fatalf("failed to get active PolicyTemplate: %v", err)
	}
	if !kuberneteshelper.HasFinalizer(updatedPolicyTemplate, kubermaticv1.PolicyTemplatePolicyBindingCleanupFinalizer) {
		t.Fatalf("expected PolicyTemplate cleanup finalizer to be added, got %v", updatedPolicyTemplate.Finalizers)
	}
}

func genPolicyBindingForTemplate(name, namespace, policyTemplateName string) *kubermaticv1.PolicyBinding {
	return &kubermaticv1.PolicyBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: kubermaticv1.PolicyBindingSpec{
			PolicyTemplateRef: corev1.ObjectReference{
				Name: policyTemplateName,
			},
		},
	}
}
