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

	v1 "k8c.io/kubermatic/v2/pkg/api/v1"

	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
)

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
			version:                  getKubernetesVersion(),
			credential:               "e2e-digitalocean",
			replicas:                 0,
			expectedRoleNames:        []string{"namespace-admin", "namespace-editor", "namespace-viewer"},
			expectedClusterRoleNames: []string{"admin", "edit", "view"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("failed to get master token: %v", err)
			}

			apiRunner := createRunner(masterToken, t)
			project, cluster := createProjectWithCluster(t, apiRunner, tc.dc, tc.credential, tc.version, tc.location, tc.replicas)
			defer cleanUpProject(t, project.ID)

			// wait for controller to provision the roles
			var roleErr error
			roleNameList := []v1.RoleName{}
			if err := wait.PollImmediate(1*time.Second, 5*time.Minute, func() (bool, error) {
				roleNameList, roleErr = apiRunner.GetRoles(project.ID, tc.dc, cluster.ID)
				return len(roleNameList) >= len(tc.expectedRoleNames), nil
			}); err != nil {
				t.Fatalf("failed to wait for roles to be created (final list of roles before giving up: %v): %v", roleNameList, roleErr)
			}

			roleNames := []string{}
			for _, roleName := range roleNameList {
				roleNames = append(roleNames, roleName.Name)
			}
			namesSet := sets.NewString(tc.expectedRoleNames...)
			if !namesSet.HasAll(roleNames...) {
				t.Fatalf("expected roles %v, got %v", tc.expectedRoleNames, roleNames)
			}

			// wait for controller to provision the cluster roles
			var clusterRoleErr error
			clusterRoleNameList := []v1.ClusterRoleName{}
			if err := wait.PollImmediate(1*time.Second, 5*time.Minute, func() (bool, error) {
				clusterRoleNameList, clusterRoleErr = apiRunner.GetClusterRoles(project.ID, tc.dc, cluster.ID)
				return len(clusterRoleNameList) >= len(tc.expectedClusterRoleNames), nil
			}); err != nil {
				t.Fatalf("failed to wait for cluster roles to be created (final list of roles before giving up: %v): %v", clusterRoleNameList, clusterRoleErr)
			}

			clusterRoleNames := []string{}
			for _, clusterRoleName := range clusterRoleNameList {
				clusterRoleNames = append(clusterRoleNames, clusterRoleName.Name)
			}
			namesSet = sets.NewString(tc.expectedClusterRoleNames...)
			if !namesSet.HasAll(clusterRoleNames...) {
				t.Fatalf("expected cluster roles %v, got %v", tc.expectedRoleNames, roleNames)
			}

			// test if default cluster role bindings were created
			clusterBindings, err := apiRunner.GetClusterBindings(project.ID, tc.dc, cluster.ID)
			if err != nil {
				t.Fatalf("failed to get cluster bindings: %v", err)
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
					t.Fatalf("failed to create binding: %v", err)
				}
				if binding.RoleRefName != roleName.Name {
					t.Fatalf("expected binding RoleRefName %q, but got %q", roleName.Name, binding.RoleRefName)
				}
			}

			for _, clusterRoleName := range clusterRoleNameList {
				binding, err := apiRunner.BindUserToClusterRole(project.ID, tc.dc, cluster.ID, clusterRoleName.Name, "test@example.com")
				if err != nil {
					t.Fatalf("failed to create cluster binding: %v", err)
				}
				if binding.RoleRefName != clusterRoleName.Name {
					t.Fatalf("expected cluster binding RoleRefName %q, but got %q", clusterRoleName.Name, binding.RoleRefName)
				}
			}

			cleanUpCluster(t, apiRunner, project.ID, tc.dc, cluster.ID)
		})
	}
}

func createProjectWithCluster(t *testing.T, apiRunner *runner, dc, credential, version, location string, replicas int32) (*v1.Project, *v1.Cluster) {
	project, err := apiRunner.CreateProject(rand.String(10))
	if err != nil {
		t.Fatalf("failed to create project %v", err)
	}

	cluster, err := apiRunner.CreateDOCluster(project.ID, dc, rand.String(10), credential, version, location, replicas)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}

	if err := apiRunner.WaitForClusterHealthy(project.ID, dc, cluster.ID); err != nil {
		t.Fatalf("cluster not ready: %v", err)
	}

	return project, cluster
}
