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
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/ee/validation/groupprojectbinding"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func requestContext(username string, groups ...string) context.Context {
	return admission.NewContextWithRequest(context.Background(), admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			UserInfo: authenticationv1.UserInfo{Username: username, Groups: groups},
		},
	})
}

// adminContext is a request from the KKP API's own ServiceAccount, as for KKP admins.
func adminContext() context.Context {
	return requestContext("system:serviceaccount:kubermatic:kubermatic-api", "system:serviceaccounts", "system:authenticated")
}

func TestValidateCreate(t *testing.T) {
	testCases := []struct {
		name            string
		existingBinding *kubermaticv1.GroupProjectBinding
		newBinding      *kubermaticv1.GroupProjectBinding
		errExpected     bool
	}{
		{
			name: "no existing binding, create succeeds",
			newBinding: &kubermaticv1.GroupProjectBinding{
				ObjectMeta: metav1.ObjectMeta{Name: "new-binding"},
				Spec:       kubermaticv1.GroupProjectBindingSpec{Group: "group1", ProjectID: "project1", Role: "viewers"},
			},
			errExpected: false,
		},
		{
			name: "same group and project already bound, create fails",
			existingBinding: &kubermaticv1.GroupProjectBinding{
				ObjectMeta: metav1.ObjectMeta{Name: "existing-binding"},
				Spec:       kubermaticv1.GroupProjectBindingSpec{Group: "group1", ProjectID: "project1", Role: "viewers"},
			},
			newBinding: &kubermaticv1.GroupProjectBinding{
				ObjectMeta: metav1.ObjectMeta{Name: "new-binding"},
				Spec:       kubermaticv1.GroupProjectBindingSpec{Group: "group1", ProjectID: "project1", Role: "editors"},
			},
			errExpected: true,
		},
		{
			name: "same group different project, create succeeds",
			existingBinding: &kubermaticv1.GroupProjectBinding{
				ObjectMeta: metav1.ObjectMeta{Name: "existing-binding"},
				Spec:       kubermaticv1.GroupProjectBindingSpec{Group: "group1", ProjectID: "project1", Role: "viewers"},
			},
			newBinding: &kubermaticv1.GroupProjectBinding{
				ObjectMeta: metav1.ObjectMeta{Name: "new-binding"},
				Spec:       kubermaticv1.GroupProjectBindingSpec{Group: "group1", ProjectID: "project2", Role: "viewers"},
			},
			errExpected: false,
		},
		{
			name: "different group same project, create succeeds",
			existingBinding: &kubermaticv1.GroupProjectBinding{
				ObjectMeta: metav1.ObjectMeta{Name: "existing-binding"},
				Spec:       kubermaticv1.GroupProjectBindingSpec{Group: "group1", ProjectID: "project1", Role: "viewers"},
			},
			newBinding: &kubermaticv1.GroupProjectBinding{
				ObjectMeta: metav1.ObjectMeta{Name: "new-binding"},
				Spec:       kubermaticv1.GroupProjectBindingSpec{Group: "group2", ProjectID: "project1", Role: "viewers"},
			},
			errExpected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var objs []ctrlruntimeclient.Object
			if tc.existingBinding != nil {
				objs = append(objs, tc.existingBinding)
			}
			client := fake.NewClientBuilder().WithObjects(objs...).Build()

			err := groupprojectbinding.ValidateCreate(adminContext(), tc.newBinding, client)
			if (err != nil) != tc.errExpected {
				t.Fatalf("expected error: %v, got: %v", tc.errExpected, err)
			}
		})
	}
}

