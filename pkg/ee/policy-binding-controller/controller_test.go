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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testClusterName      = "test-cluster"
	testClusterNamespace = "cluster-test-cluster"
	testPolicyName       = "test-policy"
)

func TestReconcile(t *testing.T) {
	log := zap.NewNop().Sugar()

	testCases := []struct {
		name        string
		binding     *kubermaticv1.PolicyBinding
		template    *kubermaticv1.PolicyTemplate
		cluster     *kubermaticv1.Cluster
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
				if readyCondition.Reason != kubermaticv1.ReasonReady {
					return fmt.Errorf("Ready reason should be %s, got %s", kubermaticv1.ReasonReady, readyCondition.Reason)
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
				if readyCondition.Reason != kubermaticv1.ReasonTemplateNotFound {
					return fmt.Errorf("Ready reason should be %s, got %s", kubermaticv1.ReasonTemplateNotFound, readyCondition.Reason)
				}

				// Verify TemplateValid condition is False
				templateValidCondition := getCondition(binding, kubermaticv1.PolicyBindingConditionTemplateValid)
				if templateValidCondition == nil {
					return fmt.Errorf("TemplateValid condition should be set")
				}
				if templateValidCondition.Status != metav1.ConditionFalse {
					return fmt.Errorf("TemplateValid condition should be False, got %s", templateValidCondition.Status)
				}
				if templateValidCondition.Reason != kubermaticv1.ReasonTemplateNotFound {
					return fmt.Errorf("TemplateValid reason should be %s, got %s", kubermaticv1.ReasonTemplateNotFound, templateValidCondition.Reason)
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
			if err := kyvernov1.AddToScheme(scheme); err != nil {
				t.Fatalf("failed to add kyverno to scheme: %v", err)
			}

			seedClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(seedObjects...).
				WithStatusSubresource(tc.binding).
				Build()

			userClient := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			r := &reconciler{
				seedClient:  seedClient,
				userClient:  userClient,
				log:         log,
				recorder:    &record.FakeRecorder{},
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

func getCondition(binding *kubermaticv1.PolicyBinding, conditionType kubermaticv1.PolicyBindingConditionType) *metav1.Condition {
	for i := range binding.Status.Conditions {
		if binding.Status.Conditions[i].Type == string(conditionType) {
			return &binding.Status.Conditions[i]
		}
	}
	return nil
}
