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
	"fmt"
	"testing"

	"github.com/go-test/deep"

	kubermaticfakeclentset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestCreateBinding(t *testing.T) {
	// test data
	kubermaticObjects := []runtime.Object{}
	impersonationClient, _, indexer, err := createFakeKubermaticClients(kubermaticObjects)
	if err != nil {
		t.Fatalf("unable to create fake clients, err = %v", err)
	}
	authenticatedUser := createAuthenitactedUser()
	existingProject := genDefaultProject()
	memberEmail := ""
	groupName := fmt.Sprintf("owners-%s", existingProject.Name)
	bindingLister := kubermaticv1lister.NewUserProjectBindingLister(indexer)
	userLister := kubermaticv1lister.NewUserLister(indexer)
	// act
	target := kubernetes.NewProjectMemberProvider(impersonationClient.CreateFakeImpersonatedClientSet, bindingLister, userLister, kubernetes.IsServiceAccount)
	result, err := target.Create(&provider.UserInfo{Email: authenticatedUser.Spec.Email, Group: fmt.Sprintf("owners-%s", existingProject.Name)}, existingProject, memberEmail, groupName)

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
				genServiceAccount("1", "test", "editors", "my-first-project-ID"),
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
			kubermaticObjects := []runtime.Object{}
			bindingObjects := []runtime.Object{}
			saObjects := []runtime.Object{}
			for _, binding := range tc.existingBindings {
				kubermaticObjects = append(kubermaticObjects, binding)
				bindingObjects = append(bindingObjects, binding)
			}
			for _, sa := range tc.existingSA {
				kubermaticObjects = append(kubermaticObjects, sa)
				saObjects = append(saObjects, sa)
			}
			kubermaticClient := kubermaticfakeclentset.NewSimpleClientset(kubermaticObjects...)
			impersonationClient := &fakeKubermaticImpersonationClient{kubermaticClient}

			bindingIndexer, err := createIndexer(bindingObjects)
			if err != nil {
				t.Fatal(err)
			}
			saIndexer, err := createIndexer(saObjects)
			if err != nil {
				t.Fatal(err)
			}

			bindingLister := kubermaticv1lister.NewUserProjectBindingLister(bindingIndexer)
			userLister := kubermaticv1lister.NewUserLister(saIndexer)
			// act
			target := kubernetes.NewProjectMemberProvider(impersonationClient.CreateFakeImpersonatedClientSet, bindingLister, userLister, kubernetes.IsServiceAccount)
			result, err := target.List(&provider.UserInfo{Email: tc.authenticatedUser.Spec.Email, Group: fmt.Sprintf("owners-%s", tc.projectToSync.Name)}, tc.projectToSync, nil)

			// validate
			if err != nil {
				t.Fatal(err)
			}
			if len(tc.expectedBindings) != len(result) {
				t.Fatalf("expected to get %d bindings, but got %d", len(tc.expectedBindings), len(result))
			}
			for _, returnedBinding := range result {
				bindingFound := false
				for _, expectedBinding := range tc.expectedBindings {
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