func TestValidateUpdate(t *testing.T) {
	testCases := []struct {
		name                   string
		oldGroupProjectBinding *kubermaticv1.GroupProjectBinding
		newGroupProjectBinding *kubermaticv1.GroupProjectBinding
		otherBindings          []ctrlruntimeclient.Object
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
		{
			name: "not allowed update: changing group collides with another binding",
			oldGroupProjectBinding: &kubermaticv1.GroupProjectBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "binding2",
				},
				Spec: kubermaticv1.GroupProjectBindingSpec{
					Group:     "group2",
					Role:      "role2",
					ProjectID: "project1",
				},
			},
			newGroupProjectBinding: &kubermaticv1.GroupProjectBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "binding2",
				},
				Spec: kubermaticv1.GroupProjectBindingSpec{
					Group:     "group1",
					Role:      "role2",
					ProjectID: "project1",
				},
			},
			otherBindings: []ctrlruntimeclient.Object{
				&kubermaticv1.GroupProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "binding1",
					},
					Spec: kubermaticv1.GroupProjectBindingSpec{
						Group:     "group1",
						Role:      "role1",
						ProjectID: "project1",
					},
				},
			},
			expectedError: fmt.Errorf("group %q is already bound to project %q", "group1", "project1"),
		},
		{
			name: "allowed no-op update: unchanged group and project do not collide with self",
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
					Group:     "group1",
					Role:      "role2",
					ProjectID: "project1",
				},
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			objs := append([]ctrlruntimeclient.Object{tc.oldGroupProjectBinding}, tc.otherBindings...)
			client := fake.NewClientBuilder().WithObjects(objs...).Build()

			err := groupprojectbinding.ValidateUpdate(adminContext(), tc.oldGroupProjectBinding, tc.newGroupProjectBinding, client)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestValidateRequesterRole(t *testing.T) {
	binding := func(role string) *kubermaticv1.GroupProjectBinding {
		return &kubermaticv1.GroupProjectBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "p1-binding"},
			Spec:       kubermaticv1.GroupProjectBindingSpec{Group: "team-x", ProjectID: "p1", Role: role},
		}
	}

	testCases := []struct {
		name        string
		ctx         context.Context
		role        string
		errExpected bool
	}{
		{
			name:        "editor cannot grant owners",
			ctx:         requestContext("editor@example.com", "editors-p1", "team-x-p1", "system:authenticated"),
			role:        "owners",
			errExpected: true,
		},
		{
			name: "editor can grant editors",
			ctx:  requestContext("editor@example.com", "editors-p1", "system:authenticated"),
			role: "editors",
		},
		{
			name:        "viewer cannot grant editors",
			ctx:         requestContext("viewer@example.com", "viewers-p1", "system:authenticated"),
			role:        "editors",
			errExpected: true,
		},
		{
			name:        "projectmanager cannot grant owners",
			ctx:         requestContext("pm@example.com", "projectmanagers-p1", "system:authenticated"),
			role:        "owners",
			errExpected: true,
		},
		{
			name: "owner can grant owners",
			ctx:  requestContext("owner@example.com", "owners-p1", "system:authenticated"),
			role: "owners",
		},
		{
			name: "highest of several roles counts",
			ctx:  requestContext("both@example.com", "viewers-p1", "owners-p1", "system:authenticated"),
			role: "owners",
		},
		{
			name:        "owner of another project does not count",
			ctx:         requestContext("other@example.com", "owners-p2", "editors-p1", "system:authenticated"),
			role:        "owners",
			errExpected: true,
		},
		{
			name: "KKP API ServiceAccount (KKP admin) can grant owners",
			ctx:  adminContext(),
			role: "owners",
		},
		{
			name:        "no admission request in context",
			ctx:         context.Background(),
			role:        "viewers",
			errExpected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewClientBuilder().Build()

			err := groupprojectbinding.ValidateCreate(tc.ctx, binding(tc.role), client)
			if tc.errExpected != (err != nil) {
				t.Fatalf("create: expected error = %v, got %v", tc.errExpected, err)
			}

			old := binding("viewers")
			old.Spec.Group = "team-y"
			client = fake.NewClientBuilder().WithObjects(old).Build()
			updated := old.DeepCopy()
			updated.Spec.Role = tc.role

			err = groupprojectbinding.ValidateUpdate(tc.ctx, old, updated, client)
			if tc.role != "viewers" && tc.errExpected != (err != nil) {
				t.Fatalf("update: expected error = %v, got %v", tc.errExpected, err)
			}
		})
	}
}

