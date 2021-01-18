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

package cluster

import (
	"bytes"
	"context"
	"testing"
	"text/template"

	logrtesting "github.com/go-logr/logr/testing"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"
)

var (
	testScheme = runtime.NewScheme()
)

func init() {
	_ = kubermaticv1.AddToScheme(testScheme)
}

func TestHandle(t *testing.T) {
	tests := []struct {
		name        string
		req         webhook.AdmissionRequest
		wantAllowed bool
		features    features.FeatureGate
	}{
		{
			name: "delete cluster succeess",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1beta1.AdmissionRequest{
					Operation: admissionv1beta1.Delete,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name:      "cluster",
					Namespace: "kubermatic",
				},
			},
			wantAllowed: true,
		},
		{
			name: "Create cluster with Tunneling expose strategy succeeds when the FeatureGate is enabled",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1beta1.AdmissionRequest{
					Operation: admissionv1beta1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name:      "foo",
					Namespace: "kubermatic",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", Namespace: "kubermatic", ExposeStrategy: "Tunneling"}.Do(),
					},
				},
			},
			wantAllowed: true,
			features:    features.FeatureGate{features.TunnelingExposeStrategy: true},
		},
		{
			name: "Create cluster with Tunneling expose strategy fails when the FeatureGate is not enabled",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1beta1.AdmissionRequest{
					Operation: admissionv1beta1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name:      "foo",
					Namespace: "kubermatic",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", Namespace: "kubermatic", ExposeStrategy: "Tunneling"}.Do(),
					},
				},
			},
			wantAllowed: false,
			features:    features.FeatureGate{features.TunnelingExposeStrategy: false},
		},
		{
			name: "Create cluster expose strategy different from Tunneling should succeed",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1beta1.AdmissionRequest{
					Operation: admissionv1beta1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name:      "foo",
					Namespace: "kubermatic",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", Namespace: "kubermatic", ExposeStrategy: "NodePort"}.Do(),
					},
				},
			},
			wantAllowed: true,
		},
		{
			name: "Unknown expose strategy",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1beta1.AdmissionRequest{
					Operation: admissionv1beta1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name:      "foo",
					Namespace: "kubermatic",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", Namespace: "kubermatic", ExposeStrategy: "ciao"}.Do(),
					},
				},
			},
			wantAllowed: false,
		},
	}
	for _, tt := range tests {
		d, err := admission.NewDecoder(testScheme)
		if err != nil {
			t.Fatalf("error occurred while creating decoder: %v", err)
		}
		handler := AdmissionHandler{
			log:      &logrtesting.NullLogger{},
			decoder:  d,
			features: tt.features,
		}
		t.Run(tt.name, func(t *testing.T) {
			if res := handler.Handle(context.TODO(), tt.req); res.Allowed != tt.wantAllowed {
				t.Errorf("Allowed %t, but wanted %t", res.Allowed, tt.wantAllowed)
				t.Logf("Response: %v", res)
			}
		})
	}
}

type rawClusterGen struct {
	Name           string
	Namespace      string
	ExposeStrategy string
}

func (r rawClusterGen) Do() []byte {
	tmpl, _ := template.New("cluster").Parse(`{
  "apiVersion": "kubermatic.k8s.io/v1",
  "kind": "Cluster",
  "metadata": {
	"name": "{{ .Name }}",
	"namespace": "{{ .Namespace}}"
  },
  "spec": {
	"exposeStrategy": "{{ .ExposeStrategy }}"
  }
}`)
	sb := bytes.Buffer{}
	_ = tmpl.Execute(&sb, r)
	return sb.Bytes()
}
