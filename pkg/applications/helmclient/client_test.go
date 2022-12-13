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
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/provenance"

	"k8c.io/kubermatic/v2/pkg/applications/test"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"sigs.k8s.io/yaml"
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

func TestDownloadChart(t *testing.T) {
	log := kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()
	chartArchiveDir := t.TempDir()

	chartGlobPath := path.Join(chartArchiveDir, "*.tgz")
	chartArchiveV1Path, chartArchiveV1Size := test.PackageChart(t, "testdata/examplechart", chartArchiveDir)
	chartArchiveV2Path, chartArchiveV2Size := test.PackageChart(t, "testdata/examplechart-v2", chartArchiveDir)
	chartArchiveV1Name := path.Base(chartArchiveV1Path)
	chartArchiveV2Name := path.Base(chartArchiveV2Path)

	httpRegistryUrl := test.StartHttpRegistryWithCleanup(t, chartGlobPath)
	httpRegistryWithAuthUrl := test.StartHttpRegistryWithAuthAndCleanup(t, chartGlobPath)

	ociRegistryUrl := test.StartOciRegistry(t, chartGlobPath)
	ociregistryWithAuthUrl, registryConfigFile := test.StartOciRegistryWithAuth(t, chartGlobPath)

	testCases := []struct {
		name               string
		repoUrl            string
		chartName          string
		chartVersion       string
		auth               AuthSettings
		expectedAchiveName string
		expectedChartSize  int64
		wantErr            bool
	}{
		{
			name:               "Download from HTTP repository should be successful",
			repoUrl:            httpRegistryUrl,
			chartName:          "examplechart",
			chartVersion:       "0.1.0",
			auth:               AuthSettings{},
			expectedAchiveName: chartArchiveV1Name,
			expectedChartSize:  chartArchiveV1Size,
			wantErr:            false,
		},
		{
			name:               "Download from HTTP repository with empty version should get the latest version",
			repoUrl:            httpRegistryUrl,
			chartName:          "examplechart",
			chartVersion:       "",
			auth:               AuthSettings{},
			expectedAchiveName: chartArchiveV2Name,
			expectedChartSize:  chartArchiveV2Size,
			wantErr:            false,
		},
		{
			name:               "Download from HTTP repository with auth should be successful",
			repoUrl:            httpRegistryWithAuthUrl,
			chartName:          "examplechart",
			chartVersion:       "0.1.0",
			auth:               AuthSettings{Username: "username", Password: "password"},
			expectedAchiveName: chartArchiveV1Name,
			expectedChartSize:  chartArchiveV1Size,
			wantErr:            false,
		},
		{
			name:              "Download from HTTP repository should fail when chart does not exist",
			repoUrl:           httpRegistryUrl,
			chartName:         "chartthatdoesnotexist",
			chartVersion:      "0.1.0",
			auth:              AuthSettings{},
			expectedChartSize: 0,
			wantErr:           true,
		},
		{
			name:              "Download from HTTP repository should fail when version is not a semversion",
			repoUrl:           httpRegistryUrl,
			chartName:         "examplechart",
			chartVersion:      "notSemver",
			auth:              AuthSettings{},
			expectedChartSize: 0,
			wantErr:           true,
		},

		{
			name:               "Download from OCI repository should be successful",
			repoUrl:            ociRegistryUrl,
			chartName:          "examplechart",
			chartVersion:       "0.1.0",
			auth:               AuthSettings{},
			expectedAchiveName: chartArchiveV1Name,
			expectedChartSize:  chartArchiveV1Size,
			wantErr:            false,
		},
		{
			name:               "Download from oci repository with empty version should get the latest version",
			repoUrl:            ociRegistryUrl,
			chartName:          "examplechart",
			chartVersion:       "",
			auth:               AuthSettings{},
			expectedAchiveName: chartArchiveV2Name,
			expectedChartSize:  chartArchiveV2Size,
			wantErr:            false,
		},
		{
			name:               "Download from OCI repository with auth should be successful",
			repoUrl:            ociregistryWithAuthUrl,
			chartName:          "examplechart",
			chartVersion:       "0.1.0",
			auth:               AuthSettings{RegistryConfigFile: registryConfigFile},
			expectedAchiveName: chartArchiveV1Name,
			expectedChartSize:  chartArchiveV1Size,
			wantErr:            false,
		},
		{
			name:              "Download from oci repository should fail when chart does not exist",
			repoUrl:           ociRegistryUrl,
			chartName:         "chartthatdoesnotexist",
			chartVersion:      "0.1.0",
			auth:              AuthSettings{},
			expectedChartSize: 0,
			wantErr:           true,
		},
		{
			name:              "Download from oci repository should fail when version is not a semversion",
			repoUrl:           ociRegistryUrl,
			chartName:         "examplechart",
			chartVersion:      "notSemver",
			auth:              AuthSettings{},
			expectedChartSize: 0,
			wantErr:           true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			func() {
				downloadDest := t.TempDir()
				settings := NewSettings(downloadDest)

				tf := cmdtesting.NewTestFactory().WithNamespace(defaultNs)
				defer tf.Cleanup()

				helmClient, err := NewClient(context.Background(), tf, settings, defaultNs, log)
				if err != nil {
					t.Fatalf("can not init helm Client: %s", err)
				}

				chartLoc, err := helmClient.DownloadChart(tc.repoUrl, tc.chartName, tc.chartVersion, downloadDest, tc.auth)

				if (err != nil) != tc.wantErr {
					t.Fatalf("DownloadChart() error = %v, wantErr %v", err, tc.wantErr)
				}

				// No need to proceed with further tests if an error is expected.
				if tc.wantErr {
					return
				}

				// Test chart is downloaded where we expect
				expectedChartLoc := downloadDest + "/" + tc.expectedAchiveName
				if chartLoc != expectedChartLoc {
					t.Fatalf("charLoc is invalid. got '%s'. expect '%s'", chartLoc, expectedChartLoc)
				}

				// Smoke test: check downloaded chart has expected size
				downloadChartInfo, err := os.Stat(chartLoc)
				if err != nil {
					t.Fatalf("can not check size of downloaded chart: %s", err)
				}

				if tc.expectedChartSize != downloadChartInfo.Size() {
					t.Errorf("size of download chart should be '%d' but was '%d'", tc.expectedChartSize, downloadChartInfo.Size())
				}
			}()
		})
	}
}

