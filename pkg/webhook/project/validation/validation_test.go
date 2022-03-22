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
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"

	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		name              string
		projectToValidate *kubermaticv1.Project
		existingUsers     []*kubermaticv1.User
		op                admissionv1.Operation
		errExpected       bool
	}{
		{
			name:              "Creating a project with a proper user owner ref should be possible",
			projectToValidate: genProject("success", test.GenDefaultUser()),
			existingUsers:     []*kubermaticv1.User{test.GenDefaultUser()},
			op:                admissionv1.Create,
		},
		{
			name:              "Creating a project with a owner ref for a user which doesn't exist should be refused",
			projectToValidate: genProject("success", test.GenDefaultUser()),
			existingUsers:     []*kubermaticv1.User{},
			op:                admissionv1.Create,
			errExpected:       true,
		},
		{
			name:              "Creating a project without a owner ref should be refused",
			projectToValidate: genProject("success", nil),
			existingUsers:     []*kubermaticv1.User{},
			op:                admissionv1.Create,
			errExpected:       true,
		},
		{
			name: "Creating a project with other owner refs but no user owner ref should be refused",
			projectToValidate: func() *kubermaticv1.Project {
				pr := genProject("success", nil)
				pr.OwnerReferences = []v1.OwnerReference{
					{Name: "bob", Kind: "Bob"},
				}
				return pr
			}(),
			existingUsers: []*kubermaticv1.User{},
			op:            admissionv1.Create,
			errExpected:   true,
		},
		{
			name:              "Deleting a project should work without any check",
			projectToValidate: genProject("success", nil),
			op:                admissionv1.Delete,
		},
		{
			name:              "Updating a project with a proper user owner ref should be possible",
			projectToValidate: genProject("success", test.GenDefaultUser()),
			existingUsers:     []*kubermaticv1.User{test.GenDefaultUser()},
			op:                admissionv1.Update,
		},
		{
			name:              "Creating a project with a owner ref for a user which doesn't exist should be refused",
			projectToValidate: genProject("success", test.GenDefaultUser()),
			existingUsers:     []*kubermaticv1.User{},
			op:                admissionv1.Update,
			errExpected:       true,
		},
		{
			name:              "Creating a project without a owner ref should be refused",
			projectToValidate: genProject("success", nil),
			existingUsers:     []*kubermaticv1.User{},
			op:                admissionv1.Update,
			errExpected:       true,
		},
		{
			name: "Creating a project with other owner refs but no user owner ref should be refused",
			projectToValidate: func() *kubermaticv1.Project {
				pr := genProject("success", nil)
				pr.OwnerReferences = []v1.OwnerReference{
					{Name: "bob", Kind: "Bob"},
				}
				return pr
			}(),
			existingUsers: []*kubermaticv1.User{},
			op:            admissionv1.Update,
			errExpected:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				obj []ctrlruntimeclient.Object
				err error
			)
			ctx := context.Background()

			for _, s := range tc.existingUsers {
				obj = append(obj, s)
			}

			client := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(obj...).
				Build()

			validator := NewValidator(client)

			switch tc.op {
			case admissionv1.Create:
				err = validator.ValidateCreate(ctx, tc.projectToValidate)
			case admissionv1.Update:
				err = validator.ValidateUpdate(ctx, nil, tc.projectToValidate)
			case admissionv1.Delete:
				err = validator.ValidateDelete(ctx, tc.projectToValidate)
			}

			if (err != nil) != tc.errExpected {
				t.Fatalf("Expected err: %t, but got err: %v", tc.errExpected, err)
			}
		})
	}
}

func genProject(name string, user *kubermaticv1.User) *kubermaticv1.Project {
	project := &kubermaticv1.Project{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.ProjectSpec{
			Name: name,
		},
	}

	if user != nil {
		project.OwnerReferences = []v1.OwnerReference{
			{
				Name: user.Name,
				Kind: kubermaticv1.UserKindName,
			},
		}
	}

	return project
}
