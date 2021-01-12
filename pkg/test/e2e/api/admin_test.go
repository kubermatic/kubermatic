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
	"reflect"
	"testing"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestGetProjectByAdmin(t *testing.T) {
	tests := []struct {
		name                        string
		expectedProjectsNumber      int
		expectedAdminProjectsNumber int
	}{
		{
			name:                        "admin can get other users' projects",
			expectedProjectsNumber:      1,
			expectedAdminProjectsNumber: 0,
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

			// change for admin user
			adminMasterToken, err := utils.RetrieveAdminMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get admin master token: %v", err)
			}

			adminTestClient := utils.NewTestClient(adminMasterToken, t)

			_, err = adminTestClient.GetProject(project.ID)
			if err != nil {
				t.Fatalf("admin failed to get other user project: %v", err)
			}

			projects, err := adminTestClient.ListProjects(true)
			if err != nil {
				t.Fatalf("admin failed to list projects: %v", err)
			}
			if len(projects) != tc.expectedProjectsNumber {
				t.Fatalf("expected %d projects, but got %d", tc.expectedProjectsNumber, len(projects))
			}

			// get only admin projects
			projects, err = adminTestClient.ListProjects(false)
			if err != nil {
				t.Fatalf("admin failed to list projects: %v", err)
			}
			if len(projects) != tc.expectedAdminProjectsNumber {
				t.Fatalf("expected %d projects, but got %d", tc.expectedAdminProjectsNumber, len(projects))
			}
		})
	}
}

func TestUpdateProjectByAdmin(t *testing.T) {
	tests := []struct {
		name           string
		newProjectName string
	}{
		{
			name:           "admin can update other users projects",
			newProjectName: "admin-test-project",
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

			// change for admin user
			adminMasterToken, err := utils.RetrieveAdminMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get admin master token: %v", err)
			}

			adminTestClient := utils.NewTestClient(adminMasterToken, t)
			project.Name = tc.newProjectName
			_, err = adminTestClient.UpdateProject(project)
			if err != nil {
				t.Fatalf("admin failed to update other user's project: %v", err)
			}

			updatedProject, err := adminTestClient.GetProject(project.ID)
			if err != nil {
				t.Fatalf("admin failed to get other user's project: %v", err)
			}
			if updatedProject.Name != tc.newProjectName {
				t.Fatalf("expected new name %q, but got %q", tc.newProjectName, updatedProject.Name)
			}
		})
	}
}

func TestDeleteProjectByAdmin(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "admin can delete other users projects",
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

			// change for admin user
			adminTestClient, err := utils.RetrieveAdminMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get admin master token: %v", err)
			}

			adminAPIRunner := utils.NewTestClient(adminTestClient, t)
			err = adminAPIRunner.DeleteProject(project.ID)
			if err != nil {
				t.Fatalf("admin failed to delete other user's project: %v", err)
			}
		})
	}
}

func TestCreateAndDeleteServiceAccountByAdmin(t *testing.T) {
	tests := []struct {
		name  string
		group string
	}{
		{
			name:  "admin can create editor SA for other users' projects",
			group: "editors",
		},
		{
			name:  "admin can create viewer SA for other users' projects",
			group: "viewers",
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

			// change for admin user
			adminMasterToken, err := utils.RetrieveAdminMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get admin master token: %v", err)
			}

			adminTestClient := utils.NewTestClient(adminMasterToken, t)

			sa, err := adminTestClient.CreateServiceAccount(rand.String(10), tc.group, project.ID)
			if err != nil {
				t.Fatalf("failed to create service account: %v", err)
			}
			saToken, err := testClient.AddTokenToServiceAccount(rand.String(10), sa.ID, project.ID)
			if err != nil {
				t.Fatalf("failed to create token: %v", err)
			}
			if err := adminTestClient.DeleteTokenForServiceAccount(saToken.ID, sa.ID, project.ID); err != nil {
				t.Fatalf("failed to delete token: %v", err)
			}
			if err := adminTestClient.DeleteServiceAccount(sa.ID, project.ID); err != nil {
				t.Fatalf("failed to delete service account: %v", err)
			}
		})
	}
}

