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

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
)

func TestValidateKubermaticConfigurationVersions(t *testing.T) {
	testcases := []struct {
		name           string
		versions       []string
		defaultVersion string
		updates        []kubermaticv1.Update
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
		{
			name:           "should allow updates with automatic update rules from concrete version",
			versions:       []string{"v1.11.1", "v1.12.2"},
			defaultVersion: "v1.11.1",
			updates: []kubermaticv1.Update{{
				From:      "v1.11.1",
				To:        "v1.12.2",
				Automatic: ptr.To(true),
			}},
			valid: true,
		},
		{
			name:           "should allow updates with automatic node update rules from concrete version",
			versions:       []string{"v1.11.1", "v1.12.2"},
			defaultVersion: "v1.11.1",
			updates: []kubermaticv1.Update{{
				From:                "v1.11.1",
				To:                  "v1.12.2",
				AutomaticNodeUpdate: ptr.To(true),
			}},
			valid: true,
		},
		{
			name:           "should allow updates with automatic update rules from wildcard version",
			versions:       []string{"v1.11.1", "v1.12.2"},
			defaultVersion: "v1.11.1",
			updates: []kubermaticv1.Update{{
				From:      "v1.11.*",
				To:        "v1.12.2",
				Automatic: ptr.To(true),
			}},
			valid: true,
		},
		{
			name:           "should allow updates with automatic node update rules from wildcard version",
			versions:       []string{"v1.11.1", "v1.12.2"},
			defaultVersion: "v1.11.1",
			updates: []kubermaticv1.Update{{
				From:                "v1.11.*",
				To:                  "v1.12.2",
				AutomaticNodeUpdate: ptr.To(true),
			}},
			valid: true,
		},
		{
			name:           "should forbid updates with automatic update rules to wildcard version",
			versions:       []string{"v1.11.1", "v1.12.2"},
			defaultVersion: "v1.11.1",
			updates: []kubermaticv1.Update{{
				From:      "v1.11.0",
				To:        "v1.12.*",
				Automatic: ptr.To(true),
			}},
			valid: false,
		},
		{
			name:           "should forbid updates with automatic node update rules to wildcard version",
			versions:       []string{"v1.11.1", "v1.12.2"},
			defaultVersion: "v1.11.1",
			updates: []kubermaticv1.Update{{
				From:                "v1.11.0",
				To:                  "v1.12.*",
				AutomaticNodeUpdate: ptr.To(true),
			}},
			valid: false,
		},
		{
			name:           "should forbid updates with automatic update rules to version with concrete automatic update rule",
			versions:       []string{"v1.11.1", "v1.12.2", "v1.13.3"},
			defaultVersion: "v1.11.1",
			updates: []kubermaticv1.Update{
				{
					From:      "v1.11.1",
					To:        "v1.12.2",
					Automatic: ptr.To(true),
				},
				{
					From:      "v1.12.2",
					To:        "v1.13.3",
					Automatic: ptr.To(true),
				},
			},
			valid: false,
		},
		{
			name:           "should forbid updates with automatic node update rules to version with concrete automatic update rule",
			versions:       []string{"v1.11.1", "v1.12.2", "v1.13.3"},
			defaultVersion: "v1.11.1",
			updates: []kubermaticv1.Update{
				{
					From:                "v1.11.1",
					To:                  "v1.12.2",
					AutomaticNodeUpdate: ptr.To(true),
				},
				{
					From:                "v1.12.2",
					To:                  "v1.13.3",
					AutomaticNodeUpdate: ptr.To(true),
				},
			},
			valid: false,
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

			config.Updates = tt.updates

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

func TestValidateMirrorImages(t *testing.T) {
	testcases := []struct {
		name         string
		mirrorImages []string
		valid        bool
	}{
		{
			name:         "valid single image",
			mirrorImages: []string{"nginx:1.21.6"},
			valid:        true,
		},
		{
			name:         "valid multiple images",
			mirrorImages: []string{"nginx:1.21.6", "quay.io/kubermatic/kubelb-manager-ee:v1.1.0"},
			valid:        true,
		},
		{
			name:         "invalid image format (missing tag)",
			mirrorImages: []string{"nginx"},
			valid:        false,
		},
		{
			name:         "invalid image format (missing repository)",
			mirrorImages: []string{":latest"},
			valid:        false,
		},
		{
			name:         "invalid image format (empty string)",
			mirrorImages: []string{""},
			valid:        false,
		},
		{
			name:         "invalid image format (extra colon)",
			mirrorImages: []string{"nginx:1.21.6:extra"},
			valid:        false,
		},
		{
			name:         "mixed valid and invalid images",
			mirrorImages: []string{"nginx:1.21.6", "invalid-image"},
			valid:        false,
		},
		{
			name:         "empty mirrorImages list",
			mirrorImages: []string{},
			valid:        true,
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			spec := &kubermaticv1.KubermaticConfigurationSpec{
				MirrorImages: tt.mirrorImages,
			}
			version := semver.NewSemverOrDie("v1.11.1")
			spec.Versions.Default = version
			spec.Versions.Versions = append(spec.Versions.Versions, *version)
			errs := ValidateKubermaticConfigurationSpec(spec)
			if tt.valid {
				if len(errs) > 0 {
					t.Fatalf("Expected configuration to be valid, but got errors: %v", errs.ToAggregate())
				}
			} else {
				if len(errs) == 0 {
					t.Fatal("Expected configuration to be invalid, but it was accepted.")
				}
			}
		})
	}
}

func TestValidateApplicationDefinitionsConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		config  kubermaticv1.ApplicationDefinitionsConfiguration
		wantErr bool
	}{
		{
			name:   "valid empty configuration",
			config: kubermaticv1.ApplicationDefinitionsConfiguration{},
		},
		{
			name: "valid default application catalog with helm credentials",
			config: kubermaticv1.ApplicationDefinitionsConfiguration{
				DefaultApplicationCatalog: kubermaticv1.DefaultApplicationCatalogSettings{
					Enable:         true,
					HelmRepository: "oci://localhost:5000/myrepo",
					HelmCredentials: &appskubermaticv1.HelmCredentials{
						Username: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
							Key:                  "username",
						},
						Password: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
							Key:                  "password",
						},
					},
				},
			},
		},
		{
			name: "valid default application catalog with registry config file",
			config: kubermaticv1.ApplicationDefinitionsConfiguration{
				DefaultApplicationCatalog: kubermaticv1.DefaultApplicationCatalogSettings{
					Enable:         true,
					HelmRepository: "oci://localhost:5000/myrepo",
					HelmRegistryConfigFile: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
						Key:                  "config.json",
					},
				},
			},
		},
		{
			name: "invalid: both helm credentials and registry config file",
			config: kubermaticv1.ApplicationDefinitionsConfiguration{
				DefaultApplicationCatalog: kubermaticv1.DefaultApplicationCatalogSettings{
					Enable:         true,
					HelmRepository: "oci://localhost:5000/myrepo",
					HelmCredentials: &appskubermaticv1.HelmCredentials{
						Username: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
							Key:                  "username",
						},
						Password: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
							Key:                  "password",
						},
					},
					HelmRegistryConfigFile: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
						Key:                  "config.json",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid: helm credentials with username but no password",
			config: kubermaticv1.ApplicationDefinitionsConfiguration{
				DefaultApplicationCatalog: kubermaticv1.DefaultApplicationCatalogSettings{
					Enable:         true,
					HelmRepository: "oci://localhost:5000/myrepo",
					HelmCredentials: &appskubermaticv1.HelmCredentials{
						Username: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
							Key:                  "username",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid: helm credentials with password but no username",
			config: kubermaticv1.ApplicationDefinitionsConfiguration{
				DefaultApplicationCatalog: kubermaticv1.DefaultApplicationCatalogSettings{
					Enable:         true,
					HelmRepository: "oci://localhost:5000/myrepo",
					HelmCredentials: &appskubermaticv1.HelmCredentials{
						Password: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
							Key:                  "password",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid: helm credentials with neither username/password nor registry config",
			config: kubermaticv1.ApplicationDefinitionsConfiguration{
				DefaultApplicationCatalog: kubermaticv1.DefaultApplicationCatalogSettings{
					Enable:          true,
					HelmRepository:  "oci://localhost:5000/myrepo",
					HelmCredentials: &appskubermaticv1.HelmCredentials{},
				},
			},
			wantErr: true,
		},
		{
			name: "valid default application catalog with insecure and plainHTTP settings",
			config: kubermaticv1.ApplicationDefinitionsConfiguration{
				DefaultApplicationCatalog: kubermaticv1.DefaultApplicationCatalogSettings{
					Enable:         true,
					HelmRepository: "oci://localhost:5000/myrepo",
					Insecure:       &[]bool{true}[0],
					PlainHTTP:      &[]bool{true}[0],
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateApplicationDefinitionsConfiguration(tt.config, field.NewPath("spec", "applications"))
			if tt.wantErr && len(errs) == 0 {
				t.Errorf("ValidateApplicationDefinitionsConfiguration() expected errors but got none")
			}
			if !tt.wantErr && len(errs) > 0 {
				t.Errorf("ValidateApplicationDefinitionsConfiguration() unexpected errors: %v", errs)
			}
		})
	}
}
