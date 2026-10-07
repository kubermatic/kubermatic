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
	"context"
	"strings"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/ee/validation/resourcequota"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestValidateCreate(t *testing.T) {
	testCases := []struct {
		name                    string
		existingResourceQuota   []*kubermaticv1.ResourceQuota
		resourceQuotaToValidate *kubermaticv1.ResourceQuota
		errExpected             bool
		errorContains           string
	}{
		{
			name: "Create ResourceQuota Success",
			existingResourceQuota: []*kubermaticv1.ResourceQuota{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existing-quota",
					},
					Spec: kubermaticv1.ResourceQuotaSpec{
						Subject: kubermaticv1.Subject{
							Name: "wwqrvcccq5",
							Kind: "project",
						},
						Quota: kubermaticv1.ResourceDetails{},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existing-quota1",
					},
					Spec: kubermaticv1.ResourceQuotaSpec{
						Subject: kubermaticv1.Subject{
							Name: "wwqrvcccq6",
							Kind: "seed",
						},
						Quota: kubermaticv1.ResourceDetails{},
					},
				},
			},
			resourceQuotaToValidate: &kubermaticv1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-quota",
				},
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{
						Name: "wwqrvcccq6",
						Kind: "project",
					},
					Quota: kubermaticv1.ResourceDetails{},
				},
			},
		},
		{
			name: "Create ResourceQuota Failure",
			existingResourceQuota: []*kubermaticv1.ResourceQuota{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existing-quota",
					},
					Spec: kubermaticv1.ResourceQuotaSpec{
						Subject: kubermaticv1.Subject{
							Name: "wwqrvcccq6",
							Kind: "project",
						},
						Quota: kubermaticv1.ResourceDetails{},
					},
				},
			},
			resourceQuotaToValidate: &kubermaticv1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-quota",
				},
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{
						Name: "wwqrvcccq6",
						Kind: "project",
					},
					Quota: kubermaticv1.ResourceDetails{},
				},
			},
			errExpected: true,
		},
		{
			name: "Create ResourceQuota with accelerator quota success",
			resourceQuotaToValidate: &kubermaticv1.ResourceQuota{
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{Name: "project-with-accelerators", Kind: kubermaticv1.ProjectSubjectKind},
					Quota: kubermaticv1.ResourceDetails{
						Accelerators: []kubermaticv1.AcceleratorQuota{{
							Provider: "kubevirt",
							Resources: corev1.ResourceList{
								"nvidia.com/GH100_H200_NVL": resource.MustParse("2"),
							},
						}},
					},
				},
			},
		},
		{
			name: "Create ResourceQuota with negative accelerator quota failure",
			resourceQuotaToValidate: &kubermaticv1.ResourceQuota{
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{Name: "project-with-negative-accelerators", Kind: kubermaticv1.ProjectSubjectKind},
					Quota: kubermaticv1.ResourceDetails{
						Accelerators: []kubermaticv1.AcceleratorQuota{{
							Provider: "kubevirt",
							Resources: corev1.ResourceList{
								"nvidia.com/GH100_H200_NVL": resource.MustParse("-1"),
							},
						}},
					},
				},
			},
			errExpected:   true,
			errorContains: `invalid accelerator quota: accelerators[0].resources[nvidia.com/GH100_H200_NVL]`,
		},
		{
			name: "Create ResourceQuota with fractional accelerator quota failure",
			resourceQuotaToValidate: &kubermaticv1.ResourceQuota{
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{Name: "project-with-fractional-accelerators", Kind: kubermaticv1.ProjectSubjectKind},
					Quota: kubermaticv1.ResourceDetails{
						Accelerators: []kubermaticv1.AcceleratorQuota{{
							Provider: "kubevirt",
							Resources: corev1.ResourceList{
								"nvidia.com/GH100_H200_NVL": resource.MustParse("500m"),
							},
						}},
					},
				},
			},
			errExpected:   true,
			errorContains: "invalid accelerator quota:",
		},
		{
			name: "Create ResourceQuota with empty accelerator key failure",
			resourceQuotaToValidate: &kubermaticv1.ResourceQuota{
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{Name: "project-with-empty-accelerator-key", Kind: kubermaticv1.ProjectSubjectKind},
					Quota: kubermaticv1.ResourceDetails{
						Accelerators: []kubermaticv1.AcceleratorQuota{{
							Provider: "kubevirt",
							Resources: corev1.ResourceList{
								"": resource.MustParse("1"),
							},
						}},
					},
				},
			},
			errExpected:   true,
			errorContains: "invalid accelerator quota:",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				obj []ctrlruntimeclient.Object
				err error
			)

			for _, rq := range tc.existingResourceQuota {
				obj = append(obj, rq)
			}

			client := fake.
				NewClientBuilder().
				WithObjects(obj...).
				Build()

			err = resourcequota.ValidateCreate(context.Background(), tc.resourceQuotaToValidate, client)
			if (err != nil) != tc.errExpected {
				t.Fatalf("Expected err: %t, but got err: %v", tc.errExpected, err)
			}
			if tc.errorContains != "" && !strings.Contains(err.Error(), tc.errorContains) {
				t.Fatalf("expected error %q to contain %q", err, tc.errorContains)
			}
		})
	}
}

