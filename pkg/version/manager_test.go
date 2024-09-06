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
	"reflect"
	"testing"

	semverlib "github.com/Masterminds/semver/v3"

	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/validation/nodeupdate"
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
			expectedVersion: &Version{Version: semverlib.MustParse("1.6.0")},
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
			expectedError: nodeupdate.VersionSkewError{
				ControlPlane: semverlib.MustParse("1.5.0"),
				Kubelet:      semverlib.MustParse("1.6.0"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := &Manager{
				updates: tc.updates,
				versions: []*Version{
					{Version: semverlib.MustParse(tc.updates[0].To)},
				},
			}

			version, err := m.AutomaticNodeUpdate(tc.fromVersion, tc.controlPlaneVersion)
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

func TestAutomaticControlPlaneUpdate(t *testing.T) {
	testCases := []struct {
		name            string
		fromVersion     string
		updates         []*Update
		versions        []*Version
		expectErr       bool
		expectedVersion *Version
	}{
		{
			name:        "automatic version update via Automatic supported",
			fromVersion: "1.5.0",
			updates: []*Update{{
				From:      "1.5.0",
				To:        "1.6.0",
				Automatic: true,
			}},
			versions: []*Version{{
				Version: semverlib.MustParse("1.6.0"),
			}},
			expectErr:       false,
			expectedVersion: &Version{Version: semverlib.MustParse("1.6.0")},
		},
		{
			name:        "automatic version update via AutomaticNodeUpdate supported",
			fromVersion: "1.5.0",
			updates: []*Update{{
				From:                "1.5.0",
				To:                  "1.6.0",
				AutomaticNodeUpdate: true,
			}},
			versions: []*Version{{
				Version: semverlib.MustParse("1.6.0"),
			}},
			expectErr:       false,
			expectedVersion: &Version{Version: semverlib.MustParse("1.6.0")},
		},
		{
			name:        "automatic version update from wildcard version supported",
			fromVersion: "1.5.0",
			updates: []*Update{{
				From:      "1.5.*",
				To:        "1.5.1",
				Automatic: true,
			}},
			versions: []*Version{{
				Version: semverlib.MustParse("1.5.1"),
			}},
			expectErr:       false,
			expectedVersion: &Version{Version: semverlib.MustParse("1.5.1")},
		},
		{
			name:        "automatic version update to wildcard version not supported",
			fromVersion: "1.5.0",
			updates: []*Update{{
				From:      "1.5.0",
				To:        "1.5.*",
				Automatic: true,
			}},
			versions: []*Version{{
				Version: semverlib.MustParse("1.5.2"),
			}},
			expectErr: true,
		},
		{
			name:        "automatic version update to non-existing version not supported",
			fromVersion: "1.5.0",
			updates: []*Update{{
				From:      "1.5.0",
				To:        "1.5.1",
				Automatic: true,
			}},
			versions: []*Version{{
				Version: semverlib.MustParse("1.5.2"),
			}},
			expectErr: true,
		},
		{
			name:        "automatic version update with multiple target versions not supported",
			fromVersion: "1.5.0",
			updates: []*Update{
				{
					From:      "1.5.0",
					To:        "1.5.1",
					Automatic: true,
				},
				{
					From:      "1.5.0",
					To:        "1.5.2",
					Automatic: true,
				},
			},
			versions: []*Version{
				{
					Version: semverlib.MustParse("1.5.1"),
				},
				{
					Version: semverlib.MustParse("1.5.2"),
				},
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := &Manager{
				updates:  tc.updates,
				versions: tc.versions,
			}

			version, err := m.AutomaticControlplaneUpdate(tc.fromVersion)

			if (err != nil) != tc.expectErr {
				t.Fatalf("expect err %v, got err %q", tc.expectErr, err)
				return
			}

			if !reflect.DeepEqual(tc.expectedVersion, version) {
				t.Errorf("expected version %v, got version %v", tc.expectedVersion, version)
			}
		})
	}
}

func TestGetVersions(t *testing.T) {
	testCases := []struct {
		name             string
		updates          []*Update
		versions         []*Version
		expectedVersions []*Version
	}{
		{
			name: "versions without automatic update included",
			updates: []*Update{{
				From: "1.5.0",
				To:   "1.6.0",
			}},
			versions: []*Version{
				{
					Version: semverlib.MustParse("1.5.0"),
				},
				{
					Version: semverlib.MustParse("1.6.0"),
				},
			},
			expectedVersions: []*Version{
				{
					Version: semverlib.MustParse("1.5.0"),
				},
				{
					Version: semverlib.MustParse("1.6.0"),
				},
			},
		},
		{
			name: "version with automatic update via Automatic excluded",
			updates: []*Update{{
				From:      "1.5.0",
				To:        "1.6.0",
				Automatic: true,
			}},
			versions: []*Version{
				{
					Version: semverlib.MustParse("1.5.0"),
				},
				{
					Version: semverlib.MustParse("1.6.0"),
				},
			},
			expectedVersions: []*Version{{
				Version: semverlib.MustParse("1.6.0"),
			}},
		},
		{
			name: "version with automatic update via AutomaticNodeUpdate excluded",
			updates: []*Update{{
				From:                "1.5.0",
				To:                  "1.6.0",
				AutomaticNodeUpdate: true,
			}},
			versions: []*Version{
				{
					Version: semverlib.MustParse("1.5.0"),
				},
				{
					Version: semverlib.MustParse("1.6.0"),
				},
			},
			expectedVersions: []*Version{{
				Version: semverlib.MustParse("1.6.0"),
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := &Manager{
				updates:  tc.updates,
				versions: tc.versions,
			}

			versions, _ := m.GetVersions()

			if !reflect.DeepEqual(tc.expectedVersions, versions) {
				t.Errorf("versions differ:\n%v", diff.ObjectDiff(tc.expectedVersions, versions))
			}
		})
	}
}
