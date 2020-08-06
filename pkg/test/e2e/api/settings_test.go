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
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/e2e/api/utils/apiclient/client/project"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/rand"
)

func TestGetDefaultGlobalSettings(t *testing.T) {
	tests := []struct {
		name             string
		expectedSettings *apiv1.GlobalSettings
	}{
		{
			name: "test, gets default settings",
			expectedSettings: &apiv1.GlobalSettings{

				CustomLinks: []kubermaticv1.CustomLink{},
				CleanupOptions: kubermaticv1.CleanupOptions{
					Enabled:  false,
					Enforced: false,
				},
				DefaultNodeCount:        10,
				ClusterTypeOptions:      kubermaticv1.ClusterTypeKubernetes,
				DisplayDemoInfo:         false,
				DisplayAPIDocs:          false,
				DisplayTermsOfService:   false,
				EnableDashboard:         true,
				EnableOIDCKubeconfig:    false,
				UserProjectsLimit:       0,
				RestrictProjectCreation: false,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var masterToken string

			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}
			apiRunner := createRunner(masterToken, t)

			settings, err := apiRunner.GetGlobalSettings()
			if err != nil {
				t.Fatalf("can not get global settings: %v", err)
			}
			if !equality.Semantic.DeepEqual(tc.expectedSettings, settings) {
				t.Fatalf("expected: %v, got %v", tc.expectedSettings, settings)
			}

		})
	}
}

func TestUserProjectsLimit(t *testing.T) {
	tests := []struct {
		name          string
		projectsLimit int
	}{
		{
			name:          "test, user reached maximum number of projects",
			projectsLimit: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var masterToken string

			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}
			apiRunner := createRunner(masterToken, t)
			// change for admin user
			adminMasterToken, err := retrieveAdminMasterToken()
			if err != nil {
				t.Fatalf("can not get admin master token due error: %v", err)
			}

			adminAPIRunner := createRunner(adminMasterToken, t)
			_, err = adminAPIRunner.UpdateGlobalSettings(json.RawMessage(fmt.Sprintf(`{"userProjectsLimit":%d}`, tc.projectsLimit)))
			if err != nil {
				t.Fatalf("can not update global settings: %v", GetErrorResponse(err))
			}

			for i := 0; i < (tc.projectsLimit + 1); i++ {
				_, err := apiRunner.CreateProject(rand.String(10))
				if err != nil && i < tc.projectsLimit {
					t.Fatalf("can not create project %v", GetErrorResponse(err))
				}
				if err == nil && i > tc.projectsLimit {
					t.Fatalf("expected error during cluster creation")
				}
			}
			_, err = adminAPIRunner.UpdateGlobalSettings(json.RawMessage(fmt.Sprintf(`{"userProjectsLimit":%d}`, 0)))
			if err != nil {
				t.Fatalf("can not update global settings: %v", GetErrorResponse(err))
			}

		})
	}
}

func TestAdminUserProjectsLimit(t *testing.T) {
	tests := []struct {
		name          string
		projectsLimit int
	}{
		{
			name:          "test, admin doesn't reach maximum number of projects",
			projectsLimit: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			adminMasterToken, err := retrieveAdminMasterToken()
			if err != nil {
				t.Fatalf("can not get admin master token due error: %v", err)
			}

			adminAPIRunner := createRunner(adminMasterToken, t)
			_, err = adminAPIRunner.UpdateGlobalSettings(json.RawMessage(fmt.Sprintf(`{"userProjectsLimit":%d}`, tc.projectsLimit)))
			if err != nil {
				t.Fatalf("can not update global settings: %v", GetErrorResponse(err))
			}

			for i := 0; i < (tc.projectsLimit + 1); i++ {
				_, err := adminAPIRunner.CreateProject(rand.String(10))
				if err != nil {
					t.Fatalf("can not create project %v", GetErrorResponse(err))
				}
			}
			_, err = adminAPIRunner.UpdateGlobalSettings(json.RawMessage(fmt.Sprintf(`{"userProjectsLimit":%d}`, 0)))
			if err != nil {
				t.Fatalf("can not update global settings: %v", GetErrorResponse(err))
			}
		})
	}
}

func TestRestrictProjectCreation(t *testing.T) {
	tests := []struct {
		name          string
		projectsLimit int
	}{
		{
			name: "test, user can not create any project, admin can create projects",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var masterToken string

			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}
			apiRunner := createRunner(masterToken, t)
			// change for admin user
			adminMasterToken, err := retrieveAdminMasterToken()
			if err != nil {
				t.Fatalf("can not get admin master token due error: %v", err)
			}

			adminAPIRunner := createRunner(adminMasterToken, t)
			_, err = adminAPIRunner.UpdateGlobalSettings(json.RawMessage(`{"restrictProjectCreation":true}`))
			if err != nil {
				t.Fatalf("can not update global settings: %v", GetErrorResponse(err))
			}

			// regular user can't create projects
			_, err = apiRunner.CreateProject(rand.String(10))
			if err == nil {
				t.Fatalf("expected error during cluster creation")
			}
			createProjectDefaultErr, ok := err.(*project.CreateProjectDefault)
			if !ok {
				t.Fatalf("create project: expected error")
			}
			if createProjectDefaultErr.Code() != http.StatusForbidden {
				t.Fatalf("create project: expected forbidden error")
			}

			// admin can create projects
			project, err := adminAPIRunner.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("admin can not creat eproject: %v", GetErrorResponse(err))
			}
			if err := adminAPIRunner.DeleteProject(project.ID); err != nil {
				t.Fatalf("admin can not delete project: %v", GetErrorResponse(err))
			}

			_, err = adminAPIRunner.UpdateGlobalSettings(json.RawMessage(`{"restrictProjectCreation":false}`))
			if err != nil {
				t.Fatalf("can not update global settings: %v", GetErrorResponse(err))
			}
		})
	}
}