func TestValidateUpdate(t *testing.T) {
	testCases := []struct {
		name             string
		oldResourceQuota *kubermaticv1.ResourceQuota
		newResourceQuota *kubermaticv1.ResourceQuota
		errExpected      bool
		errorContains    string
	}{
		{
			name: "Update ResourceQuota Subject Failure",
			oldResourceQuota: &kubermaticv1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing-quota",
				},
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{
						Name: "wwqrvcccq6",
						Kind: "project",
					},
					Quota: kubermaticv1.ResourceDetails{},
				},
			},
			newResourceQuota: &kubermaticv1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing-quota",
				},
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{
						Name: "wwqrvcccq7",
						Kind: "project",
					},
					Quota: kubermaticv1.ResourceDetails{},
				},
			},
			errExpected:   true,
			errorContains: "Operation not permitted: updating ResourceQuota Subject is not allowed!",
		},
		{
			name: "new fractional accelerator quota is rejected",
			oldResourceQuota: &kubermaticv1.ResourceQuota{
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{Name: "project-with-accelerators", Kind: kubermaticv1.ProjectSubjectKind},
				},
			},
			newResourceQuota: &kubermaticv1.ResourceQuota{
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{Name: "project-with-accelerators", Kind: kubermaticv1.ProjectSubjectKind},
					Quota: kubermaticv1.ResourceDetails{
						Accelerators: acceleratorQuota("kubevirt", "nvidia.com/GH100_H200_NVL", "1.5"),
					},
				},
			},
			errExpected:   true,
			errorContains: `invalid accelerator quota: accelerators[0].resources[nvidia.com/GH100_H200_NVL]`,
		},
		{
			name: "existing quota without accelerators permits finalizer removal",
			oldResourceQuota: &kubermaticv1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "existing-quota",
					Finalizers: []string{"kubermatic.k8c.io/test"},
				},
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{Name: "project-with-accelerators", Kind: kubermaticv1.ProjectSubjectKind},
				},
			},
			newResourceQuota: &kubermaticv1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{Name: "existing-quota"},
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{Name: "project-with-accelerators", Kind: kubermaticv1.ProjectSubjectKind},
				},
			},
		},
		{
			name: "unsupported provider is rejected",
			oldResourceQuota: &kubermaticv1.ResourceQuota{
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{Name: "project-with-accelerators", Kind: kubermaticv1.ProjectSubjectKind},
				},
			},
			newResourceQuota: &kubermaticv1.ResourceQuota{
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{Name: "project-with-accelerators", Kind: kubermaticv1.ProjectSubjectKind},
					Quota: kubermaticv1.ResourceDetails{
						Accelerators: acceleratorQuota("aws", "accelerators.kubermatic.io/h100", "2"),
					},
				},
			},
			errExpected:   true,
			errorContains: `invalid accelerator quota: accelerators[0].provider: Unsupported value: "aws"`,
		},
		{
			name: "malformed resource name is rejected",
			oldResourceQuota: &kubermaticv1.ResourceQuota{
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{Name: "project-with-accelerators", Kind: kubermaticv1.ProjectSubjectKind},
				},
			},
			newResourceQuota: &kubermaticv1.ResourceQuota{
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{Name: "project-with-accelerators", Kind: kubermaticv1.ProjectSubjectKind},
					Quota: kubermaticv1.ResourceDetails{
						Accelerators: acceleratorQuota("kubevirt", "bad", "2"),
					},
				},
			},
			errExpected:   true,
			errorContains: `invalid accelerator quota: accelerators[0].resources[bad]`,
		},
		{
			name: "duplicate provider is rejected",
			oldResourceQuota: &kubermaticv1.ResourceQuota{
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{Name: "project-with-accelerators", Kind: kubermaticv1.ProjectSubjectKind},
					Quota: kubermaticv1.ResourceDetails{
						Accelerators: acceleratorQuota("kubevirt", "nvidia.com/GPU", "1"),
					},
				},
			},
			newResourceQuota: &kubermaticv1.ResourceQuota{
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{Name: "project-with-accelerators", Kind: kubermaticv1.ProjectSubjectKind},
					Quota: kubermaticv1.ResourceDetails{Accelerators: []kubermaticv1.AcceleratorQuota{
						{Provider: "kubevirt", Resources: corev1.ResourceList{"nvidia.com/GPU": resource.MustParse("1")}},
						{Provider: "kubevirt", Resources: corev1.ResourceList{"nvidia.com/A100": resource.MustParse("1")}},
					}},
				},
			},
			errExpected:   true,
			errorContains: `invalid accelerator quota: accelerators[1].provider: Duplicate value: "kubevirt"`,
		},
		{
			name: "valid accelerator quota is accepted",
			oldResourceQuota: &kubermaticv1.ResourceQuota{
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{Name: "project-with-accelerators", Kind: kubermaticv1.ProjectSubjectKind},
				},
			},
			newResourceQuota: &kubermaticv1.ResourceQuota{
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{Name: "project-with-accelerators", Kind: kubermaticv1.ProjectSubjectKind},
					Quota: kubermaticv1.ResourceDetails{
						Accelerators: acceleratorQuota("kubevirt", "nvidia.com/GH100_H200_NVL", "1"),
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := resourcequota.ValidateUpdate(context.Background(), tc.oldResourceQuota, tc.newResourceQuota)
			if (err != nil) != tc.errExpected {
				t.Fatalf("Expected err: %t, but got err: %v", tc.errExpected, err)
			}
			if tc.errorContains != "" && !strings.Contains(err.Error(), tc.errorContains) {
				t.Fatalf("expected error %q to contain %q", err, tc.errorContains)
			}
		})
	}
}

func acceleratorQuota(provider, resourceName, quantity string) []kubermaticv1.AcceleratorQuota {
	return []kubermaticv1.AcceleratorQuota{{
		Provider: provider,
		Resources: corev1.ResourceList{
			corev1.ResourceName(resourceName): resource.MustParse(quantity),
		},
	}}
}
