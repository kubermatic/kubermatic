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

package seedresourcesuptodatecondition

import (
	"context"
	"errors"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// There is a good chance someone wants simplify the code
// and make an if err := r.reconcile(); err != nil {} simplification,
// accidentally shortcircuiting the workqeue and retrying.
func TestReconcileReturnsError(t *testing.T) {
	ctx := context.Background()
	r := &reconciler{
		client: &fakeClientThatErrorsOnGet{fake.NewClientBuilder().Build()},
	}
	expectedErr := `failed to get cluster "": erroring on get as requested`
	if _, err := r.Reconcile(ctx, reconcile.Request{}); err == nil || err.Error() != expectedErr {
		t.Fatalf("Expected error %v, got error %v", expectedErr, err)
	}
}

type fakeClientThatErrorsOnGet struct {
	ctrlruntimeclient.Client
}

func (f *fakeClientThatErrorsOnGet) Get(
	_ context.Context,
	key ctrlruntimeclient.ObjectKey,
	_ ctrlruntimeclient.Object,
	_ ...ctrlruntimeclient.GetOption,
) error {
	return errors.New("erroring on get as requested")
}

func TestSetSeedResourcesUpToDateCondition(t *testing.T) {
	var (
		inProgressStatefulSet = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sts-in-progress",
				Namespace: "cluster-test",
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: ptr.To[int32](2),
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas:   1,
				UpdatedReplicas: 2,
				CurrentReplicas: 2,
			},
		}
		inProgressDeployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deployment-in-progress",
				Namespace: "cluster-test",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: ptr.To[int32](2),
			},
			Status: appsv1.DeploymentStatus{
				UpdatedReplicas:   2,
				ReadyReplicas:     1,
				AvailableReplicas: 2,
			},
		}
		readyStatefulSet = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sts-ready",
				Namespace: "cluster-test",
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: ptr.To[int32](2),
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas:   2,
				UpdatedReplicas: 2,
				CurrentReplicas: 2,
			},
		}
		readyDeployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deployment-ready",
				Namespace: "cluster-test",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: ptr.To[int32](2),
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
		resources                 []ctrlruntimeclient.Object
		expectedHasConditionValue bool
	}{
		{
			name:    "statefulSet resources are not yet updated",
			cluster: cluster(),
			resources: []ctrlruntimeclient.Object{
				inProgressStatefulSet,
			},
			expectedHasConditionValue: false,
		},
		{
			name:    "deployments resources are not yet updated",
			cluster: cluster(),
			resources: []ctrlruntimeclient.Object{
				inProgressDeployment,
			},
			expectedHasConditionValue: false,
		},
		{
			name:    "cluster resources have finished updating successfully",
			cluster: cluster(),
			resources: []ctrlruntimeclient.Object{
				readyStatefulSet,
				readyDeployment,
			},
			expectedHasConditionValue: true,
		},
	}

	for _, testCase := range testcases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			client := fake.
				NewClientBuilder().
				WithObjects(append(testCase.resources, testCase.cluster)...).
				Build()

			r := &reconciler{
				client: client,
			}
			if err := r.reconcile(ctx, testCase.cluster); err != nil {
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
			Name: "test",
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: kubernetes.NamespaceName("test"),
		},
	}
}
