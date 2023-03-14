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

package test

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/registry"
	"github.com/phayes/freeport"
	"golang.org/x/crypto/bcrypt"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/pusher"
	registry2 "helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/repo/repotest"
	"helm.sh/helm/v3/pkg/uploader"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
)

// Const relative to pkg/applications/helmclient/testdata/examplechart chart.
const (
	// ConfigmapName is the Name of the config map deployed by the examplechart and examplechart-v2.
	ConfigmapName = "testcm"

	// ConfigmapName2 is the Name of the config map deployed by the examplechart and examplechart2.
	ConfigmapName2 = "testcm-2"

	// CmDataKey is the key in the values.yaml of examplechart / examplechart-v2 that holds custom configmap data.
	CmDataKey = "cmData"

	// VersionLabelKey is the key in the values.yaml of examplechart / examplechart-v2 that holds custom version label value. it's also the name of the label in the configmap.
	VersionLabelKey = "versionLabel"

	// DefaultVerionLabel is the default value of the version label the configmap deployed by pkg/applications/helmclient/testdata/examplechart chart.
	DefaultVerionLabel = "1.0"

	// DefaultVerionLabelV2 is the default value of the version label the configmap deployed by pkg/applications/helmclient/testdata/examplechart-v2 chart.
	DefaultVerionLabelV2 = "2.0"

	// EnableDNSLabelKey is the label key on cmData examplechart that allow test the enableDns deployOption.
	EnableDNSLabelKey = "enableDNSLabel"

	// SvcName is the name of the service deployed by chart chart-with-lb.
	SvcName = "svc-1"

	// DeploySvcKey is the key values.yaml of examplechart-v2 to enable / disabled the deployment of the service.
	DeploySvcKey = "deployService"
)

var (
	// DefaultData contains the default data of the configmap deployed by pkg/applications/helmclient/testdata/examplechart chart.
	DefaultData = map[string]string{"foo": "bar"}

	// DefaultDataV2 contains the default data of the configmap deployed by pkg/applications/helmclient/testdata/examplechart-v2 chart.
	DefaultDataV2 = map[string]string{"foo-version-2": "bar-version-2"}
)

// PackageChart packages the chart in chartDir into a chart archive file (i.e. a tgz) in destDir directory and returns
// the full path and the size of the archive.
func PackageChart(t *testing.T, chartDir string, destDir string) (string, int64) {
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

// StartHttpRegistryWithCleanup starts a Helm http registry and uploads charts archives matching glob and returns the registry URL.
func StartHttpRegistryWithCleanup(t *testing.T, glob string) string {
	srv, err := repotest.NewTempServerWithCleanup(t, glob)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		srv.Stop()
	})
	if err := srv.CreateIndex(); err != nil {
		t.Fatal(err)
	}
	return srv.URL()
}

// StartHttpRegistryWithAuthAndCleanup starts a Helm http registry with basic auth enabled and uploads charts archives matching glob and returns the registry URL.
// the basic auth is hardcoded to Username: "username", Password: "password".
func StartHttpRegistryWithAuthAndCleanup(t *testing.T, glob string) string {
	srvWithAuth := repotest.NewTempServerWithCleanupAndBasicAuth(t, glob)
	t.Cleanup(func() {
		srvWithAuth.Stop()
	})
	if err := srvWithAuth.CreateIndex(); err != nil {
		t.Fatal(err)
	}
	return srvWithAuth.URL()
}

// StartOciRegistry start an oci registry and uploads charts archives matching glob and returns the registry URL.
func StartOciRegistry(t *testing.T, glob string) string {
	registryURL, _ := newOciRegistry(t, glob, false)
	return registryURL
}

// StartOciRegistryWithAuth start an oci registry with authentication, uploads charts archives matching glob,
// returns the registry URL and registryConfigFile.
// registryConfigFile contains the credentials of the registry.
func StartOciRegistryWithAuth(t *testing.T, glob string) (string, string) {
	return newOciRegistry(t, glob, true)
}

