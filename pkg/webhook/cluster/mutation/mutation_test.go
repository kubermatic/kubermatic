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

package mutation

import (
	"bytes"
	"context"
	"testing"
	"text/template"

	logrtesting "github.com/go-logr/logr/testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
		name            string
		req             webhook.AdmissionRequest
		wantAllowed     bool
		wantAnnotations bool
		wantUseOctavia  bool
	}{
		{
			name: "Create cluster success",
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
						Raw: rawClusterGen{Name: "foo", CloudProvider: "openstack", ExternalCloudProvider: true}.Do(),
					},
				},
			},
			wantAllowed: true,
		},
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
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", CloudProvider: "openstack", ExternalCloudProvider: true}.Do(),
					},
				},
			},
			wantAllowed: true,
		},
		{
			name: "Update OpenStack cluster to enable the CCM/CSI migration",
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
						Raw: rawClusterGen{Name: "foo", CloudProvider: "openstack", ExternalCloudProvider: true}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", CloudProvider: "openstack", ExternalCloudProvider: false}.Do(),
					},
				},
			},
			wantAllowed:     true,
			wantAnnotations: true,
			wantUseOctavia:  true,
		},
		{
			name: "Update OpenStack cluster with enabled CCM/CSI migration",
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
						Raw: rawClusterGen{Name: "foo", CloudProvider: "openstack", ExternalCloudProvider: true}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", CloudProvider: "openstack", ExternalCloudProvider: true}.Do(),
					},
				},
			},
			wantAllowed:     true,
			wantAnnotations: false,
			wantUseOctavia:  false,
		},
		{
			name: "Update non-OpenStack cluster to enable CCM/CSI migration",
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
						Raw: rawClusterGen{Name: "foo", CloudProvider: "hetzner", ExternalCloudProvider: true}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", CloudProvider: "hetzner", ExternalCloudProvider: false}.Do(),
					},
				},
			},
			wantAllowed:     true,
			wantAnnotations: false,
			wantUseOctavia:  false,
		},
	}
	for _, tt := range tests {
		d, err := admission.NewDecoder(testScheme)
		if err != nil {
			t.Fatalf("error occurred while creating decoder: %v", err)
		}
		handler := AdmissionHandler{
			log:     &logrtesting.NullLogger{},
			decoder: d,
		}
		t.Run(tt.name, func(t *testing.T) {
			res := handler.Handle(context.TODO(), tt.req)
			if res.Allowed != tt.wantAllowed {
				t.Logf("Response: %v", res)
				t.Fatalf("Allowed %t, but wanted %t", res.Allowed, tt.wantAllowed)
			}

			foundPatchCCMAnnotation := false
			foundPatchCSIAnnotation := false
			foundPatchUseOctavia := false

			for _, patch := range res.Patches {
				switch {
				case patch.Operation == "add" && patch.Path == "/metadata/annotations" && patch.Value != nil:
					if m, ok := patch.Value.(map[string]interface{}); ok {
						for k := range m {
							if k == kubermaticv1.CCMMigrationNeededAnnotation {
								foundPatchCCMAnnotation = true
							}
							if k == kubermaticv1.CSIMigrationNeededAnnotation {
								foundPatchCSIAnnotation = true
							}
						}
					}
				case patch.Operation == "add" && patch.Path == "/spec/cloud/openstack/useOctavia":
					if v, ok := patch.Value.(bool); ok {
						if v {
							foundPatchUseOctavia = true
						}
					}
				}
			}

			if tt.wantAnnotations != foundPatchCCMAnnotation {
				t.Errorf("ccm-migration.k8c.io/migration-needed: expected: %v, found: %v", tt.wantAnnotations, foundPatchCCMAnnotation)
			}
			if tt.wantAnnotations != foundPatchCSIAnnotation {
				t.Errorf("csi-migration.k8c.io/migration-needed: expected: %v, found: %v", tt.wantAnnotations, foundPatchCSIAnnotation)
			}
			if tt.wantUseOctavia != foundPatchUseOctavia {
				t.Errorf(".spec.Cloud.openstack.UseOctavia: expected: %v, found: %v", tt.wantAnnotations, foundPatchUseOctavia)
			}
		})
	}
}

type rawClusterGen struct {
	Name                  string
	CloudProvider         string
	ExternalCloudProvider bool
}

func (r rawClusterGen) Do() []byte {
	tmpl, _ := template.New("cluster").Parse(`
	{
	"apiVersion": "kubermatic.k8s.io/v1",
	"kind": "Cluster",
	"metadata": {
	  "name": "{{ .Name }}"
	},
	"spec": {
	  "features": {
		"externalCloudProvider": {{ .ExternalCloudProvider }}
	  },
      "cloud": {
        "{{ .CloudProvider }}": {}
      }
	}
}`)
	sb := bytes.Buffer{}
	_ = tmpl.Execute(&sb, r)
	return sb.Bytes()
}
