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

package groupprojectbinding_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/ee/validation/groupprojectbinding"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateUpdate(t *testing.T) {
	testCases := []struct {
		name                   string
		oldGroupProjectBinding *kubermaticv1.GroupProjectBinding
		newGroupProjectBinding *kubermaticv1.GroupProjectBinding
		expectedError          error
	}{
		{
			name: "allowed update",
			oldGroupProjectBinding: &kubermaticv1.GroupProjectBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "binding1",
				},
				Spec: kubermaticv1.GroupProjectBindingSpec{
					Group:     "group1",
					Role:      "role1",
					ProjectID: "project1",
				},
			},
			newGroupProjectBinding: &kubermaticv1.GroupProjectBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "binding1",
				},
				Spec: kubermaticv1.GroupProjectBindingSpec{
					Group:     "group2",
					Role:      "role2",
					ProjectID: "project1",
				},
			},
			expectedError: nil,
		},
		{
			name: "not allowed update: changing project id",
			oldGroupProjectBinding: &kubermaticv1.GroupProjectBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "binding1",
				},
				Spec: kubermaticv1.GroupProjectBindingSpec{
					Group:     "group1",
					Role:      "role1",
					ProjectID: "project1",
				},
			},
			newGroupProjectBinding: &kubermaticv1.GroupProjectBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "binding1",
				},
				Spec: kubermaticv1.GroupProjectBindingSpec{
					Group:     "group2",
					Role:      "role2",
					ProjectID: "project2",
				},
			},
			expectedError: errors.New("attribute \"projectID\" cannot be updated for existing GroupProjectBinding resource"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := groupprojectbinding.ValidateUpdate(tc.oldGroupProjectBinding, tc.newGroupProjectBinding)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}