// newOciRegistry starts an oci registry, uploads charts archives matching glob, returns the registry URL and
// registryConfigFile if authentication is enabled.
func newOciRegistry(t *testing.T, glob string, enableAuth bool) (string, string) {
	t.Helper()

	// Registry config
	config := &configuration.Configuration{}
	credentialDir := t.TempDir()

	var username, password, registryConfigFile string

	if enableAuth {
		username = "someuser"
		password = "somepassword"

		encrypedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			t.Fatalf("failed to generate encrypt password: %s", err)
		}
		authHtpasswd := filepath.Join(credentialDir, "auth.htpasswd")
		err = os.WriteFile(authHtpasswd, []byte(fmt.Sprintf("%s:%s\n", username, string(encrypedPassword))), 0600)
		if err != nil {
			t.Fatalf("failed to write auth.htpasswd file: %s", err)
		}

		config.Auth = configuration.Auth{
			"htpasswd": configuration.Parameters{
				"realm": "localhost",
				"path":  authHtpasswd,
			},
		}
	}

	port, err := freeport.GetFreePort()
	if err != nil {
		t.Fatalf("error finding free port for test registry")
	}

	config.HTTP.Addr = fmt.Sprintf(":%d", port)
	config.HTTP.DrainTimeout = time.Duration(10) * time.Second
	config.Storage = map[string]configuration.Parameters{"inmemory": map[string]interface{}{}}

	ociRegistryUrl := fmt.Sprintf("oci://localhost:%d/helm-charts", port)

	r, err := registry.NewRegistry(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := r.ListenAndServe(); err != nil {
			t.Errorf("can not start OCI registry: %s", err)
			return
		}
	}()

	// Ensure registry is listening
	var lastError error
	if !utils.WaitFor(500*time.Millisecond, 5*time.Second, func() bool {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
		defer cancel()
		request, err := http.NewRequestWithContext(ctx, http.MethodHead, strings.Replace(ociRegistryUrl, "oci://", "http://", 1), nil)
		if err != nil {
			lastError = fmt.Errorf("failed to created request to test oci registry is up: %w", err)
			return false
		}

		resp, err := http.DefaultClient.Do(request)
		lastError = err
		defer func() {
			if resp != nil {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Failed to close response body from testing oci registry :%s", err)
				}
			}
		}()

		return lastError == nil
	}) {
		t.Fatalf("failed to check if oci registry is up: %s", lastError)
	}

	if glob != "" {
		options := []registry2.ClientOption{registry2.ClientOptWriter(os.Stdout)}
		if enableAuth {
			registryConfigFile = filepath.Join(credentialDir, "reg-cred")
			// to generate auth field :  echo '<user>:<password>' | base64
			auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
			if err := os.WriteFile(registryConfigFile, []byte(fmt.Sprintf(`{"auths":{"localhost:%d":{"username":"%s","password":"%s","auth":"%s"}}}`, port, username, password, auth)), 0600); err != nil {
				t.Fatal(err)
			}
			options = append(options, registry2.ClientOptCredentialsFile(registryConfigFile))
		}
		regClient, err := registry2.NewClient(options...)
		if err != nil {
			t.Fatal(err)
		}

		chartUploader := uploader.ChartUploader{
			Out:     os.Stdout,
			Pushers: pusher.All(&cli.EnvSettings{}),
			Options: []pusher.Option{pusher.WithRegistryClient(regClient)},
		}

		files, err := filepath.Glob(glob)
		if err != nil {
			t.Fatalf("failed to upload chart, invalid blob: %s", err)
		}
		for i := range files {
			err = chartUploader.UploadTo(files[i], ociRegistryUrl)
			if err != nil {
				t.Fatalf("can not push chart '%s' to oci registry: %s", files[i], err)
			}
		}
	}

	return ociRegistryUrl, registryConfigFile
}

// CopyDir coypy source dir to destination dir.
func CopyDir(source string, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		// error may occurred if path is not accessible
		if err != nil {
			return err
		}
		relPath := strings.Replace(path, source, "", 1)
		if info.IsDir() {
			return os.Mkdir(filepath.Join(destination, relPath), 0755)
		} else {
			data, err := os.ReadFile(filepath.Join(source, relPath))
			if err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(destination, relPath), data, 0755)
		}
	})
}
