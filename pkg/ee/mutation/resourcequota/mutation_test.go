//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
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
			name: "Add missing OwnershipReference and Labels to a new ResourceQuota",
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
				jsonpatch.NewOperation(
					"add",
					"/metadata/labels",
					map[string]interface{}{
						kubermaticv1.ResourceQuotaSubjectKindLabelKey: kubermaticv1.ProjectSubjectKind,
						kubermaticv1.ResourceQuotaSubjectNameLabelKey: "xxtestxx",
					},
				),
			},
		},
		{
			name: "Overwrite inappropriate labels to a new ResourceQuota",
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
							Labels: map[string]string{
								kubermaticv1.ResourceQuotaSubjectKindLabelKey: "NOT-project",
								kubermaticv1.ResourceQuotaSubjectNameLabelKey: "WRONG-name",
							},
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
				jsonpatch.NewOperation(
					"replace",
					"/metadata/labels/subject-name",
					"xxtestxx",
				),
				jsonpatch.NewOperation(
					"replace",
					"/metadata/labels/subject-kind",
					"project",
				),
			},
		},
		{
			name: "Update immutable subject-name label of a ResourceQuota",
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
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
							Labels: map[string]string{
								kubermaticv1.ResourceQuotaSubjectNameLabelKey: "should-not-be-changed",
							},
						}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawResourceQuotaGen{
							Name:        "project-xxtestxx",
							ProjectName: "xxtestxx",
							Labels: map[string]string{
								kubermaticv1.ResourceQuotaSubjectKindLabelKey: kubermaticv1.ProjectSubjectKind,
								kubermaticv1.ResourceQuotaSubjectNameLabelKey: "xxtestxx",
							},
						}.Do(),
					},
				},
			},
			wantError: true,
		},
		{
			name: "Update immutable subject-kind label of a ResourceQuota",
			req: admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
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
							Labels: map[string]string{
								kubermaticv1.ResourceQuotaSubjectKindLabelKey: "should-not-be-changed",
							},
						}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawResourceQuotaGen{
							Name:        "project-xxtestxx",
							ProjectName: "xxtestxx",
							Labels: map[string]string{
								kubermaticv1.ResourceQuotaSubjectKindLabelKey: kubermaticv1.ProjectSubjectKind,
								kubermaticv1.ResourceQuotaSubjectNameLabelKey: "xxtestxx",
							},
						}.Do(),
					},
				},
			},
			wantError: true,
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
	Labels      map[string]string
}

func (r rawResourceQuotaGen) Do() []byte {
	setting := kubermaticv1.ResourceQuota{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubermatic.k8c.io/v1",
			Kind:       "ResourceQuota",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   r.Name,
			Labels: r.Labels,
		},
		Spec: kubermaticv1.ResourceQuotaSpec{
			Subject: kubermaticv1.Subject{
				Name: r.ProjectName,
				Kind: kubermaticv1.ProjectSubjectKind,
			},
		},
	}

	s := json.NewSerializerWithOptions(json.DefaultMetaFactory, testScheme, testScheme, json.SerializerOptions{Pretty: true})
	buff := bytes.NewBuffer([]byte{})
	_ = s.Encode(&setting, buff)

	return buff.Bytes()
}
