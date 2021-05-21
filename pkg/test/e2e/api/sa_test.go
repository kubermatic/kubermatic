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
	"strings"
	"testing"

	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	"k8s.io/apimachinery/pkg/util/rand"
)

func TestCreateSA(t *testing.T) {
	tests := []struct {
		name  string
		group string
	}{
		{
			name:  "create SA with token for editors group",
			group: rbac.EditorGroupNamePrefix,
		},
		{
			name:  "create SA with token for viewers group",
			group: rbac.ViewerGroupNamePrefix,
		},
		{
			name:  "create SA with token for projectmanagers group",
			group: rbac.ProjectManagerGroupNamePrefix,
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

			sa, err := testClient.CreateServiceAccount(rand.String(10), tc.group, project.ID)
			if err != nil {
				t.Fatalf("failed to create service account: %v", err)
			}

			if _, err := testClient.AddTokenToServiceAccount(rand.String(10), sa.ID, project.ID); err != nil {
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
			group: rbac.EditorGroupNamePrefix,
		},
		{
			name:  "test project access when token has viewer rights",
			group: rbac.ViewerGroupNamePrefix,
		},
		{
			name:  "test project access when token has projectmanagers rights",
			group: rbac.ProjectManagerGroupNamePrefix,
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

			sa, err := testClient.CreateServiceAccount(rand.String(10), tc.group, project.ID)
			if err != nil {
				t.Fatalf("failed to create service account: %v", err)
			}

			sa, err = testClient.GetServiceAccount(sa.ID, project.ID)
			if err != nil {
				t.Fatalf("failed to get service account: %v", err)
			}

			saToken, err := testClient.AddTokenToServiceAccount(rand.String(10), sa.ID, project.ID)
			if err != nil {
				t.Fatalf("failed to create token: %v", err)
			}

			apiRunnerWithSAToken := utils.NewTestClient(saToken.Token, t)

			project, err = apiRunnerWithSAToken.GetProject(project.ID)
			if err != nil {
				t.Fatalf("failed to get project: %v", err)
			}

			// check if SA can add member to the project
			_, err = apiRunnerWithSAToken.AddProjectUser(project.ID, "roxy2@loodse.com", "roxy2", "viewers")
			switch tc.group {
			case rbac.ViewerGroupNamePrefix:
			case rbac.EditorGroupNamePrefix:
				if err == nil {
					t.Fatalf("expected error, SA can not manage members for the group %s", tc.group)
				}
				if !strings.Contains(err.Error(), "403") {
					t.Fatalf("expected error status 403 Forbidden, but was: %v", err)
				}
			case rbac.ProjectManagerGroupNamePrefix:
				if err != nil {
					t.Fatal("SA in projectmanagers group should add a new member to the project")
				}
			}

			// check update project
			newProjectName := rand.String(10)
			project.Name = newProjectName
			project, err = apiRunnerWithSAToken.UpdateProject(project)
			switch tc.group {
			case rbac.ViewerGroupNamePrefix:
				if err == nil {
					t.Fatal("expected error")
				}

				if !strings.Contains(err.Error(), "403") {
					t.Fatalf("expected error status 403 Forbidden, but was: %v", err)
				}
			case rbac.EditorGroupNamePrefix:
			case rbac.ProjectManagerGroupNamePrefix:
				if err != nil {
					t.Fatalf("failed to update project: %v", err)
				}
				if project.Name != newProjectName {
					t.Fatalf("expected name %q, but got %q", newProjectName, project.Name)
				}
			}

			// check if SA can create a new project
			switch tc.group {
			case rbac.ViewerGroupNamePrefix:
			case rbac.EditorGroupNamePrefix:
				_, err := apiRunnerWithSAToken.CreateProject(rand.String(10))
				if err == nil {
					t.Fatal("expected error, SA can not create a project")
				}
				if !strings.Contains(err.Error(), "403") {
					t.Fatalf("expected error status 403 Forbidden, but was: %v", err)
				}
			case rbac.ProjectManagerGroupNamePrefix:
				newSAproject, err := apiRunnerWithSAToken.CreateProjectBySA(rand.String(10), []string{"roxy2@loodse.com", "roxy@loodse.com"})
				if err != nil {
					t.Fatalf("service account in projectmanagers should create a project %v", err)
				}
				defer cleanupProject(t, newSAproject.ID)
			}

			// check access to not owned project
			notOwnedProject, err := testClient.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("failed to create project: %v", err)
			}
			defer cleanupProject(t, notOwnedProject.ID)

			_, err = apiRunnerWithSAToken.GetProject(notOwnedProject.ID)
			if err == nil {
				t.Fatal("expected error, SA token can't access not owned project")
			}

			if !strings.Contains(err.Error(), "403") {
				t.Fatalf("expected error status 403 Forbidden, but was: %v", err)
			}

		})
	}
}
