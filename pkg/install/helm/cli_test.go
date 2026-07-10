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

package helm

import (
	"os/exec"
	"testing"
	"time"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestGuessReleaseVersion(t *testing.T) {
	testcases := []struct {
		input           string
		expectedVersion *semverlib.Version
		expectedChart   string
	}{
		{
			input:           "foo",
			expectedVersion: nil,
			expectedChart:   "",
		},
		{
			input:           "foo-bar",
			expectedVersion: nil,
			expectedChart:   "",
		},
		{
			input:           "foo-1.2.3",
			expectedVersion: semverlib.MustParse("1.2.3"),
			expectedChart:   "foo",
		},
		{
			input:           "foo-bar-1.2.3",
			expectedVersion: semverlib.MustParse("1.2.3"),
			expectedChart:   "foo-bar",
		},
		{
			input:           "foo-bar-super-long-release-name-1.2.3",
			expectedVersion: semverlib.MustParse("1.2.3"),
			expectedChart:   "foo-bar-super-long-release-name",
		},
		{
			input:           "foo-bar-super-long-release-name-1.2.3-suffix-really-long",
			expectedVersion: semverlib.MustParse("1.2.3-suffix-really-long"),
			expectedChart:   "foo-bar-super-long-release-name",
		},
		{
			input:           "this-is-not-a-version",
			expectedVersion: nil,
			expectedChart:   "",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.input, func(t *testing.T) {
			version, chart, err := guessChartName(testcase.input)
			if testcase.expectedVersion == nil && err == nil {
				t.Fatalf("Expected an error, but got version %v and chart %q.", version, chart)
			}
			if testcase.expectedVersion != nil {
				if !version.Equal(testcase.expectedVersion) {
					t.Fatalf("Expected version %v, but got version %v.", testcase.expectedVersion, version)
				}

				if testcase.expectedChart != chart {
					t.Fatalf("Expected chart %q, but got chart %q.", testcase.expectedChart, chart)
				}
			}
		})
	}
}

func TestListReleasesUsesCorrectFlagsForHelmVersion(t *testing.T) {
	testcases := []struct {
		name         string
		helmVersion  string
		expectedArgs []string
	}{
		{
			name:        "helm v3 adds --all",
			helmVersion: "3.19.0",
			expectedArgs: []string{
				"helm",
				"--namespace", "test-namespace",
				"list", "-o", "json", "--all",
			},
		},
		{
			name:        "helm v4 does not add --all",
			helmVersion: "4.0.0",
			expectedArgs: []string{
				"helm",
				"--namespace", "test-namespace",
				"list", "-o", "json",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			originalRunCmd := runCmd
			t.Cleanup(func() {
				runCmd = originalRunCmd
			})

			capturedArgs := []string{}
			runCmd = func(cmd *exec.Cmd) ([]byte, error) {
				capturedArgs = append([]string{}, cmd.Args...)
				return []byte(`[{"name":"release","namespace":"test-namespace","chart":"test-chart-1.2.3"}]`), nil
			}

			c := &cli{
				binary:     "helm",
				kubeconfig: "test-kubeconfig",
				timeout:    30 * time.Second,
				version:    *semverlib.MustParse(tc.helmVersion),
				logger:     logrus.New(),
			}

			releases, err := c.ListReleases("test-namespace")
			require.NoError(t, err)
			require.Equal(t, tc.expectedArgs, capturedArgs)
			require.Len(t, releases, 1)
			require.Equal(t, "test-chart", releases[0].Chart)
			require.Equal(t, "1.2.3", releases[0].Version.String())
		})
	}
}

func TestInstallChartUsesCorrectFlagsForHelmVersion(t *testing.T) {
	testcases := []struct {
		name         string
		helmVersion  string
		flags        []string
		expectedArgs []string
	}{
		{
			name:        "helm v3 does not enable server-side apply and keeps --atomic",
			helmVersion: "3.19.0",
			flags:       []string{"--atomic"},
			expectedArgs: []string{
				"helm",
				"--namespace", "test-namespace",
				"upgrade", "--install",
				"--timeout", "30s",
				"--values", "values.yaml",
				"--atomic",
				"release", "chart-dir",
			},
		},
		{
			name:        "helm v4.0 enables server-side apply and translates --atomic",
			helmVersion: "4.0.0",
			flags:       []string{"--atomic"},
			expectedArgs: []string{
				"helm",
				"--namespace", "test-namespace",
				"upgrade", "--install",
				"--timeout", "30s",
				"--server-side=true", "--force-conflicts",
				"--values", "values.yaml",
				"--rollback-on-failure",
				"release", "chart-dir",
			},
		},
		{
			name:        "helm v4.1 enables server-side apply without extra flags",
			helmVersion: "4.1.1",
			flags:       nil,
			expectedArgs: []string{
				"helm",
				"--namespace", "test-namespace",
				"upgrade", "--install",
				"--timeout", "30s",
				"--server-side=true", "--force-conflicts",
				"--values", "values.yaml",
				"release", "chart-dir",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			originalRunCmd := runCmd
			t.Cleanup(func() {
				runCmd = originalRunCmd
			})

			capturedArgs := []string{}
			runCmd = func(cmd *exec.Cmd) ([]byte, error) {
				capturedArgs = append([]string{}, cmd.Args...)
				return nil, nil
			}

			c := &cli{
				binary:     "helm",
				kubeconfig: "test-kubeconfig",
				timeout:    30 * time.Second,
				version:    *semverlib.MustParse(tc.helmVersion),
				logger:     logrus.New(),
			}

			err := c.InstallChart("test-namespace", "release", "chart-dir", "values.yaml", nil, tc.flags)
			require.NoError(t, err)
			require.Equal(t, tc.expectedArgs, capturedArgs)
		})
	}
}
