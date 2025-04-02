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
	"time"

	"github.com/go-test/deep"
	"go.uber.org/zap"
	jsonpatch "gomodules.xyz/jsonpatch/v2"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/fake"

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

const (
	defaultAppName    = "app-def-1"
	defaultAppVersion = "1.2.3"
)

func init() {
	_ = appskubermaticv1.AddToScheme(testScheme)
}

func getApplicationDefinition(name string) *appskubermaticv1.ApplicationDefinition {
	return &appskubermaticv1.ApplicationDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appskubermaticv1.ApplicationDefinitionSpec{
			Description: "Description",
			Versions: []appskubermaticv1.ApplicationVersion{
				{
					Version: defaultAppVersion,
				},
			},
			DefaultNamespace: &appskubermaticv1.AppNamespaceSpec{
				Name:   "default",
				Create: true,
			},
		},
	}
}

func TestHandle(t *testing.T) {
	ad := getApplicationDefinition(defaultAppName)
	fakeClient := fake.
		NewClientBuilder().
		WithObjects(ad).
		Build()

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
						Raw: toByteWithSerOpts(
							func() *appskubermaticv1.ApplicationInstallation {
								ai := commonAppInstall()
								ai.Spec.DeployOptions = &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true}}
								ai.Spec.Values = runtime.RawExtension{Raw: []byte(`{"not-empty":"value"}`)}
								return ai
							}(),
						),
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
						Raw: toByteWithSerOpts(
							func() *appskubermaticv1.ApplicationInstallation {
								ai := commonAppInstall()
								ai.Spec.DeployOptions = &appskubermaticv1.DeployOptions{}
								ai.Spec.Values = runtime.RawExtension{Raw: []byte(`{"not-empty":"value"}`)}
								return ai
							}(),
						),
					},
					Object: runtime.RawExtension{
						Raw: toByteWithSerOpts(
							func() *appskubermaticv1.ApplicationInstallation {
								ai := commonAppInstall()
								ai.Spec.DeployOptions = &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: false, Wait: true}}
								ai.Spec.Values = runtime.RawExtension{Raw: []byte(`{"not-empty":"value"}`)}
								return ai
							}(),
						),
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
					Operation: admissionv1.Delete,
					RequestKind: &metav1.GroupVersionKind{
						Group:   appskubermaticv1.GroupName,
						Version: appskubermaticv1.GroupVersion,
						Kind:    "ApplicationInstallation",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: toByteWithSerOpts(
							func() *appskubermaticv1.ApplicationInstallation {
								ai := commonAppInstall()
								ai.Spec.DeployOptions = &appskubermaticv1.DeployOptions{}
								ai.Spec.Values = runtime.RawExtension{Raw: []byte(`{"not-empty":"value"}`)}
								return ai
							}(),
						),
					},
				},
			},
			wantPatches: []jsonpatch.Operation{},
		},
		{
			name: "Create ApplicationInstallation without values --> values should be defaulted",
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
						Raw: toByteWithSerOpts(
							func() *appskubermaticv1.ApplicationInstallation {
								ai := commonAppInstall()
								ai.Spec.DeployOptions = &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Timeout: metav1.Duration{Duration: 5 * time.Minute}}}
								ai.Spec.Values = runtime.RawExtension{}
								return ai
							}(),
						),
					},
				},
			},
			wantPatches: []jsonpatch.Operation{
				jsonpatch.NewOperation("replace", "/spec/values", map[string]interface{}{}),
			},
		},
		{
			name: "Create ApplicationInstallation with values --> values remain unchanged",
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
						Raw: toByteWithSerOpts(
							func() *appskubermaticv1.ApplicationInstallation {
								ai := commonAppInstall()
								ai.Spec.DeployOptions = &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Timeout: metav1.Duration{Duration: 5 * time.Minute}}}
								ai.Spec.Values = runtime.RawExtension{Raw: []byte(`{"not-empty":"value"}`)}
								return ai
							}(),
						),
					},
				},
			},
			wantPatches: []jsonpatch.Operation{},
		},
		{
			name: "Create ApplicationInstallation without app namespace configured --> default application namespace should be used",
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
						Raw: toByteWithSerOpts(
							func() *appskubermaticv1.ApplicationInstallation {
								ai := commonAppInstall()
								ai.Spec.Namespace = nil
								ai.Spec.DeployOptions = &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Timeout: metav1.Duration{Duration: 5 * time.Minute}}}
								ai.Spec.Values = runtime.RawExtension{Raw: []byte(`{"not-empty":"value"}`)}
								return ai
							}(),
						),
					},
				},
			},
			wantPatches: []jsonpatch.Operation{
				jsonpatch.NewOperation("add", "/spec/namespace", map[string]interface{}{"name": "default", "create": true}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := AdmissionHandler{
				log:     zap.NewNop().Sugar(),
				decoder: admission.NewDecoder(testScheme),
				client:  fakeClient,
			}
			res := handler.Handle(context.Background(), tt.req)
			if res.Result != nil && (res.Result.Code == http.StatusInternalServerError || res.Result.Code == http.StatusBadRequest) {
				t.Fatalf("Request failed: %v", res.Result.Message)
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

func commonAppInstall() *appskubermaticv1.ApplicationInstallation {
	return &appskubermaticv1.ApplicationInstallation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appskubermaticv1.GroupName + "/" + appskubermaticv1.GroupVersion,
			Kind:       "ApplicationInstallation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-1",
			Namespace: "default",
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			Namespace: &appskubermaticv1.AppNamespaceSpec{
				Name: "default",
			},
			ApplicationRef: appskubermaticv1.ApplicationRef{
				Name:    "app-def-1",
				Version: "v1.0.0",
			},
		},
	}
}

func toByteWithSerOpts(o runtime.Object) []byte {
	s := json.NewSerializerWithOptions(json.DefaultMetaFactory, testScheme, testScheme, json.SerializerOptions{Pretty: true})
	buff := bytes.NewBuffer([]byte{})
	_ = s.Encode(o, buff)

	return buff.Bytes()
}
