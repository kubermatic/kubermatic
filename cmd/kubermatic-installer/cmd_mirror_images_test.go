/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/sets"
)

func TestParseProviderFilter(t *testing.T) {
	testCases := []struct {
		name          string
		input         []string
		expectedCount int
		expectError   bool
	}{
		{
			name:          "empty filter returns nil",
			input:         []string{},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "nil filter returns nil",
			input:         nil,
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "single valid provider",
			input:         []string{"aws"},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:          "multiple valid providers",
			input:         []string{"aws", "azure", "kubevirt"},
			expectedCount: 3,
			expectError:   false,
		},
		{
			name:          "mixed case providers",
			input:         []string{"AWS", "Azure", "KubeVirt"},
			expectedCount: 3,
			expectError:   false,
		},
		{
			name:          "providers with spaces",
			input:         []string{" aws ", " azure ", " kubevirt "},
			expectedCount: 3,
			expectError:   false,
		},
		{
			name:          "invalid provider",
			input:         []string{"aws", "invalid-provider", "azure"},
			expectedCount: 0,
			expectError:   true,
		},
		{
			name:          "duplicate providers",
			input:         []string{"aws", "azure", "aws"},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "empty strings in slice",
			input:         []string{"aws", "", "azure"},
			expectedCount: 2,
			expectError:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseProviderFilter(tc.input)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected an error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tc.expectedCount == 0 && result != nil {
				t.Errorf("expected nil result for empty filter, got %v", result)
				return
			}

			if tc.expectedCount > 0 {
				if result == nil {
					t.Errorf("expected non-nil result, got nil")
					return
				}

				if result.Len() != tc.expectedCount {
					t.Errorf("expected %d providers, got %d: %v", tc.expectedCount, result.Len(), sets.List(result))
				}
			}
		})
	}
}

