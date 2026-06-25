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
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type userClusterClientProviderFunc func(context.Context, *kubermaticv1.Cluster, ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error)

func (f userClusterClientProviderFunc) GetClient(ctx context.Context, cluster *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error) {
	return f(ctx, cluster, options...)
}

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
		WithStatusSubresource(&kubermaticv1.PolicyBinding{}).
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

func TestHandleKyvernoCleanupClearsPolicyBindingFinalizersOnLiveDisable(t *testing.T) {
	ctx := context.Background()
	const clusterNamespace = "cluster-test"

	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test",
			Finalizers: []string{CleanupFinalizer},
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: clusterNamespace,
		},
	}
	binding := &kubermaticv1.PolicyBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "with-finalizer",
			Namespace:  clusterNamespace,
			Finalizers: []string{kubermaticv1.PolicyBindingCleanupFinalizer},
		},
		Spec: kubermaticv1.PolicyBindingSpec{
			PolicyTemplateRef: corev1.ObjectReference{Name: "policy-template"},
		},
	}

	seedClient := fake.NewClientBuilder().
		WithScheme(fake.NewScheme()).
		WithObjects(cluster, binding).
		WithStatusSubresource(&kubermaticv1.PolicyBinding{}).
		Build()

	userClusterScheme := fake.NewScheme()
	if err := apiextensionsv1.AddToScheme(userClusterScheme); err != nil {
		t.Fatalf("failed to add apiextensions to scheme: %v", err)
	}
	if err := admissionregistrationv1.AddToScheme(userClusterScheme); err != nil {
		t.Fatalf("failed to add admissionregistration to scheme: %v", err)
	}

	userClusterClient := fake.NewClientBuilder().
		WithScheme(userClusterScheme).
		Build()

	r := &reconciler{
		Client: seedClient,
		userClusterConnectionProvider: userClusterClientProviderFunc(func(context.Context, *kubermaticv1.Cluster, ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error) {
			return userClusterClient, nil
		}),
	}

	if err := r.handleKyvernoCleanup(ctx, cluster); err != nil {
		t.Fatalf("handleKyvernoCleanup failed: %v", err)
	}

	updatedBinding := &kubermaticv1.PolicyBinding{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: binding.Name, Namespace: binding.Namespace}, updatedBinding); err != nil {
		t.Fatalf("failed to get updated binding: %v", err)
	}
	if len(updatedBinding.Finalizers) != 0 {
		t.Fatalf("expected live-disable cleanup to remove PolicyBinding finalizer, got %v", updatedBinding.Finalizers)
	}
	if updatedBinding.Status.Active == nil || *updatedBinding.Status.Active {
		t.Fatalf("expected live-disable cleanup to mark PolicyBinding inactive, got %#v", updatedBinding.Status.Active)
	}
	readyCondition := getCondition(updatedBinding, kubermaticv1.PolicyBindingConditionReady)
	if readyCondition == nil || readyCondition.Status != metav1.ConditionFalse || readyCondition.Reason != policyBindingReasonKyvernoDisabled {
		t.Fatalf("expected Ready=False/%s, got %#v", policyBindingReasonKyvernoDisabled, readyCondition)
	}
	appliedCondition := getCondition(updatedBinding, kubermaticv1.PolicyBindingConditionKyvernoPolicyApplied)
	if appliedCondition == nil || appliedCondition.Status != metav1.ConditionFalse || appliedCondition.Reason != policyBindingReasonKyvernoDisabled {
		t.Fatalf("expected KyvernoPolicyApplied=False/%s, got %#v", policyBindingReasonKyvernoDisabled, appliedCondition)
	}
}

func TestRemovePolicyBindingCleanupFinalizersUsesFallbackNamespace(t *testing.T) {
	ctx := context.Background()

	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}
	clusterNamespace := kubernetesprovider.NamespaceName(cluster.Name)
	binding := &kubermaticv1.PolicyBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "with-finalizer",
			Namespace:  clusterNamespace,
			Finalizers: []string{kubermaticv1.PolicyBindingCleanupFinalizer},
		},
		Spec: kubermaticv1.PolicyBindingSpec{
			PolicyTemplateRef: corev1.ObjectReference{Name: "policy-template"},
		},
	}

	seedClient := fake.NewClientBuilder().
		WithScheme(fake.NewScheme()).
		WithObjects(binding).
		WithStatusSubresource(&kubermaticv1.PolicyBinding{}).
		Build()

	r := &reconciler{Client: seedClient}
	if err := r.removePolicyBindingCleanupFinalizers(ctx, cluster); err != nil {
		t.Fatalf("removePolicyBindingCleanupFinalizers failed: %v", err)
	}

	updatedBinding := &kubermaticv1.PolicyBinding{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: binding.Name, Namespace: binding.Namespace}, updatedBinding); err != nil {
		t.Fatalf("failed to get updated binding: %v", err)
	}
	for _, finalizer := range updatedBinding.Finalizers {
		if finalizer == kubermaticv1.PolicyBindingCleanupFinalizer {
			t.Fatalf("expected cleanup finalizer to be removed from fallback namespace, got %v", updatedBinding.Finalizers)
		}
	}
}

func getCondition(binding *kubermaticv1.PolicyBinding, conditionType kubermaticv1.PolicyBindingConditionType) *metav1.Condition {
	for i := range binding.Status.Conditions {
		if binding.Status.Conditions[i].Type == string(conditionType) {
			return &binding.Status.Conditions[i]
		}
	}
	return nil
}
