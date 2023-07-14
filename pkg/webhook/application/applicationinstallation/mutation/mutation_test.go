/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package mutation

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/go-test/deep"
	"go.uber.org/zap"
	jsonpatch "gomodules.xyz/jsonpatch/v2"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	testScheme = runtime.NewScheme()
)

func init() {
	_ = appskubermaticv1.AddToScheme(testScheme)
}

func TestHandle(t *testing.T) {
	tests := []struct {
		name        string
		req         webhook.AdmissionRequest
		wantPatches []jsonpatch.JsonPatchOperation
	}{
		{
			name: "Create applicationInstallation with DeployOps.Helm.Atomic=true --> DeployOps.Helm.Wait and DeployOps.Helm.timeout should be defaulted",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   appskubermaticv1.GroupName,
						Version: appskubermaticv1.GroupVersion,
						Kind:    "ApplicationInstallation",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawAppInstallGen{
							DeployOps: &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true}},
						}.Do(),
					},
				},
			},
			wantPatches: []jsonpatch.Operation{
				jsonpatch.NewOperation("add", "/spec/deployOptions/helm/wait", true),
				jsonpatch.NewOperation("replace", "/spec/deployOptions/helm/timeout", "5m0s"),
			},
		},
		{
			name: "Update applicationInstallation with DeployOps.Helm.Wait=true-->  DeployOps.Helm.timeout should be defaulted",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   appskubermaticv1.GroupName,
						Version: appskubermaticv1.GroupVersion,
						Kind:    "ApplicationInstallation",
					},
					Name: "foo",
					OldObject: runtime.RawExtension{
						Raw: rawAppInstallGen{
							DeployOps: &appskubermaticv1.DeployOptions{},
						}.Do(),
					},
					Object: runtime.RawExtension{
						Raw: rawAppInstallGen{
							DeployOps: &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: false, Wait: true}},
						}.Do(),
					},
				},
			},
			wantPatches: []jsonpatch.Operation{
				jsonpatch.NewOperation("replace", "/spec/deployOptions/helm/timeout", "5m0s"),
			},
		},
		{
			name: "Delete applicationInstallation should not generate path",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   appskubermaticv1.GroupName,
						Version: appskubermaticv1.GroupVersion,
						Kind:    "ApplicationInstallation",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawAppInstallGen{
							DeployOps: &appskubermaticv1.DeployOptions{},
						}.Do(),
					},
				},
			},
			wantPatches: []jsonpatch.Operation{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := AdmissionHandler{
				log:     zap.NewNop().Sugar(),
				decoder: admission.NewDecoder(testScheme),
			}
			res := handler.Handle(context.Background(), tt.req)
			if res.AdmissionResponse.Result != nil && (res.AdmissionResponse.Result.Code == http.StatusInternalServerError || res.AdmissionResponse.Result.Code == http.StatusBadRequest) {
				t.Fatalf("Request failed: %v", res.AdmissionResponse.Result.Message)
			}

			a := map[string]jsonpatch.JsonPatchOperation{}
			for _, p := range res.Patches {
				a[p.Path] = p
			}
			w := map[string]jsonpatch.JsonPatchOperation{}
			for _, p := range tt.wantPatches {
				w[p.Path] = p
			}
			if diff := deep.Equal(a, w); len(diff) > 0 {
				t.Errorf("Diff found between wanted and actual patches: %+v", diff)
			}
		})
	}
}

type rawAppInstallGen struct {
	DeployOps *appskubermaticv1.DeployOptions
}

func (r rawAppInstallGen) Do() []byte {
	key := appskubermaticv1.ApplicationInstallation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appskubermaticv1.GroupName + "/" + appskubermaticv1.GroupVersion,
			Kind:       "ApplicationInstallation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-1",
			Namespace: "default",
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			Namespace: appskubermaticv1.AppNamespaceSpec{
				Name: "default",
			},
			ApplicationRef: appskubermaticv1.ApplicationRef{
				Name:    "app-def-1",
				Version: "v1.0.0",
			},
			DeployOptions: r.DeployOps,
		},
	}

	s := json.NewSerializerWithOptions(json.DefaultMetaFactory, testScheme, testScheme, json.SerializerOptions{Pretty: true})
	buff := bytes.NewBuffer([]byte{})
	_ = s.Encode(&key, buff)

	return buff.Bytes()
}
