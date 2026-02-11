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
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/containerd/containerd/v2/core/remotes/docker"
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

	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
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

	// DefaultVersionLabel is the default value of the version label the configmap deployed by pkg/applications/helmclient/testdata/examplechart chart.
	DefaultVersionLabel = "1.0"

	// DefaultVersionLabelV2 is the default value of the version label the configmap deployed by pkg/applications/helmclient/testdata/examplechart-v2 chart.
	DefaultVersionLabelV2 = "2.0"

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

	// registryHostname must be configured using the -registry-hostname flag to anything *but*
	// "localhost" to enable registry tests. "localhost" is not a realistic value in production
	// (loading Helm charts from localhost in a controller-manager Pod is meaningless) and causes
	// special behaviour in Helm/containerd to enforce plaintext HTTP, unless many, many hoops are
	// jumped through.
	registryHostname string
)

func init() {
	flag.StringVar(&registryHostname, "registry-hostname", "", "a host alias for localhost")
}

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

// StartHTTPRegistryWithCleanup starts a Helm http registry and uploads charts archives matching
// glob and returns the registry URL.
func StartHTTPRegistryWithCleanup(t *testing.T, glob string) string {
	u, _ := startRegistryWithCleanup(t, glob, false)
	return u
}

// StartHTTPSRegistryWithCleanup is the same as StartHttpRegistryWithCleanup, but starts a TLS
// server instead. Any package calling this must provide a "testdata" directory with the Helm
// test PKI.
func StartHTTPSRegistryWithCleanup(t *testing.T, glob string) (string, *PKI) {
	return startRegistryWithCleanup(t, glob, true)
}

func startRegistryWithCleanup(t *testing.T, glob string, secure bool) (string, *PKI) {
	srv, err := repotest.NewTempServerWithCleanup(t, glob)
	if err != nil {
		t.Fatal(err)
	}

	var pki *PKI
	if secure {
		pki = switchServerToTLS(t, srv)
	}

	t.Cleanup(func() {
		srv.Stop()
	})
	if err := srv.CreateIndex(); err != nil {
		t.Fatal(err)
	}
	return srv.URL(), pki
}

// StartHTTPRegistryWithAuthAndCleanup starts a Helm http registry with basic auth enabled and
// uploads charts archives matching glob and returns the registry URL.
// the basic auth is hardcoded to Username: "username", Password: "password".
func StartHTTPRegistryWithAuthAndCleanup(t *testing.T, glob string) string {
	u, _ := startRegistryWithAuthAndCleanup(t, glob, false)
	return u
}

// StartHTTPSRegistryWithAuthAndCleanup is the same as StartHttpRegistryWithAuthAndCleanup, but
// starts a TLS server instead. Any package calling this must provide a "testdata" directory with
// the Helm test PKI.
func StartHTTPSRegistryWithAuthAndCleanup(t *testing.T, glob string) (string, *PKI) {
	return startRegistryWithAuthAndCleanup(t, glob, true)
}

func startRegistryWithAuthAndCleanup(t *testing.T, glob string, secure bool) (string, *PKI) {
	srvWithAuth := repotest.NewTempServerWithCleanupAndBasicAuth(t, glob)

	var pki *PKI
	if secure {
		pki = switchServerToTLS(t, srvWithAuth)
	}

	t.Cleanup(func() {
		srvWithAuth.Stop()
	})
	if err := srvWithAuth.CreateIndex(); err != nil {
		t.Fatal(err)
	}
	return srvWithAuth.URL(), pki
}

func switchServerToTLS(t *testing.T, srv *repotest.Server) *PKI {
	// turn plaintext into secure server; Helm always expects the keypair in "../../testdata/",
	// so in order not to put a random "testdata" directory in our codebase, we temporarily switch
	// the working dir to a dummy, so the path resolves to ./testdata.
	srv.Stop()

	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	if err := os.Chdir("testdata/dummy"); err != nil {
		t.Fatalf("To use this function, you need to have copied the Helm testdata PKI into your package: %v", err)
	}

	testdata, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("Failed to determine testdata directory: %v", err)
	}

	srv.StartTLS()

	if err := os.Chdir(pwd); err != nil {
		t.Fatalf("Failed to change working directory back: %v", err)
	}

	return &PKI{
		CAFile:          filepath.Join(testdata, "rootca.crt"),
		CertificateFile: filepath.Join(testdata, "crt.pem"),
		KeyFile:         filepath.Join(testdata, "key.pem"),
	}
}

