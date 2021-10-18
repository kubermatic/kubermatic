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
	"testing"

	v1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	"k8s.io/apimachinery/pkg/util/rand"
)

func TestCreateUpdateDOCluster(t *testing.T) {
	tests := []createCluster{
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
			testCluster(ctx, project, cluster, testClient, tc, t)
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
