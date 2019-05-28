// +build e2e

package e2e

import (
	"k8s.io/apimachinery/pkg/util/rand"
	"strings"
	"testing"
)

func cleanUpProject(id string) func(t *testing.T) {
	return func(t *testing.T) {
		masterToken, err := GetMasterToken()
		if err != nil {
			t.Fatalf("can not get master token due error: %v", err)
		}
		apiRunner := CreateAPIRunner(masterToken, t)

		if err := apiRunner.DeleteProject(id); err != nil {
			t.Fatalf("can not delete project due error: %v", err)
		}
	}
}

func TestCreateSA(t *testing.T) {
	tests := []struct {
		name  string
		group string
	}{
		{
			name:  "create SA with token for editors group",
			group: "editors",
		},
		{
			name:  "create SA with token for viewers group",
			group: "viewers",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := GetMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			apiRunner := CreateAPIRunner(masterToken, t)
			project, err := apiRunner.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("can not create project due error: %v", err)
			}
			teardown := cleanUpProject(project.ID)
			defer teardown(t)

			sa, err := apiRunner.CreateServiceAccount(rand.String(10), tc.group, project.ID)
			if err != nil {
				t.Fatalf("can not create service account due error: %v", err)
			}

			if _, err := apiRunner.AddTokenToServiceAccount(rand.String(10), sa.ID, project.ID); err != nil {
				t.Fatalf("can not create token due error: %v", err)
			}
		})
	}
}

func TestTokenAccessForProject(t *testing.T) {
	tests := []struct {
		name  string
		group string
	}{
		{
			name:  "test project access when token has editor rights",
			group: "editors",
		},
		{
			name:  "test project access when token has viewer rights",
			group: "viewers",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := GetMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}
			apiRunner := CreateAPIRunner(masterToken, t)
			project, err := apiRunner.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("can not create project due error: %v", err)
			}
			teardown := cleanUpProject(project.ID)
			defer teardown(t)

			sa, err := apiRunner.CreateServiceAccount(rand.String(10), tc.group, project.ID)
			if err != nil {
				t.Fatalf("can not create service account due error: %v", err)
			}

			saToken, err := apiRunner.AddTokenToServiceAccount(rand.String(10), sa.ID, project.ID)
			if err != nil {
				t.Fatalf("can not create token due error: %v", err)
			}

			apiRunnerWithSAToken := CreateAPIRunner(saToken.Token, t)

			project, err = apiRunnerWithSAToken.GetProject(project.ID, 1)
			if err != nil {
				t.Fatalf("can not get project due error: %v", err)
			}

			newProjectName := rand.String(10)
			project.Name = newProjectName

			project, err = apiRunnerWithSAToken.UpdateProject(project)

			if tc.group == "viewers" {
				if err == nil {
					t.Fatalf("expected error")
				}

				if !strings.Contains(err.Error(), "403") {
					t.Fatalf("expected error status 403 Forbidden was %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("can not update project due error: %v", err)
				}

				if project.Name != newProjectName {
					t.Fatalf("expected name %s got %s", newProjectName, project.Name)
				}
			}

			// check access to not owned project
			notOwnedProject, err := apiRunner.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("can not create project due error: %v", err)
			}
			teardownNotOwnedProject := cleanUpProject(notOwnedProject.ID)
			defer teardownNotOwnedProject(t)

			_, err = apiRunnerWithSAToken.GetProject(notOwnedProject.ID, 1)
			if err == nil {
				t.Fatalf("expected error, SA token can't access not owned project")
			}

			if !strings.Contains(err.Error(), "403") {
				t.Fatalf("expected error status 403 Forbidden was %v", err)
			}
		})
	}
}
