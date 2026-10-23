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

package helmclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/release"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
)

func TestShouldUpgradeUsesReleaseNameForLookup(t *testing.T) {
	const namespace = "app-system"
	const releaseName = "kube-system-example"
	const template = `{{- $secret := lookup "v1" "Secret" .Release.Namespace (printf "%s-token" .Release.Name) -}}
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}
data:
  token: {{ if $secret }}{{ index $secret.data "token" | quote }}{{ else }}{{ randAlphaNum 32 | quote }}{{ end }}
  message: {{ .Values.message | quote }}
`
	for _, tc := range []struct {
		name        string
		values      map[string]interface{}
		template    string
		wantUpgrade bool
	}{
		{name: "unchanged release", template: template},
		{name: "changed values", template: template, values: map[string]interface{}{"message": "updated"}, wantUpgrade: true},
		{name: "changed template", template: template + "  extra: changed\n", wantUpgrade: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var lookups atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				var response interface{}
				switch r.URL.Path {
				case "/api/v1":
					response = &metav1.APIResourceList{
						GroupVersion: "v1",
						APIResources: []metav1.APIResource{{Name: "secrets", Kind: "Secret", Namespaced: true}},
					}
				case "/api/v1/namespaces/" + namespace + "/secrets/" + releaseName + "-token":
					lookups.Add(1)
					response = &corev1.Secret{
						TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
						ObjectMeta: metav1.ObjectMeta{Name: releaseName + "-token", Namespace: namespace},
						Data:       map[string][]byte{"token": []byte("existing-token")},
					}
				default:
					w.WriteHeader(http.StatusNotFound)
					response = &metav1.Status{
						TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"},
						Status:   metav1.StatusFailure, Reason: metav1.StatusReasonNotFound, Code: http.StatusNotFound,
					}
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Errorf("encode API response: %v", err)
				}
			}))
			defer server.Close()

			factory := cmdtesting.NewTestFactory().WithNamespace(namespace)
			defer factory.Cleanup()
			factory.ClientConfigVal = &rest.Config{Host: server.URL}
			h := HelmClient{restClientGetter: factory, targetNamespace: namespace}
			newChart := func(body string) *chart.Chart {
				return &chart.Chart{
					Metadata:  &chart.Metadata{APIVersion: "v2", Name: "example", Version: "1.0.0", AppVersion: "1.0.0"},
					Values:    map[string]interface{}{"message": "original"},
					Raw:       []*chart.File{{Name: "values.yaml", Data: []byte("message: original\n")}},
					Templates: []*chart.File{{Name: "templates/configmap.yaml", Data: []byte(body)}},
				}
			}
			dir := t.TempDir()
			if err := chartutil.SaveDir(newChart(tc.template), dir); err != nil {
				t.Fatal(err)
			}
			current := &release.Release{Name: releaseName, Namespace: namespace, Chart: newChart(template)}
			upgrade, err := h.shouldUpgrade(filepath.Join(dir, "example"), current, tc.values)
			if err != nil {
				t.Fatal(err)
			}
			if upgrade != tc.wantUpgrade {
				t.Errorf("shouldUpgrade() = %v, want %v", upgrade, tc.wantUpgrade)
			}
			if got := lookups.Load(); got != 2 {
				t.Errorf("expected both charts to look up the installed release's Secret; got %d lookups", got)
			}
		})
	}
}
