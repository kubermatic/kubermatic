//go:build ee

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

package resourcequota_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-test/deep"
	jsonpatch "gomodules.xyz/jsonpatch/v2"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/webhook/resourcequota/mutation"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
	testcases := []struct {
		name         string
		req          webhook.AdmissionRequest
		existingObjs []ctrlruntimeclient.Object
		wantError    bool
		wantPatches  []jsonpatch.JsonPatchOperation
	}{
		{
			name: "Add missing OwnershipReference to a new ResourceQuota",
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "ResourceQuota",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawResourceQuotaGen{
							Name:        "project-xxtestxx",
							ProjectName: "xxtestxx",
						}.Do(),
					},
				},
			},
			existingObjs: []ctrlruntimeclient.Object{
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "xxtestxx",
						UID:  "bar",
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "boo",
					},
				},
			},
			wantError: false,
			wantPatches: []jsonpatch.Operation{
				jsonpatch.NewOperation(
					"add",
					"/metadata/ownerReferences",
					[]interface{}{map[string]interface{}{
						"apiVersion":         "kubermatic.k8c.io/v1",
						"kind":               "Project",
						"name":               "xxtestxx",
						"uid":                "bar",
						"controller":         true,
						"blockOwnerDeletion": true,
					}},
				),
			},
		},
	}

	for _, tc := range testcases {
		d, err := admission.NewDecoder(testScheme)
		if err != nil {
			t.Fatalf("error occurred while creating decoder: %v", err)
		}

		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithObjects(tc.existingObjs...).Build()

			handler := mutation.AdmissionHandler{
				Log:     logr.Discard(),
				Decoder: d,
				Client:  client,
			}

			res := handler.Handle(context.Background(), tc.req)
			if res.AdmissionResponse.Result != nil && res.AdmissionResponse.Result.Code == http.StatusBadRequest {
				if tc.wantError {
					return
				}

				t.Fatalf("Request failed: %v", res.AdmissionResponse.Result.Message)
			}

			a := map[string]jsonpatch.JsonPatchOperation{}
			for _, p := range res.Patches {
				a[p.Path] = p
			}
			fmt.Println(a)

			w := map[string]jsonpatch.JsonPatchOperation{}
			for _, p := range tc.wantPatches {
				w[p.Path] = p
			}
			if diff := deep.Equal(a, w); len(diff) > 0 {
				t.Errorf("Diff found between wanted and actual patches: %+v", diff)
			}
		})
	}
}

type rawResourceQuotaGen struct {
	Name        string
	ProjectName string
}

func (r rawResourceQuotaGen) Do() []byte {
	setting := kubermaticv1.ResourceQuota{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubermatic.k8c.io/v1",
			Kind:       "ResourceQuota",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Name,
			Namespace: resources.KubermaticNamespace,
		},
		Spec: kubermaticv1.ResourceQuotaSpec{
			Subject: kubermaticv1.Subject{
				Name: r.ProjectName,
				Kind: "project",
			},
		},
	}

	s := json.NewSerializerWithOptions(json.DefaultMetaFactory, testScheme, testScheme, json.SerializerOptions{Pretty: true})
	buff := bytes.NewBuffer([]byte{})
	_ = s.Encode(&setting, buff)

	return buff.Bytes()
}
