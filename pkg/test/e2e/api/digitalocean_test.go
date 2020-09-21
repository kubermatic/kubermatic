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
	"regexp"
	"testing"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/rand"
)

func TestCreateUpdateDOCluster(t *testing.T) {
	tests := []struct {
		name           string
		dc             string
		location       string
		version        string
		credential     string
		replicas       int32
		patch          PatchCluster
		expectedName   string
		expectedLabels map[string]string
		sshKeyName     string
		publicKey      string
	}{
		{
			name:       "create cluster on DigitalOcean",
			dc:         "kubermatic",
			location:   "do-fra1",
			version:    getKubernetesVersion(),
			credential: "e2e-digitalocean",
			replicas:   1,
			patch: PatchCluster{
				Name:   "newName",
				Labels: map[string]string{"a": "b"},
			},
			expectedName:   "newName",
			expectedLabels: map[string]string{"a": "b"},
			sshKeyName:     "test",
			publicKey:      "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== user@example.com ",
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

			sshKey, err := apiRunner.CreateUserSSHKey(project.ID, tc.sshKeyName, tc.publicKey)
			if err != nil {
				t.Fatalf("failed to get create SSH key: %v", err)
			}
			cluster, err := apiRunner.CreateDOCluster(project.ID, tc.dc, rand.String(10), tc.credential, tc.version, tc.location, tc.replicas)
			if err != nil {
				t.Fatalf("failed to create cluster: %v", err)
			}

			if err := apiRunner.WaitForClusterHealthy(project.ID, tc.dc, cluster.ID); err != nil {
				if err := apiRunner.PrintClusterEvents(project.ID, tc.dc, cluster.ID); err != nil {
					t.Errorf("failed to print cluster events: %v", err)
				}

				t.Fatalf("cluster not ready: %v", err)
			}

			if err := apiRunner.WaitForClusterNodeDeploymentsToByReady(project.ID, tc.dc, cluster.ID, tc.replicas); err != nil {
				t.Fatalf("cluster nodes not ready: %v", err)
			}

			_, err = apiRunner.UpdateCluster(project.ID, tc.dc, cluster.ID, tc.patch)
			if err != nil {
				t.Fatalf("failed to update cluster: %v", err)
			}

			updatedCluster, err := apiRunner.GetCluster(project.ID, tc.dc, cluster.ID)
			if err != nil {
				t.Fatalf("failed to get cluster: %v", err)
			}

			if updatedCluster.Name != tc.expectedName {
				t.Fatalf("expected new name %q, but got %q", tc.expectedName, updatedCluster.Name)
			}

			if !equality.Semantic.DeepEqual(updatedCluster.Labels, tc.expectedLabels) {
				t.Fatalf("expected labels %v, but got %v", tc.expectedLabels, updatedCluster.Labels)
			}
			if err := apiRunner.DetachSSHKeyFromClusterParams(project.ID, cluster.ID, tc.dc, sshKey.ID); err != nil {
				t.Fatalf("failed to detach SSH key to the cluster: %v", err)
			}

			cleanUpCluster(t, apiRunner, project.ID, tc.dc, cluster.ID)
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
			version:    getKubernetesVersion(),
			credential: "e2e-digitalocean",
			replicas:   0,
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

			cluster, err := apiRunner.CreateDOCluster(project.ID, tc.dc, rand.String(10), tc.credential, tc.version, tc.location, tc.replicas)
			if err != nil {
				t.Fatalf("failed to create cluster: %v", err)
			}

			healthStatus, err := apiRunner.GetClusterHealthStatus(project.ID, tc.dc, cluster.ID)
			if err != nil {
				t.Fatalf("failed to get health status: %v", err)
			}
			if IsHealthyCluster(healthStatus) {
				t.Fatal("Cluster is ready too fast")
			}

			cleanUpCluster(t, apiRunner, project.ID, tc.dc, cluster.ID)
		})
	}
}

func TestGetClusterKubeconfig(t *testing.T) {
	tests := []struct {
		name       string
		dc         string
		location   string
		version    string
		credential string
		replicas   int32
	}{
		{
			name:       "kubeconfig contains token",
			dc:         "kubermatic",
			location:   "do-fra1",
			version:    getKubernetesVersion(),
			credential: "e2e-digitalocean",
			replicas:   0,
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

			cluster, err := apiRunner.CreateDOCluster(project.ID, tc.dc, rand.String(10), tc.credential, tc.version, tc.location, tc.replicas)
			if err != nil {
				t.Fatalf("failed to create cluster: %v", err)
			}

			if err := apiRunner.WaitForClusterHealthy(project.ID, tc.dc, cluster.ID); err != nil {
				if err := apiRunner.PrintClusterEvents(project.ID, tc.dc, cluster.ID); err != nil {
					t.Errorf("failed to print cluster events: %v", err)
				}

				t.Fatalf("cluster not ready: %v", err)
			}

			kubeconfig, err := apiRunner.GetKubeconfig(tc.dc, project.ID, cluster.ID)
			if err != nil {
				t.Fatalf("failed to get kubeconfig: %v", err)
			}
			regex := regexp.MustCompile(`token: [a-z0-9]{6}\.[a-z0-9]{16}`)
			matches := regex.FindAllString(kubeconfig, -1)
			if len(matches) != 1 {
				t.Fatalf("expected token in kubeconfig, got %q", kubeconfig)
			}

			cleanUpCluster(t, apiRunner, project.ID, tc.dc, cluster.ID)
		})
	}
}
