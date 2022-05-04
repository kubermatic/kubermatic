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

package validation

import (
	"bytes"
	"context"
	"testing"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"

	appkubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	defaultAppName    = "app"
	defaultAppVersion = "1.2.3"
)

var (
	testScheme = runtime.NewScheme()
)

func init() {
	_ = appkubermaticv1.AddToScheme(testScheme)
}

func TestValidateApplicationInsallation(t *testing.T) {
	ad := getApplicationDefinition(defaultAppName)
	fakeClient := fakectrlruntimeclient.
		NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(ad).
		Build()

	ai := getApplicationInstallation(defaultAppName, defaultAppName, defaultAppVersion)
	validRaw := applicationInstallationToRawExt(*ai)

	ai.Spec.Namespace.Create = false
	invalidUpdateRaw := applicationInstallationToRawExt(*ai)

	ai.Spec.ApplicationRef.Name = "invalid"
	invalidRaw := applicationInstallationToRawExt(*ai)

	tests := []struct {
		name        string
		req         webhook.AdmissionRequest
		wantAllowed bool
	}{
		{
			name: "Create ApplicationInstallation Success",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   appkubermaticv1.GroupName,
						Version: appkubermaticv1.GroupVersion,
						Kind:    "ApplicationInstallation",
					},
					Name:   "default",
					Object: validRaw,
				},
			},
			wantAllowed: true,
		},
		{
			name: "Create ApplicationInstallation Failure",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   appkubermaticv1.GroupName,
						Version: appkubermaticv1.GroupVersion,
						Kind:    "ApplicationInstallation",
					},
					Name:   "default",
					Object: invalidRaw,
				},
			},
			wantAllowed: false,
		},
		{
			name: "Delete ApplicationInstallation Success",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Delete,
					RequestKind: &metav1.GroupVersionKind{
						Group:   appkubermaticv1.GroupName,
						Version: appkubermaticv1.GroupVersion,
						Kind:    "ApplicationInstallation",
					},
					Name:   "default",
					Object: validRaw,
				},
			},
			wantAllowed: true,
		},
		{
			name: "Update ApplicationInstallation Failure",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   appkubermaticv1.GroupName,
						Version: appkubermaticv1.GroupVersion,
						Kind:    "ApplicationInstallation",
					},
					Name:      "default",
					Object:    validRaw,
					OldObject: invalidUpdateRaw,
				},
			},
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := admission.NewDecoder(testScheme)
			if err != nil {
				t.Fatalf("error occurred while creating decoder: %v", err)
			}

			handler := AdmissionHandler{
				log:     logr.Discard(),
				decoder: d,
				client:  fakeClient,
			}

			if res := handler.Handle(context.Background(), tt.req); res.Allowed != tt.wantAllowed {
				t.Errorf("Allowed %t, but wanted %t", res.Allowed, tt.wantAllowed)
				t.Logf("Response: %v", res)
			}
		})
	}
}

func getApplicationDefinition(name string) *appkubermaticv1.ApplicationDefinition {
	return &appkubermaticv1.ApplicationDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appkubermaticv1.ApplicationDefinitionSpec{
			Description: "Description",
			Versions: []appkubermaticv1.ApplicationVersion{
				{
					Version: defaultAppVersion,
				},
			},
		},
	}
}

func getApplicationInstallation(name string, appName string, appVersion string) *appkubermaticv1.ApplicationInstallation {
	return &appkubermaticv1.ApplicationInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "kube-system",
		},
		Spec: appkubermaticv1.ApplicationInstallationSpec{
			Namespace: appkubermaticv1.NamespaceSpec{
				Name:   "default",
				Create: true,
			},
			ApplicationRef: appkubermaticv1.ApplicationRef{
				Name:    appName,
				Version: appkubermaticv1.Version{Version: *semverlib.MustParse(appVersion)},
			},
		},
	}
}

func applicationInstallationToRawExt(ai appkubermaticv1.ApplicationInstallation) runtime.RawExtension {
	s := json.NewSerializer(json.DefaultMetaFactory, testScheme, testScheme, true)
	buff := bytes.NewBuffer([]byte{})
	_ = s.Encode(&ai, buff)

	return runtime.RawExtension{
		Raw: buff.Bytes(),
	}
}
