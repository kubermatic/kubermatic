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
	"encoding/json"
	"errors"
	"os/exec"
	"slices"
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
	const (
		namespace   = "test-namespace"
		releaseName = "release"
	)

	// Helm rejects --force-conflicts unless server-side apply is enabled, and
	// --server-side defaults to "auto", which inherits the apply method from the
	// release's newest revision. So the flags depend on release state, not just on
	// the helm version.
	testcases := []struct {
		name        string
		helmVersion string
		flags       []string

		// metadataStatus and metadataApplyMethod are what `helm get metadata`
		// reports. An empty apply method means helm never recorded one, which is the
		// case for every release installed by Helm 3.
		metadataStatus      string
		metadataApplyMethod string
		// metadataFails simulates a release helm cannot report on, either because it
		// does not exist or because the read failed. releaseListed then decides
		// which of the two the fallback `helm list` sees.
		metadataFails bool
		releaseListed bool
		listFails     bool

		expectedArgs []string
		// expectMetadataProbe guards against probing with a helm that has no such
		// subcommand. expectListProbe guards against listing when the metadata call
		// already answered the question.
		expectMetadataProbe bool
		expectListProbe     bool
	}{
		{
			name:                "helm 3 never enables server-side apply and keeps --atomic",
			helmVersion:         "3.19.0",
			flags:               []string{"--atomic"},
			metadataStatus:      "deployed",
			metadataApplyMethod: "csa",
			expectedArgs: []string{
				"helm",
				"--namespace", namespace,
				"upgrade", "--install",
				"--timeout", "30s",
				"--values", "values.yaml",
				"--atomic",
				releaseName, "chart-dir",
			},
			expectMetadataProbe: false,
			expectListProbe:     false,
		},
		{
			name:          "helm 4 forces conflicts on a fresh install and translates --atomic",
			helmVersion:   "4.0.0",
			flags:         []string{"--atomic"},
			metadataFails: true,
			releaseListed: false,
			expectedArgs: []string{
				"helm",
				"--namespace", namespace,
				"upgrade", "--install",
				"--timeout", "30s",
				"--server-side=true", "--force-conflicts",
				"--values", "values.yaml",
				"--rollback-on-failure",
				releaseName, "chart-dir",
			},
			expectMetadataProbe: true,
			expectListProbe:     true,
		},
		{
			// --server-side=true is load-bearing next to --rollback-on-failure: helm
			// forwards both values into the rollback, which resolves the apply method
			// against the rollback target's revision. Leaving --server-side at auto
			// there can make the rollback itself fail on the invalid flag pair and
			// mask the real upgrade error.
			name:                "helm 4 keeps the full pair alongside the rollback flag",
			helmVersion:         "4.1.1",
			flags:               []string{"--atomic"},
			metadataStatus:      "deployed",
			metadataApplyMethod: "ssa",
			expectedArgs: []string{
				"helm",
				"--namespace", namespace,
				"upgrade", "--install",
				"--timeout", "30s",
				"--server-side=true", "--force-conflicts",
				"--values", "values.yaml",
				"--rollback-on-failure",
				releaseName, "chart-dir",
			},
			expectMetadataProbe: true,
			expectListProbe:     false,
		},
		{
			name:                "helm 4 forces conflicts when the release already applies server-side",
			helmVersion:         "4.1.1",
			metadataStatus:      "deployed",
			metadataApplyMethod: "ssa",
			expectedArgs: []string{
				"helm",
				"--namespace", namespace,
				"upgrade", "--install",
				"--timeout", "30s",
				"--server-side=true", "--force-conflicts",
				"--values", "values.yaml",
				releaseName, "chart-dir",
			},
			expectMetadataProbe: true,
			expectListProbe:     false,
		},
		{
			name:                "helm 4 leaves a client-side release alone",
			helmVersion:         "4.1.1",
			metadataStatus:      "deployed",
			metadataApplyMethod: "csa",
			expectedArgs: []string{
				"helm",
				"--namespace", namespace,
				"upgrade", "--install",
				"--timeout", "30s",
				"--values", "values.yaml",
				releaseName, "chart-dir",
			},
			expectMetadataProbe: true,
			expectListProbe:     false,
		},
		{
			// The regression that shipped in v2.30.6: a release installed by Helm 3
			// has no recorded apply method, so helm resolves "auto" to client-side and
			// refuses --force-conflicts.
			name:                "helm 4 leaves a release with no recorded apply method alone",
			helmVersion:         "4.1.1",
			metadataStatus:      "deployed",
			metadataApplyMethod: "",
			expectedArgs: []string{
				"helm",
				"--namespace", namespace,
				"upgrade", "--install",
				"--timeout", "30s",
				"--values", "values.yaml",
				releaseName, "chart-dir",
			},
			expectMetadataProbe: true,
			expectListProbe:     false,
		},
		{
			// helm treats a release whose newest revision is uninstalled as a fresh
			// install, which applies server-side no matter what that revision
			// recorded. Deciding on the apply method alone would withhold
			// --force-conflicts exactly where a conflict can occur.
			name:                "helm 4 forces conflicts on an uninstalled release with a client-side revision",
			helmVersion:         "4.1.1",
			metadataStatus:      "uninstalled",
			metadataApplyMethod: "csa",
			expectedArgs: []string{
				"helm",
				"--namespace", namespace,
				"upgrade", "--install",
				"--timeout", "30s",
				"--server-side=true", "--force-conflicts",
				"--values", "values.yaml",
				releaseName, "chart-dir",
			},
			expectMetadataProbe: true,
			expectListProbe:     false,
		},
		{
			name:          "helm 4 does not force conflicts when an existing release cannot be read",
			helmVersion:   "4.1.1",
			metadataFails: true,
			releaseListed: true,
			expectedArgs: []string{
				"helm",
				"--namespace", namespace,
				"upgrade", "--install",
				"--timeout", "30s",
				"--values", "values.yaml",
				releaseName, "chart-dir",
			},
			expectMetadataProbe: true,
			expectListProbe:     true,
		},
		{
			name:          "helm 4 does not force conflicts when releases cannot be listed either",
			helmVersion:   "4.1.1",
			metadataFails: true,
			listFails:     true,
			expectedArgs: []string{
				"helm",
				"--namespace", namespace,
				"upgrade", "--install",
				"--timeout", "30s",
				"--values", "values.yaml",
				releaseName, "chart-dir",
			},
			expectMetadataProbe: true,
			expectListProbe:     true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			originalRunCmd := runCmd
			t.Cleanup(func() {
				runCmd = originalRunCmd
			})

			upgradeCalls := [][]string{}
			metadataArgs := []string{}
			listProbed := false

			runCmd = func(cmd *exec.Cmd) ([]byte, error) {
				switch {
				case containsArg(cmd.Args, "metadata"):
					metadataArgs = append([]string{}, cmd.Args...)
					if tc.metadataFails {
						return nil, errors.New("simulated helm get metadata failure")
					}
					payload := map[string]any{
						"name":     releaseName,
						"revision": 1,
						"status":   tc.metadataStatus,
					}
					// helm omits the field entirely when it was never recorded
					if tc.metadataApplyMethod != "" {
						payload["applyMethod"] = tc.metadataApplyMethod
					}
					encoded, err := json.Marshal(payload)
					require.NoError(t, err)
					return encoded, nil

				case containsArg(cmd.Args, "list"):
					listProbed = true
					if tc.listFails {
						return nil, errors.New("simulated helm list failure")
					}
					if !tc.releaseListed {
						return []byte(`[]`), nil
					}
					return []byte(`[{"name":"` + releaseName + `","namespace":"` + namespace + `","chart":"test-chart-1.2.3"}]`), nil

				default:
					upgradeCalls = append(upgradeCalls, append([]string{}, cmd.Args...))
					return nil, nil
				}
			}

			c := &cli{
				binary:     "helm",
				kubeconfig: "test-kubeconfig",
				timeout:    30 * time.Second,
				version:    *semverlib.MustParse(tc.helmVersion),
				logger:     logrus.New(),
			}

			err := c.InstallChart(namespace, releaseName, "chart-dir", "values.yaml", nil, tc.flags)
			require.NoError(t, err)

			require.Len(t, upgradeCalls, 1, "expected exactly one helm upgrade invocation")
			upgradeArgs := upgradeCalls[0]
			require.Equal(t, tc.expectedArgs, upgradeArgs)

			require.Equal(t, tc.expectMetadataProbe, len(metadataArgs) > 0,
				"unexpected `helm get metadata` probing behaviour")
			require.Equal(t, tc.expectListProbe, listProbed,
				"unexpected `helm list` probing behaviour")

			if len(metadataArgs) > 0 {
				// The probe must read the same revision helm's own upgrade reads, which
				// means no --revision, and it must be namespaced and JSON.
				require.Equal(t, []string{
					"helm",
					"--namespace", namespace,
					"get", "metadata", releaseName, "-o", "json",
				}, metadataArgs)
			}

			// The invariant the v2.30.6 regression violated: helm rejects
			// --force-conflicts unless server-side apply is enabled alongside it.
			if containsArg(upgradeArgs, "--force-conflicts") {
				require.Contains(t, upgradeArgs, "--server-side=true",
					"--force-conflicts must never be emitted without --server-side=true")
			}
		})
	}
}