// TestClearRepositoryOverrides covers the helper behind mirror-images
// --ignore-repository-overrides. Every DockerRepository override and the three
// DockerTagSuffix fields (API, UI, Addons) must be blanked so DefaultConfiguration
// re-fills them with upstream defaults; the development-only DockerTag override must
// survive. Leaving the addons suffix in place was the cause of issue #15868, where the
// upstream addons repository was paired with a customer-only tag (e.g. v2.30.0-17) and
// failed with MANIFEST_UNKNOWN.
func TestClearRepositoryOverrides(t *testing.T) {
	testCases := []struct {
		name   string
		config *kubermaticv1.KubermaticConfiguration
		// assertCleared lists fields that must be empty after the call.
		assertCleared func(t *testing.T, c *kubermaticv1.KubermaticConfiguration)
		// assertPreserved checks fields that must keep their value after the call.
		assertPreserved func(t *testing.T, c *kubermaticv1.KubermaticConfiguration)
	}{
		{
			name: "every repository override and every tag suffix is cleared",
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					API: kubermaticv1.KubermaticAPIConfiguration{
						DockerRepository: "myreg.io/kubermatic",
						DockerTagSuffix:  "17",
					},
					UI: kubermaticv1.KubermaticUIConfiguration{
						DockerRepository: "myreg.io/dashboard",
						DockerTagSuffix:  "17",
					},
					MasterController: kubermaticv1.KubermaticMasterControllerConfiguration{
						DockerRepository: "myreg.io/master-controller",
					},
					SeedController: kubermaticv1.KubermaticSeedControllerConfiguration{
						DockerRepository: "myreg.io/seed-controller",
					},
					Webhook: kubermaticv1.KubermaticWebhookConfiguration{
						DockerRepository: "myreg.io/webhook",
					},
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						KubermaticDockerRepository:     "myreg.io/ucc",
						DNATControllerDockerRepository: "myreg.io/dnat",
						EtcdLauncherDockerRepository:   "myreg.io/etcd-launcher",
						Addons: kubermaticv1.KubermaticAddonsConfiguration{
							DockerRepository: "myreg.io/addons",
							DockerTagSuffix:  "17",
						},
					},
					VerticalPodAutoscaler: kubermaticv1.KubermaticVPAConfiguration{
						Recommender:         kubermaticv1.KubermaticVPAComponent{DockerRepository: "myreg.io/vpa-recommender"},
						Updater:             kubermaticv1.KubermaticVPAComponent{DockerRepository: "myreg.io/vpa-updater"},
						AdmissionController: kubermaticv1.KubermaticVPAComponent{DockerRepository: "myreg.io/vpa-admission"},
					},
				},
			},
			assertCleared: func(t *testing.T, c *kubermaticv1.KubermaticConfiguration) {
				assert.Empty(t, c.Spec.API.DockerRepository)
				assert.Empty(t, c.Spec.UI.DockerRepository)
				assert.Empty(t, c.Spec.MasterController.DockerRepository)
				assert.Empty(t, c.Spec.SeedController.DockerRepository)
				assert.Empty(t, c.Spec.Webhook.DockerRepository)
				assert.Empty(t, c.Spec.UserCluster.KubermaticDockerRepository)
				assert.Empty(t, c.Spec.UserCluster.DNATControllerDockerRepository)
				assert.Empty(t, c.Spec.UserCluster.EtcdLauncherDockerRepository)
				assert.Empty(t, c.Spec.UserCluster.Addons.DockerRepository)
				assert.Empty(t, c.Spec.VerticalPodAutoscaler.Recommender.DockerRepository)
				assert.Empty(t, c.Spec.VerticalPodAutoscaler.Updater.DockerRepository)
				assert.Empty(t, c.Spec.VerticalPodAutoscaler.AdmissionController.DockerRepository)

				assert.Empty(t, c.Spec.API.DockerTagSuffix)
				assert.Empty(t, c.Spec.UI.DockerTagSuffix)
				assert.Empty(t, c.Spec.UserCluster.Addons.DockerTagSuffix)
			},
		},
		{
			// the exact #15868 scenario: only the addons suffix is set on top of a
			// repository override; both must be cleared so the upstream addons image
			// resolves with its plain version tag instead of v<version>-17.
			name: "issue 15868: addons repository and tag suffix are cleared together",
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						Addons: kubermaticv1.KubermaticAddonsConfiguration{
							DockerRepository: "myreg.io/addons",
							DockerTagSuffix:  "17",
						},
					},
				},
			},
			assertCleared: func(t *testing.T, c *kubermaticv1.KubermaticConfiguration) {
				assert.Empty(t, c.Spec.UserCluster.Addons.DockerRepository)
				assert.Empty(t, c.Spec.UserCluster.Addons.DockerTagSuffix)
			},
		},
		{
			// DockerTag is a development-only full-tag override and is outside the
			// repository-override contract, so it must survive on both API and UI.
			name: "development-only DockerTag is preserved on API and UI",
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					API: kubermaticv1.KubermaticAPIConfiguration{
						DockerRepository: "myreg.io/kubermatic",
						DockerTag:        "dev-api",
					},
					UI: kubermaticv1.KubermaticUIConfiguration{
						DockerRepository: "myreg.io/dashboard",
						DockerTag:        "dev-ui",
					},
				},
			},
			assertCleared: func(t *testing.T, c *kubermaticv1.KubermaticConfiguration) {
				assert.Empty(t, c.Spec.API.DockerRepository)
				assert.Empty(t, c.Spec.UI.DockerRepository)
			},
			assertPreserved: func(t *testing.T, c *kubermaticv1.KubermaticConfiguration) {
				assert.Equal(t, "dev-api", c.Spec.API.DockerTag)
				assert.Equal(t, "dev-ui", c.Spec.UI.DockerTag)
			},
		},
		{
			name:   "empty config stays empty",
			config: &kubermaticv1.KubermaticConfiguration{},
			assertCleared: func(t *testing.T, c *kubermaticv1.KubermaticConfiguration) {
				assert.Equal(t, &kubermaticv1.KubermaticConfiguration{}, c)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clearRepositoryOverrides(tc.config)

			if tc.assertCleared != nil {
				tc.assertCleared(t, tc.config)
			}
			if tc.assertPreserved != nil {
				tc.assertPreserved(t, tc.config)
			}
		})
	}
}
