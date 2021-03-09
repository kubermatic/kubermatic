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

package validation

import (
	"bytes"
	"context"
	"testing"
	"text/template"

	logrtesting "github.com/go-logr/logr/testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
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
		client      ctrlruntimeclient.Client
	}{
		{
			name: "Delete cluster success",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Delete,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "cluster",
				},
			},
			wantAllowed: true,
		},
		{
			name: "Create cluster with Tunneling expose strategy succeeds when the FeatureGate is enabled",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
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
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
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
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
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
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", Namespace: "kubermatic", ExposeStrategy: "ciao"}.Do(),
					},
				},
			},
			wantAllowed: false,
		},
		{
			name: "Reject EnableUserSSHKey agent update",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", Namespace: "kubermatic", ExposeStrategy: "NodePort", EnableUserSSHKey: true}.Do(),
					},
				},
			},
			wantAllowed: false,
			client: ctrlruntimefakeclient.NewClientBuilder().WithObjects(
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "foo",
					},
					Spec: kubermaticv1.ClusterSpec{
						EnableUserSSHKeyAgent: true,
					},
				},
			).Build(),
		},
		{
			name: "Accept EnableUserSSHKey agent update when the value did not change",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", Namespace: "kubermatic", ExposeStrategy: "NodePort", EnableUserSSHKey: false}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", Namespace: "kubermatic", ExposeStrategy: "NodePort", EnableUserSSHKey: false}.Do(),
					},
				},
			},
			wantAllowed: true,
			client: ctrlruntimefakeclient.NewClientBuilder().WithObjects(
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "foo",
					},
					Spec: kubermaticv1.ClusterSpec{
						EnableUserSSHKeyAgent: false,
					},
				},
			).Build(),
		},
		{
			name: "Accept a cluster create request with externalCloudProvider disabled",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", Namespace: "kubermatic", ExposeStrategy: "NodePort", ExternalCloudProvider: false}.Do(),
					},
				},
			},
			wantAllowed: true,
			client: ctrlruntimefakeclient.NewClientBuilder().WithObjects(
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "foo",
					},
					Spec: kubermaticv1.ClusterSpec{
						Features: map[string]bool{
							kubermaticv1.ClusterFeatureExternalCloudProvider: false,
						},
					},
				},
			).Build(),
		},
		{
			name: "Accept a cluster create request with externalCloudProvider enabled",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", Namespace: "kubermatic", ExposeStrategy: "NodePort", ExternalCloudProvider: true}.Do(),
					},
				},
			},
			wantAllowed: true,
			client: ctrlruntimefakeclient.NewClientBuilder().WithObjects(
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "foo",
					},
					Spec: kubermaticv1.ClusterSpec{
						Features: map[string]bool{
							kubermaticv1.ClusterFeatureExternalCloudProvider: true,
						},
					},
				},
			).Build(),
		},
		{
			name: "Accept enabling the externalCloudProvider feature",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", Namespace: "kubermatic", ExposeStrategy: "NodePort", ExternalCloudProvider: true}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", Namespace: "kubermatic", ExposeStrategy: "NodePort", ExternalCloudProvider: false}.Do(),
					},
				},
			},
			wantAllowed: true,
			client: ctrlruntimefakeclient.NewClientBuilder().WithObjects(
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "foo",
					},
					Spec: kubermaticv1.ClusterSpec{
						Features: map[string]bool{
							kubermaticv1.ClusterFeatureExternalCloudProvider: true,
						},
					},
				},
			).Build(),
		},
		{
			name: "Reject disabling the externalCloudProvider feature",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", Namespace: "kubermatic", ExposeStrategy: "NodePort", ExternalCloudProvider: false}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", Namespace: "kubermatic", ExposeStrategy: "NodePort", ExternalCloudProvider: true}.Do(),
					},
				},
			},
			wantAllowed: false,
			client: ctrlruntimefakeclient.NewClientBuilder().WithObjects(
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "foo",
					},
					Spec: kubermaticv1.ClusterSpec{
						Features: map[string]bool{
							kubermaticv1.ClusterFeatureExternalCloudProvider: true,
						},
					},
				},
			).Build(),
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
			client:   tt.client,
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
	Name                  string
	Namespace             string
	ExposeStrategy        string
	EnableUserSSHKey      bool
	ExternalCloudProvider bool
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
	"exposeStrategy": "{{ .ExposeStrategy }}",
	"enableUserSSHKey": {{ .EnableUserSSHKey }},
	"features": {
		"externalCloudProvider": {{ .ExternalCloudProvider }}
	}
  }
}`)
	sb := bytes.Buffer{}
	_ = tmpl.Execute(&sb, r)
	return sb.Bytes()
}
