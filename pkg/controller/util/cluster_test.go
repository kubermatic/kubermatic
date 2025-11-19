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

package util

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/utils/ptr"
)

func TestSetClusterCondition(t *testing.T) {
	conditionType := kubermaticv1.ClusterConditionSeedResourcesUpToDate
	versions := kubermatic.GetFakeVersions()

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
			cluster: getCluster(conditionType, &kubermaticv1.ClusterCondition{
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
			cluster:                 getCluster("", nil),
			conditionStatus:         corev1.ConditionTrue,
			conditionReason:         "my-reason",
			conditionMessage:        "my-message",
			conditionChangeExpected: true,
		},
		{
			name: "Update because of Kubermatic version",
			cluster: getCluster(conditionType, &kubermaticv1.ClusterCondition{
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
			cluster: getCluster(conditionType, &kubermaticv1.ClusterCondition{
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
			cluster: getCluster(conditionType, &kubermaticv1.ClusterCondition{
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
			cluster: getCluster(conditionType, &kubermaticv1.ClusterCondition{
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

func getCluster(conditionType kubermaticv1.ClusterConditionType, condition *kubermaticv1.ClusterCondition) *kubermaticv1.Cluster {
	c := &kubermaticv1.Cluster{}
	if condition != nil {
		c.Status.Conditions = map[kubermaticv1.ClusterConditionType]kubermaticv1.ClusterCondition{
			conditionType: *condition,
		}
	}

	return c
}

func TestGetKyvernoEnforcement(t *testing.T) {
	testCases := []struct {
		name             string
		dcSettings       *kubermaticv1.KyvernoConfigurations
		seedSettings     *kubermaticv1.KyvernoConfigurations
		globalSettings   *kubermaticv1.KyvernoConfigurations
		expectedEnforced bool
		expectedSource   KyvernoEnforcementSource
	}{
		{
			name:             "all parameters nil - no enforcement",
			dcSettings:       nil,
			seedSettings:     nil,
			globalSettings:   nil,
			expectedEnforced: false,
			expectedSource:   KyvernoEnforcementSourceNone,
		},
		{
			name:             "all non-nil but Enforced fields are nil - no enforcement",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: nil},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: nil},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: nil},
			expectedEnforced: false,
			expectedSource:   KyvernoEnforcementSourceNone,
		},
		{
			name:             "only DC enforces true",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			seedSettings:     nil,
			globalSettings:   nil,
			expectedEnforced: true,
			expectedSource:   KyvernoEnforcementSourceDatacenter,
		},
		{
			name:             "only DC explicitly disables (false)",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			seedSettings:     nil,
			globalSettings:   nil,
			expectedEnforced: false,
			expectedSource:   KyvernoEnforcementSourceDatacenter,
		},
		{
			name:             "only Seed enforces true",
			dcSettings:       nil,
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			globalSettings:   nil,
			expectedEnforced: true,
			expectedSource:   KyvernoEnforcementSourceSeed,
		},
		{
			name:             "only Seed explicitly disables (false)",
			dcSettings:       nil,
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			globalSettings:   nil,
			expectedEnforced: false,
			expectedSource:   KyvernoEnforcementSourceSeed,
		},
		{
			name:             "only Global enforces true",
			dcSettings:       nil,
			seedSettings:     nil,
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			expectedEnforced: true,
			expectedSource:   KyvernoEnforcementSourceGlobal,
		},
		{
			name:             "only Global explicitly disables (false)",
			dcSettings:       nil,
			seedSettings:     nil,
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			expectedEnforced: false,
			expectedSource:   KyvernoEnforcementSourceGlobal,
		},
		{
			name:             "DC overrides Seed enforcement",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			globalSettings:   nil,
			expectedEnforced: false,
			expectedSource:   KyvernoEnforcementSourceDatacenter,
		},
		{
			name:             "DC overrides Global enforcement",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			seedSettings:     nil,
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			expectedEnforced: false,
			expectedSource:   KyvernoEnforcementSourceDatacenter,
		},
		{
			name:             "DC overrides both Seed and Global",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			expectedEnforced: false,
			expectedSource:   KyvernoEnforcementSourceDatacenter,
		},
		{
			name:             "Seed overrides Global enforcement",
			dcSettings:       nil,
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			expectedEnforced: false,
			expectedSource:   KyvernoEnforcementSourceSeed,
		},
		{
			name:             "Seed with true overrides Global with false",
			dcSettings:       nil,
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			expectedEnforced: true,
			expectedSource:   KyvernoEnforcementSourceSeed,
		},
		{
			name:             "DC with true overrides Seed with false",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			globalSettings:   nil,
			expectedEnforced: true,
			expectedSource:   KyvernoEnforcementSourceDatacenter,
		},
		{
			name:             "DC exists with nil Enforced, Seed enforces",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: nil},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			globalSettings:   nil,
			expectedEnforced: true,
			expectedSource:   KyvernoEnforcementSourceSeed,
		},
		{
			name:             "DC and Seed exist with nil Enforced, Global enforces",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: nil},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: nil},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			expectedEnforced: true,
			expectedSource:   KyvernoEnforcementSourceGlobal,
		},
		{
			name:             "all exist but only Global has opinion",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: nil},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: nil},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			expectedEnforced: false,
			expectedSource:   KyvernoEnforcementSourceGlobal,
		},
		{
			name:             "all three enforce",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			expectedEnforced: true,
			expectedSource:   KyvernoEnforcementSourceDatacenter,
		},
		{
			name:             "all three disable",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			expectedEnforced: false,
			expectedSource:   KyvernoEnforcementSourceDatacenter,
		},
		{
			name:             "mixed - DC=true, Seed=false, Global=true",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			expectedEnforced: true,
			expectedSource:   KyvernoEnforcementSourceDatacenter,
		},
		{
			name:             "global enforcement with Seed opt-out (admin enforces globally, dev seed opts out)",
			dcSettings:       nil,
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			expectedEnforced: false,
			expectedSource:   KyvernoEnforcementSourceSeed,
		},
		{
			name:             "global enforcement with DC override (global enforcement, legacy DC needs exception)",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			seedSettings:     nil,
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			expectedEnforced: false,
			expectedSource:   KyvernoEnforcementSourceDatacenter,
		},
		{
			name:             "Seed enforces, DC opts out (prod seed enforces, special DC for testing)",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			globalSettings:   nil,
			expectedEnforced: false,
			expectedSource:   KyvernoEnforcementSourceDatacenter,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := GetKyvernoEnforcement(tc.dcSettings, tc.seedSettings, tc.globalSettings)
			if result.Enforced != tc.expectedEnforced {
				t.Errorf("expected Enforced=%v, got %v", tc.expectedEnforced, result.Enforced)
			}
			if result.Source != tc.expectedSource {
				t.Errorf("expected Source=%q, got %q", tc.expectedSource, result.Source)
			}
		})
	}
}
