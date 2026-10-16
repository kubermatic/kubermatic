/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package source

import (
	"context"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
)

// TestHelmSourceUsesCABundle verifies that a chart can be downloaded from an HTTPS
// repository whose certificate is only trusted via the configured CA bundle file.
func TestHelmSourceUsesCABundle(t *testing.T) {
	chartPath, err := chartutil.Save(&chart.Chart{
		Metadata: &chart.Metadata{APIVersion: chart.APIVersionV2, Name: "examplechart", Version: "0.1.0"},
	}, t.TempDir())
	if err != nil {
		t.Fatalf("failed to create chart: %v", err)
	}
	chartData, err := os.ReadFile(chartPath)
	if err != nil {
		t.Fatalf("failed to read chart: %v", err)
	}

	var serverURL string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.yaml":
			fmt.Fprintf(w, "apiVersion: v1\nentries:\n  examplechart:\n  - apiVersion: v2\n    name: examplechart\n    version: 0.1.0\n    urls:\n    - %s/examplechart-0.1.0.tgz\n", serverURL)
		case "/examplechart-0.1.0.tgz":
			_, _ = w.Write(chartData)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	serverURL = srv.URL

	caFile := filepath.Join(t.TempDir(), "ca.pem")
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	if err := os.WriteFile(caFile, caPEM, 0600); err != nil {
		t.Fatalf("failed to write CA bundle: %v", err)
	}

	testCases := []struct {
		name         string
		caBundleFile string
		wantErr      bool
	}{
		{name: "without CA bundle the certificate is rejected", wantErr: true},
		{name: "with CA bundle the chart is downloaded", caBundleFile: caFile},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := HelmSource{
				Ctx:          context.Background(),
				CacheDir:     t.TempDir(),
				Log:          zap.NewNop().Sugar(),
				Kubeconfig:   "",
				CABundleFile: tc.caBundleFile,
				Source: &appskubermaticv1.HelmSource{
					URL:          srv.URL,
					ChartName:    "examplechart",
					ChartVersion: "0.1.0",
				},
			}

			_, err := h.DownloadSource(t.TempDir())
			if (err != nil) != tc.wantErr {
				t.Fatalf("DownloadSource() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
