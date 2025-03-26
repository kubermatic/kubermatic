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

package version

import (
	"testing"

	semverlib "github.com/Masterminds/semver/v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

func TestAutomaticUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		manager         *Manager
		versionFrom     string
		expectedVersion string
		expectedError   string
	}{
		{
			name:            "test best automatic update for kubernetes cluster",
			versionFrom:     "1.10.0",
			expectedVersion: "1.10.1",
			manager: New([]*Version{
				{
					Version: semverlib.MustParse("1.10.0"),
					Default: false,
				},
				{
					Version: semverlib.MustParse("1.10.1"),
					Default: true,
				},
			}, []*Update{
				{
					From:      "1.10.0",
					To:        "1.10.1",
					Automatic: true,
				},
			}, nil),
		},
		{
			name:            "test Kubernetes best automatic update with wild card for kubernetes cluster",
			versionFrom:     "1.10.0",
			expectedVersion: "1.10.1",
			manager: New([]*Version{
				{
					Version: semverlib.MustParse("1.10.0"),
					Default: false,
				},
				{
					Version: semverlib.MustParse("1.10.1"),
					Default: true,
				},
			}, []*Update{
				{
					From:      "1.10.*",
					To:        "1.10.1",
					Automatic: true,
				},
			}, nil),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			updateVersion, err := tc.manager.AutomaticControlplaneUpdate(tc.versionFrom)

			if len(tc.expectedError) > 0 {
				if err == nil {
					t.Fatalf("Expected error")
				}
				if tc.expectedError != err.Error() {
					t.Fatalf("Expected error: %s got %v", tc.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if updateVersion.Version.String() != tc.expectedVersion {
					t.Fatalf("Unexpected update version to be %s. Got=%v", tc.expectedVersion, updateVersion)
				}
			}
		})
	}
}

func TestProviderIncompatibilitiesVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		manager          *Manager
		conditions       []kubermaticv1.ConditionType
		provider         kubermaticv1.ProviderType
		expectedVersions []*Version
	}{
		{
			name:     "No incompatibility for given provider",
			provider: kubermaticv1.AWSCloudProvider,
			manager: New([]*Version{
				{
					Version: semverlib.MustParse("1.21.0"),
					Default: true,
				},
				{
					Version: semverlib.MustParse("1.22.0"),
					Default: false,
				},
			}, nil,
				[]*ProviderIncompatibility{
					{
						Provider:  kubermaticv1.VSphereCloudProvider,
						Version:   "1.22.*",
						Operation: kubermaticv1.CreateOperation,
					},
				}),
			expectedVersions: []*Version{
				{
					Version: semverlib.MustParse("1.21.0"),
				},
				{
					Version: semverlib.MustParse("1.22.0"),
				},
			},
		},
		{
			name:     "Always Incompatibility for given provider",
			provider: kubermaticv1.VSphereCloudProvider,
			manager: New([]*Version{
				{
					Version: semverlib.MustParse("1.21.0"),
					Default: true,
				},
				{
					Version: semverlib.MustParse("1.22.0"),
					Default: false,
				},
			}, nil,
				[]*ProviderIncompatibility{
					{
						Provider:  kubermaticv1.VSphereCloudProvider,
						Version:   "1.22.*",
						Operation: kubermaticv1.CreateOperation,
						Condition: kubermaticv1.AlwaysCondition,
					},
				}),
			expectedVersions: []*Version{
				{
					Version: semverlib.MustParse("1.21.0"),
				},
			},
		},
		{
			name:       "Matching Incompatibility for given provider",
			provider:   kubermaticv1.VSphereCloudProvider,
			conditions: []kubermaticv1.ConditionType{kubermaticv1.ExternalCloudProviderCondition},
			manager: New([]*Version{
				{
					Version: semverlib.MustParse("1.21.0"),
					Default: true,
				},
				{
					Version: semverlib.MustParse("1.22.0"),
					Default: false,
				},
			}, nil,
				[]*ProviderIncompatibility{
					{
						Provider:  kubermaticv1.VSphereCloudProvider,
						Version:   "1.22.*",
						Operation: kubermaticv1.CreateOperation,
						Condition: kubermaticv1.ExternalCloudProviderCondition,
					},
				}),
			expectedVersions: []*Version{
				{
					Version: semverlib.MustParse("1.21.0"),
				},
			},
		},
		{
			name:     "Multiple Incompatibilities for different providers",
			provider: kubermaticv1.VSphereCloudProvider,
			manager: New([]*Version{
				{
					Version: semverlib.MustParse("1.21.0"),
					Default: true,
				},
				{
					Version: semverlib.MustParse("1.22.0"),
					Default: false,
				},
			}, nil,
				[]*ProviderIncompatibility{
					{
						Provider:  kubermaticv1.VSphereCloudProvider,
						Version:   "1.21.*",
						Operation: kubermaticv1.CreateOperation,
						Condition: kubermaticv1.AlwaysCondition,
					},
					{
						Provider:  kubermaticv1.AWSCloudProvider,
						Version:   "1.22.*",
						Operation: kubermaticv1.CreateOperation,
						Condition: kubermaticv1.AlwaysCondition,
					},
				}),
			expectedVersions: []*Version{
				{
					Version: semverlib.MustParse("1.22.0"),
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			availableVersions, err := tc.manager.GetVersionsForProvider(tc.provider, tc.conditions...)
			if err != nil {
				t.Fatalf("unexpected error %s", err)
			}
			for _, ev := range tc.expectedVersions {
				var found bool
				for _, av := range availableVersions {
					if av.Version.String() == ev.Version.String() {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected version %s not found in availableVersions", ev.Version)
				}
			}

			for _, av := range availableVersions {
				var found bool
				for _, ev := range tc.expectedVersions {
					if av.Version.String() == ev.Version.String() {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("unexpected version %s is available", av.Version)
				}
			}
		})
	}
}

func TestProviderIncompatibilitiesUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		manager          *Manager
		provider         kubermaticv1.ProviderType
		fromVersion      string
		conditions       []kubermaticv1.ConditionType
		expectedVersions []*Version
	}{
		{
			name:        "Check with no incompatibility for given provider",
			provider:    kubermaticv1.AWSCloudProvider,
			fromVersion: "1.21.0",
			manager: New([]*Version{
				{
					Version: semverlib.MustParse("1.21.0"),
					Default: true,
				},
				{
					Version: semverlib.MustParse("1.22.0"),
					Default: false,
				},
			}, []*Update{
				{
					From:      "1.21.*",
					To:        "1.22.*",
					Automatic: true,
				},
			}, []*ProviderIncompatibility{
				{
					Provider:  kubermaticv1.VSphereCloudProvider,
					Version:   "1.22.*",
					Condition: kubermaticv1.AlwaysCondition,
					Operation: kubermaticv1.CreateOperation,
				},
			}),
			expectedVersions: []*Version{
				{
					Version: semverlib.MustParse("1.22.0"),
				},
			},
		},
		{
			name:        "Check with conditioned Incompatibility for given provider",
			provider:    kubermaticv1.VSphereCloudProvider,
			fromVersion: "1.21.0",
			conditions:  []kubermaticv1.ConditionType{kubermaticv1.ExternalCloudProviderCondition},
			manager: New([]*Version{
				{
					Version: semverlib.MustParse("1.21.0"),
					Default: true,
				},
				{
					Version: semverlib.MustParse("1.22.0"),
					Default: false,
				},
			}, []*Update{
				{
					From:      "1.21.*",
					To:        "1.22.*",
					Automatic: true,
				},
			}, []*ProviderIncompatibility{
				{
					Provider:  kubermaticv1.VSphereCloudProvider,
					Version:   "1.22.*",
					Operation: kubermaticv1.UpdateOperation,
					Condition: kubermaticv1.ExternalCloudProviderCondition,
				},
			}),
			expectedVersions: []*Version{},
		},
		{
			name:        "Check with unconditioned Incompatibility for given provider",
			provider:    kubermaticv1.VSphereCloudProvider,
			fromVersion: "1.21.0",
			manager: New([]*Version{
				{
					Version: semverlib.MustParse("1.21.0"),
					Default: true,
				},
				{
					Version: semverlib.MustParse("1.22.0"),
					Default: false,
				},
			}, []*Update{
				{
					From:      "1.21.*",
					To:        "1.22.*",
					Automatic: true,
				},
			}, []*ProviderIncompatibility{
				{
					Provider:  kubermaticv1.VSphereCloudProvider,
					Version:   "1.22.*",
					Condition: kubermaticv1.ExternalCloudProviderCondition,
				},
			}),
			expectedVersions: []*Version{
				{
					Version: semverlib.MustParse("1.22.0"),
				},
			},
		},
		{
			name:        "Check with InTreeCloudProvider Incompatibility for OpenStack 1.26",
			provider:    kubermaticv1.OpenstackCloudProvider,
			fromVersion: "1.25.0",
			conditions:  []kubermaticv1.ConditionType{kubermaticv1.InTreeCloudProviderCondition},
			manager: New([]*Version{
				{
					Version: semverlib.MustParse("1.25.0"),
					Default: true,
				},
				{
					Version: semverlib.MustParse("1.26.0"),
					Default: false,
				},
			}, []*Update{
				{
					From:      "1.25.*",
					To:        "1.26.*",
					Automatic: true,
				},
			}, []*ProviderIncompatibility{
				{
					Provider:  kubermaticv1.OpenstackCloudProvider,
					Version:   "1.26.0",
					Operation: kubermaticv1.UpdateOperation,
					Condition: kubermaticv1.InTreeCloudProviderCondition,
				},
			}),
			expectedVersions: []*Version{},
		},
		{
			name:        "Check with InTreeCloudProvider Incompatibility for OpenStack 1.26 with external CCM",
			provider:    kubermaticv1.OpenstackCloudProvider,
			fromVersion: "1.25.0",
			conditions:  []kubermaticv1.ConditionType{kubermaticv1.ExternalCloudProviderCondition},
			manager: New([]*Version{
				{
					Version: semverlib.MustParse("1.25.0"),
					Default: true,
				},
				{
					Version: semverlib.MustParse("1.26.0"),
					Default: false,
				},
			}, []*Update{
				{
					From:      "1.25.*",
					To:        "1.26.*",
					Automatic: true,
				},
			}, []*ProviderIncompatibility{
				{
					Provider:  kubermaticv1.OpenstackCloudProvider,
					Version:   "1.26.0",
					Operation: kubermaticv1.UpdateOperation,
					Condition: kubermaticv1.InTreeCloudProviderCondition,
				},
			}),
			expectedVersions: []*Version{
				{
					Version: semverlib.MustParse("1.26.0"),
				},
			},
		},
		{
			name:        "Check with InTreeCloudProvider Incompatibility for vSphere 1.25 with external CCM",
			provider:    kubermaticv1.VSphereCloudProvider,
			fromVersion: "1.24.0",
			conditions:  []kubermaticv1.ConditionType{kubermaticv1.ExternalCloudProviderCondition},
			manager: New([]*Version{
				{
					Version: semverlib.MustParse("1.24.0"),
					Default: true,
				},
				{
					Version: semverlib.MustParse("1.25.0"),
					Default: false,
				},
			}, []*Update{
				{
					From:      "1.24.*",
					To:        "1.25.*",
					Automatic: true,
				},
			}, []*ProviderIncompatibility{
				{
					Provider:  kubermaticv1.VSphereCloudProvider,
					Version:   "1.25.0",
					Operation: kubermaticv1.UpdateOperation,
					Condition: kubermaticv1.InTreeCloudProviderCondition,
				},
			}),
			expectedVersions: []*Version{
				{
					Version: semverlib.MustParse("1.25.0"),
				},
			},
		},
		{
			name:        "Check with InTreeCloudProvider Incompatibility for vSphere 1.25",
			provider:    kubermaticv1.VSphereCloudProvider,
			fromVersion: "1.24.0",
			conditions:  []kubermaticv1.ConditionType{kubermaticv1.InTreeCloudProviderCondition},
			manager: New([]*Version{
				{
					Version: semverlib.MustParse("1.24.0"),
					Default: true,
				},
				{
					Version: semverlib.MustParse("1.25.0"),
					Default: false,
				},
			}, []*Update{
				{
					From:      "1.24.*",
					To:        "1.25.*",
					Automatic: true,
				},
			}, []*ProviderIncompatibility{
				{
					Provider:  kubermaticv1.VSphereCloudProvider,
					Version:   "1.25.0",
					Operation: kubermaticv1.UpdateOperation,
					Condition: kubermaticv1.InTreeCloudProviderCondition,
				},
			}),
			expectedVersions: []*Version{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			availableVersions, err := tc.manager.GetPossibleUpdates(tc.fromVersion, tc.provider, tc.conditions...)
			if err != nil {
				t.Fatalf("unexpected error %s", err)
			}
			for _, ev := range tc.expectedVersions {
				var found bool
				for _, av := range availableVersions {
					if av.Version.String() == ev.Version.String() {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected version %s not found in availableVersions", ev.Version)
				}
			}

			for _, av := range availableVersions {
				var found bool
				for _, ev := range tc.expectedVersions {
					if av.Version.String() == ev.Version.String() {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("unexpected version %s is available", av.Version)
				}
			}
		})
	}
}
