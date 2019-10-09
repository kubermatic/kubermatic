package util

import (
	"context"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilpointer "k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestConcurrencyLimitReached(t *testing.T) {
	concurrencyLimitReachedTestCases := []struct {
		name                 string
		maxConcurrentLimit   int
		expectedLimitReached bool
	}{
		{
			name:                 "concurrency limit has not reached",
			maxConcurrentLimit:   2,
			expectedLimitReached: false,
		},
		{
			name:                 "concurrency limit has reached",
			maxConcurrentLimit:   1,
			expectedLimitReached: true,
		},
	}

	for _, testCase := range concurrencyLimitReachedTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			reached, err := ConcurrencyLimitReached(context.Background(), fake.NewFakeClient(&kubermaticv1.Cluster{}), testCase.maxConcurrentLimit)

			if err != nil {
				t.Fatalf("failed to run test: %v with error: %v", testCase.name, err)
			}

			if reached != testCase.expectedLimitReached {
				t.Fatalf("failed to run test: %v, expects: %v, got: %v", testCase.name, testCase.expectedLimitReached, reached)
			}
		})
	}
}

func TestSetSeedResourcesUpToDateCondition(t *testing.T) {
	var (
		cluster = &kubermaticv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "testing-namespace",
			},
		}
		updateFailingCluster = &kubermaticv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failed-cluster",
				Namespace: "testing-namespace",
			},
			Status: kubermaticv1.ClusterStatus{
				Conditions: []kubermaticv1.ClusterCondition{
					{
						Type:   kubermaticv1.SeedResourcesUpToDate,
						Status: corev1.ConditionTrue,
					},
				},
			},
		}
		inProgressStatefulSet = &appv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "testing-namespace",
			},
			Spec: appv1.StatefulSetSpec{
				Replicas: utilpointer.Int32Ptr(2),
			},
			Status: appv1.StatefulSetStatus{
				ReadyReplicas:   1,
				UpdatedReplicas: 2,
				CurrentReplicas: 2,
			},
		}
		inProgressDeployment = &appv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "testing-namespace",
			},
			Spec: appv1.DeploymentSpec{
				Replicas: utilpointer.Int32Ptr(2),
			},
			Status: appv1.DeploymentStatus{
				UpdatedReplicas:   2,
				ReadyReplicas:     1,
				AvailableReplicas: 2,
			},
		}
		readyStatefulSet = &appv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "testing-namespace",
			},
			Spec: appv1.StatefulSetSpec{
				Replicas: utilpointer.Int32Ptr(2),
			},
			Status: appv1.StatefulSetStatus{
				ReadyReplicas:   2,
				UpdatedReplicas: 2,
				CurrentReplicas: 2,
			},
		}
		readyDeployment = &appv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "testing-namespace",
			},
			Spec: appv1.DeploymentSpec{
				Replicas: utilpointer.Int32Ptr(2),
			},
			Status: appv1.DeploymentStatus{
				UpdatedReplicas:   2,
				ReadyReplicas:     2,
				AvailableReplicas: 2,
			},
		}
	)
	testcases := []struct {
		name                      string
		resources                 []runtime.Object
		successfullyReconciled    bool
		expectedHasConditionValue bool
	}{
		{
			name: "statefulSet resources are not yet updated",
			resources: []runtime.Object{
				cluster,
				inProgressStatefulSet,
			},
			successfullyReconciled:    true,
			expectedHasConditionValue: false,
		},
		{
			name: "deployments resources are not yet updated",
			resources: []runtime.Object{
				cluster,
				inProgressDeployment,
			},
			successfullyReconciled:    true,
			expectedHasConditionValue: false,
		},
		{
			name: "cluster resources have finished updating successfully",
			resources: []runtime.Object{
				cluster,
				readyStatefulSet,
				readyDeployment,
			},
			successfullyReconciled:    true,
			expectedHasConditionValue: true,
		},
		{
			name: "cluster reconcile has failed updating",
			resources: []runtime.Object{
				updateFailingCluster,
			},
			successfullyReconciled:    false,
			expectedHasConditionValue: false,
		},
	}

	for _, testCase := range testcases {
		t.Run(testCase.name, func(t *testing.T) {
			cluster, ok := testCase.resources[0].(*kubermaticv1.Cluster)
			if !ok {
				t.Fatalf("expected first resource in testcase %v to be a cluster, but was %T", testCase.name, testCase.resources[0])
			}

			err := SetSeedResourcesUpToDateCondition(context.Background(), cluster, fake.NewFakeClient(testCase.resources...), testCase.successfullyReconciled)
			if err != nil {
				t.Fatalf("failed to run test: %v with error: %v", testCase.name, err)
			}

			clusterConditionValue := cluster.Status.HasConditionValue(kubermaticv1.SeedResourcesUpToDate, corev1.ConditionTrue)
			if clusterConditionValue != testCase.expectedHasConditionValue {
				t.Fatalf("failed to run test: %v, expects: %v, got: %v", testCase.name, testCase.expectedHasConditionValue, clusterConditionValue)
			}
		})
	}
}
