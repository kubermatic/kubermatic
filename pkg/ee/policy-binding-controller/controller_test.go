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

package policybindingcontroller

import (
	"context"
	"fmt"
	"testing"

	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	ctrlruntimeevent "sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testClusterName      = "test-cluster"
	testClusterNamespace = "cluster-test-cluster"
	testPolicyName       = "test-policy"
)

//nolint:gocyclo
func TestReconcile(t *testing.T) {
	log := zap.NewNop().Sugar()

	testCases := []struct {
		name        string
		binding     *kubermaticv1.PolicyBinding
		template    *kubermaticv1.PolicyTemplate
		cluster     *kubermaticv1.Cluster
		userObjects []ctrlruntimeclient.Object
		expectError bool
		validate    func(t *testing.T, seedClient, userClient ctrlruntimeclient.Client, binding *kubermaticv1.PolicyBinding) error
	}{
		{
			name:     "creates ClusterPolicy and sets Ready status",
			binding:  genPolicyBinding(testPolicyName, testClusterNamespace, testPolicyName),
			template: genPolicyTemplate(testPolicyName, false),
			cluster:  genCluster(testClusterName, true),
			validate: func(t *testing.T, seedClient, userClient ctrlruntimeclient.Client, binding *kubermaticv1.PolicyBinding) error {
				ctx := context.Background()

				// Verify ClusterPolicy was created
				clusterPolicy := &kyvernov1.ClusterPolicy{}
				if err := userClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: testPolicyName}, clusterPolicy); err != nil {
					return fmt.Errorf("ClusterPolicy should be created: %w", err)
				}

				if clusterPolicy.Labels[LabelPolicyBinding] != testPolicyName {
					return fmt.Errorf("ClusterPolicy should have binding label, got: %v", clusterPolicy.Labels)
				}
				if clusterPolicy.Labels[LabelPolicyTemplate] != testPolicyName {
					return fmt.Errorf("ClusterPolicy should have template label, got: %v", clusterPolicy.Labels)
				}

				readyCondition := getCondition(binding, kubermaticv1.PolicyBindingConditionReady)
				if readyCondition == nil {
					return fmt.Errorf("Ready condition should be set")
				}
				if readyCondition.Status != metav1.ConditionTrue {
					return fmt.Errorf("Ready condition should be True, got %s", readyCondition.Status)
				}
				if readyCondition.Reason != kubermaticv1.PolicyBindingReasonReady {
					return fmt.Errorf("Ready reason should be %s, got %s", kubermaticv1.PolicyBindingReasonReady, readyCondition.Reason)
				}

				appliedCondition := getCondition(binding, kubermaticv1.PolicyBindingConditionKyvernoPolicyApplied)
				if appliedCondition == nil {
					return fmt.Errorf("KyvernoPolicyApplied condition should be set")
				}
				if appliedCondition.Status != metav1.ConditionTrue {
					return fmt.Errorf("KyvernoPolicyApplied condition should be True, got %s", appliedCondition.Status)
				}

				if binding.Status.Active == nil || !*binding.Status.Active {
					return fmt.Errorf("Active should be true")
				}

				if binding.Status.ObservedGeneration != binding.Generation {
					return fmt.Errorf("ObservedGeneration should be %d, got %d", binding.Generation, binding.Status.ObservedGeneration)
				}

				return nil
			},
		},
		{
			name:     "namespaced policy without namespace remains inactive",
			binding:  genPolicyBinding(testPolicyName, testClusterNamespace, testPolicyName),
			template: genPolicyTemplate(testPolicyName, true),
			cluster:  genCluster(testClusterName, true),
			validate: func(t *testing.T, seedClient, userClient ctrlruntimeclient.Client, binding *kubermaticv1.PolicyBinding) error {
				ctx := context.Background()

				if binding == nil {
					return fmt.Errorf("binding should still exist")
				}

				policyList := &kyvernov1.PolicyList{}
				if err := userClient.List(ctx, policyList); err != nil {
					return fmt.Errorf("failed to list namespaced policies: %w", err)
				}
				if len(policyList.Items) != 0 {
					return fmt.Errorf("expected no namespaced policies, got %d", len(policyList.Items))
				}

				clusterPolicy := &kyvernov1.ClusterPolicy{}
				err := userClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: testPolicyName}, clusterPolicy)
				if err == nil {
					return fmt.Errorf("ClusterPolicy should not be created for namespaced policy without namespace")
				}

				readyCondition := getCondition(binding, kubermaticv1.PolicyBindingConditionReady)
				if readyCondition == nil {
					return fmt.Errorf("Ready condition should be set")
				}
				if readyCondition.Status != metav1.ConditionFalse {
					return fmt.Errorf("Ready condition should be False, got %s", readyCondition.Status)
				}
				if readyCondition.Reason != kubermaticv1.PolicyBindingReasonPolicyNamespaceMissing {
					return fmt.Errorf("Ready reason should be %s, got %s", kubermaticv1.PolicyBindingReasonPolicyNamespaceMissing, readyCondition.Reason)
				}

				appliedCondition := getCondition(binding, kubermaticv1.PolicyBindingConditionKyvernoPolicyApplied)
				if appliedCondition == nil {
					return fmt.Errorf("KyvernoPolicyApplied condition should be set")
				}
				if appliedCondition.Status != metav1.ConditionFalse {
					return fmt.Errorf("KyvernoPolicyApplied condition should be False, got %s", appliedCondition.Status)
				}
				if appliedCondition.Reason != kubermaticv1.PolicyBindingReasonPolicyNamespaceMissing {
					return fmt.Errorf("KyvernoPolicyApplied reason should be %s, got %s", kubermaticv1.PolicyBindingReasonPolicyNamespaceMissing, appliedCondition.Reason)
				}

				if binding.Status.Active == nil || *binding.Status.Active {
					return fmt.Errorf("Active should be false")
				}

				return nil
			},
		},
		{
			name:     "namespaced policy without namespace deletes stale generated resources",
			binding:  genPolicyBinding(testPolicyName, testClusterNamespace, testPolicyName),
			template: genPolicyTemplate(testPolicyName, true),
			cluster:  genCluster(testClusterName, true),
			userObjects: []ctrlruntimeclient.Object{
				genClusterPolicy(testPolicyName, testPolicyName),
				genPolicy(testPolicyName, "old-namespace", testPolicyName),
			},
			validate: func(t *testing.T, seedClient, userClient ctrlruntimeclient.Client, binding *kubermaticv1.PolicyBinding) error {
				ctx := context.Background()

				clusterPolicyList := &kyvernov1.ClusterPolicyList{}
				if err := userClient.List(ctx, clusterPolicyList, ctrlruntimeclient.MatchingLabels{LabelPolicyBinding: testPolicyName}); err != nil {
					return fmt.Errorf("failed to list stale ClusterPolicies: %w", err)
				}
				if len(clusterPolicyList.Items) != 0 {
					return fmt.Errorf("expected stale ClusterPolicies to be deleted, got %d", len(clusterPolicyList.Items))
				}

				policyList := &kyvernov1.PolicyList{}
				if err := userClient.List(ctx, policyList, ctrlruntimeclient.MatchingLabels{LabelPolicyBinding: testPolicyName}); err != nil {
					return fmt.Errorf("failed to list stale Policies: %w", err)
				}
				if len(policyList.Items) != 0 {
					return fmt.Errorf("expected stale Policies to be deleted, got %d", len(policyList.Items))
				}

				readyCondition := getCondition(binding, kubermaticv1.PolicyBindingConditionReady)
				if readyCondition == nil || readyCondition.Status != metav1.ConditionFalse || readyCondition.Reason != kubermaticv1.PolicyBindingReasonPolicyNamespaceMissing {
					return fmt.Errorf("expected Ready=False/%s, got %#v", kubermaticv1.PolicyBindingReasonPolicyNamespaceMissing, readyCondition)
				}

				if binding.Status.Active == nil || *binding.Status.Active {
					return fmt.Errorf("Active should be false")
				}

				return nil
			},
		},
		{
			name:     "kyverno disabled triggers cleanup",
			binding:  genPolicyBindingWithFinalizer(testPolicyName, testClusterNamespace, testPolicyName),
			template: genPolicyTemplate(testPolicyName, false),
			cluster:  genCluster(testClusterName, false),
			validate: func(t *testing.T, seedClient, userClient ctrlruntimeclient.Client, binding *kubermaticv1.PolicyBinding) error {
				ctx := context.Background()

				clusterPolicy := &kyvernov1.ClusterPolicy{}
				err := userClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: testPolicyName}, clusterPolicy)
				if err == nil {
					return fmt.Errorf("ClusterPolicy should be deleted during cleanup")
				}

				if len(binding.Finalizers) > 0 {
					return fmt.Errorf("Finalizer should be removed, got: %v", binding.Finalizers)
				}

				return nil
			},
		},
		{
			name:     "binding deletion triggers cleanup and removes finalizer",
			binding:  genPolicyBindingWithDeletionTimestamp(testPolicyName, testClusterNamespace, testPolicyName),
			template: genPolicyTemplate(testPolicyName, false),
			cluster:  genCluster(testClusterName, true),
			validate: func(t *testing.T, seedClient, userClient ctrlruntimeclient.Client, binding *kubermaticv1.PolicyBinding) error {
				ctx := context.Background()

				// Verify ClusterPolicy was deleted (cleanup happened)
				clusterPolicy := &kyvernov1.ClusterPolicy{}
				err := userClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: testPolicyName}, clusterPolicy)
				if err == nil {
					return fmt.Errorf("ClusterPolicy should be deleted during cleanup")
				}

				// When finalizer is removed and DeletionTimestamp is set, the binding is deleted.
				return nil
			},
		},
		{
			name:     "binding deletion cleans stale labeled namespaced policy without namespace in spec",
			binding:  genPolicyBindingWithDeletionTimestamp(testPolicyName, testClusterNamespace, testPolicyName),
			template: genPolicyTemplate(testPolicyName, true),
			cluster:  genCluster(testClusterName, true),
			userObjects: []ctrlruntimeclient.Object{
				genPolicy(testPolicyName, "old-namespace", testPolicyName),
			},
			validate: func(t *testing.T, seedClient, userClient ctrlruntimeclient.Client, binding *kubermaticv1.PolicyBinding) error {
				ctx := context.Background()

				policyList := &kyvernov1.PolicyList{}
				if err := userClient.List(ctx, policyList, ctrlruntimeclient.MatchingLabels{LabelPolicyBinding: testPolicyName}); err != nil {
					return fmt.Errorf("failed to list stale Policies: %w", err)
				}
				if len(policyList.Items) != 0 {
					return fmt.Errorf("expected stale Policies to be deleted, got %d", len(policyList.Items))
				}

				return nil
			},
		},
		{
			name:     "template not found sets status conditions and triggers cleanup",
			binding:  genPolicyBindingWithFinalizer(testPolicyName, testClusterNamespace, "non-existent-template"),
			template: nil, // No template exists
			cluster:  genCluster(testClusterName, true),
			validate: func(t *testing.T, seedClient, userClient ctrlruntimeclient.Client, binding *kubermaticv1.PolicyBinding) error {
				if binding == nil {
					return fmt.Errorf("binding should still exist")
				}

				// Verify Ready condition is False with TemplateNotFound reason
				readyCondition := getCondition(binding, kubermaticv1.PolicyBindingConditionReady)
				if readyCondition == nil {
					return fmt.Errorf("Ready condition should be set")
				}
				if readyCondition.Status != metav1.ConditionFalse {
					return fmt.Errorf("Ready condition should be False, got %s", readyCondition.Status)
				}
				if readyCondition.Reason != kubermaticv1.PolicyBindingReasonTemplateNotFound {
					return fmt.Errorf("Ready reason should be %s, got %s", kubermaticv1.PolicyBindingReasonTemplateNotFound, readyCondition.Reason)
				}

				// Verify TemplateValid condition is False
				templateValidCondition := getCondition(binding, kubermaticv1.PolicyBindingConditionTemplateValid)
				if templateValidCondition == nil {
					return fmt.Errorf("TemplateValid condition should be set")
				}
				if templateValidCondition.Status != metav1.ConditionFalse {
					return fmt.Errorf("TemplateValid condition should be False, got %s", templateValidCondition.Status)
				}
				if templateValidCondition.Reason != kubermaticv1.PolicyBindingReasonTemplateNotFound {
					return fmt.Errorf("TemplateValid reason should be %s, got %s", kubermaticv1.PolicyBindingReasonTemplateNotFound, templateValidCondition.Reason)
				}

				// Verify Active is false
				if binding.Status.Active == nil || *binding.Status.Active {
					return fmt.Errorf("Active should be false")
				}

				// Verify finalizer was removed (cleanup happened)
				if len(binding.Finalizers) > 0 {
					return fmt.Errorf("Finalizer should be removed, got: %v", binding.Finalizers)
				}

				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			// Build seed client objects
			seedObjects := []ctrlruntimeclient.Object{tc.cluster}
			if tc.template != nil {
				seedObjects = append(seedObjects, tc.template)
			}
			seedObjects = append(seedObjects, tc.binding)

			scheme := fake.NewScheme()
			if err := kyvernov1.Install(scheme); err != nil {
				t.Fatalf("failed to add kyverno to scheme: %v", err)
			}

			seedClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(seedObjects...).
				WithStatusSubresource(tc.binding).
				Build()

			userClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tc.userObjects...).
				Build()

			r := &reconciler{
				seedClient:  seedClient,
				userClient:  userClient,
				log:         log,
				recorder:    &events.FakeRecorder{},
				namespace:   testClusterNamespace,
				clusterName: testClusterName,
				clusterIsPaused: func(ctx context.Context) (bool, error) {
					return false, nil
				},
			}

			req := reconcile.Request{
				NamespacedName: ctrlruntimeclient.ObjectKey{
					Namespace: tc.binding.Namespace,
					Name:      tc.binding.Name,
				},
			}

			_, err := r.Reconcile(ctx, req)
			if tc.expectError && err == nil {
				t.Error("expected error but got none")
				return
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Get updated binding for validation (may not exist if deleted)
			updatedBinding := &kubermaticv1.PolicyBinding{}
			getErr := seedClient.Get(ctx, req.NamespacedName, updatedBinding)
			if getErr != nil {
				// Binding might have been deleted - pass nil to validator
				updatedBinding = nil
			}

			if tc.validate != nil {
				if err := tc.validate(t, seedClient, userClient, updatedBinding); err != nil {
					t.Errorf("validation failed: %v", err)
				}
			}
		})
	}
}

