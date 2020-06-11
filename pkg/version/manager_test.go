package version

import (
	"reflect"
	"testing"

	"github.com/Masterminds/semver"

	"github.com/kubermatic/kubermatic/pkg/validation/nodeupdate"
)

func TestAutomaticNodeUpdate(t *testing.T) {
	testCases := []struct {
		name                string
		fromVersion         string
		controlPlaneVersion string
		updates             []*Update
		expectedError       error
		expectedVersion     *Version
	}{
		{
			name:                "Happy path, we get a version",
			fromVersion:         "1.5.0",
			controlPlaneVersion: "1.6.0",
			updates: []*Update{{
				From:                "1.5.0",
				To:                  "1.6.0",
				AutomaticNodeUpdate: true,
			}},
			expectedVersion: &Version{Version: semver.MustParse("1.6.0")},
		},
		{
			name:                "Node compatibility check fails, error",
			fromVersion:         "1.5.0",
			controlPlaneVersion: "1.5.0",
			updates: []*Update{{
				From:                "1.5.0",
				To:                  "1.6.0",
				AutomaticNodeUpdate: true,
			}},
			expectedError: nodeupdate.ErrVersionSkew{
				ControlPlane: semver.MustParse("1.5.0"),
				Kubelet:      semver.MustParse("1.6.0"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := &Manager{
				updates: tc.updates,
				versions: []*Version{
					{Version: semver.MustParse(tc.updates[0].To)},
				},
			}
			version, err := m.AutomaticNodeUpdate(tc.fromVersion, "", tc.controlPlaneVersion)
			// a simple err comparison considers them different, because they contain different
			// semver pointers, even thought their value is equal
			if !reflect.DeepEqual(err, tc.expectedError) {
				t.Fatalf("expected err %v, got err %v", tc.expectedError, err)
			}
			if err != nil {
				return
			}

			if !version.Version.Equal(tc.expectedVersion.Version) {
				t.Errorf("expected version %q, got version %q", tc.expectedVersion.Version.String(), version.Version.String())
			}
		})
	}
}
