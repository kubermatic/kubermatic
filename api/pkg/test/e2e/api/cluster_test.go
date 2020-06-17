// +build cloud

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
	"os"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/rand"
)

const getAWSMaxAttempts = 12

func getSecretAccessKey() string {
	return os.Getenv("AWS_E2E_TESTS_SECRET")
}

func getAccessKeyID() string {
	return os.Getenv("AWS_E2E_TESTS_KEY_ID")
}

func getKubernetesVersion() string {
	version := os.Getenv("VERSIONS_TO_TEST")
	if len(version) > 0 {
		return version
	}
	return "v1.14.2"
}

func TestCreateAWSCluster(t *testing.T) {
	tests := []struct {
		name             string
		dc               string
		location         string
		availabilityZone string
		replicas         int32
	}{
		{
			name:             "create cluster on AWS",
			dc:               "europe-west3-c-1",
			location:         "aws-eu-central-1a",
			availabilityZone: "eu-central-1a",
			replicas:         1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Log("Getting master token")
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token: %v", err)
			}
			t.Log("Got master token")

			apiRunner := createRunner(masterToken, t)

			t.Log("Creating project")
			project, err := apiRunner.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("can not create project: %v", GetErrorResponse(err))
			}
			t.Logf("Successfully created project %q", project.ID)
			teardown := cleanUpProject(project.ID, 1)
			defer teardown(t)

			t.Log("Creating cluster")
			cluster, err := apiRunner.CreateAWSCluster(project.ID, tc.dc, rand.String(10), getSecretAccessKey(), getAccessKeyID(), getKubernetesVersion(), tc.location, tc.availabilityZone, tc.replicas)
			if err != nil {
				t.Fatalf("can not create cluster due to error: %v", GetErrorResponse(err))
			}
			t.Logf("Successfully created cluster %q", cluster.ID)

			t.Logf("Waiting for cluster %q to get ready", cluster.ID)
			var clusterReady bool
			for attempt := 1; attempt <= getAWSMaxAttempts; attempt++ {
				healthStatus, err := apiRunner.GetClusterHealthStatus(project.ID, tc.dc, cluster.ID)
				if err != nil {
					t.Fatalf("can not get health status: %v", GetErrorResponse(err))
				}

				if IsHealthyCluster(healthStatus) {
					clusterReady = true
					break
				}
				time.Sleep(30 * time.Second)
			}

			if !clusterReady {
				if err := apiRunner.PrintClusterEvents(project.ID, tc.dc, cluster.ID); err != nil {
					t.Errorf("failed to print cluster events: %v", err)
				}
				t.Fatalf("cluster is not ready after %d attempts", getAWSMaxAttempts)
			}
			t.Logf("Cluster %q got ready", cluster.ID)

			t.Log("Waiting for nodeDeployments to get ready")
			var ndReady bool
			for attempt := 1; attempt <= getAWSMaxAttempts; attempt++ {
				ndList, err := apiRunner.GetClusterNodeDeployment(project.ID, tc.dc, cluster.ID)
				if err != nil {
					t.Fatalf("can not get node deployments: %v", GetErrorResponse(err))
				}

				if len(ndList) == 1 {
					ndReady = true
					break
				}
				time.Sleep(30 * time.Second)
			}
			if !ndReady {
				t.Fatalf("node deployment is not redy after %d attempts", getAWSMaxAttempts)
			}
			t.Log("NodeDeployments got ready")

			t.Log("Waiting for all nodes to get ready")
			var replicasReady bool
			for attempt := 1; attempt <= getAWSMaxAttempts; attempt++ {
				ndList, err := apiRunner.GetClusterNodeDeployment(project.ID, tc.dc, cluster.ID)
				if err != nil {
					t.Fatalf("can not get node deployments: %v", GetErrorResponse(err))
				}

				if ndList[0].Status.AvailableReplicas == tc.replicas {
					replicasReady = true
					break
				}
				time.Sleep(30 * time.Second)
			}
			if !replicasReady {
				t.Fatalf("number of nodes is not as expected")
			}
			t.Log("all nodes got ready")

			cleanUpCluster(t, apiRunner, project.ID, tc.dc, cluster.ID)

		})
	}
}