func TestValidateRequesterRoleFromGroupBinding(t *testing.T) {
	gpb := func(name, group, role string) *kubermaticv1.GroupProjectBinding {
		return &kubermaticv1.GroupProjectBinding{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec:       kubermaticv1.GroupProjectBindingSpec{Group: group, ProjectID: "p1", Role: role},
		}
	}

	// the requester's role comes only from the GroupProjectBinding of its IdP group
	editorByGroup := gpb("p1-editors", "team-editors", "editors")
	ownerByGroup := gpb("p1-owners", "team-owners", "owners")

	testCases := []struct {
		name        string
		ctx         context.Context
		role        string
		errExpected bool
	}{
		{
			name:        "editor through a group binding cannot grant owners",
			ctx:         requestContext("editor@example.com", "team-editors-p1", "system:authenticated"),
			role:        "owners",
			errExpected: true,
		},
		{
			name: "editor through a group binding can grant editors",
			ctx:  requestContext("editor@example.com", "team-editors-p1", "system:authenticated"),
			role: "editors",
		},
		{
			name: "owner through a group binding can grant owners",
			ctx:  requestContext("owner@example.com", "team-owners-p1", "system:authenticated"),
			role: "owners",
		},
		{
			name:        "group bound in another project does not count",
			ctx:         requestContext("other@example.com", "team-owners-p2", "team-editors-p1", "system:authenticated"),
			role:        "owners",
			errExpected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithObjects(editorByGroup, ownerByGroup).Build()

			err := groupprojectbinding.ValidateCreate(tc.ctx, gpb("p1-new", "team-new", tc.role), client)
			if tc.errExpected != (err != nil) {
				t.Fatalf("expected error = %v, got %v", tc.errExpected, err)
			}
		})
	}
}

func TestValidateUpdateGroupChange(t *testing.T) {
	owners := &kubermaticv1.GroupProjectBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "p1-owners"},
		Spec:       kubermaticv1.GroupProjectBindingSpec{Group: "team-owners", ProjectID: "p1", Role: "owners"},
	}

	testCases := []struct {
		name        string
		ctx         context.Context
		mutate      func(*kubermaticv1.GroupProjectBinding)
		errExpected bool
	}{
		{
			name:        "projectmanager cannot point an owners binding at another group",
			ctx:         requestContext("pm@example.com", "projectmanagers-p1", "system:authenticated"),
			mutate:      func(b *kubermaticv1.GroupProjectBinding) { b.Spec.Group = "team-pm" },
			errExpected: true,
		},
		{
			name:        "projectmanager cannot lower an owners binding",
			ctx:         requestContext("pm@example.com", "projectmanagers-p1", "system:authenticated"),
			mutate:      func(b *kubermaticv1.GroupProjectBinding) { b.Spec.Role = "viewers" },
			errExpected: true,
		},
		{
			name:   "owner can point an owners binding at another group",
			ctx:    requestContext("owner@example.com", "owners-p1", "system:authenticated"),
			mutate: func(b *kubermaticv1.GroupProjectBinding) { b.Spec.Group = "team-other" },
		},
		{
			name:   "projectmanager may update an owners binding without changing group or role",
			ctx:    requestContext("pm@example.com", "projectmanagers-p1", "system:authenticated"),
			mutate: func(b *kubermaticv1.GroupProjectBinding) { b.Labels = map[string]string{"a": "b"} },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithObjects(owners).Build()
			updated := owners.DeepCopy()
			tc.mutate(updated)

			err := groupprojectbinding.ValidateUpdate(tc.ctx, owners, updated, client)
			if tc.errExpected != (err != nil) {
				t.Fatalf("expected error = %v, got %v", tc.errExpected, err)
			}
		})
	}
}
