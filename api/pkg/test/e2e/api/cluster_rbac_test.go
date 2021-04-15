// +build create

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
	"testing"
	"time"

	v1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"

	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
)

const getMaxAttempts = 24

func TestCreateClusterRoleBinding(t *testing.T) {
	tests := []struct {
		name                     string
		dc                       string
		location                 string
		version                  string
		credential               string
		replicas                 int32
		expectedRoleNames        []string
		expectedClusterRoleNames []string
	}{
		{
			name:                     "create cluster/role binding",
			dc:                       "kubermatic",
			location:                 "do-fra1",
			version:                  "v1.15.6",
			credential:               "e2e-digitalocean",
			replicas:                 1,
			expectedRoleNames:        []string{"namespace-admin", "namespace-editor", "namespace-viewer"},
			expectedClusterRoleNames: []string{"admin", "edit", "view"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token %v", err)
			}

			apiRunner := createRunner(masterToken, t)
			project, cluster := createProjectWithCluster(t, apiRunner, tc.dc, tc.credential, tc.version, tc.location, tc.replicas)
			teardown := cleanUpProject(project.ID, getMaxAttempts)
			defer teardown(t)

			roleNameList := []v1.RoleName{}
			// wait for controller
			for attempt := 1; attempt <= getMaxAttempts; attempt++ {
				roleNameList, err = apiRunner.GetRoles(project.ID, tc.dc, cluster.ID)
				if err != nil {
					t.Fatalf("can not get user cluster roles due to error: %v", err)
				}

				if len(roleNameList) == len(tc.expectedRoleNames) {
					break
				}
				time.Sleep(2 * time.Second)
			}

			if len(roleNameList) != len(tc.expectedRoleNames) {
				t.Fatalf("expectd length list is different then returned")
			}

			roleNames := []string{}
			for _, roleName := range roleNameList {
				roleNames = append(roleNames, roleName.Name)
			}
			namesSet := sets.NewString(tc.expectedRoleNames...)
			if !namesSet.HasAll(roleNames...) {
				t.Fatalf("expects roles %v, got %v", tc.expectedRoleNames, roleNames)
			}

			// test cluster roles
			clusterRoleNameList := []v1.ClusterRoleName{}
			// wait for controller
			for attempt := 1; attempt <= getMaxAttempts; attempt++ {
				clusterRoleNameList, err = apiRunner.GetClusterRoles(project.ID, tc.dc, cluster.ID)
				if err != nil {
					t.Fatalf("can not get cluster roles due to error: %v", err)
				}

				if len(clusterRoleNameList) == len(tc.expectedClusterRoleNames) {
					break
				}
				time.Sleep(2 * time.Second)
			}

			if len(clusterRoleNameList) != len(tc.expectedClusterRoleNames) {
				t.Fatalf("expectd length list is different then returned")
			}

			clusterRoleNames := []string{}
			for _, clusterRoleName := range clusterRoleNameList {
				clusterRoleNames = append(clusterRoleNames, clusterRoleName.Name)
			}
			namesSet = sets.NewString(tc.expectedClusterRoleNames...)
			if !namesSet.HasAll(clusterRoleNames...) {
				t.Fatalf("expects cluster roles %v, got %v", tc.expectedRoleNames, roleNames)
			}

			// test if default cluster role bindings were created
			clusterBindings, err := apiRunner.GetClusterBindings(project.ID, tc.dc, cluster.ID)
			if err != nil {
				t.Fatalf("can not get cluster bindings due to error: %v", err)
			}

			namesSet = sets.NewString(tc.expectedClusterRoleNames...)
			for _, clusterBinding := range clusterBindings {
				if !namesSet.Has(clusterBinding.RoleRefName) {
					t.Fatalf("expected role reference name %s in the cluster binding", clusterBinding.RoleRefName)
				}
			}

			for _, roleName := range roleNameList {
				binding, err := apiRunner.BindUserToRole(project.ID, tc.dc, cluster.ID, roleName.Name, "default", "test@example.com")
				if err != nil {
					t.Fatalf("can not create binding due to error: %v", err)
				}
				if binding.RoleRefName != roleName.Name {
					t.Fatalf("expected binding RoleRefName %s got %s", roleName.Name, binding.RoleRefName)
				}
			}

			for _, clusterRoleName := range clusterRoleNameList {
				binding, err := apiRunner.BindUserToClusterRole(project.ID, tc.dc, cluster.ID, clusterRoleName.Name, "test@example.com")
				if err != nil {
					t.Fatalf("can not create cluster binding due to error: %v", err)
				}
				if binding.RoleRefName != clusterRoleName.Name {
					t.Fatalf("expected cluster binding RoleRefName %s got %s", clusterRoleName.Name, binding.RoleRefName)
				}
			}

			cleanUpCluster(t, apiRunner, project.ID, tc.dc, cluster.ID)
		})
	}
}

func createProjectWithCluster(t *testing.T, apiRunner *runner, dc, credential, version, location string, replicas int32) (*v1.Project, *v1.Cluster) {
	project, err := apiRunner.CreateProject(rand.String(10))
	if err != nil {
		t.Fatalf("can not create project %v", err)
	}

	cluster, err := apiRunner.CreateDOCluster(project.ID, dc, rand.String(10), credential, version, location, replicas)
	if err != nil {
		t.Fatalf("can not create cluster due to error: %v", GetErrorResponse(err))
	}

	var clusterReady bool
	for attempt := 1; attempt <= getMaxAttempts; attempt++ {
		healthStatus, err := apiRunner.GetClusterHealthStatus(project.ID, dc, cluster.ID)
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
		t.Fatalf("cluster not ready after %d attempts", getMaxAttempts)
	}

	return project, cluster
}