func TestBuildDependencies(t *testing.T) {
	log := kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()

	chartArchiveDir := t.TempDir()
	chartGlobPath := path.Join(chartArchiveDir, "*.tgz")

	test.PackageChart(t, "testdata/examplechart", chartArchiveDir)
	test.PackageChart(t, "testdata/examplechart2", chartArchiveDir)

	httpRegistryUrl := test.StartHttpRegistryWithCleanup(t, chartGlobPath)
	httpRegistryWithAuthUrl := test.StartHttpRegistryWithAuthAndCleanup(t, chartGlobPath)

	ociRegistryUrl := test.StartOciRegistry(t, chartGlobPath)
	ociRegistryWithAuthUrl, registryConfigFile := test.StartOciRegistryWithAuth(t, chartGlobPath)

	const fileDepChartName = "filedepchart"
	const fileDepChartVersion = "2.3.4"
	testCases := []struct {
		name         string
		dependencies []*chart.Dependency
		hasLockFile  bool
		auth         AuthSettings
		wantErr      bool
	}{
		{
			name:         "no dependencies",
			dependencies: []*chart.Dependency{},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "http dependencies with Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpRegistryUrl}},
			hasLockFile:  true,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "http dependencies without Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpRegistryUrl}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "oci dependencies with Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: ociRegistryUrl}},
			hasLockFile:  true,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "oci dependencies without Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: ociRegistryUrl}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "file dependencies with Chat.lock file",
			dependencies: []*chart.Dependency{{Name: fileDepChartName, Version: fileDepChartVersion, Repository: "file://../" + fileDepChartName}},
			hasLockFile:  true,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "file dependencies without Chat.lock file",
			dependencies: []*chart.Dependency{{Name: fileDepChartName, Version: fileDepChartVersion, Repository: "file://../" + fileDepChartName}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "http and oci dependencies with Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpRegistryUrl}, {Name: "examplechart2", Version: "0.1.0", Repository: ociRegistryUrl}},
			hasLockFile:  true,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "http and oci dependencies without Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpRegistryUrl}, {Name: "examplechart2", Version: "0.1.0", Repository: ociRegistryUrl}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      false,
		},
		{
			name:         "http dependencies with Chat.lock file and auth",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpRegistryWithAuthUrl}},
			hasLockFile:  true,
			auth:         AuthSettings{Username: "username", Password: "password"},
			wantErr:      false,
		},
		{
			name:         "http dependencies without Chat.lock file and auth",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: httpRegistryWithAuthUrl}},
			hasLockFile:  false,
			auth:         AuthSettings{Username: "username", Password: "password"},
			wantErr:      false,
		},
		{
			name:         "oci dependencies with Chat.lock file and auth",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: ociRegistryWithAuthUrl}},
			hasLockFile:  true,
			auth:         AuthSettings{RegistryConfigFile: registryConfigFile},
			wantErr:      false,
		},
		{
			name:         "oci dependencies without Chat.lock file and auth",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: ociRegistryWithAuthUrl}},
			hasLockFile:  false,
			auth:         AuthSettings{RegistryConfigFile: registryConfigFile},
			wantErr:      false,
		},
		{
			name:         "http dependency with empty version should fail",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "", Repository: httpRegistryUrl}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      true,
		},
		{
			name:         "oci dependency with empty version should fail",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "", Repository: ociRegistryWithAuthUrl}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      true,
		},
		{
			name:         "http dependency with non semver should fail",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "", Repository: httpRegistryUrl}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      true,
		},
		{
			name:         "oci dependency with non semver should fail",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "", Repository: ociRegistryWithAuthUrl}},
			hasLockFile:  false,
			auth:         AuthSettings{},
			wantErr:      true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			func() {
				tempDir := t.TempDir()
				settings := NewSettings(tempDir)

				tf := cmdtesting.NewTestFactory().WithNamespace(defaultNs)
				defer tf.Cleanup()

				// This chart may be used as a file dependency by testingChart.
				fileDepChart := &chart.Chart{
					Metadata: &chart.Metadata{
						APIVersion: chart.APIVersionV2,
						Name:       fileDepChartName,
						Version:    fileDepChartVersion,
					},
				}
				if err := chartutil.SaveDir(fileDepChart, tempDir); err != nil {
					t.Fatal(err)
				}

				chartName := "testing-chart"
				testingChart := &chart.Chart{
					Metadata: &chart.Metadata{
						APIVersion:   chart.APIVersionV2,
						Name:         chartName,
						Version:      "1.2.3",
						Dependencies: tc.dependencies,
					},
				}
				if err := chartutil.SaveDir(testingChart, tempDir); err != nil {
					t.Fatal(err)
				}

				lockFile := filepath.Join(tempDir, chartName, "Chart.lock")
				generatedTime := time.Now()
				// chartutil.SaveDir does not save Chart.lock so we do it manually
				if tc.hasLockFile {
					digest, err := HashReq(tc.dependencies, tc.dependencies)
					if err != nil {
						t.Fatal(err)
					}
					loc := &chart.Lock{
						Generated:    generatedTime,
						Digest:       digest,
						Dependencies: tc.dependencies,
					}
					out, err := yaml.Marshal(loc)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(lockFile, out, 0644); err != nil {
						t.Fatal(err)
					}
				}

				helmClient, err := NewClient(context.Background(), tf, settings, defaultNs, log)
				if err != nil {
					t.Fatalf("can not init helm client: %s", err)
				}

				chartUnderTest, err := helmClient.buildDependencies(filepath.Join(tempDir, chartName), tc.auth)

				if (err != nil) != tc.wantErr {
					t.Fatalf("buildDependencies() error = %v, wantErr %v", err, tc.wantErr)
				}

				// No need to proceed with further tests if an error is expected.
				if tc.wantErr {
					return
				}

				// check dependencies
				for _, dep := range tc.dependencies {
					depArchiveName := dep.Name + "-" + dep.Version + ".tgz"
					desiredDependency := filepath.Join(tempDir, chartName, "charts", depArchiveName)
					if _, err := os.Stat(desiredDependency); err != nil {
						t.Fatalf("dependency %v has not been downloaded in charts directory: %s", dep, err)
					}
					assertDependencyLoaded(chartUnderTest, dep, t)
				}
				if tc.hasLockFile {
					actualLock := &chart.Lock{}
					in, err := os.ReadFile(lockFile)
					if err != nil {
						t.Fatalf("can not read actual Chart.lock: %s", err)
					}
					if err := yaml.Unmarshal(in, actualLock); err != nil {
						t.Fatalf("can not unmarshamm Chart.lock: %s", err)
					}
					if !generatedTime.Equal(actualLock.Generated) {
						t.Fatalf("lock file should not have been modified. expected generatedTime:%v, actual generatedTime:%v", generatedTime, actualLock.Generated)
					}
				}
			}()
		})
	}
}

