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

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	concurrencyLimitReachedTestCases = []struct {
		name  string
		wants struct {
			ctx    context.Context
			client *mockClient
			limit  int
		}
		expects struct {
			limitReached bool
			err          error
		}
	}{
		{
			name: "concurrency limit has not reached",
			wants: struct {
				ctx    context.Context
				client *mockClient
				limit  int
			}{
				ctx:    context.Background(),
				client: &mockClient{},
				limit:  1,
			},

			expects: struct {
				limitReached bool
				err          error
			}{

				limitReached: false,
				err:          nil,
			},
		},
		{
			name: "concurrency limit has reached",
			wants: struct {
				ctx    context.Context
				client *mockClient
				limit  int
			}{
				ctx: context.Background(),
				client: &mockClient{
					limitReached: true,
				},
				limit: 2,
			},

			expects: struct {
				limitReached bool
				err          error
			}{

				limitReached: true,
				err:          nil,
			},
		},
	}

	setClusterUpdatedSuccessfullyConditionTestCases = []struct {
		name  string
		wants struct {
			ctx                    context.Context
			client                 *mockClient
			cluster                kubermaticv1.Cluster
			successfullyReconciled bool
		}
		expects struct {
			hasConditionValue bool
			err               error
		}
	}{
		{
			name: "statefulSet resources are not yet updated",
			wants: struct {
				ctx                    context.Context
				client                 *mockClient
				cluster                kubermaticv1.Cluster
				successfullyReconciled bool
			}{
				ctx: context.Background(),
				client: &mockClient{
					inProgressStatefulSet: true,
				},
				cluster:                kubermaticv1.Cluster{},
				successfullyReconciled: true,
			},
			expects: struct {
				hasConditionValue bool
				err               error
			}{
				hasConditionValue: false,
				err:               nil,
			},
		},
		{
			name: "deployments resources are not yet updated",
			wants: struct {
				ctx                    context.Context
				client                 *mockClient
				cluster                kubermaticv1.Cluster
				successfullyReconciled bool
			}{
				ctx: context.Background(),
				client: &mockClient{
					inProgressDeployments: true,
				},
				cluster:                kubermaticv1.Cluster{},
				successfullyReconciled: true,
			},
			expects: struct {
				hasConditionValue bool
				err               error
			}{
				hasConditionValue: false,
				err:               nil,
			},
		},
		{
			name: "cluster resources have finished updating successfully",
			wants: struct {
				ctx                    context.Context
				client                 *mockClient
				cluster                kubermaticv1.Cluster
				successfullyReconciled bool
			}{
				ctx:                    context.Background(),
				client:                 &mockClient{},
				cluster:                kubermaticv1.Cluster{},
				successfullyReconciled: true,
			},
			expects: struct {
				hasConditionValue bool
				err               error
			}{
				hasConditionValue: true,
				err:               nil,
			},
		},
		{
			name: "cluster reconcile has failed updating",
			wants: struct {
				ctx                    context.Context
				client                 *mockClient
				cluster                kubermaticv1.Cluster
				successfullyReconciled bool
			}{
				ctx:    context.Background(),
				client: &mockClient{},
				cluster: kubermaticv1.Cluster{
					Status: kubermaticv1.ClusterStatus{
						Conditions: []kubermaticv1.ClusterCondition{
							{
								Type:   kubermaticv1.ClusterConditionControllerFinishedUpdatingSuccessfully,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				successfullyReconciled: false,
			},
			expects: struct {
				hasConditionValue bool
				err               error
			}{
				hasConditionValue: false,
				err:               nil,
			},
		},
	}
)

func TestConcurrencyLimitReached(t *testing.T) {
	for _, testCase := range concurrencyLimitReachedTestCases {
		reached, err := ConcurrencyLimitReached(testCase.wants.ctx, testCase.wants.client, testCase.wants.limit)

		if err != testCase.expects.err {
			t.Fatalf("failed to run test: %v with error: %v", testCase.name, err)
		}

		if reached != testCase.expects.limitReached {
			t.Fatalf("failed to run test: %v, expects: %v, got: %v", testCase.name, testCase.expects.limitReached, reached)
		}
	}
}

func TestSetClusterUpdatedSuccessfullyCondition(t *testing.T) {
	for _, testCase := range setClusterUpdatedSuccessfullyConditionTestCases {
		err := SetClusterUpdatedSuccessfullyCondition(testCase.wants.ctx, &testCase.wants.cluster, testCase.wants.client, testCase.wants.successfullyReconciled)
		if err != testCase.expects.err {
			t.Fatalf("failed to run test: %v with error: %v", testCase.name, err)
		}

		clusterConditionValue := hasConditionValue(testCase.wants.cluster,
			kubermaticv1.ClusterConditionControllerFinishedUpdatingSuccessfully,
			corev1.ConditionTrue)

		if clusterConditionValue != testCase.expects.hasConditionValue {
			t.Fatalf("failed to run test: %v, expects: %v, got: %v", testCase.name, testCase.expects.hasConditionValue, clusterConditionValue)
		}
	}
}

type mockClient struct {
	ctrlruntimeclient.Client
	limitReached          bool
	inProgressDeployments bool
	inProgressStatefulSet bool
}

func (m *mockClient) List(ctx context.Context, opts *ctrlruntimeclient.ListOptions, list runtime.Object) error {
	testingData := struct {
		readyClusters             []kubermaticv1.Cluster
		inProgressClusters        []kubermaticv1.Cluster
		readyStatefulSetList      []appv1.StatefulSet
		inProgressStatefulSetList []appv1.StatefulSet
		readyDeploymentList       []appv1.Deployment
		inProgressDeploymentList  []appv1.Deployment
	}{
		readyClusters: []kubermaticv1.Cluster{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ready-cluster-1",
				},
				Status: kubermaticv1.ClusterStatus{
					Conditions: []kubermaticv1.ClusterCondition{
						{
							Type:   kubermaticv1.ClusterConditionControllerFinishedUpdatingSuccessfully,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ready-cluster-2",
				},
				Status: kubermaticv1.ClusterStatus{
					Conditions: []kubermaticv1.ClusterCondition{
						{
							Type:   kubermaticv1.ClusterConditionControllerFinishedUpdatingSuccessfully,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		},
		inProgressClusters: []kubermaticv1.Cluster{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "in-progress-cluster-1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "in-progress-cluster-2",
				},
			},
		},
		readyStatefulSetList: []appv1.StatefulSet{
			{
				Spec: appv1.StatefulSetSpec{
					Replicas: utilpointer.Int32Ptr(2),
				},
				Status: appv1.StatefulSetStatus{
					ReadyReplicas:   2,
					UpdatedReplicas: 2,
					CurrentReplicas: 2,
				},
			},
		},
		inProgressStatefulSetList: []appv1.StatefulSet{
			{
				Spec: appv1.StatefulSetSpec{
					Replicas: utilpointer.Int32Ptr(2),
				},
				Status: appv1.StatefulSetStatus{
					ReadyReplicas:   1,
					UpdatedReplicas: 2,
					CurrentReplicas: 2,
				},
			},
		},
		readyDeploymentList: []appv1.Deployment{
			{
				Spec: appv1.DeploymentSpec{
					Replicas: utilpointer.Int32Ptr(2),
				},
				Status: appv1.DeploymentStatus{
					UpdatedReplicas:   2,
					ReadyReplicas:     2,
					AvailableReplicas: 2,
				},
			},
		},
		inProgressDeploymentList: []appv1.Deployment{
			{
				Spec: appv1.DeploymentSpec{
					Replicas: utilpointer.Int32Ptr(2),
				},
				Status: appv1.DeploymentStatus{
					UpdatedReplicas:   1,
					ReadyReplicas:     2,
					AvailableReplicas: 2,
				},
			},
		},
	}

	switch obj := list.(type) {
	case *kubermaticv1.ClusterList:
		if m.limitReached {
			obj.Items = testingData.inProgressClusters
		} else {
			obj.Items = testingData.readyClusters
		}
	case *appv1.DeploymentList:
		if m.inProgressDeployments {
			obj.Items = testingData.inProgressDeploymentList
		} else {
			obj.Items = testingData.readyDeploymentList
		}

	case *appv1.StatefulSetList:
		if m.inProgressStatefulSet {
			obj.Items = testingData.inProgressStatefulSetList
		} else {
			obj.Items = testingData.readyStatefulSetList
		}
	}

	return nil
}

func (m *mockClient) Update(ctx context.Context, obj runtime.Object) error {
	return nil
}