// StartOciRegistry start an oci registry and uploads charts archives matching glob
// and returns the registry URL.
func StartOciRegistry(t *testing.T, glob string) string {
	registryURL, _, _ := newOciRegistry(t, glob, false, false)
	return registryURL
}

// StartSecureOciRegistry start an oci registry and uploads charts archives matching glob
// and returns the registry URL and the generated PKI.
func StartSecureOciRegistry(t *testing.T, glob string) (string, *PKI) {
	registryURL, _, pki := newOciRegistry(t, glob, false, true)
	return registryURL, pki
}

// StartOciRegistryWithAuth start an oci registry with authentication, uploads charts archives matching glob,
// returns the registry URL and registryConfigFile.
// registryConfigFile contains the credentials of the registry.
func StartOciRegistryWithAuth(t *testing.T, glob string) (string, string) {
	ociRegistryURL, registryConfigFile, _ := newOciRegistry(t, glob, true, false)

	return ociRegistryURL, registryConfigFile
}

// StartOciRegistryWithAuth start an oci registry with authentication, uploads charts archives matching glob,
// returns the registry URL and registryConfigFile.
// registryConfigFile contains the credentials of the registry.
func StartSecureOciRegistryWithAuth(t *testing.T, glob string) (string, string, *PKI) {
	return newOciRegistry(t, glob, true, true)
}

type PKI struct {
	CAFile          string
	CertificateFile string
	KeyFile         string
}

// newOciRegistry starts an oci registry, uploads charts archives matching glob, returns the registry URL and
// registryConfigFile if authentication is enabled.
func newOciRegistry(t *testing.T, glob string, enableAuth bool, secure bool) (string, string, *PKI) {
	t.Helper()

	if registryHostname == "" {
		t.Fatal("Must set -registry-hostname to an alias for localhost")
	}

	isLocalhost, err := docker.MatchLocalhost(registryHostname)
	if err != nil {
		t.Fatalf("Failed to test -registry-hostname to be a loopback address: %v", err)
	}

	if isLocalhost {
		t.Fatal("-registry-hostname must not be a loopback address, but an alias")
	}

	// Registry config
	config := &configuration.Configuration{}
	credentialDir := t.TempDir()

	var username, password, registryConfigFile string

	if enableAuth {
		username = "someuser"
		password = "somepassword"

		encryptedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			t.Fatalf("failed to generate encrypt password: %s", err)
		}
		authHtpasswd := filepath.Join(credentialDir, "auth.htpasswd")
		err = os.WriteFile(authHtpasswd, []byte(fmt.Sprintf("%s:%s\n", username, string(encryptedPassword))), 0600)
		if err != nil {
			t.Fatalf("failed to write auth.htpasswd file: %s", err)
		}

		config.Auth = configuration.Auth{
			"htpasswd": configuration.Parameters{
				"realm": registryHostname,
				"path":  authHtpasswd,
			},
		}
	}

	var (
		pki    *PKI
		caCert []byte
	)

	if secure {
		pki = setupPKI(t, credentialDir)
		caCert, _ = os.ReadFile(pki.CAFile)

		config.HTTP.TLS.Key = pki.KeyFile
		config.HTTP.TLS.Certificate = pki.CertificateFile
	}

	port, err := freeport.GetFreePort()
	if err != nil {
		t.Fatalf("error finding free port for test registry")
	}

	config.HTTP.Addr = fmt.Sprintf(":%d", port)
	config.HTTP.DrainTimeout = time.Duration(10) * time.Second
	config.Storage = map[string]configuration.Parameters{"inmemory": map[string]interface{}{}}

	fullHost := net.JoinHostPort(registryHostname, strconv.Itoa(port))
	ociRegistryURL := fmt.Sprintf("oci://%s/helm-charts", fullHost)

	ctx := context.Background()

	r, err := registry.NewRegistry(ctx, config)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := r.ListenAndServe(); err != nil {
			t.Errorf("can not start OCI registry: %s", err)
			return
		}
	}()

	httpClient := http.Client{}
	if secure {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			t.Fatal("could not add CA cert to CA bundle")
		}

		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
			},
		}
	}

	// ensure registry is listening
	waitForRegistry(t, ctx, &httpClient, ociRegistryURL, secure)

	// preload registry
	if glob != "" {
		options := []registry2.ClientOption{registry2.ClientOptWriter(os.Stdout)}
		if enableAuth {
			registryConfigFile = filepath.Join(credentialDir, "reg-cred")
			// to generate auth field:
			// echo '<user>:<password>' | base64
			auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
			if err := os.WriteFile(registryConfigFile, []byte(fmt.Sprintf(`{"auths":{"%s:%d":{"username":"%s","password":"%s","auth":"%s"}}}`, registryHostname, port, username, password, auth)), 0600); err != nil {
				t.Fatal(err)
			}
			options = append(options, registry2.ClientOptCredentialsFile(registryConfigFile))
		}

		if secure {
			options = append(options, registry2.ClientOptHTTPClient(&httpClient))
		} else {
			options = append(options, registry2.ClientOptPlainHTTP())
		}

		regClient, err := registry2.NewClient(options...)
		if err != nil {
			t.Fatal(err)
		}

		chartUploader := uploader.ChartUploader{
			Out:     os.Stdout,
			Pushers: pusher.All(&cli.EnvSettings{}),
			Options: []pusher.Option{
				pusher.WithRegistryClient(regClient),
				pusher.WithPlainHTTP(!secure),
			},
		}

		files, err := filepath.Glob(glob)
		if err != nil {
			t.Fatalf("failed to upload chart, invalid blob: %s", err)
		}
		for i := range files {
			err = chartUploader.UploadTo(files[i], ociRegistryURL)
			if err != nil {
				t.Fatalf("can not push chart '%s' to oci registry: %s", files[i], err)
			}
		}
	}

	return ociRegistryURL, registryConfigFile, pki
}

