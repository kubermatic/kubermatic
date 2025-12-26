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

package kyverno

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/utils/ptr"
)

func TestGetEnforcement(t *testing.T) {
	testCases := []struct {
		name             string
		dcSettings       *kubermaticv1.KyvernoConfigurations
		seedSettings     *kubermaticv1.KyvernoConfigurations
		globalSettings   *kubermaticv1.KyvernoConfigurations
		expectedEnforced bool
		expectedSource   EnforcementSource
	}{
		{
			name:             "all parameters nil - no enforcement",
			dcSettings:       nil,
			seedSettings:     nil,
			globalSettings:   nil,
			expectedEnforced: false,
			expectedSource:   EnforcementSourceNone,
		},
		{
			name:             "all non-nil but Enforced fields are nil - no enforcement",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: nil},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: nil},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: nil},
			expectedEnforced: false,
			expectedSource:   EnforcementSourceNone,
		},
		{
			name:             "only DC enforces true",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			seedSettings:     nil,
			globalSettings:   nil,
			expectedEnforced: true,
			expectedSource:   EnforcementSourceDatacenter,
		},
		{
			name:             "only DC explicitly disables (false)",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			seedSettings:     nil,
			globalSettings:   nil,
			expectedEnforced: false,
			expectedSource:   EnforcementSourceDatacenter,
		},
		{
			name:             "only Seed enforces true",
			dcSettings:       nil,
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			globalSettings:   nil,
			expectedEnforced: true,
			expectedSource:   EnforcementSourceSeed,
		},
		{
			name:             "only Seed explicitly disables (false)",
			dcSettings:       nil,
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			globalSettings:   nil,
			expectedEnforced: false,
			expectedSource:   EnforcementSourceSeed,
		},
		{
			name:             "only Global enforces true",
			dcSettings:       nil,
			seedSettings:     nil,
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			expectedEnforced: true,
			expectedSource:   EnforcementSourceGlobal,
		},
		{
			name:             "only Global explicitly disables (false)",
			dcSettings:       nil,
			seedSettings:     nil,
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			expectedEnforced: false,
			expectedSource:   EnforcementSourceGlobal,
		},
		{
			name:             "DC overrides Seed enforcement",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			globalSettings:   nil,
			expectedEnforced: false,
			expectedSource:   EnforcementSourceDatacenter,
		},
		{
			name:             "DC overrides Global enforcement",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			seedSettings:     nil,
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			expectedEnforced: false,
			expectedSource:   EnforcementSourceDatacenter,
		},
		{
			name:             "DC overrides both Seed and Global",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			expectedEnforced: false,
			expectedSource:   EnforcementSourceDatacenter,
		},
		{
			name:             "Seed overrides Global enforcement",
			dcSettings:       nil,
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			expectedEnforced: false,
			expectedSource:   EnforcementSourceSeed,
		},
		{
			name:             "Seed with true overrides Global with false",
			dcSettings:       nil,
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			expectedEnforced: true,
			expectedSource:   EnforcementSourceSeed,
		},
		{
			name:             "DC with true overrides Seed with false",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			globalSettings:   nil,
			expectedEnforced: true,
			expectedSource:   EnforcementSourceDatacenter,
		},
		{
			name:             "DC exists with nil Enforced, Seed enforces",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: nil},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			globalSettings:   nil,
			expectedEnforced: true,
			expectedSource:   EnforcementSourceSeed,
		},
		{
			name:             "DC and Seed exist with nil Enforced, Global enforces",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: nil},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: nil},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			expectedEnforced: true,
			expectedSource:   EnforcementSourceGlobal,
		},
		{
			name:             "all exist but only Global has opinion",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: nil},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: nil},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			expectedEnforced: false,
			expectedSource:   EnforcementSourceGlobal,
		},
		{
			name:             "all three enforce",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			expectedEnforced: true,
			expectedSource:   EnforcementSourceDatacenter,
		},
		{
			name:             "all three disable",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			expectedEnforced: false,
			expectedSource:   EnforcementSourceDatacenter,
		},
		{
			name:             "mixed - DC=true, Seed=false, Global=true",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			expectedEnforced: true,
			expectedSource:   EnforcementSourceDatacenter,
		},
		{
			name:             "global enforcement with Seed opt-out (admin enforces globally, dev seed opts out)",
			dcSettings:       nil,
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			expectedEnforced: false,
			expectedSource:   EnforcementSourceSeed,
		},
		{
			name:             "global enforcement with DC override (global enforcement, legacy DC needs exception)",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			seedSettings:     nil,
			globalSettings:   &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			expectedEnforced: false,
			expectedSource:   EnforcementSourceDatacenter,
		},
		{
			name:             "Seed enforces, DC opts out (prod seed enforces, special DC for testing)",
			dcSettings:       &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(false)},
			seedSettings:     &kubermaticv1.KyvernoConfigurations{Enforced: ptr.To(true)},
			globalSettings:   nil,
			expectedEnforced: false,
			expectedSource:   EnforcementSourceDatacenter,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := GetEnforcement(tc.dcSettings, tc.seedSettings, tc.globalSettings)
			if result.Enforced != tc.expectedEnforced {
				t.Errorf("expected Enforced=%v, got %v", tc.expectedEnforced, result.Enforced)
			}
			if result.Source != tc.expectedSource {
				t.Errorf("expected Source=%q, got %q", tc.expectedSource, result.Source)
			}
		})
	}
}
