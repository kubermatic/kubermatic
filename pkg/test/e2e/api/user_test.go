// +build e2e

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

package api

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/rand"
)

func TestDeleteProjectOwner(t *testing.T) {
	tests := []struct {
		name          string
		expectedUsers []string
	}{
		{
			name:          "test, delete project owner",
			expectedUsers: []string{"roxy@loodse.com"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			apiRunner := createRunner(masterToken, t)
			project, err := apiRunner.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("can not create project due error: %v", GetErrorResponse(err))
			}
			teardown := cleanUpProject(project.ID, 10)
			defer teardown(t)

			projectUsers, err := apiRunner.GetProjectUsers(project.ID)
			if err != nil {
				t.Fatalf("can not get the project user due error: %v", err)
			}

			if len(projectUsers) != len(tc.expectedUsers) {
				t.Fatalf("the number of user is different than expected")
			}

			for _, user := range projectUsers {
				if !contains(tc.expectedUsers, user.Email) {
					t.Fatalf("the user %s doesn't belong to the expected user list", user.Email)
				}
			}

			err = apiRunner.DeleteUserFromProject(project.ID, projectUsers[0].ID)
			if err == nil {
				t.Fatalf("expected error when delete owner of the project")
			}

		})
	}
}

func TestAddUserToProject(t *testing.T) {
	tests := []struct {
		name          string
		newUserEmail  string
		newUserName   string
		newUserGroup  string
		expectedUsers []string
	}{
		{
			name:          "test, add user to project",
			newUserEmail:  "roxy2@loodse.com",
			newUserName:   "roxy2",
			newUserGroup:  "viewers",
			expectedUsers: []string{"roxy@loodse.com", "roxy2@loodse.com"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			apiRunner := createRunner(masterToken, t)
			project, err := apiRunner.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("can not create project due error: %v", err)
			}
			teardown := cleanUpProject(project.ID, 10)
			defer teardown(t)

			_, err = apiRunner.AddProjectUser(project.ID, tc.newUserEmail, tc.newUserName, tc.newUserGroup)
			if err != nil {
				t.Fatalf("can not add user to project due error: %v", GetErrorResponse(err))
			}

			projectUsers, err := apiRunner.GetProjectUsers(project.ID)
			if err != nil {
				t.Fatalf("can not get the project user due error: %v", err)
			}

			if len(projectUsers) != len(tc.expectedUsers) {
				t.Fatalf("the number of user is different than expected")
			}

			for _, user := range projectUsers {
				if !contains(tc.expectedUsers, user.Email) {
					t.Fatalf("the user %s doesn't belong to the expected user list", user.Email)
				}
			}

		})
	}
}

func contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}
