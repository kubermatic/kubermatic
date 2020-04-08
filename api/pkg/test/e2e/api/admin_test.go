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
