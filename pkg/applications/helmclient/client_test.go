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

package helmclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/kube"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/provenance"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"

	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
)

const defaultNs = "default"

func TestNewShouldFailWhenRESTClientGetterNamespaceIsDifferentThanTargetNamespace(t *testing.T) {
	log := kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()
	tempDir := t.TempDir()
	settings := NewSettings(tempDir)

	tf := cmdtesting.NewTestFactory().WithNamespace(defaultNs)
	defer tf.Cleanup()

	_, err := NewClient(context.Background(), tf, settings, "another-ns", log)
	if err == nil {
		t.Fatalf("helmclient.NewClient() should fail when RESTClientGetter namespace is different than targetNamespace : %s", err)
	}
	expectedErrMsg := "namespace set in RESTClientGetter should be the same as targetNamespace"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Fatalf("helmclient.NewClient() fails for the wrong reason. expected error message: '%s' to contain: '%s'", err, expectedErrMsg)
	}
}

func TestRollbackUsesLatestSuccessfulRevision(t *testing.T) {
	tests := []struct {
		name             string
		releases         []*release.Release
		expectedRollback int
	}{
		{
			name: "pending upgrade with superseded revision and no deployed revision",
			releases: []*release.Release{
				testRelease("test-release", "default", 1, release.StatusSuperseded),
				testRelease("test-release", "default", 2, release.StatusPendingUpgrade),
			},
			expectedRollback: 1,
		},
		{
			name: "pending rollback with deployed revision",
			releases: []*release.Release{
				testRelease("test-release", "default", 1, release.StatusDeployed),
				testRelease("test-release", "default", 2, release.StatusPendingRollback),
			},
			expectedRollback: 1,
		},
		{
			name: "latest successful revision is selected",
			releases: []*release.Release{
				testRelease("test-release", "default", 1, release.StatusDeployed),
				testRelease("test-release", "default", 2, release.StatusSuperseded),
				testRelease("test-release", "default", 3, release.StatusPendingUpgrade),
			},
			expectedRollback: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helmClient := newTestHelmClient(t, tt.releases, &kubefake.PrintingKubeClient{Out: io.Discard})

			if err := helmClient.Rollback("test-release"); err != nil {
				t.Fatalf("expected rollback to succeed, got %v", err)
			}

			latestRelease, err := helmClient.actionConfig.Releases.Last("test-release")
			if err != nil {
				t.Fatalf("failed to get latest release: %v", err)
			}
			if latestRelease.Info.Status != release.StatusDeployed {
				t.Fatalf("expected rollback release to be deployed, got %s", latestRelease.Info.Status)
			}
			expectedDescription := "Rollback to " + strconv.Itoa(tt.expectedRollback)
			if latestRelease.Info.Description != expectedDescription {
				t.Fatalf("expected rollback description %q, got %q", expectedDescription, latestRelease.Info.Description)
			}
		})
	}
}

func TestRollbackUninstallsOnlyWhenNoSuccessfulRevisionExists(t *testing.T) {
	tests := []struct {
		name     string
		releases []*release.Release
	}{
		{
			name: "pending upgrade after failed revision",
			releases: []*release.Release{
				testRelease("test-release", "default", 1, release.StatusFailed),
				testRelease("test-release", "default", 2, release.StatusPendingUpgrade),
			},
		},
		{
			name: "pending install without any successful revision",
			releases: []*release.Release{
				testRelease("test-release", "default", 1, release.StatusPendingInstall),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helmClient := newTestHelmClient(t, tt.releases, &kubefake.PrintingKubeClient{Out: io.Discard})

			if err := helmClient.Rollback("test-release"); err != nil {
				t.Fatalf("expected uninstall fallback to succeed, got %v", err)
			}

			if _, err := helmClient.actionConfig.Releases.History("test-release"); !errors.Is(err, driver.ErrReleaseNotFound) {
				t.Fatalf("expected uninstall fallback to purge release history, got %v", err)
			}
		})
	}
}

func TestRollbackDoesNotUninstallWhenRollbackExecutionFails(t *testing.T) {
	helmClient := newTestHelmClient(t, []*release.Release{
		testRelease("test-release", "default", 1, release.StatusSuperseded),
		testRelease("test-release", "default", 2, release.StatusPendingUpgrade),
	}, &kubefake.FailingKubeClient{
		PrintingKubeClient: kubefake.PrintingKubeClient{Out: io.Discard},
		UpdateError:        errors.New("update failed"),
	})

	if err := helmClient.Rollback("test-release"); err == nil {
		t.Fatal("expected rollback error")
	}

	history, err := helmClient.actionConfig.Releases.History("test-release")
	if err != nil {
		t.Fatalf("expected release history to remain after rollback failure, got %v", err)
	}
	if len(history) == 0 {
		t.Fatal("expected release history to remain after rollback failure")
	}
}

