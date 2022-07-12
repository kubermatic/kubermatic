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

package kubernetes_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-test/deep"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateBinding(t *testing.T) {
	// test data
	kubermaticObjects := []ctrlruntimeclient.Object{}
	fakeClient := fakectrlruntimeclient.
		NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(kubermaticObjects...).
		Build()
	authenticatedUser := createAuthenitactedUser()
	existingProject := genDefaultProject()
	memberEmail := ""
	groupName := fmt.Sprintf("owners-%s", existingProject.Name)
	fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
		return fakeClient, nil
	}
	// act
	target := kubernetes.NewProjectMemberProvider(fakeImpersonationClient, fakeClient)
	result, err := target.Create(context.Background(), &provider.UserInfo{Email: authenticatedUser.Spec.Email, Groups: []string{fmt.Sprintf("owners-%s", existingProject.Name)}}, existingProject, memberEmail, groupName)

	// validate
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("received an empty response")
	}
	if result.Spec.Group != groupName {
		t.Fatalf("unexpected result returned, expected %s, got %s", groupName, result.Spec.Group)
	}
	if result.Spec.UserEmail != memberEmail {
		t.Fatalf("unexpected result returned, expected %s, got %s", memberEmail, result.Spec.UserEmail)
	}
	if result.Spec.ProjectID != existingProject.Name {
		t.Fatalf("unexpected result returned, expected %s, got %s", existingProject.Name, result.Spec.ProjectID)
	}
}

func TestListBinding(t *testing.T) {
	// test data
	testcases := []struct {
		name              string
		authenticatedUser *kubermaticv1.User
		projectToSync     *kubermaticv1.Project
		existingBindings  []*kubermaticv1.UserProjectBinding
		existingSA        []*kubermaticv1.User
		expectedBindings  []*kubermaticv1.UserProjectBinding
	}{
		{
			name:              "scenario 1: list bindings for the given project",
			authenticatedUser: createAuthenitactedUser(),
			projectToSync:     genDefaultProject(),
			existingBindings: []*kubermaticv1.UserProjectBinding{
				createBinding("abcdBinding", "my-first-project-ID", "bob@acme.com", "owners"),
				createBinding("cdBinding", "my-first-project-ID", "bob@acme.com", "owners"),
				createBinding("differentProjectBinding", "abcd", "bob@acme.com", "owners"),
			},
			expectedBindings: []*kubermaticv1.UserProjectBinding{
				createBinding("abcdBinding", "my-first-project-ID", "bob@acme.com", "owners"),
				createBinding("cdBinding", "my-first-project-ID", "bob@acme.com", "owners"),
			},
		},
		{
			name:              "scenario 1: filter out service accounts from bindings for the given project",
			authenticatedUser: createAuthenitactedUser(),
			projectToSync:     genDefaultProject(),
			existingSA: []*kubermaticv1.User{
				genProjectServiceAccount("1", "test", "editors", "my-first-project-ID"),
			},
			existingBindings: []*kubermaticv1.UserProjectBinding{

				createBinding("abcdBinding", "my-first-project-ID", "bob@acme.com", "owners"),
				createBinding("cdBinding", "my-first-project-ID", "bob@acme.com", "owners"),
				// binding for service account
				createBinding("test", "my-first-project-ID", "serviceaccount-1@localhost", "editors"),
				// binding for regular user with email pattern for service account
				createBinding("fakeServiceAccount", "my-first-project-ID", "serviceaccount-test@localhost", "editors"),
				createBinding("differentProjectBinding", "abcd", "bob@acme.com", "owners"),
			},
			expectedBindings: []*kubermaticv1.UserProjectBinding{
				createBinding("abcdBinding", "my-first-project-ID", "bob@acme.com", "owners"),
				createBinding("cdBinding", "my-first-project-ID", "bob@acme.com", "owners"),
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			kubermaticObjects := []ctrlruntimeclient.Object{}
			for _, binding := range tc.existingBindings {
				kubermaticObjects = append(kubermaticObjects, binding)
			}
			for _, sa := range tc.existingSA {
				kubermaticObjects = append(kubermaticObjects, sa)
			}
			fakeClient := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(kubermaticObjects...).
				Build()

			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return fakeClient, nil
			}
			// act
			target := kubernetes.NewProjectMemberProvider(fakeImpersonationClient, fakeClient)
			result, err := target.List(context.Background(), &provider.UserInfo{Email: tc.authenticatedUser.Spec.Email, Groups: []string{fmt.Sprintf("owners-%s", tc.projectToSync.Name)}}, tc.projectToSync, nil)

			// validate
			if err != nil {
				t.Fatal(err)
			}
			if len(tc.expectedBindings) != len(result) {
				t.Fatalf("expected to get %d bindings, but got %d", len(tc.expectedBindings), len(result))
			}
			for _, returnedBinding := range result {
				returnedBinding.ResourceVersion = ""
				bindingFound := false
				for _, expectedBinding := range tc.expectedBindings {
					expectedBinding.ResourceVersion = ""
					if diff := deep.Equal(returnedBinding, expectedBinding); diff == nil {
						bindingFound = true
						break
					}
				}
				if !bindingFound {
					t.Fatalf("returned binding was not found on the list of expected ones, binding = %#v", returnedBinding)
				}
			}
		})
	}
}

