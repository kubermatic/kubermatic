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

package helper

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

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
				t.Errorf("expected index %d, got index %d", tc.expectedIndex, index)
			}
			if !apiequality.Semantic.DeepEqual(condition, tc.expectedCondition) {
				t.Error("conditions are not equal")
			}
		})
	}
}

func TestSetClusterCondition_sorts(t *testing.T) {
	cluster := &kubermaticv1.Cluster{
		Status: kubermaticv1.ClusterStatus{
			Conditions: []kubermaticv1.ClusterCondition{
				{Type: kubermaticv1.ClusterConditionCloudControllerReconcilingSuccess},
				{Type: kubermaticv1.ClusterConditionAddonControllerReconcilingSuccess},
			},
		},
	}
	SetClusterCondition(
		cluster,
		kubermatic.NewDefaultVersions(),
		kubermaticv1.ClusterConditionUpdateControllerReconcilingSuccess,
		corev1.ConditionTrue,
		"",
		"",
	)
	if cluster.Status.Conditions[0].Type != kubermaticv1.ClusterConditionAddonControllerReconcilingSuccess {
		t.Fatal("ClusterConditions are unsorted")
	}
}

func TestSetClusterCondition(t *testing.T) {
	conditionType := kubermaticv1.ClusterConditionSeedResourcesUpToDate
	versions := kubermatic.NewFakeVersions()

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
				KubermaticVersion: uniqueVersion(versions),
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
				KubermaticVersion: uniqueVersion(versions),
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
				KubermaticVersion: uniqueVersion(versions),
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
				KubermaticVersion: uniqueVersion(versions),
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
			SetClusterCondition(tc.cluster, versions, conditionType, tc.conditionStatus, tc.conditionReason, tc.conditionMessage)
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
