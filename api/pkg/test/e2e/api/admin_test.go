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
	"reflect"
	"testing"
	"time"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"

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
			name:                        "admin can get other users projects",
			expectedProjectsNumber:      1,
			expectedAdminProjectsNumber: 0,
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
			teardown := cleanUpProject(project.ID, 3)
			defer teardown(t)

			// change for admin user
			adminMasterToken, err := retrieveAdminMasterToken()
			if err != nil {
				t.Fatalf("can not get admin master token due error: %v", err)
			}

			adminAPIRunner := createRunner(adminMasterToken, t)

			_, err = adminAPIRunner.GetProject(project.ID, 1)
			if err != nil {
				t.Fatalf("admin can not get other user project: %v", err)
			}
			projects, err := adminAPIRunner.ListProjects(true, 1)
			if err != nil {
				t.Fatalf("admin can not list projects: %v", err)
			}
			if len(projects) != tc.expectedProjectsNumber {
				t.Fatalf("expected projects number: %d got %d", tc.expectedProjectsNumber, len(projects))
			}

			// get only admin projects
			projects, err = adminAPIRunner.ListProjects(false, 1)
			if err != nil {
				t.Fatalf("admin can not list projects: %v", err)
			}
			if len(projects) != tc.expectedAdminProjectsNumber {
				t.Fatalf("expected projects number: %d got %d", tc.expectedAdminProjectsNumber, len(projects))
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
			teardown := cleanUpProject(project.ID, 3)
			defer teardown(t)

			// change for admin user
			adminMasterToken, err := retrieveAdminMasterToken()
			if err != nil {
				t.Fatalf("can not get admin master token due error: %v", err)
			}

			adminAPIRunner := createRunner(adminMasterToken, t)
			project.Name = tc.newProjectName
			_, err = adminAPIRunner.UpdateProject(project)
			if err != nil {
				t.Fatalf("admin can not update other user project: %v", err)
			}
			updatedProject, err := adminAPIRunner.GetProject(project.ID, 1)
			if err != nil {
				t.Fatalf("admin can not get other user project: %v", err)
			}
			if updatedProject.Name != tc.newProjectName {
				t.Fatalf("expected new name %s got %s", tc.newProjectName, updatedProject.Name)
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

			// change for admin user
			adminMasterToken, err := retrieveAdminMasterToken()
			if err != nil {
				t.Fatalf("can not get admin master token due error: %v", err)
			}

			adminAPIRunner := createRunner(adminMasterToken, t)
			err = adminAPIRunner.DeleteProject(project.ID)
			if err != nil {
				t.Fatalf("admin can not delete other user project: %v", err)
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
			name:  "admin can create SA for other users projects",
			group: "editors",
		},
		{
			name:  "admin can create SA for other users projects",
			group: "viewers",
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
			teardown := cleanUpProject(project.ID, 3)
			defer teardown(t)

			// change for admin user
			adminMasterToken, err := retrieveAdminMasterToken()
			if err != nil {
				t.Fatalf("can not get admin master token due error: %v", err)
			}

			adminAPIRunner := createRunner(adminMasterToken, t)
			sa, err := adminAPIRunner.CreateServiceAccount(rand.String(10), tc.group, project.ID)
			if err != nil {
				t.Fatalf("can not create service account due error: %v", err)
			}
			saToken, err := apiRunner.AddTokenToServiceAccount(rand.String(10), sa.ID, project.ID)
			if err != nil {
				t.Fatalf("can not create token due error: %v", err)
			}

			if err := adminAPIRunner.DeleteTokenForServiceAccount(saToken.ID, sa.ID, project.ID); err != nil {
				t.Fatalf("can not delete token due error: %v", err)
			}
			if err := adminAPIRunner.DeleteServiceAccount(sa.ID, project.ID); err != nil {
				t.Fatalf("can not delete service account due error: %v", err)
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
			teardown := cleanUpProject(project.ID, 3)
			defer teardown(t)

			// change for admin user
			adminMasterToken, err := retrieveAdminMasterToken()
			if err != nil {
				t.Fatalf("can not get admin master token due error: %v", err)
			}

			adminAPIRunner := createRunner(adminMasterToken, t)
			projectUsers, err := adminAPIRunner.GetProjectUsers(project.ID)
			if err != nil {
				t.Fatalf("can not get the project user due error: %v", err)
			}

			if len(projectUsers) != len(tc.expectedUsers) {
				t.Fatalf("expected %d got %d: %v", len(tc.expectedUsers), len(projectUsers), projectUsers)
			}

			for _, user := range projectUsers {
				if !tc.expectedUsers.Has(user.Email) {
					t.Fatalf("the user %s doesn't belong to the expected user list", user.Email)
				}
			}
			err = adminAPIRunner.DeleteUserFromProject(project.ID, projectUsers[0].ID)
			if err != nil {
				t.Fatalf("admin can not delete user from the project %v", err)
			}

		})
	}
}