func TestDeletingPolicyBindingPreservesFinalizerWhenCleanupFails(t *testing.T) {
	ctx := context.Background()
	log := zap.NewNop().Sugar()

	binding := genPolicyBindingWithDeletionTimestamp(testPolicyName, testClusterNamespace, testPolicyName)
	template := genPolicyTemplate(testPolicyName, false)
	cluster := genCluster(testClusterName, true)

	scheme := fake.NewScheme()
	if err := kyvernov1.Install(scheme); err != nil {
		t.Fatalf("failed to add kyverno to scheme: %v", err)
	}

	seedClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, template, binding).
		WithStatusSubresource(binding).
		Build()
	userClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(ctx context.Context, client ctrlruntimeclient.WithWatch, obj ctrlruntimeclient.Object, opts ...ctrlruntimeclient.DeleteOption) error {
				if _, ok := obj.(*kyvernov1.ClusterPolicy); ok {
					return fmt.Errorf("delete failed")
				}

				return client.Delete(ctx, obj, opts...)
			},
		}).
		Build()

	r := &reconciler{
		seedClient: seedClient,
		userClient: userClient,
	}

	if err := r.reconcile(ctx, log, binding, cluster); err == nil {
		t.Fatal("expected cleanup error")
	}

	updatedBinding := &kubermaticv1.PolicyBinding{}
	if err := seedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: binding.Name, Namespace: binding.Namespace}, updatedBinding); err != nil {
		t.Fatalf("expected deleting binding to keep finalizer after cleanup failure: %v", err)
	}
	if len(updatedBinding.Finalizers) != 1 || updatedBinding.Finalizers[0] != cleanupFinalizer {
		t.Fatalf("expected cleanup finalizer to be preserved, got %v", updatedBinding.Finalizers)
	}

	readyCondition := getCondition(updatedBinding, kubermaticv1.PolicyBindingConditionReady)
	if readyCondition == nil || readyCondition.Status != metav1.ConditionFalse || readyCondition.Reason != kubermaticv1.PolicyBindingReasonApplyFailed {
		t.Fatalf("expected Ready=False/%s, got %#v", kubermaticv1.PolicyBindingReasonApplyFailed, readyCondition)
	}
}

