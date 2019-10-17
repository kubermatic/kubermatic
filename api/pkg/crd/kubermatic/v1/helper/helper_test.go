package helper

import (
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
)

func TestGetClusterCondition(t *testing.T) {
	testCases := []struct {
		name              string
		cluster           *kubermaticv1.Cluster
		conditionType     kubermaticv1.ClusterConditionType
		expectedIndex     int
		expectedCondition *kubermaticv1.ClusterCondition
	}{
		{
			name:          "No condition",
			cluster:       &kubermaticv1.Cluster{},
			conditionType: kubermaticv1.ClusterConditionSeedResourcesUpToDate,
			expectedIndex: -1,
		},
		{
			name: "Condition exists",
			cluster: &kubermaticv1.Cluster{
				Status: kubermaticv1.ClusterStatus{
					Conditions: []kubermaticv1.ClusterCondition{{
						Type: kubermaticv1.ClusterConditionSeedResourcesUpToDate,
					}},
				},
			},
			conditionType: kubermaticv1.ClusterConditionSeedResourcesUpToDate,
			expectedIndex: 0,
			expectedCondition: &kubermaticv1.ClusterCondition{
				Type: kubermaticv1.ClusterConditionSeedResourcesUpToDate,
			},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			index, condition := GetClusterCondition(tc.cluster, tc.conditionType)
			if index != tc.expectedIndex {
				t.Errorf("expected index %d, got index %d", index, tc.expectedIndex)
			}
			if !apiequality.Semantic.DeepEqual(condition, tc.expectedCondition) {
				t.Error("conditions are not equal")
			}
		})
	}
}

func TestSetClusterCondition(t *testing.T) {
	conditionType := kubermaticv1.ClusterConditionSeedResourcesUpToDate
	testCases := []struct {
		name                    string
		cluster                 *kubermaticv1.Cluster
		conditionStatus         corev1.ConditionStatus
		conditionReason         string
		conditionMessage        string
		conditionChangeExpected bool
	}{
		{
			name: "Condition already exists, nothing to do",
			cluster: getCluster(&kubermaticv1.ClusterCondition{
				Type:              conditionType,
				Status:            corev1.ConditionTrue,
				KubermaticVersion: resources.KUBERMATICGITTAG + "-" + resources.KUBERMATICCOMMIT,
				Reason:            "my-reason",
				Message:           "my-message",
			}),
			conditionStatus:         corev1.ConditionTrue,
			conditionReason:         "my-reason",
			conditionMessage:        "my-message",
			conditionChangeExpected: false,
		},
		{
			name:                    "Condition doesn't exist and is created",
			cluster:                 getCluster(nil),
			conditionStatus:         corev1.ConditionTrue,
			conditionReason:         "my-reason",
			conditionMessage:        "my-message",
			conditionChangeExpected: true,
		},
		{
			name: "Update because of Kubermatic version",
			cluster: getCluster(&kubermaticv1.ClusterCondition{
				Type:              conditionType,
				Status:            corev1.ConditionTrue,
				KubermaticVersion: "outdated",
				Reason:            "my-reason",
				Message:           "my-message",
			}),
			conditionStatus:         corev1.ConditionTrue,
			conditionReason:         "my-reason",
			conditionMessage:        "my-message",
			conditionChangeExpected: true,
		},
		{
			name: "Update because of status",
			cluster: getCluster(&kubermaticv1.ClusterCondition{
				Type:              conditionType,
				Status:            corev1.ConditionFalse,
				KubermaticVersion: resources.KUBERMATICGITTAG + "-" + resources.KUBERMATICCOMMIT,
				Reason:            "my-reason",
				Message:           "my-message",
			}),
			conditionStatus:         corev1.ConditionTrue,
			conditionReason:         "my-reason",
			conditionMessage:        "my-message",
			conditionChangeExpected: true,
		},
		{
			name: "Update because of reason",
			cluster: getCluster(&kubermaticv1.ClusterCondition{
				Type:              conditionType,
				Status:            corev1.ConditionTrue,
				KubermaticVersion: resources.KUBERMATICGITTAG + "-" + resources.KUBERMATICCOMMIT,
				Reason:            "outdated-reason",
				Message:           "my-message",
			}),
			conditionStatus:         corev1.ConditionTrue,
			conditionReason:         "my-reason",
			conditionMessage:        "my-message",
			conditionChangeExpected: true,
		},
		{
			name: "Update because of message",
			cluster: getCluster(&kubermaticv1.ClusterCondition{
				Type:              conditionType,
				Status:            corev1.ConditionTrue,
				KubermaticVersion: resources.KUBERMATICGITTAG + "-" + resources.KUBERMATICCOMMIT,
				Reason:            "my-reason",
				Message:           "outdated-message",
			}),
			conditionStatus:         corev1.ConditionTrue,
			conditionReason:         "my-reason",
			conditionMessage:        "my-message",
			conditionChangeExpected: true,
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			initialCluster := tc.cluster.DeepCopy()
			SetClusterCondition(tc.cluster, conditionType, tc.conditionStatus, tc.conditionReason, tc.conditionMessage)
			hasChanged := !apiequality.Semantic.DeepEqual(initialCluster, tc.cluster)
			if hasChanged != tc.conditionChangeExpected {
				t.Errorf("Change doesn't match expectation: hasChanged: %t: changeExpected: %t", hasChanged, tc.conditionChangeExpected)
			}
		})
	}
}

func getCluster(condition *kubermaticv1.ClusterCondition) *kubermaticv1.Cluster {
	c := &kubermaticv1.Cluster{}
	if condition != nil {
		c.Status.Conditions = []kubermaticv1.ClusterCondition{*condition}
	}
	return c
}