func TestManageClusterByAdmin(t *testing.T) {
	tests := []struct {
		name           string
		dc             string
		location       string
		version        string
		credential     string
		replicas       int32
		patch          PatchCluster
		expectedName   string
		expectedLabels map[string]string
	}{
		{
			name:       "create cluster on DigitalOcean",
			dc:         "kubermatic",
			location:   "do-fra1",
			version:    "v1.15.6",
			credential: "e2e-digitalocean",
			replicas:   1,
			patch: PatchCluster{
				Name:   "newName",
				Labels: map[string]string{"a": "b"},
			},
			expectedName:   "newName",
			expectedLabels: map[string]string{"a": "b"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var maxAttempts = 24
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token %v", err)
			}

			apiRunner := createRunner(masterToken, t)
			project, err := apiRunner.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("can not create project %v", err)
			}
			teardown := cleanUpProject(project.ID, 5)
			defer teardown(t)

			cluster, err := apiRunner.CreateDOCluster(project.ID, tc.dc, rand.String(10), tc.credential, tc.version, tc.location, tc.replicas)
			if err != nil {
				t.Fatalf("can not create cluster due to error: %v", err)
			}
			// change for admin user
			adminMasterToken, err := retrieveAdminMasterToken()
			if err != nil {
				t.Fatalf("can not get admin master token due error: %v", err)
			}

			adminAPIRunner := createRunner(adminMasterToken, t)
			var clusterReady bool
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				healthStatus, err := adminAPIRunner.GetClusterHealthStatus(project.ID, tc.dc, cluster.ID)
				if err != nil {
					t.Fatalf("can not get health status %v", GetErrorResponse(err))
				}

				if IsHealthyCluster(healthStatus) {
					clusterReady = true
					break
				}
				time.Sleep(30 * time.Second)
			}

			if !clusterReady {
				t.Fatalf("cluster not ready after %d attempts", maxAttempts)
			}

			var ndReady bool
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				ndList, err := adminAPIRunner.GetClusterNodeDeployment(project.ID, tc.dc, cluster.ID)
				if err != nil {
					t.Fatalf("can not get node deployments %v", GetErrorResponse(err))
				}

				if len(ndList) == 1 {
					ndReady = true
					break
				}
				time.Sleep(30 * time.Second)
			}
			if !ndReady {
				t.Fatalf("node deployment is not redy after %d attempts", maxAttempts)
			}

			var replicasReady bool
			var ndList []apiv1.NodeDeployment
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				ndList, err = adminAPIRunner.GetClusterNodeDeployment(project.ID, tc.dc, cluster.ID)
				if err != nil {
					t.Fatalf("can not get node deployments %v", GetErrorResponse(err))
				}

				if ndList[0].Status.AvailableReplicas == tc.replicas {
					replicasReady = true
					break
				}
				time.Sleep(30 * time.Second)
			}
			if !replicasReady {
				t.Fatalf("the number of nodes is not as expected, available replicas %d", ndList[0].Status.AvailableReplicas)
			}

			_, err = adminAPIRunner.UpdateCluster(project.ID, tc.dc, cluster.ID, tc.patch)
			if err != nil {
				t.Fatalf("can not update cluster %v", GetErrorResponse(err))
			}

			updatedCluster, err := adminAPIRunner.GetCluster(project.ID, tc.dc, cluster.ID)
			if err != nil {
				t.Fatalf("can not get cluster %v", GetErrorResponse(err))
			}

			if updatedCluster.Name != tc.expectedName {
				t.Fatalf("expected new name %s got %s", tc.expectedName, updatedCluster.Name)
			}

			if !equality.Semantic.DeepEqual(updatedCluster.Labels, tc.expectedLabels) {
				t.Fatalf("expected labels %v got %v", tc.expectedLabels, updatedCluster.Labels)
			}

			cleanUpCluster(t, apiRunner, project.ID, tc.dc, cluster.ID)

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
			publicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== lukasz@loodse.com ",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due to error: %v", err)
			}

			apiRunner := createRunner(masterToken, t)
			project, err := apiRunner.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("can not create project due error: %v", err)
			}
			teardown := cleanUpProject(project.ID, 1)
			defer teardown(t)

			// change for admin user
			adminMasterToken, err := retrieveAdminMasterToken()
			if err != nil {
				t.Fatalf("can not get admin master token due error: %v", err)
			}

			adminAPIRunner := createRunner(adminMasterToken, t)
			sshKey, err := adminAPIRunner.CreateUserSSHKey(project.ID, tc.keyName, tc.publicKey)
			if err != nil {
				t.Fatalf("can not get create SSH key due error: %v", err)
			}
			sshKeys, err := adminAPIRunner.ListUserSSHKey(project.ID)
			if err != nil {
				t.Fatalf("can not list SSH keys due error: %v", err)
			}
			if len(sshKeys) != 1 {
				t.Fatalf("expected one SSH key, got %v", sshKeys)
			}
			if !reflect.DeepEqual(sshKeys[0], sshKey) {
				t.Fatalf("expected %v, got %v", sshKey, sshKeys[0])
			}
			if err := adminAPIRunner.DeleteUserSSHKey(project.ID, sshKey.ID); err != nil {
				t.Fatalf("can not delete SSH key due error: %v", err)
			}
			sshKeys, err = adminAPIRunner.ListUserSSHKey(project.ID)
			if err != nil {
				t.Fatalf("can not list SSH keys due error: %v", err)
			}
			if len(sshKeys) != 0 {
				t.Fatalf("found SSH key")
			}
		})
	}
}