func TestDeletingPolicyBindingRemovesFinalizerWhenKyvernoAPIsAreMissing(t *testing.T) {
	ctx := context.Background()
	log := zap.NewNop().Sugar()

	binding := genPolicyBindingWithDeletionTimestamp(testPolicyName, testClusterNamespace, testPolicyName)
	binding.Spec.KyvernoPolicyNamespace = &kubermaticv1.KyvernoPolicyNamespace{Name: "old-namespace"}
	template := genPolicyTemplate(testPolicyName, true)
	cluster := genCluster(testClusterName, true)

	scheme := fake.NewScheme()
	if err := kyvernov1.Install(scheme); err != nil {
		t.Fatalf("failed to add kyverno to scheme: %v", err)
	}

	seedClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, template, binding).
		WithStatusSubresource(binding).
		Build()
	userClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(ctx context.Context, client ctrlruntimeclient.WithWatch, obj ctrlruntimeclient.Object, opts ...ctrlruntimeclient.DeleteOption) error {
				switch obj.(type) {
				case *kyvernov1.ClusterPolicy:
					return noKindMatchError("ClusterPolicy")
				case *kyvernov1.Policy:
					return noKindMatchError("Policy")
				default:
					return client.Delete(ctx, obj, opts...)
				}
			},
			List: func(ctx context.Context, client ctrlruntimeclient.WithWatch, list ctrlruntimeclient.ObjectList, opts ...ctrlruntimeclient.ListOption) error {
				switch list.(type) {
				case *kyvernov1.ClusterPolicyList:
					return noKindMatchError("ClusterPolicy")
				case *kyvernov1.PolicyList:
					return noKindMatchError("Policy")
				default:
					return client.List(ctx, list, opts...)
				}
			},
		}).
		Build()

	r := &reconciler{
		seedClient: seedClient,
		userClient: userClient,
	}

	if err := r.reconcile(ctx, log, binding, cluster); err != nil {
		t.Fatalf("expected missing Kyverno APIs to be treated as completed cleanup, got: %v", err)
	}

	updatedBinding := &kubermaticv1.PolicyBinding{}
	err := seedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: binding.Name, Namespace: binding.Namespace}, updatedBinding)
	if apierrors.IsNotFound(err) {
		return
	}
	if err != nil {
		t.Fatalf("failed to get updated binding: %v", err)
	}
	if len(updatedBinding.Finalizers) > 0 {
		t.Fatalf("expected cleanup finalizer to be removed, got %v", updatedBinding.Finalizers)
	}
}