func TestManageProjectMembersByAdmin(t *testing.T) {
	tests := []struct {
		name          string
		group         string
		expectedUsers sets.String
	}{
		{
			name:          "admin can manage project members for any project",
			expectedUsers: sets.NewString("roxy@loodse.com"),
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

			// change for admin user
			adminMasterToken, err := utils.RetrieveAdminMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get admin master token: %v", err)
			}

			adminTestClient := utils.NewTestClient(adminMasterToken, t)
			projectUsers, err := adminTestClient.GetProjectUsers(project.ID)
			if err != nil {
				t.Fatalf("failed to get the project user: %v", err)
			}

			if len(projectUsers) != len(tc.expectedUsers) {
				t.Fatalf("expected %d users, but got %d: %v", len(tc.expectedUsers), len(projectUsers), projectUsers)
			}

			for _, user := range projectUsers {
				if !tc.expectedUsers.Has(user.Email) {
					t.Fatalf("the user %q doesn't belong to the expected user list", user.Email)
				}
			}

			err = adminTestClient.DeleteUserFromProject(project.ID, projectUsers[0].ID)
			if err != nil {
				t.Fatalf("admin failed to delete user from the project: %v", err)
			}
		})
	}
}

// creates project + cluster + nodes
func TestManageClusterByAdmin(t *testing.T) {
	tests := []struct {
		name           string
		dc             string
		location       string
		version        string
		credential     string
		replicas       int32
		patch          utils.PatchCluster
		expectedName   string
		expectedLabels map[string]string
	}{
		{
			name:       "create cluster on DigitalOcean",
			dc:         "kubermatic",
			location:   "do-fra1",
			version:    utils.KubernetesVersion(),
			credential: "e2e-digitalocean",
			replicas:   0,
			patch: utils.PatchCluster{
				Name:   "newName",
				Labels: map[string]string{"a": "b"},
			},
			expectedName:   "newName",
			expectedLabels: map[string]string{"a": "b"},
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

			cluster, err := testClient.CreateDOCluster(project.ID, tc.dc, rand.String(10), tc.credential, tc.version, tc.location, tc.replicas)
			if err != nil {
				t.Fatalf("failed to create cluster: %v", err)
			}

			// change for admin user
			adminMasterToken, err := utils.RetrieveAdminMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get admin master token: %v", err)
			}

			adminTestClient := utils.NewTestClient(adminMasterToken, t)

			if err := adminTestClient.WaitForClusterHealthy(project.ID, tc.dc, cluster.ID); err != nil {
				t.Fatalf("cluster not ready: %v", err)
			}

			_, err = adminTestClient.UpdateCluster(project.ID, tc.dc, cluster.ID, tc.patch)
			if err != nil {
				t.Fatalf("failed to update cluster: %v", err)
			}

			updatedCluster, err := adminTestClient.GetCluster(project.ID, tc.dc, cluster.ID)
			if err != nil {
				t.Fatalf("failed to get cluster: %v", err)
			}

			if updatedCluster.Name != tc.expectedName {
				t.Fatalf("expected new name %q, but got %q", tc.expectedName, updatedCluster.Name)
			}

			if !equality.Semantic.DeepEqual(updatedCluster.Labels, tc.expectedLabels) {
				t.Fatalf("expected labels %v, but got %v", tc.expectedLabels, updatedCluster.Labels)
			}

			testClient.CleanupCluster(t, project.ID, tc.dc, cluster.ID)
		})
	}
}

func TestManageSSHKeyByAdmin(t *testing.T) {
	tests := []struct {
		name      string
		keyName   string
		publicKey string
	}{
		{
			name:      "admin can manage SSH keys for any project",
			keyName:   "test",
			publicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== user@example.com ",
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

			// change for admin user
			adminMasterToken, err := utils.RetrieveAdminMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get admin master token: %v", err)
			}

			adminTestClient := utils.NewTestClient(adminMasterToken, t)
			sshKey, err := adminTestClient.CreateUserSSHKey(project.ID, tc.keyName, tc.publicKey)
			if err != nil {
				t.Fatalf("failed to get create SSH key: %v", err)
			}
			sshKeys, err := adminTestClient.ListUserSSHKey(project.ID)
			if err != nil {
				t.Fatalf("failed to list SSH keys: %v", err)
			}
			if len(sshKeys) != 1 {
				t.Fatalf("expected one SSH key, got %v", sshKeys)
			}
			if !reflect.DeepEqual(sshKeys[0], sshKey) {
				t.Fatalf("expected %v, got %v", sshKey, sshKeys[0])
			}
			if err := adminTestClient.DeleteUserSSHKey(project.ID, sshKey.ID); err != nil {
				t.Fatalf("failed to delete SSH key: %v", err)
			}
			sshKeys, err = adminTestClient.ListUserSSHKey(project.ID)
			if err != nil {
				t.Fatalf("failed to list SSH keys: %v", err)
			}
			if len(sshKeys) != 0 {
				t.Fatalf("found SSH key, even though it should have been removed")
			}
		})
	}
}
