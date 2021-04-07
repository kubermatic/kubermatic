// +build e2e

/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	"k8s.io/apimachinery/pkg/util/rand"
)

func TestCreateAutomationAccount(t *testing.T) {
	tests := []struct {
		name  string
		group string
	}{
		{
			name:  "create Automation Account with token for owners group",
			group: "owners",
		},
		{
			name:  "create Automation Account with token for editors group",
			group: "editors",
		},
		{
			name:  "create Automation Account with token for viewers group",
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

			sa, err := testClient.CreateAutomationAccount(rand.String(10), tc.group)
			if err != nil {
				t.Fatalf("failed to create automation account: %v", err)
			}

			if _, err := testClient.AddTokenToAutomationAccount(rand.String(10), sa.ID); err != nil {
				t.Fatalf("failed to create token: %v", err)
			}
		})
	}
}

func TestAutomationAccountTokenAccessForProject(t *testing.T) {
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
		{
			name:  "test project access when token has owner rights",
			group: "owners",
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

			sa, err := testClient.CreateAutomationAccount(rand.String(10), tc.group)
			if err != nil {
				t.Fatalf("failed to create automation account: %v", err)
			}

			saToken, err := testClient.AddTokenToAutomationAccount(rand.String(10), sa.ID)
			if err != nil {
				t.Fatalf("failed to create token: %v", err)
			}

			apiRunnerWithSAToken := utils.NewTestClient(saToken.Token, t)

			project, err := apiRunnerWithSAToken.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("failed to create project: %v", err)
			}
			defer cleanupProject(t, project.ID)

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

		})
	}
}
