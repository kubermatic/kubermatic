/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

	"github.com/go-logr/logr"

	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

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
	_ = osmv1alpha1.AddToScheme(testScheme)
}

// TestHandle tests the admission handler.
func TestHandle(t *testing.T) {
	osp := getOperatingSystemProfile()
	ospRaw := ospToRawExt(osp)

	osp.Spec.OSVersion = "fake"
	ospRawInvalidUpdate := ospToRawExt(osp)

	osp.Spec.Version = "fake"
	ospRawValidUpdate := ospToRawExt(osp)

	tests := []struct {
		name        string
		req         webhook.AdmissionRequest
		wantAllowed bool
	}{
		{
			name: "Delete osp success",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Delete,
					RequestKind: &metav1.GroupVersionKind{
						Group:   osmv1alpha1.GroupName,
						Version: osmv1alpha1.GroupVersion,
						Kind:    "OperatingSystemProfile",
					},
					Name: "osp",
				},
			},
			wantAllowed: true,
		},
		{
			name: "Create osp success",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   osmv1alpha1.GroupName,
						Version: osmv1alpha1.GroupVersion,
						Kind:    "OperatingSystemProfile",
					},
					Name:   "osp",
					Object: ospRaw,
				},
			},
			wantAllowed: true,
		},
		{
			name: "Update osp rejected",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   osmv1alpha1.GroupName,
						Version: osmv1alpha1.GroupVersion,
						Kind:    "OperatingSystemProfile",
					},
					Name:      "osp",
					Object:    ospRaw,
					OldObject: ospRawInvalidUpdate,
				},
			},
			wantAllowed: false,
		},
		{
			name: "Update osp success",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   osmv1alpha1.GroupName,
						Version: osmv1alpha1.GroupVersion,
						Kind:    "OperatingSystemProfile",
					},
					Name:      "osp",
					Object:    ospRaw,
					OldObject: ospRawValidUpdate,
				},
			},
			wantAllowed: true,
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
			}

			if res := handler.Handle(context.Background(), tt.req); res.Allowed != tt.wantAllowed {
				t.Errorf("Allowed %t, but wanted %t", res.Allowed, tt.wantAllowed)
				t.Logf("Response: %v", res)
			}
		})
	}
}

func getOperatingSystemProfile() osmv1alpha1.OperatingSystemProfile {
	return osmv1alpha1.OperatingSystemProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: osmv1alpha1.OperatingSystemProfileSpec{
			OSName:    "ubuntu",
			OSVersion: "20.04",
			ProvisioningConfig: osmv1alpha1.OSPConfig{
				Files: []osmv1alpha1.File{
					{
						Path:        "/opt/bin/test.service",
						Permissions: 700,
						Content: osmv1alpha1.FileContent{
							Inline: &osmv1alpha1.FileContentInline{
								Data: "    #!/bin/bash\n    set -xeuo pipefail\n    cloud-init clean\n    cloud-init init\n    systemctl start provision.service",
							},
						},
					},
				},
			},
		},
	}
}

func ospToRawExt(osp osmv1alpha1.OperatingSystemProfile) runtime.RawExtension {
	s := json.NewSerializer(json.DefaultMetaFactory, testScheme, testScheme, true)
	buff := bytes.NewBuffer([]byte{})
	_ = s.Encode(&osp, buff)

	return runtime.RawExtension{
		Raw: buff.Bytes(),
	}
}
