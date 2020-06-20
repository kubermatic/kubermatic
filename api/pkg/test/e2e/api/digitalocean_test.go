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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"testing"
	"time"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/rand"
)

const getDOMaxAttempts = 24

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
	}{
		{
			name:       "create cluster on DigitalOcean",
			dc:         "prow-build-cluster",
			location:   "do-fra1",
			version:    "v1.15.6",
			credential: "e2e-digitalocean",
			replicas:   1,
			patch: PatchCluster{
				Name:   "newName",
				Labels: map[string]string{"a": "b"},
			},
			expectedName:   "newName",
			expectedLabels: map[string]string{"a": "b"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token %v", err)
			}

			apiRunner := createRunner(masterToken, t)
			project, err := apiRunner.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("can not create project %v", err)
			}
			teardown := cleanUpProject(project.ID, getDOMaxAttempts)
			defer teardown(t)

			cluster, err := apiRunner.CreateDOCluster(project.ID, tc.dc, rand.String(10), tc.credential, tc.version, tc.location, tc.replicas)
			if err != nil {
				t.Fatalf("can not create cluster due to error: %v", err)
			}

			var clusterReady bool
			for attempt := 1; attempt <= getDOMaxAttempts; attempt++ {
				healthStatus, err := apiRunner.GetClusterHealthStatus(project.ID, tc.dc, cluster.ID)
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
				t.Fatalf("cluster not ready after %d attempts", getDOMaxAttempts)
			}

			var ndReady bool
			for attempt := 1; attempt <= getDOMaxAttempts; attempt++ {
				ndList, err := apiRunner.GetClusterNodeDeployment(project.ID, tc.dc, cluster.ID)
				if err != nil {
					t.Fatalf("can not get node deployments %v", GetErrorResponse(err))
				}

				if len(ndList) == 1 {
					ndReady = true
					break
				}
				time.Sleep(30 * time.Second)
			}
			if !ndReady {
				t.Fatalf("node deployment is not redy after %d attempts", getDOMaxAttempts)
			}

			var replicasReady bool
			var ndList []apiv1.NodeDeployment
			for attempt := 1; attempt <= getDOMaxAttempts; attempt++ {
				ndList, err = apiRunner.GetClusterNodeDeployment(project.ID, tc.dc, cluster.ID)
				if err != nil {
					t.Fatalf("can not get node deployments %v", GetErrorResponse(err))
				}

				if ndList[0].Status.AvailableReplicas == tc.replicas {
					replicasReady = true
					break
				}
				time.Sleep(30 * time.Second)
			}
			if !replicasReady {
				t.Fatalf("the number of nodes is not as expected, available replicas %d", ndList[0].Status.AvailableReplicas)
			}

			_, err = apiRunner.UpdateCluster(project.ID, tc.dc, cluster.ID, tc.patch)
			if err != nil {
				t.Fatalf("can not update cluster %v", GetErrorResponse(err))
			}

			updatedCluster, err := apiRunner.GetCluster(project.ID, tc.dc, cluster.ID)
			if err != nil {
				t.Fatalf("can not get cluster %v", GetErrorResponse(err))
			}

			if updatedCluster.Name != tc.expectedName {
				t.Fatalf("expected new name %s got %s", tc.expectedName, updatedCluster.Name)
			}

			if !equality.Semantic.DeepEqual(updatedCluster.Labels, tc.expectedLabels) {
				t.Fatalf("expected labels %v got %v", tc.expectedLabels, updatedCluster.Labels)
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
			dc:         "prow-build-cluster",
			location:   "do-fra1",
			version:    "v1.15.6",
			credential: "e2e-digitalocean",
			replicas:   1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token %v", err)
			}

			apiRunner := createRunner(masterToken, t)
			project, err := apiRunner.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("can not create project %v", GetErrorResponse(err))
			}
			teardown := cleanUpProject(project.ID, getDOMaxAttempts)
			defer teardown(t)

			cluster, err := apiRunner.CreateDOCluster(project.ID, tc.dc, rand.String(10), tc.credential, tc.version, tc.location, tc.replicas)
			if err != nil {
				t.Fatalf("can not create cluster due to error: %v", err)
			}

			healthStatus, err := apiRunner.GetClusterHealthStatus(project.ID, tc.dc, cluster.ID)
			if err != nil {
				t.Fatalf("can not get health status %v", GetErrorResponse(err))
			}
			if IsHealthyCluster(healthStatus) {
				t.Fatal("Cluster is ready too fast")
			}

			time.Sleep(5 * time.Second)

			cleanUpCluster(t, apiRunner, project.ID, tc.dc, cluster.ID)

		})
	}
}

func TestGetClusterKubeconfig(t *testing.T) {
	tests := []struct {
		name         string
		dc           string
		location     string
		version      string
		credential   string
		replicas     int32
		path         string
		expectedCode int
	}{
		{
			name:         "kubeconfig contains token",
			dc:           "prow-build-cluster",
			location:     "do-fra1",
			version:      "v1.15.6",
			credential:   "e2e-digitalocean",
			replicas:     1,
			path:         "/api/v1/projects/%s/dc/%s/clusters/%s/kubeconfig",
			expectedCode: http.StatusOK,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token %v", err)
			}

			apiRunner := createRunner(masterToken, t)
			project, err := apiRunner.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("can not create project %v", err)
			}
			teardown := cleanUpProject(project.ID, getDOMaxAttempts)
			defer teardown(t)

			cluster, err := apiRunner.CreateDOCluster(project.ID, tc.dc, rand.String(10), tc.credential, tc.version, tc.location, tc.replicas)
			if err != nil {
				t.Fatalf("can not create cluster due to error: %v", err)
			}

			var clusterReady bool
			for attempt := 1; attempt <= getDOMaxAttempts; attempt++ {
				healthStatus, err := apiRunner.GetClusterHealthStatus(project.ID, tc.dc, cluster.ID)
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
				t.Fatalf("cluster not ready after %d attempts", getDOMaxAttempts)
			}

			var u url.URL
			u.Host = getHost()
			u.Scheme = getScheme()
			u.Path = fmt.Sprintf(tc.path, project.ID, tc.dc, cluster.ID)

			req, err := http.NewRequest("GET", u.String(), nil)
			if err != nil {
				t.Fatalf("can not make GET call due error: %v", err)
			}

			req.Header.Set("Cache-Control", "no-cache")
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", masterToken))

			client := &http.Client{Timeout: time.Second * 10}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatal("error reading response. ", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedCode {
				t.Fatalf("expected code %d, got %d", tc.expectedCode, resp.StatusCode)
			}

			bodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			kubeconfig := string(bodyBytes)
			regex := regexp.MustCompile(`token: [a-z0-9]{6}\.[a-z0-9]{16}`)
			matches := regex.FindAllString(kubeconfig, -1)
			if len(matches) != 1 {
				t.Fatalf("expected token in kubeconfig, got %s", kubeconfig)
			}

			cleanUpCluster(t, apiRunner, project.ID, tc.dc, cluster.ID)
		})
	}
}
