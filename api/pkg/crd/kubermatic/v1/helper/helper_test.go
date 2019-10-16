package helper

import (
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

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
	testCases := []struct {
		name                    string
		cluster                 *kubermaticv1.Cluster
		expectedCondition       kubermaticv1.ClusterCondition
		conditionChangeExpected bool
	}{
		{
			name: "Condition already exists, nothing to do",
		},
		{
			name: "Condition doesn't exist and is created",
		},
		{
			name: "Update because of Kubermatic version",
		},
		{
			name: "Update because of Kubermatic version",
		},
		{
			name: "Update because of status",
		},
		{
			name: "Update because of reason",
		},
		{
			name: "Update because of message",
		},
	}
}
