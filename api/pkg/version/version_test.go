package version

import (
	"testing"

	"github.com/Masterminds/semver"
	"github.com/go-test/deep"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
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
			manager: New([]*MasterVersion{
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
			}, []*MasterUpdate{
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
			manager: New([]*MasterVersion{
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
			}, []*MasterUpdate{
				{
					From:      "1.10.*",
					To:        "1.10.1",
					Automatic: true,
					Type:      apiv1.KubernetesClusterType,
				},
			}),
		},
		{
			name:          "test required update for kubernetes cluster type doesn't exist",
			versionFrom:   "1.10.0",
			expectedError: "failed to get MasterVersion for 1.10.1: version not found",
			clusterType:   apiv1.KubernetesClusterType,
			manager: New([]*MasterVersion{
				{
					Version: semver.MustParse("1.10.0"),
					Default: false,
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.10.1"),
					Default: true,
					Type:    apiv1.OpenShiftClusterType,
				},
			}, []*MasterUpdate{
				{
					From:      "1.10.0",
					To:        "1.10.1",
					Automatic: true,
					Type:      apiv1.KubernetesClusterType,
				},
			}),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			updateVersion, err := tc.manager.AutomaticUpdate(tc.versionFrom, tc.clusterType)

			if len(tc.expectedError) > 0 {
				if err == nil {
					t.Fatalf("Epected error")
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

func TestGetMasterVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		manager          *Manager
		expectedVersions []*MasterVersion
	}{
		{
			name: "test OpenShift versions without automatic updates",
			manager: New([]*MasterVersion{
				{
					Version: semver.MustParse("1.13.5"),
					Default: true,
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("3.11.5"),
					Default: true,
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("3.11"),
					Default: false,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("4.1"),
					Default: true,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("4.2"),
					Default: true,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("4.3"),
					Default: true,
					Type:    apiv1.OpenShiftClusterType,
				},
			}, []*MasterUpdate{},
			),
			expectedVersions: []*MasterVersion{
				{
					Version: semver.MustParse("3.11"),
					Default: false,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("4.1"),
					Default: true,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("4.2"),
					Default: true,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("4.3"),
					Default: true,
					Type:    apiv1.OpenShiftClusterType,
				},
			},
		},
		{
			name: "test OpenShift versions with automatic updates",
			manager: New([]*MasterVersion{
				{
					Version: semver.MustParse("1.13.5"),
					Default: true,
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("3.11.5"),
					Default: true,
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("3.11"),
					Default: false,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("4.1"),
					Default: true,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("4.2"),
					Default: true,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("4.3"),
					Default: true,
					Type:    apiv1.OpenShiftClusterType,
				},
			}, []*MasterUpdate{
				{
					From:      "4.*",
					To:        "4.1",
					Automatic: true,
					Type:      apiv1.OpenShiftClusterType,
				},
			}),
			expectedVersions: []*MasterVersion{
				{
					Version: semver.MustParse("3.11"),
					Default: false,
					Type:    apiv1.OpenShiftClusterType,
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			versions, err := tc.manager.GetMasterVersions(apiv1.OpenShiftClusterType)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// equality.Semantic.DeepEqual panic comparing version.MasterVersion types
			if diff := deep.Equal(versions, tc.expectedVersions); diff != nil {
				t.Fatalf("version list %v is not the same as expected %v", versions, tc.expectedVersions)
			}
		})
	}
}