func newTestHelmClient(t *testing.T, releases []*release.Release, kubeClient kube.Interface) HelmClient {
	t.Helper()

	actionConfig := &action.Configuration{
		Releases:   storage.Init(driver.NewMemory()),
		KubeClient: kubeClient,
		Log:        func(string, ...interface{}) {},
	}
	for _, rel := range releases {
		if err := actionConfig.Releases.Create(rel); err != nil {
			t.Fatalf("failed to create test release: %v", err)
		}
	}

	return HelmClient{
		actionConfig: actionConfig,
		logger:       kubermaticlog.Logger,
	}
}

func testRelease(name, namespace string, version int, status release.Status) *release.Release {
	return &release.Release{
		Name:      name,
		Namespace: namespace,
		Version:   version,
		Chart:     &chart.Chart{},
		Info: &release.Info{
			Status: status,
		},
	}
}

func TestNewDeploySettings(t *testing.T) {
	tests := []struct {
		name      string
		wait      bool
		timeout   time.Duration
		atomic    bool
		enableDNS bool
		want      *DeployOpts
		wantErr   bool
	}{
		{
			name:      "test valid: no wait, timeout, atomic and enableDNS",
			wait:      false,
			timeout:   0,
			atomic:    false,
			enableDNS: false,
			want: &DeployOpts{
				wait:      false,
				timeout:   0,
				atomic:    false,
				enableDNS: false,
			},
			wantErr: false,
		},
		{
			name:      "test valid: wait=true timeout=10s and no atomic",
			wait:      true,
			timeout:   10 * time.Second,
			atomic:    false,
			enableDNS: false,
			want: &DeployOpts{
				wait:      true,
				timeout:   10 * time.Second,
				atomic:    false,
				enableDNS: false,
			},
			wantErr: false,
		},
		{
			name:      "test valid: wait=true timeout=10s atomic=true",
			wait:      true,
			timeout:   10 * time.Second,
			atomic:    true,
			enableDNS: false,
			want: &DeployOpts{
				wait:      true,
				timeout:   10 * time.Second,
				atomic:    true,
				enableDNS: false,
			},
			wantErr: false,
		},
		{
			name:      "test valid: wait=true timeout=10s atomic=true enableDns=true",
			wait:      true,
			timeout:   10 * time.Second,
			atomic:    true,
			enableDNS: true,
			want: &DeployOpts{
				wait:      true,
				timeout:   10 * time.Second,
				atomic:    true,
				enableDNS: true,
			},
			wantErr: false,
		},
		{
			name:      "test valid: wait=false timeout=0 atomic=false enableDns=true",
			wait:      false,
			timeout:   0,
			atomic:    false,
			enableDNS: true,
			want: &DeployOpts{
				wait:      false,
				timeout:   0,
				atomic:    false,
				enableDNS: true,
			},
			wantErr: false,
		},
		{
			name:      "test invalid: wait=true without timeout",
			wait:      true,
			timeout:   0,
			atomic:    false,
			enableDNS: false,
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "test invalid: atomic=true without wait",
			wait:      false,
			timeout:   10 * time.Second,
			atomic:    true,
			enableDNS: false,
			want:      nil,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDeployOpts(tt.wait, tt.timeout, tt.atomic, tt.enableDNS)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewDeployOpts() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				if tt.want.wait != got.wait {
					t.Errorf("want DeployOpts.wait=%v, got %v", tt.want.wait, got.wait)
				}
				if tt.want.timeout != got.timeout {
					t.Errorf("want DeployOpts.timeout=%v, got %v", tt.want.timeout, got.timeout)
				}
				if tt.want.atomic != got.atomic {
					t.Errorf("want DeployOpts.atomic=%v, got %v", tt.want.atomic, got.atomic)
				}
				if tt.want.enableDNS != got.enableDNS {
					t.Errorf("want DeployOpts.enableDNS=%v, got %v", tt.want.enableDNS, got.enableDNS)
				}
			}
		})
	}
}

// assertDependencyLoaded checks that the given dependency has been loaded into the chart.
func assertDependencyLoaded(chartUnderTest *chart.Chart, dep *chart.Dependency, t *testing.T) {
	t.Helper()
	found := false
	for _, chartDep := range chartUnderTest.Dependencies() {
		if chartDep.Metadata.Name == dep.Name && chartDep.Metadata.Version == dep.Version {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("dependency  %v has not been loaded into chart", dep)
	}
}

// HashReq generates a hash of the dependencies.
//
// This should be used only to compare against another hash generated by this
// function.
// borrowed to https://github.com/helm/helm/blob/49819b4ef782e80b0c7f78c30bd76b51ebb56dc8/internal/resolver/resolver.go#L215
// because it's internal.
func HashReq(req, lock []*chart.Dependency) (string, error) {
	data, err := json.Marshal([2][]*chart.Dependency{req, lock})
	if err != nil {
		return "", err
	}
	s, err := provenance.Digest(bytes.NewBuffer(data))
	return "sha256:" + s, err
}