func TestNewDeploySettings(t *testing.T) {
	tests := []struct {
		name    string
		wait    bool
		timeout time.Duration
		atomic  bool
		want    *DeployOpts
		wantErr bool
	}{
		{
			name:    "test valid: no wait, timeout and atomic",
			wait:    false,
			timeout: 0,
			atomic:  false,
			want: &DeployOpts{
				wait:    false,
				timeout: 0,
				atomic:  false,
			},
			wantErr: false,
		},
		{
			name:    "test valid: wait=true timeout=10s and no atomic",
			wait:    true,
			timeout: 10 * time.Second,
			atomic:  false,
			want: &DeployOpts{
				wait:    true,
				timeout: 10 * time.Second,
				atomic:  false,
			},
			wantErr: false,
		},
		{
			name:    "test valid: wait=true timeout=10s atomic=true",
			wait:    true,
			timeout: 10 * time.Second,
			atomic:  true,
			want: &DeployOpts{
				wait:    true,
				timeout: 10 * time.Second,
				atomic:  true,
			},
			wantErr: false,
		},
		{
			name:    "test invalid: wait=true without timeout",
			wait:    true,
			timeout: 0,
			atomic:  false,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "test invalid: atomic=true without wait",
			wait:    false,
			timeout: 10 * time.Second,
			atomic:  true,
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDeployOpts(tt.wait, tt.timeout, tt.atomic)
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