func TestInstallChartPropagatesUpgradeFailure(t *testing.T) {
	testcases := []struct {
		name        string
		applyMethod string
		// expectHint asserts the error names the apply method, which is the only way
		// an operator can tell a field conflict apart from any other failure.
		expectHint bool
	}{
		{
			name:        "server-side release fails without the client-side hint",
			applyMethod: "ssa",
			expectHint:  false,
		},
		{
			name:        "client-side release explains that conflicts were not forced",
			applyMethod: "csa",
			expectHint:  true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			originalRunCmd := runCmd
			t.Cleanup(func() {
				runCmd = originalRunCmd
			})

			runCmd = func(cmd *exec.Cmd) ([]byte, error) {
				if containsArg(cmd.Args, "metadata") {
					return []byte(`{"name":"release","revision":1,"status":"deployed","applyMethod":"` + tc.applyMethod + `"}`), nil
				}
				return nil, errors.New("conflict occurred while applying object")
			}

			c := &cli{
				binary:     "helm",
				kubeconfig: "test-kubeconfig",
				timeout:    30 * time.Second,
				version:    *semverlib.MustParse("4.1.1"),
				logger:     logrus.New(),
			}

			err := c.InstallChart("test-namespace", "release", "chart-dir", "values.yaml", nil, nil)
			require.Error(t, err)
			require.Contains(t, err.Error(), "conflict occurred while applying object",
				"the underlying helm error must survive wrapping")

			if tc.expectHint {
				require.Contains(t, err.Error(), "applies client-side")
				require.Contains(t, err.Error(), "helm get metadata release --namespace test-namespace")
			} else {
				require.NotContains(t, err.Error(), "applies client-side")
			}
		})
	}
}

func containsArg(args []string, want string) bool {
	return slices.Contains(args, want)
}
