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
			}),
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
			}),
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
