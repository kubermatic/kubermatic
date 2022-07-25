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
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/distribution/distribution/v3/configuration"
	ociregistry "github.com/distribution/distribution/v3/registry"
	"github.com/phayes/freeport"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/provenance"
	"helm.sh/helm/v3/pkg/pusher"
	helmregistry "helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/repo/repotest"
	"helm.sh/helm/v3/pkg/uploader"

	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"sigs.k8s.io/yaml"
)

const defaultNs = "default"

func TestNewShouldFailWhenRESTClientGetterNamespaceIsDifferentThanTargetNamespace(t *testing.T) {
	log := kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()
	downloadDest, settings := createHelmConfiguration(t)
	defer os.RemoveAll(downloadDest)

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
	chartArchivePath, chartArchiveSize := packageChart(t, "testdata/examplechart", chartArchiveDir)

	srv, err := repotest.NewTempServerWithCleanup(t, path.Join(chartArchiveDir, "*.tgz"))
	defer srv.Stop()
	if err != nil {
		t.Fatal(err)
	}

	ociRegistryUrl := startOciRegistry(t, chartArchivePath)

	testCases := []struct {
		name              string
		repoUrl           string
		chartName         string
		chartVersion      string
		expectedChartSize int64
		wantErr           bool
	}{
		{
			name:              "Download from HTTPshould be successful",
			repoUrl:           srv.URL(),
			chartName:         "examplechart",
			chartVersion:      "0.1.0",
			expectedChartSize: chartArchiveSize,
			wantErr:           false,
		},
		{
			name:              "Download from HTTP repository should fail when chart does not exist",
			repoUrl:           srv.URL(),
			chartName:         "chartthatdoesnotexist",
			chartVersion:      "0.1.0",
			expectedChartSize: 0,
			wantErr:           true,
		},

		{
			name:              "Download from OCI repository should be successful",
			repoUrl:           ociRegistryUrl,
			chartName:         "examplechart",
			chartVersion:      "0.1.0",
			expectedChartSize: chartArchiveSize,
			wantErr:           false,
		},
		{
			name:              "Download from oci repository should fail when chart does not exist",
			repoUrl:           ociRegistryUrl,
			chartName:         "chartthatdoesnotexist",
			chartVersion:      "0.1.0",
			expectedChartSize: 0,
			wantErr:           true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			func() {
				downloadDest, settings := createHelmConfiguration(t)
				defer os.RemoveAll(downloadDest)

				tf := cmdtesting.NewTestFactory().WithNamespace(defaultNs)
				defer tf.Cleanup()

				helmClient, err := NewClient(context.Background(), tf, settings, defaultNs, log)
				if err != nil {
					t.Fatalf("can not init helm Client: %s", err)
				}

				chartLoc, err := helmClient.DownloadChart(tc.repoUrl, tc.chartName, tc.chartVersion, downloadDest)

				if (err != nil) != tc.wantErr {
					t.Fatalf("DownloadChart() error = %v, wantErr %v", err, tc.wantErr)
				}

				// No need to proceed with further tests if an error is expected.
				if tc.wantErr {
					return
				}

				// Test chart is downloaded where we expect
				chartArchiveName := tc.chartName + "-" + tc.chartVersion + ".tgz"
				expectedChartLoc := downloadDest + "/" + chartArchiveName
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
	chartArchivePath, _ := packageChart(t, "testdata/examplechart", chartArchiveDir)

	srv, err := repotest.NewTempServerWithCleanup(t, path.Join(chartArchiveDir, "*.tgz"))
	defer srv.Stop()
	if err != nil {
		t.Fatal(err)
	}

	ociRegistryUrl := startOciRegistry(t, chartArchivePath)

	const fileDepChartName = "filedepchart"
	const fileDepChartVersion = "2.3.4"
	testCases := []struct {
		name         string
		dependencies []*chart.Dependency
		hasLockFile  bool
		wantErr      bool
	}{
		{
			name:         "no dependencies",
			dependencies: []*chart.Dependency{},
			hasLockFile:  false,
			wantErr:      false,
		},
		{
			name:         "http dependencies with Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: srv.URL()}},
			hasLockFile:  true,
			wantErr:      false,
		},
		{
			name:         "http dependencies without Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: srv.URL()}},
			hasLockFile:  false,
			wantErr:      false,
		},
		{
			name:         "oci dependencies with Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: ociRegistryUrl}},
			hasLockFile:  true,
			wantErr:      false,
		},
		{
			name:         "oci dependencies without Chat.lock file",
			dependencies: []*chart.Dependency{{Name: "examplechart", Version: "0.1.0", Repository: ociRegistryUrl}},
			hasLockFile:  false,
			wantErr:      false,
		},
		{
			name:         "file dependencies with Chat.lock file",
			dependencies: []*chart.Dependency{{Name: fileDepChartName, Version: fileDepChartVersion, Repository: "file://../" + fileDepChartName}},
			hasLockFile:  true,
			wantErr:      false,
		},
		{
			name:         "file dependencies without Chat.lock file",
			dependencies: []*chart.Dependency{{Name: fileDepChartName, Version: fileDepChartVersion, Repository: "file://../" + fileDepChartName}},
			hasLockFile:  false,
			wantErr:      false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			func() {
				tempDir, settings := createHelmConfiguration(t)
				defer os.RemoveAll(tempDir)

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

				chartUnderTest, err := helmClient.buildDependencies(filepath.Join(tempDir, chartName))

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

// createHelmConfiguration creates the temporary directory where helm caches and chart will be download and the
// corresponding HelmSettings.It returns the path to the temporary directory and the HelmSettings.
func createHelmConfiguration(t *testing.T) (string, HelmSettings) {
	t.Helper()

	downloadDest, err := os.MkdirTemp("", "helmClientTest-")
	if err != nil {
		t.Fatalf("can not create temp dir where chart will be downloaded: %s", err)
	}
	return downloadDest, NewSettings(downloadDest)
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

// startOciRegistry and upload chart archive located at chartLoc.
func startOciRegistry(t *testing.T, chartLoc string) string {
	t.Helper()

	// Registry config
	config := &configuration.Configuration{}
	port, err := freeport.GetFreePort()
	if err != nil {
		t.Fatalf("error finding free port for test registry")
	}

	config.HTTP.Addr = fmt.Sprintf(":%d", port)
	config.HTTP.DrainTimeout = time.Duration(10) * time.Second
	config.Storage = map[string]configuration.Parameters{"inmemory": map[string]interface{}{}}

	ociRegistryUrl := fmt.Sprintf("oci://localhost:%d/helm-charts", port)

	r, err := ociregistry.NewRegistry(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := r.ListenAndServe(); err != nil {
			t.Errorf("can not start http registry: %s", err)
			return
		}
	}()

	regClient, err := helmregistry.NewClient(helmregistry.ClientOptWriter(os.Stdout))
	if err != nil {
		t.Fatal(err)
	}
	chartUploader := uploader.ChartUploader{
		Out:            os.Stdout,
		Pushers:        pusher.All(&cli.EnvSettings{}),
		RegistryClient: regClient,
	}

	err = chartUploader.UploadTo(chartLoc, ociRegistryUrl)
	if err != nil {
		t.Fatalf("can not push chart to oci registry: %s", err)
	}
	return ociRegistryUrl
}

// packageChart packages the chart in chartDir into a chart archive file (i.e. a tgz) in destDir directory.
func packageChart(t *testing.T, chartDir string, destDir string) (string, int64) {
	ch, err := loader.LoadDir(chartDir)
	if err != nil {
		t.Fatalf("failed to load chart '%s': %s", chartDir, err)
	}

	if reqs := ch.Metadata.Dependencies; reqs != nil {
		if err := action.CheckDependencies(ch, reqs); err != nil {
			t.Fatalf("invalid dependencies for chart '%s': %s", chartDir, err)
		}
	}

	archivePath, err := chartutil.Save(ch, destDir)
	if err != nil {
		t.Fatalf("failed to package chart '%s': %s", chartDir, err)
	}

	expectedChartInfo, err := os.Stat(archivePath)
	if err != nil {
		t.Fatalf("can get size chart archive %s", err)
	}

	return archivePath, expectedChartInfo.Size()
}
