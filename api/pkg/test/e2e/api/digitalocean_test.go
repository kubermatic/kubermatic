// +build create

package e2e

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/rand"
)

const getDOMaxAttempts = 24

func TestCreateDOCluster(t *testing.T) {
	tests := []struct {
		name       string
		dc         string
		location   string
		version    string
		credential string
		replicas   int32
	}{
		{
			name:       "create cluster on DigitalOcean",
			dc:         "prow-build-cluster",
			location:   "do-fra1",
			version:    "v1.15.6",
			credential: "loodse",
			replicas:   1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := GetMasterToken()
			if err != nil {
				t.Fatalf("can not get master token %v", err)
			}

			apiRunner := CreateAPIRunner(masterToken, t)
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
			for attempt := 1; attempt <= getDOMaxAttempts; attempt++ {
				ndList, err := apiRunner.GetClusterNodeDeployment(project.ID, tc.dc, cluster.ID)
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
				t.Fatalf("number of nodes is not as expected")
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
			credential: "loodse",
			replicas:   1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := GetMasterToken()
			if err != nil {
				t.Fatalf("can not get master token %v", err)
			}

			apiRunner := CreateAPIRunner(masterToken, t)
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
			credential:   "loodse",
			replicas:     1,
			path:         "/api/v1/projects/%s/dc/%s/clusters/%s/kubeconfig",
			expectedCode: http.StatusOK,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			masterToken, err := GetMasterToken()
			if err != nil {
				t.Fatalf("can not get master token %v", err)
			}

			apiRunner := CreateAPIRunner(masterToken, t)
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