func TestNamespacedPolicyWithoutNamespaceReportsStaleCleanupFailure(t *testing.T) {
	ctx := context.Background()
	log := zap.NewNop().Sugar()

	active := true
	binding := genPolicyBinding(testPolicyName, testClusterNamespace, testPolicyName)
	binding.Status.Active = &active

	template := genPolicyTemplate(testPolicyName, true)
	cluster := genCluster(testClusterName, true)

	scheme := fake.NewScheme()
	if err := kyvernov1.Install(scheme); err != nil {
		t.Fatalf("failed to add kyverno to scheme: %v", err)
	}

	seedClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, template, binding).
		WithStatusSubresource(binding).
		Build()
	userClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(ctx context.Context, client ctrlruntimeclient.WithWatch, list ctrlruntimeclient.ObjectList, opts ...ctrlruntimeclient.ListOption) error {
				if _, ok := list.(*kyvernov1.ClusterPolicyList); ok {
					return fmt.Errorf("stale cleanup failed")
				}

				return client.List(ctx, list, opts...)
			},
		}).
		Build()

	r := &reconciler{
		seedClient: seedClient,
		userClient: userClient,
	}

	if err := r.reconcile(ctx, log, binding, cluster); err == nil {
		t.Fatal("expected stale cleanup error")
	}

	updatedBinding := &kubermaticv1.PolicyBinding{}
	if err := seedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: binding.Name, Namespace: binding.Namespace}, updatedBinding); err != nil {
		t.Fatalf("failed to get updated binding: %v", err)
	}

	readyCondition := getCondition(updatedBinding, kubermaticv1.PolicyBindingConditionReady)
	if readyCondition == nil || readyCondition.Status != metav1.ConditionFalse || readyCondition.Reason != kubermaticv1.PolicyBindingReasonApplyFailed {
		t.Fatalf("expected Ready=False/%s, got %#v", kubermaticv1.PolicyBindingReasonApplyFailed, readyCondition)
	}

	appliedCondition := getCondition(updatedBinding, kubermaticv1.PolicyBindingConditionKyvernoPolicyApplied)
	if appliedCondition == nil || appliedCondition.Status != metav1.ConditionFalse || appliedCondition.Reason != kubermaticv1.PolicyBindingReasonApplyFailed {
		t.Fatalf("expected KyvernoPolicyApplied=False/%s, got %#v", kubermaticv1.PolicyBindingReasonApplyFailed, appliedCondition)
	}

	if updatedBinding.Status.Active == nil || !*updatedBinding.Status.Active {
		t.Fatal("expected Active to remain true while stale cleanup failed")
	}
}

