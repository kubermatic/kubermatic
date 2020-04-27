// +build e2e

package api

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/rand"
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
			teardown := cleanUpProject(project.ID, 1)
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
			teardown := cleanUpProject(project.ID, 1)
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
			teardown := cleanUpProject(project.ID, 1)
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
