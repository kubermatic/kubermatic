//go:build create

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
	"regexp"
	"strings"
	"testing"
	"time"

	v1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestCreateUpdateDOCluster(t *testing.T) {
	tests := []struct {
		name                     string
		dc                       string
		location                 string
		version                  string
		credential               string
		replicas                 int32
		patch                    utils.PatchCluster
		patchAdmin               utils.PatchCluster
		expectedName             string
		expectedAdminName        string
		expectedLabels           map[string]string
		sshKeyName               string
		publicKey                string
		expectedRoleNames        []string
		expectedClusterRoleNames []string
	}{
		{
			name:       "create cluster on DigitalOcean",
			dc:         "kubermatic",
			location:   "do-fra1",
			version:    utils.KubernetesVersion(),
			credential: "e2e-digitalocean",
			replicas:   1,
			patch: utils.PatchCluster{
				Name:   "newName",
				Labels: map[string]string{"a": "b"},
			},
			patchAdmin: utils.PatchCluster{
				Name: "newAdminName",
			},
			expectedName:             "newName",
			expectedAdminName:        "newAdminName",
			expectedLabels:           map[string]string{"a": "b"},
			sshKeyName:               "test",
			publicKey:                "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== user@example.com ",
			expectedRoleNames:        []string{"namespace-admin", "namespace-editor", "namespace-viewer"},
			expectedClusterRoleNames: []string{"cluster-admin", "edit", "view"},
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
			project, cluster := createProjectWithCluster(t, testClient, tc.dc, tc.credential, tc.version, tc.location, tc.replicas)
			defer cleanupProject(t, project.ID)

			sshKey, err := testClient.CreateUserSSHKey(project.ID, tc.sshKeyName, tc.publicKey)
			if err != nil {
				t.Fatalf("failed to get create SSH key: %v", err)
			}

			_, err = testClient.UpdateCluster(project.ID, tc.dc, cluster.ID, tc.patch)
			if err != nil {
				t.Fatalf("failed to update cluster: %v", err)
			}

			updatedCluster, err := testClient.GetCluster(project.ID, tc.dc, cluster.ID)
			if err != nil {
				t.Fatalf("failed to get cluster: %v", err)
			}

			if updatedCluster.Name != tc.expectedName {
				t.Fatalf("expected new name %q, but got %q", tc.expectedName, updatedCluster.Name)
			}

			if !equality.Semantic.DeepEqual(updatedCluster.Labels, tc.expectedLabels) {
				t.Fatalf("expected labels %v, but got %v", tc.expectedLabels, updatedCluster.Labels)
			}
			if err := testClient.DetachSSHKeyFromClusterParams(project.ID, cluster.ID, tc.dc, sshKey.ID); err != nil {
				t.Fatalf("failed to detach SSH key to the cluster: %v", err)
			}

			kubeconfig, err := testClient.GetKubeconfig(tc.dc, project.ID, cluster.ID)
			if err != nil {
				t.Fatalf("failed to get kubeconfig: %v", err)
			}
			regex := regexp.MustCompile(`token: [a-z0-9]{6}\.[a-z0-9]{16}`)
			matches := regex.FindAllString(kubeconfig, -1)
			if len(matches) != 1 {
				t.Fatalf("expected token in kubeconfig, got %q", kubeconfig)
			}

			// wait for controller to provision the roles
			var roleErr error
			roleNameList := []v1.RoleName{}
			if err := wait.PollImmediate(1*time.Second, 5*time.Minute, func() (bool, error) {
				roleNameList, roleErr = testClient.GetRoles(project.ID, tc.dc, cluster.ID)
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
				clusterRoleNameList, clusterRoleErr = testClient.GetClusterRoles(project.ID, tc.dc, cluster.ID)
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
			clusterBindings, err := testClient.GetClusterBindings(project.ID, tc.dc, cluster.ID)
			if err != nil {
				t.Fatalf("failed to get cluster bindings: %v", err)
			}

			namesSet = sets.NewString(tc.expectedClusterRoleNames...)
			for _, clusterBinding := range clusterBindings {
				if !namesSet.Has(clusterBinding.RoleRefName) {
					t.Fatalf("expected role reference name %s in the cluster binding", clusterBinding.RoleRefName)
				}
			}

			// change for admin user
			adminMasterToken, err := utils.RetrieveAdminMasterToken(ctx)
			if err != nil {
				t.Fatalf("failed to get admin master token: %v", err)
			}

			adminTestClient := utils.NewTestClient(adminMasterToken, t)

			_, err = adminTestClient.UpdateCluster(project.ID, tc.dc, cluster.ID, tc.patchAdmin)
			if err != nil {
				t.Fatalf("failed to update cluster: %v", err)
			}

			updatedCluster, err = adminTestClient.GetCluster(project.ID, tc.dc, cluster.ID)
			if err != nil {
				t.Fatalf("failed to get cluster: %v", err)
			}

			if strings.Compare(updatedCluster.Name, tc.expectedAdminName) != 0 {
				t.Fatalf("expected new name %q, but got %q", tc.expectedAdminName, updatedCluster.Name)
			}

			testClient.CleanupCluster(t, project.ID, tc.dc, cluster.ID)
		})
	}
}

func TestDeleteClusterBeforeIsUp(t *testing.T) {
	tests := []struct {
		name       string
		dc         string
		location   string
		version    string
		credential string
		replicas   int32
	}{
		{
			name:       "delete cluster before is up",
			dc:         "kubermatic",
			location:   "do-fra1",
			version:    utils.KubernetesVersion(),
			credential: "e2e-digitalocean",
			replicas:   0,
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

			cluster, err := testClient.CreateDOCluster(project.ID, tc.dc, rand.String(10), tc.credential, tc.version, tc.location, tc.replicas)
			if err != nil {
				t.Fatalf("failed to create cluster: %v", err)
			}

			healthStatus, err := testClient.GetClusterHealthStatus(project.ID, tc.dc, cluster.ID)
			if err != nil {
				t.Fatalf("failed to get health status: %v", err)
			}
			if utils.IsHealthyCluster(healthStatus) {
				t.Fatal("Cluster is ready too fast")
			}

			testClient.CleanupCluster(t, project.ID, tc.dc, cluster.ID)
		})
	}
}

func createProjectWithCluster(t *testing.T, testClient *utils.TestClient, dc, credential, version, location string, replicas int32) (*v1.Project, *v1.Cluster) {
	project, err := testClient.CreateProject(rand.String(10))
	if err != nil {
		t.Fatalf("failed to create project %v", err)
	}

	cluster, err := testClient.CreateDOCluster(project.ID, dc, rand.String(10), credential, version, location, replicas)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}

	if err := testClient.WaitForClusterHealthy(project.ID, dc, cluster.ID); err != nil {
		t.Fatalf("cluster not ready: %v", err)
	}

	if err := testClient.WaitForClusterNodeDeploymentsToByReady(project.ID, dc, cluster.ID, replicas); err != nil {
		t.Fatalf("cluster nodes not ready: %v", err)
	}

	return project, cluster
}
