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

	"github.com/Masterminds/semver/v3"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
)

func TestAutomaticUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		manager         *Manager
		clusterType     string
		versionFrom     string
		expectedVersion string
		expectedError   string
	}{
		{
			name:            "test best automatic update for kubernetes cluster",
			versionFrom:     "1.10.0",
			expectedVersion: "1.10.1",
			clusterType:     apiv1.KubernetesClusterType,
			manager: New([]*Version{
				{
					Version: semver.MustParse("1.10.0"),
					Default: false,
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.10.1"),
					Default: true,
					Type:    apiv1.KubernetesClusterType,
				},
			}, []*Update{
				{
					From:      "1.10.0",
					To:        "1.10.1",
					Automatic: true,
					Type:      apiv1.KubernetesClusterType,
				},
			}, nil),
		},
		{
			name:            "test Kubernetes best automatic update with wild card for kubernetes cluster",
			versionFrom:     "1.10.0",
			expectedVersion: "1.10.1",
			clusterType:     apiv1.KubernetesClusterType,
			manager: New([]*Version{
				{
					Version: semver.MustParse("1.10.0"),
					Default: false,
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.10.1"),
					Default: true,
					Type:    apiv1.KubernetesClusterType,
				},
			}, []*Update{
				{
					From:      "1.10.*",
					To:        "1.10.1",
					Automatic: true,
					Type:      apiv1.KubernetesClusterType,
				},
			}, nil),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			updateVersion, err := tc.manager.AutomaticControlplaneUpdate(tc.versionFrom, tc.clusterType)

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
		clusterType      string
		conditions       []ConditionType
		provider         kubermaticv1.ProviderType
		expectedVersions []*Version
	}{
		{
			name:        "No incompatibility for given provider",
			provider:    kubermaticv1.ProviderAWS,
			clusterType: apiv1.KubernetesClusterType,
			manager: New([]*Version{
				{
					Version: semver.MustParse("1.21.0"),
					Default: true,
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.22.0"),
					Default: false,
					Type:    apiv1.KubernetesClusterType,
				},
			}, nil,
				[]*ProviderIncompatibility{
					{
						Provider:  kubermaticv1.ProviderVSphere,
						Version:   "1.22.*",
						Operation: CreateOperation,
						Type:      apiv1.KubernetesClusterType,
					},
				}),
			expectedVersions: []*Version{
				{
					Version: semver.MustParse("1.21.0"),
				},
				{
					Version: semver.MustParse("1.22.0"),
				},
			},
		},
		{
			name:        "Always Incompatibility for given provider",
			provider:    kubermaticv1.ProviderVSphere,
			clusterType: apiv1.KubernetesClusterType,
			manager: New([]*Version{
				{
					Version: semver.MustParse("1.21.0"),
					Default: true,
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.22.0"),
					Default: false,
					Type:    apiv1.KubernetesClusterType,
				},
			}, nil,
				[]*ProviderIncompatibility{
					{
						Provider:  kubermaticv1.ProviderVSphere,
						Version:   "1.22.*",
						Operation: CreateOperation,
						Condition: AlwaysCondition,
						Type:      apiv1.KubernetesClusterType,
					},
				}),
			expectedVersions: []*Version{
				{
					Version: semver.MustParse("1.21.0"),
				},
			},
		},
		{
			name:        "Matching Incompatibility for given provider",
			provider:    kubermaticv1.ProviderVSphere,
			clusterType: apiv1.KubernetesClusterType,
			conditions:  []ConditionType{ExternalCloudProviderCondition},
			manager: New([]*Version{
				{
					Version: semver.MustParse("1.21.0"),
					Default: true,
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.22.0"),
					Default: false,
					Type:    apiv1.KubernetesClusterType,
				},
			}, nil,
				[]*ProviderIncompatibility{
					{
						Provider:  kubermaticv1.ProviderVSphere,
						Version:   "1.22.*",
						Operation: CreateOperation,
						Condition: ExternalCloudProviderCondition,
						Type:      apiv1.KubernetesClusterType,
					},
				}),
			expectedVersions: []*Version{
				{
					Version: semver.MustParse("1.21.0"),
				},
			},
		},
		{
			name:        "Multiple Incompatibilities for different providers",
			provider:    kubermaticv1.ProviderVSphere,
			clusterType: apiv1.KubernetesClusterType,
			manager: New([]*Version{
				{
					Version: semver.MustParse("1.21.0"),
					Default: true,
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.22.0"),
					Default: false,
					Type:    apiv1.KubernetesClusterType,
				},
			}, nil,
				[]*ProviderIncompatibility{
					{
						Provider:  kubermaticv1.ProviderVSphere,
						Version:   "1.21.*",
						Operation: CreateOperation,
						Condition: AlwaysCondition,
						Type:      apiv1.KubernetesClusterType,
					},
					{
						Provider:  kubermaticv1.ProviderAWS,
						Version:   "1.22.*",
						Operation: CreateOperation,
						Condition: AlwaysCondition,
						Type:      apiv1.KubernetesClusterType,
					},
				}),
			expectedVersions: []*Version{
				{
					Version: semver.MustParse("1.22.0"),
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			availableVersions, err := tc.manager.GetVersionsV2(tc.clusterType, tc.provider, tc.conditions...)
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
		clusterType      string
		provider         kubermaticv1.ProviderType
		fromVersion      string
		conditions       []ConditionType
		expectedVersions []*Version
	}{
		{
			name:        "Check with no incompatibility for given provider",
			provider:    kubermaticv1.ProviderAWS,
			clusterType: apiv1.KubernetesClusterType,
			fromVersion: "1.21.0",
			manager: New([]*Version{
				{
					Version: semver.MustParse("1.21.0"),
					Default: true,
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.22.0"),
					Default: false,
					Type:    apiv1.KubernetesClusterType,
				},
			}, []*Update{
				{
					From:      "1.21.*",
					To:        "1.22.*",
					Automatic: true,
					Type:      apiv1.KubernetesClusterType,
				},
			}, []*ProviderIncompatibility{
				{
					Provider:  kubermaticv1.ProviderVSphere,
					Version:   "1.22.*",
					Type:      apiv1.KubernetesClusterType,
					Condition: AlwaysCondition,
					Operation: CreateOperation,
				},
			}),
			expectedVersions: []*Version{
				{
					Version: semver.MustParse("1.22.0"),
				},
			},
		},
		{
			name:        "Check with conditioned Incompatibility for given provider",
			provider:    kubermaticv1.ProviderVSphere,
			clusterType: apiv1.KubernetesClusterType,
			fromVersion: "1.21.0",
			conditions:  []ConditionType{ExternalCloudProviderCondition},
			manager: New([]*Version{
				{
					Version: semver.MustParse("1.21.0"),
					Default: true,
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.22.0"),
					Default: false,
					Type:    apiv1.KubernetesClusterType,
				},
			}, []*Update{
				{
					From:      "1.21.*",
					To:        "1.22.*",
					Automatic: true,
					Type:      apiv1.KubernetesClusterType,
				},
			}, []*ProviderIncompatibility{
				{
					Provider:  kubermaticv1.ProviderVSphere,
					Version:   "1.22.*",
					Type:      apiv1.KubernetesClusterType,
					Operation: UpdateOperation,
					Condition: ExternalCloudProviderCondition,
				},
			}),
			expectedVersions: []*Version{},
		},
		{
			name:        "Check with unconditioned Incompatibility for given provider",
			provider:    kubermaticv1.ProviderVSphere,
			clusterType: apiv1.KubernetesClusterType,
			fromVersion: "1.21.0",
			manager: New([]*Version{
				{
					Version: semver.MustParse("1.21.0"),
					Default: true,
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.22.0"),
					Default: false,
					Type:    apiv1.KubernetesClusterType,
				},
			}, []*Update{
				{
					From:      "1.21.*",
					To:        "1.22.*",
					Automatic: true,
					Type:      apiv1.KubernetesClusterType,
				},
			}, []*ProviderIncompatibility{
				{
					Provider:  kubermaticv1.ProviderVSphere,
					Version:   "1.22.*",
					Type:      apiv1.KubernetesClusterType,
					Condition: ExternalCloudProviderCondition,
				},
			}),
			expectedVersions: []*Version{
				{
					Version: semver.MustParse("1.22.0"),
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			availableVersions, err := tc.manager.GetPossibleUpdates(tc.fromVersion, tc.clusterType, tc.provider, tc.conditions...)
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