func waitForRegistry(t *testing.T, ctx context.Context, httpClient *http.Client, url string, secure bool) {
	var lastError error
	if !utils.WaitFor(ctx, 500*time.Millisecond, 5*time.Second, func() bool {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		var protocol string
		if secure {
			protocol = "https://"
		} else {
			protocol = "http://"
		}

		request, err := http.NewRequestWithContext(ctx, http.MethodHead, strings.Replace(url, "oci://", protocol, 1), nil)
		if err != nil {
			lastError = fmt.Errorf("failed to create request to test oci registry is up: %w", err)
			return false
		}

		resp, err := httpClient.Do(request)
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
}

func setupPKI(t *testing.T, credentialDir string) *PKI {
	ca, err := triple.NewCA("test")
	if err != nil {
		t.Fatalf("failed to create dummy CA: %s", err)
	}

	// service name, namespace and DNS domain are irrelevant for these testcases
	keypair, err := triple.NewServerKeyPair(ca, registryHostname, "dummy", "dummyns", "local", nil, []string{registryHostname})
	if err != nil {
		t.Fatalf("failed to create dummy keypair: %s", err)
	}

	caFile := filepath.Join(credentialDir, "ca.crt")
	if err := os.WriteFile(caFile, triple.EncodeCertPEM(ca.Cert), 0644); err != nil {
		t.Fatalf("failed to write dummy CA: %s", err)
	}

	certFile := filepath.Join(credentialDir, "server.crt")
	if err := os.WriteFile(certFile, triple.EncodeCertPEM(keypair.Cert), 0644); err != nil {
		t.Fatalf("failed to write dummy certificate: %s", err)
	}

	keyFile := filepath.Join(credentialDir, "server.key")
	if err := os.WriteFile(keyFile, triple.EncodePrivateKeyPEM(keypair.Key), 0600); err != nil {
		t.Fatalf("failed to write dummy certificate key: %s", err)
	}

	return &PKI{
		CAFile:          caFile,
		CertificateFile: certFile,
		KeyFile:         keyFile,
	}
}

// CopyDir copies source dir to destination dir.
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
