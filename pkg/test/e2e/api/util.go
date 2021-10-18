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
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	v1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
)

type createCluster struct {
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
}

func cleanupProject(t *testing.T, id string) {
	// use a dedicated context so that cleanups always run, even
	// if the context inside a test was already cancelled
	token, err := utils.RetrieveAdminMasterToken(context.Background())
	if err != nil {
		t.Fatalf("failed to get master token: %v", err)
	}

	utils.NewTestClient(token, t).CleanupProject(t, id)
}

// getErrorResponse converts the client error response to string
func getErrorResponse(err error) string {
	rawData, newErr := json.Marshal(err)
	if newErr != nil {
		return err.Error()
	}
	return string(rawData)
}

func testCluster(ctx context.Context, project *v1.Project, cluster *v1.Cluster, testClient *utils.TestClient, tc createCluster, t *testing.T) {
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
}
