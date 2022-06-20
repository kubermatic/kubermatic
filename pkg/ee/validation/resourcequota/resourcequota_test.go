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

package resourcequota

import (
	"context"
	"fmt"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/scheme"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestValidateResourceQuotaInsallation(t *testing.T) {

	testCases := []struct {
		name                    string
		existingResourceQuota   []*kubermaticv1.ResourceQuota
		resourceQuotaToValidate *kubermaticv1.ResourceQuota
		errExpected             bool
	}{
		{
			name: "Create ApplicationInstallation Success",
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
			name: "Create ApplicationInstallation Failure",
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			client := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				Build()

			err := ValidateResourceQuota(context.Background(), tc.resourceQuotaToValidate, client)
			fmt.Println(err)
			if (err != nil) != tc.errExpected {
				t.Fatalf("Expected err: %t, but got err: %v", tc.errExpected, err)
			}
		})
	}
}