func TestPolicyBindingDeletionTimestampChangedPredicate(t *testing.T) {
	predicate := policyBindingDeletionTimestampChangedPredicate()

	oldBinding := genPolicyBinding(testPolicyName, testClusterNamespace, testPolicyName)
	newBinding := oldBinding.DeepCopy()
	now := metav1.Now()
	newBinding.DeletionTimestamp = &now

	if !predicate.Update(ctrlruntimeevent.TypedUpdateEvent[*kubermaticv1.PolicyBinding]{
		ObjectOld: oldBinding,
		ObjectNew: newBinding,
	}) {
		t.Fatal("expected deletion timestamp transition to pass predicate")
	}

	if predicate.Update(ctrlruntimeevent.TypedUpdateEvent[*kubermaticv1.PolicyBinding]{
		ObjectOld: oldBinding,
		ObjectNew: oldBinding.DeepCopy(),
	}) {
		t.Fatal("expected unchanged deletion timestamp to be filtered")
	}
}

func TestUpdateStatusIgnoresDeletedPolicyBinding(t *testing.T) {
	ctx := context.Background()

	oldBinding := genPolicyBinding(testPolicyName, testClusterNamespace, testPolicyName)
	binding := oldBinding.DeepCopy()
	binding.SetStatusFields(nil, false)

	seedClient := fake.NewClientBuilder().
		WithScheme(fake.NewScheme()).
		Build()

	r := &reconciler{seedClient: seedClient}
	if err := r.updateStatus(ctx, oldBinding, binding); err != nil {
		t.Fatalf("expected NotFound status patch to be ignored, got: %v", err)
	}
}