func TestRoleMapper(t *testing.T) {
	// test data
	testcases := []struct {
		name                         string
		authenticatedUser            *kubermaticv1.User
		existingUserProjectBindings  []*kubermaticv1.UserProjectBinding
		existingGroupProjectBindings []*kubermaticv1.GroupProjectBinding
		projectID                    string
		expectedRoles                sets.String
	}{
		{
			name:              "scenario 1: map user to roles from user binding and group bindings",
			authenticatedUser: createUserWithGroups("admins", "editors"),
			projectID:         "my-first-project-ID",
			existingUserProjectBindings: []*kubermaticv1.UserProjectBinding{
				createBinding("userBinding", "my-first-project-ID", "john@acme.com", "owners"),
			},
			existingGroupProjectBindings: []*kubermaticv1.GroupProjectBinding{
				genGroupProjectBinding("adminsBinding", "my-first-project-ID", "admins", "owners"),
				genGroupProjectBinding("devsBinding", "my-first-project-ID", "editors", "editors"),
				genGroupProjectBinding("guestBindingSecondProject", "my-second-project-ID", "viewers", "viewers"),
				genGroupProjectBinding("guestBinding", "my-first-project-ID", "guest", "viewers"),
			},
			expectedRoles: genRoles("owners", "editors"),
		},
		{
			name:              "scenario 2: map user to roles just from user binding",
			authenticatedUser: createUserWithGroups("admins", "editors"),
			projectID:         "my-first-project-ID",
			existingUserProjectBindings: []*kubermaticv1.UserProjectBinding{
				createBinding("userBinding", "my-first-project-ID", "john@acme.com", "viewers"),
			},
			existingGroupProjectBindings: []*kubermaticv1.GroupProjectBinding{},
			expectedRoles:                genRoles("viewers"),
		},
		{
			name:                        "scenario 3: map user to roles just from group bindings",
			authenticatedUser:           createUserWithGroups("admins", "editors"),
			projectID:                   "my-first-project-ID",
			existingUserProjectBindings: []*kubermaticv1.UserProjectBinding{},
			existingGroupProjectBindings: []*kubermaticv1.GroupProjectBinding{
				genGroupProjectBinding("adminsBinding", "my-first-project-ID", "admins", "owners"),
				genGroupProjectBinding("devsBinding", "my-first-project-ID", "editors", "editors"),
				genGroupProjectBinding("guestBindingSecondProject", "my-second-project-ID", "viewers", "viewers"),
				genGroupProjectBinding("guestBinding", "my-first-project-ID", "guest", "viewers"),
			},
			expectedRoles: genRoles("owners", "editors"),
		},
		{
			name:                         "scenario 4: map user to no bindings",
			authenticatedUser:            createUserWithGroups("admins", "editors"),
			projectID:                    "my-first-project-ID",
			existingUserProjectBindings:  []*kubermaticv1.UserProjectBinding{},
			existingGroupProjectBindings: []*kubermaticv1.GroupProjectBinding{},
			expectedRoles:                genRoles(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			kubermaticObjects := []ctrlruntimeclient.Object{}
			for _, upb := range tc.existingUserProjectBindings {
				kubermaticObjects = append(kubermaticObjects, upb)
			}
			for _, gpb := range tc.existingGroupProjectBindings {
				kubermaticObjects = append(kubermaticObjects, gpb)
			}
			fakeClient := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(kubermaticObjects...).
				Build()

			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return fakeClient, nil
			}
			// act
			pmp := kubernetes.NewProjectMemberProvider(fakeImpersonationClient, fakeClient)
			resultRoles, err := pmp.MapUserToRoles(context.Background(), tc.authenticatedUser, tc.projectID)
			if err != nil {
				t.Fatal(err)
			}

			if !resultRoles.Equal(tc.expectedRoles) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(resultRoles, tc.expectedRoles))
			}
		})
	}
}

func genRoles(roles ...string) sets.String {
	roleSet := sets.String{}
	roleSet.Insert(roles...)
	return roleSet
}

func createUserWithGroups(groups ...string) *kubermaticv1.User {
	user := createAuthenitactedUser()
	user.Spec.Groups = groups
	return user
}
