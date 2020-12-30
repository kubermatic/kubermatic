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
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/rand"
)

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
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("failed to get master token: %v", err)
			}

			apiRunner := createRunner(masterToken, t)
			project, err := apiRunner.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("failed to create project: %v", err)
			}
			defer cleanUpProject(t, project.ID)

			sa, err := apiRunner.CreateServiceAccount(rand.String(10), tc.group, project.ID)
			if err != nil {
				t.Fatalf("failed to create service account: %v", err)
			}

			if _, err := apiRunner.AddTokenToServiceAccount(rand.String(10), sa.ID, project.ID); err != nil {
				t.Fatalf("failed to create token: %v", err)
			}
		})
	}
}

func TestTokenAccessForProject(t *testing.T) {
	t.Parallel()
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
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("failed to get master token: %v", err)
			}

			apiRunner := createRunner(masterToken, t)
			project, err := apiRunner.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("failed to create project: %v", err)
			}
			defer cleanUpProject(t, project.ID)

			sa, err := apiRunner.CreateServiceAccount(rand.String(10), tc.group, project.ID)
			if err != nil {
				t.Fatalf("failed to create service account: %v", err)
			}

			sa, err = apiRunner.GetServiceAccount(sa.ID, project.ID)
			if err != nil {
				t.Fatalf("failed to get service account: %v", err)
			}

			saToken, err := apiRunner.AddTokenToServiceAccount(rand.String(10), sa.ID, project.ID)
			if err != nil {
				t.Fatalf("failed to create token: %v", err)
			}

			apiRunnerWithSAToken := createRunner(saToken.Token, t)

			project, err = apiRunnerWithSAToken.GetProject(project.ID)
			if err != nil {
				t.Fatalf("failed to get project: %v", err)
			}

			newProjectName := rand.String(10)
			project.Name = newProjectName

			project, err = apiRunnerWithSAToken.UpdateProject(project)

			if tc.group == "viewers" {
				if err == nil {
					t.Fatal("expected error")
				}

				if !strings.Contains(err.Error(), "403") {
					t.Fatalf("expected error status 403 Forbidden, but was: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("failed to update project: %v", err)
				}

				if project.Name != newProjectName {
					t.Fatalf("expected name %q, but got %q", newProjectName, project.Name)
				}
			}

			// check access to not owned project
			notOwnedProject, err := apiRunner.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("failed to create project: %v", err)
			}
			defer cleanUpProject(t, notOwnedProject.ID)

			_, err = apiRunnerWithSAToken.GetProject(notOwnedProject.ID)
			if err == nil {
				t.Fatal("expected error, SA token can't access not owned project")
			}

			if !strings.Contains(err.Error(), "403") {
				t.Fatalf("expected error status 403 Forbidden, but was: %v", err)
			}

			// check if SA can create a new project
			_, err = apiRunnerWithSAToken.CreateProject(rand.String(10))
			if err == nil {
				t.Fatal("expected error, SA can not create a project")
			}

			if !strings.Contains(err.Error(), "403") {
				t.Fatalf("expected error status 403 Forbidden, but was: %v", err)
			}

		})
	}
}