func TestPolicyBindingRelevantClusterChangedPredicate(t *testing.T) {
	predicate := policyBindingRelevantClusterChangedPredicate(testClusterName)

	oldCluster := genCluster(testClusterName, true)
	newCluster := genCluster(testClusterName, false)
	if !predicate.Update(ctrlruntimeevent.TypedUpdateEvent[*kubermaticv1.Cluster]{
		ObjectOld: oldCluster,
		ObjectNew: newCluster,
	}) {
		t.Fatal("expected Kyverno state change to pass predicate")
	}

	unchangedCluster := genCluster(testClusterName, true)
	if predicate.Update(ctrlruntimeevent.TypedUpdateEvent[*kubermaticv1.Cluster]{
		ObjectOld: oldCluster,
		ObjectNew: unchangedCluster,
	}) {
		t.Fatal("expected unchanged Kyverno state to be filtered")
	}

	deletingCluster := oldCluster.DeepCopy()
	now := metav1.Now()
	deletingCluster.DeletionTimestamp = &now
	if !predicate.Update(ctrlruntimeevent.TypedUpdateEvent[*kubermaticv1.Cluster]{
		ObjectOld: oldCluster,
		ObjectNew: deletingCluster,
	}) {
		t.Fatal("expected deletion timestamp transition to pass predicate")
	}

	otherCluster := genCluster("other-cluster", false)
	if predicate.Update(ctrlruntimeevent.TypedUpdateEvent[*kubermaticv1.Cluster]{
		ObjectOld: genCluster("other-cluster", true),
		ObjectNew: otherCluster,
	}) {
		t.Fatal("expected other cluster to be filtered")
	}

	if predicate.Create(ctrlruntimeevent.TypedCreateEvent[*kubermaticv1.Cluster]{
		Object: newCluster,
	}) {
		t.Fatal("expected create events to be filtered")
	}

	if predicate.Delete(ctrlruntimeevent.TypedDeleteEvent[*kubermaticv1.Cluster]{
		Object: newCluster,
	}) {
		t.Fatal("expected delete events to be filtered")
	}

	if predicate.Generic(ctrlruntimeevent.TypedGenericEvent[*kubermaticv1.Cluster]{
		Object: newCluster,
	}) {
		t.Fatal("expected generic events to be filtered")
	}
}

