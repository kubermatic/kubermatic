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
	"context"
	"testing"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
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

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := utils.RetrieveMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get master token: %v", err)
			}

			testClient := utils.NewTestClient(masterToken, t)
			project, err := testClient.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("failed to create project: %v", err)
			}
			defer cleanupProject(t, project.ID)

			projectUsers, err := testClient.GetProjectUsers(project.ID)
			if err != nil {
				t.Fatalf("failed to get the project user: %v", err)
			}

			if len(projectUsers) != len(tc.expectedUsers) {
				t.Fatal("the number of user is different than expected")
			}

			for _, user := range projectUsers {
				if !contains(tc.expectedUsers, user.Email) {
					t.Fatalf("the user %q doesn't belong to the expected user list", user.Email)
				}
			}

			err = testClient.DeleteUserFromProject(project.ID, projectUsers[0].ID)
			if err == nil {
				t.Fatal("expected error when delete owner of the project")
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

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := utils.RetrieveMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get master token: %v", err)
			}

			testClient := utils.NewTestClient(masterToken, t)
			project, err := testClient.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("failed to create project: %v", err)
			}
			defer cleanupProject(t, project.ID)

			_, err = testClient.AddProjectUser(project.ID, tc.newUserEmail, tc.newUserName, tc.newUserGroup)
			if err != nil {
				t.Fatalf("failed to add user to project: %v", err)
			}

			projectUsers, err := testClient.GetProjectUsers(project.ID)
			if err != nil {
				t.Fatalf("failed to get the project users: %v", err)
			}

			if len(projectUsers) != len(tc.expectedUsers) {
				t.Fatal("the number of user is different than expected")
			}

			for _, user := range projectUsers {
				if !contains(tc.expectedUsers, user.Email) {
					t.Fatalf("the user %q doesn't belong to the expected user list", user.Email)
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
