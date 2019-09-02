package util

import (
	"context"
	"errors"
	"fmt"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilpointer "k8s.io/utils/pointer"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	testSetup = struct {
		limit                     int
		readyClusters             []kubermaticv1.Cluster
		inProgressClusters        []kubermaticv1.Cluster
		readyStatefulSetList      []v1.StatefulSet
		inProgressStatefulSetList []v1.StatefulSet
		readyDeploymentList       []v1.Deployment
		inProgressDeploymentList  []v1.Deployment
	}{
		limit: 2,
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
		readyStatefulSetList: []v1.StatefulSet{
			{
				Spec: v1.StatefulSetSpec{
					Replicas: utilpointer.Int32Ptr(2),
				},
				Status: v1.StatefulSetStatus{
					ReadyReplicas:   2,
					UpdatedReplicas: 2,
					CurrentReplicas: 2,
				},
			},
		},
		inProgressStatefulSetList: []v1.StatefulSet{
			{
				Spec: v1.StatefulSetSpec{
					Replicas: utilpointer.Int32Ptr(2),
				},
				Status: v1.StatefulSetStatus{
					ReadyReplicas:   1,
					UpdatedReplicas: 2,
					CurrentReplicas: 2,
				},
			},
		},
		readyDeploymentList: []v1.Deployment{
			{
				Spec: v1.DeploymentSpec{
					Replicas: utilpointer.Int32Ptr(2),
				},
				Status: v1.DeploymentStatus{
					UpdatedReplicas:   2,
					ReadyReplicas:     2,
					AvailableReplicas: 2,
				},
			},
		},
		inProgressDeploymentList: []v1.Deployment{
			{
				Spec: v1.DeploymentSpec{
					Replicas: utilpointer.Int32Ptr(2),
				},
				Status: v1.DeploymentStatus{
					UpdatedReplicas:   1,
					ReadyReplicas:     2,
					AvailableReplicas: 2,
				},
			},
		},
	}
)

func TestConcurrencyLimitReached(t *testing.T) {
	var (
		ctx    = context.Background()
		client = &mockClient{
			errorOccurred: true,
		}
	)

	reached, err := ConcurrencyLimitReached(ctx, client, testSetup.limit)
	if err == nil || !reached {
		t.Fatal("error should not be nil and reached should be true")
	}

	client.errorOccurred = false
	reached, err = ConcurrencyLimitReached(ctx, client, testSetup.limit)
	if err != nil {
		t.Fatal(err)
	}

	if reached {
		t.Fatal("max concurrent updates should not have been reached")
	}

	client.limitReached = true
	reached, err = ConcurrencyLimitReached(ctx, client, testSetup.limit)
	if err != nil {
		t.Fatal(err)
	}

	if !reached {
		t.Fatal("max concurrent updates should have been reached")
	}
}

func TestSetClusterUpdatedSuccessfullyCondition(t *testing.T) {
	var (
		ctx    = context.Background()
		client = &mockClient{
			inProgressStatefulSet: true,
			inProgressDeployments: true,
		}
		testCluster = kubermaticv1.Cluster{}
	)

	err := SetClusterUpdatedSuccessfullyCondition(ctx, &testCluster, client)
	if err != nil {
		t.Fatal("error has occurred: ", err)
	}

	if ok := hasConditionValue(testCluster,
		kubermaticv1.ClusterConditionControllerFinishedUpdatingSuccessfully,
		corev1.ConditionTrue); ok {
		t.Fatal(fmt.Sprintf("resources are still updating, condition %v shouldn't be there",
			kubermaticv1.ClusterConditionControllerFinishedUpdatingSuccessfully))
	}

	client.inProgressStatefulSet = false
	err = SetClusterUpdatedSuccessfullyCondition(ctx, &testCluster, client)
	if err != nil {
		t.Fatal("error has occurred: ", err)
	}

	if ok := hasConditionValue(testCluster,
		kubermaticv1.ClusterConditionControllerFinishedUpdatingSuccessfully,
		corev1.ConditionTrue); ok {
		t.Fatal(fmt.Sprintf("resources are still updating, condition %v shouldn't be there",
			kubermaticv1.ClusterConditionControllerFinishedUpdatingSuccessfully))
	}

	client.inProgressDeployments = false
	err = SetClusterUpdatedSuccessfullyCondition(ctx, &testCluster, client)
	if err != nil {
		t.Fatal("error has occurred: ", err)
	}

	if ok := hasConditionValue(testCluster,
		kubermaticv1.ClusterConditionControllerFinishedUpdatingSuccessfully,
		corev1.ConditionTrue); !ok {
		t.Fatal(fmt.Sprintf("resources are still updating, condition %v shouldn't be there",
			kubermaticv1.ClusterConditionControllerFinishedUpdatingSuccessfully))
	}
}

type mockClient struct {
	ctrlruntimeclient.Client
	limitReached          bool
	errorOccurred         bool
	inProgressDeployments bool
	inProgressStatefulSet bool
}

func (m *mockClient) List(ctx context.Context, opts *ctrlruntimeclient.ListOptions, list runtime.Object) error {
	if m.errorOccurred {
		return errors.New("error occurred while listing the clusters")
	}

	switch obj := list.(type) {
	case *kubermaticv1.ClusterList:
		if m.limitReached {
			obj.Items = testSetup.inProgressClusters
		} else {
			obj.Items = testSetup.readyClusters
		}
	case *v1.DeploymentList:
		if m.inProgressDeployments {
			obj.Items = testSetup.inProgressDeploymentList
		} else {
			obj.Items = testSetup.readyDeploymentList
		}

	case *v1.StatefulSetList:
		if m.inProgressStatefulSet {
			obj.Items = testSetup.inProgressStatefulSetList
		} else {
			obj.Items = testSetup.readyStatefulSetList
		}
	}

	return nil
}

func (m *mockClient) Update(ctx context.Context, obj runtime.Object) error {
	return nil
}