func TestMapClusterToPolicyBindings(t *testing.T) {
	ctx := context.Background()
	log := zap.NewNop().Sugar()

	matchingBinding := genPolicyBinding("matching-binding", testClusterNamespace, testPolicyName)
	secondMatchingBinding := genPolicyBinding("second-matching-binding", testClusterNamespace, testPolicyName)
	otherNamespaceBinding := genPolicyBinding("other-binding", "cluster-other", testPolicyName)

	seedClient := fake.NewClientBuilder().
		WithScheme(fake.NewScheme()).
		WithObjects(matchingBinding, secondMatchingBinding, otherNamespaceBinding).
		Build()

	requests := mapClusterToPolicyBindings(seedClient, testClusterNamespace, testClusterName, log)(ctx, genCluster(testClusterName, false))
	got := map[string]bool{}
	for _, req := range requests {
		got[req.String()] = true
	}

	expected := []string{
		testClusterNamespace + "/matching-binding",
		testClusterNamespace + "/second-matching-binding",
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d requests, got %d: %v", len(expected), len(got), got)
	}
	for _, key := range expected {
		if !got[key] {
			t.Fatalf("expected request %q, got %v", key, got)
		}
	}

	requests = mapClusterToPolicyBindings(seedClient, testClusterNamespace, testClusterName, log)(ctx, genCluster("other-cluster", false))
	if len(requests) != 0 {
		t.Fatalf("expected other cluster to map to no requests, got %v", requests)
	}
}

func genCluster(name string, kyvernoEnabled bool) *kubermaticv1.Cluster {
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.ClusterSpec{},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: "cluster-" + name,
		},
	}
	if kyvernoEnabled {
		cluster.Spec.Kyverno = &kubermaticv1.KyvernoSettings{
			Enabled: true,
		}
	}
	return cluster
}

func genPolicyTemplate(name string, namespaced bool) *kubermaticv1.PolicyTemplate {
	return &kubermaticv1.PolicyTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.PolicyTemplateSpec{
			Title:            "Test Policy",
			Description:      "A test policy template",
			NamespacedPolicy: namespaced,
			PolicySpec: runtime.RawExtension{
				Raw: []byte(`{"rules":[]}`),
			},
		},
	}
}

func genPolicyBinding(name, namespace, templateName string) *kubermaticv1.PolicyBinding {
	return &kubermaticv1.PolicyBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: kubermaticv1.PolicyBindingSpec{
			PolicyTemplateRef: corev1.ObjectReference{
				Name: templateName,
			},
		},
	}
}

func genPolicyBindingWithFinalizer(name, namespace, templateName string) *kubermaticv1.PolicyBinding {
	binding := genPolicyBinding(name, namespace, templateName)
	binding.Finalizers = []string{cleanupFinalizer}
	return binding
}

func genPolicyBindingWithDeletionTimestamp(name, namespace, templateName string) *kubermaticv1.PolicyBinding {
	binding := genPolicyBindingWithFinalizer(name, namespace, templateName)
	now := metav1.Now()
	binding.DeletionTimestamp = &now
	return binding
}

func genClusterPolicy(name, bindingName string) *kyvernov1.ClusterPolicy {
	return &kyvernov1.ClusterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				LabelPolicyBinding: bindingName,
			},
		},
	}
}

func genPolicy(name, namespace, bindingName string) *kyvernov1.Policy {
	return &kyvernov1.Policy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				LabelPolicyBinding: bindingName,
			},
		},
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

func noKindMatchError(kind string) error {
	return &meta.NoKindMatchError{
		GroupKind:        schema.GroupKind{Group: "kyverno.io", Kind: kind},
		SearchedVersions: []string{"v1"},
	}
}
