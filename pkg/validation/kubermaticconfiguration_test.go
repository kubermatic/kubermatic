/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package validation

import (
	"testing"

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	"k8c.io/api/v2/pkg/semver"
)

func TestValidateKubermaticConfigurationVersions(t *testing.T) {
	testcases := []struct {
		name           string
		versions       []string
		defaultVersion string
		valid          bool
	}{
		{
			name:           "vanilla, single version",
			versions:       []string{"v1.10.5"},
			defaultVersion: "v1.10.5",
			valid:          true,
		},
		{
			name:           "regular version update",
			versions:       []string{"v1.10.5", "v1.11.4"},
			defaultVersion: "v1.11.4",
			valid:          true,
		},
		{
			name:           "order does not matter",
			versions:       []string{"v1.11.4", "v1.12.3", "1.9.2", "v1.10.8"},
			defaultVersion: "v1.10.8",
			valid:          true,
		},
		{
			name:           "missing v1.12",
			versions:       []string{"v1.11.4", "v1.13.3"},
			defaultVersion: "v1.13.3",
			valid:          false,
		},
		{
			name:           "order does not matter",
			versions:       []string{"v1.13.3", "v1.11.4"},
			defaultVersion: "v1.11.4",
			valid:          false,
		},
		{
			name:           "large gaps are also detected",
			versions:       []string{"v1.15.4", "v1.11.4"},
			defaultVersion: "v1.11.4",
			valid:          false,
		},
		{
			name:           "no default configured",
			versions:       []string{"v1.15.4", "v1.11.4"},
			defaultVersion: "",
			valid:          false,
		},
		{
			name:           "invalid default configured",
			versions:       []string{"v1.2.2", "v1.2.4"},
			defaultVersion: "v1.2.3",
			valid:          false,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			config := kubermaticv1.KubermaticVersioningConfiguration{}
			if tt.defaultVersion != "" {
				config.Default = semver.NewSemverOrDie(tt.defaultVersion)
			}

			for _, v := range tt.versions {
				version := semver.NewSemverOrDie(v)
				config.Versions = append(config.Versions, *version)
			}

			errs := ValidateKubermaticVersioningConfiguration(config, nil)
			if tt.valid {
				if len(errs) > 0 {
					t.Fatalf("Expected configuration to be valid, but got err: %v", errs.ToAggregate())
				}
			} else {
				if len(errs) == 0 {
					t.Fatal("Expected configuration to be invalid, but it was accepted.")
				}
			}
		})
	}
}
