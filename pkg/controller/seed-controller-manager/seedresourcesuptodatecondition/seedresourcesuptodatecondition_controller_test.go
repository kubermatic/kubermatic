package seedresourcesuptodatecondition

import (
	"context"
	"errors"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilpointer "k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// There is a good chance someone wants simplify the code
// and make an if err := r.reconcile(); err != nil {} simplication,
// accidentally shortcircuiting the workqeue and retrying
func TestReconcileReturnsError(t *testing.T) {
	r := &reconciler{
		client: &fakeClientThatErrorsOnGet{fakectrlruntimeclient.NewFakeClient()},
	}
	expectedErr := `failed to get cluster "": erroring on get as requested`
	if _, err := r.Reconcile(reconcile.Request{}); err == nil || err.Error() != expectedErr {
		t.Fatalf("Expected error %v, got error %v", expectedErr, err)
	}
}

type fakeClientThatErrorsOnGet struct {
	ctrlruntimeclient.Client
}

func (f *fakeClientThatErrorsOnGet) Get(
	_ context.Context,
	key ctrlruntimeclient.ObjectKey,
	_ runtime.Object,
) error {
	return errors.New("erroring on get as requested")
}

func TestSetSeedResourcesUpToDateCondition(t *testing.T) {
	var (
		inProgressStatefulSet = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "testing-namespace",
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: utilpointer.Int32Ptr(2),
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas:   1,
				UpdatedReplicas: 2,
				CurrentReplicas: 2,
			},
		}
		inProgressDeployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "testing-namespace",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: utilpointer.Int32Ptr(2),
			},
			Status: appsv1.DeploymentStatus{
				UpdatedReplicas:   2,
				ReadyReplicas:     1,
				AvailableReplicas: 2,
			},
		}
		readyStatefulSet = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "testing-namespace",
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: utilpointer.Int32Ptr(2),
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas:   2,
				UpdatedReplicas: 2,
				CurrentReplicas: 2,
			},
		}
		readyDeployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "testing-namespace",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: utilpointer.Int32Ptr(2),
			},
			Status: appsv1.DeploymentStatus{
				UpdatedReplicas:   2,
				ReadyReplicas:     2,
				AvailableReplicas: 2,
			},
		}
	)
	testcases := []struct {
		name                      string
		cluster                   *kubermaticv1.Cluster
		resources                 []runtime.Object
		expectedHasConditionValue bool
	}{
		{
			name:    "statefulSet resources are not yet updated",
			cluster: cluster(),
			resources: []runtime.Object{
				inProgressStatefulSet,
			},
			expectedHasConditionValue: false,
		},
		{
			name:    "deployments resources are not yet updated",
			cluster: cluster(),
			resources: []runtime.Object{
				inProgressDeployment,
			},
			expectedHasConditionValue: false,
		},
		{
			name:    "cluster resources have finished updating successfully",
			cluster: cluster(),
			resources: []runtime.Object{
				readyStatefulSet,
				readyDeployment,
			},
			expectedHasConditionValue: true,
		},
	}

	ctx := context.Background()
	for _, testCase := range testcases {
		t.Run(testCase.name, func(t *testing.T) {
			client := fakectrlruntimeclient.NewFakeClient(append(testCase.resources, testCase.cluster)...)
			r := &reconciler{
				client: client,
			}
			if err := r.reconcile(testCase.cluster); err != nil {
				t.Fatalf("Error calling reconcile: %v", err)
			}

			newCluster := &kubermaticv1.Cluster{}
			if err := client.Get(ctx, types.NamespacedName{Name: testCase.cluster.Name}, newCluster); err != nil {
				t.Fatalf("failed to get cluster after it was updated: %v", err)
			}
			clusterConditionValue := newCluster.Status.HasConditionValue(kubermaticv1.ClusterConditionSeedResourcesUpToDate, corev1.ConditionTrue)
			if clusterConditionValue != testCase.expectedHasConditionValue {
				t.Fatalf("condition doesn't have expected value, expects: %v, got: %v", testCase.expectedHasConditionValue, clusterConditionValue)
			}
		})
	}
}

func cluster() *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "testing-namespace",
		},
	}
}
