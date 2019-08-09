// +build create

package e2e

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/rand"
)

const getDOMaxAttempts = 24

func cleanUpProject(id string) func(t *testing.T) {
	return func(t *testing.T) {
		masterToken, err := GetMasterToken()
		if err != nil {
			t.Fatalf("can not get master token due error: %v", err)
		}
		apiRunner := CreateAPIRunner(masterToken, t)

		if err := apiRunner.DeleteProject(id); err != nil {
			t.Fatalf("can not delete project due error: %v", err)
		}
		for attempt := 1; attempt <= getDOMaxAttempts; attempt++ {
			_, err := apiRunner.GetProject(id, 5)
			if err != nil {
				break
			}
			time.Sleep(3 * time.Second)
		}
		_, err = apiRunner.GetProject(id, 5)
		if err == nil {
			t.Fatalf("can not delete the project")
		}
	}
}

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
			version:    "v1.14.2",
			credential: "digitalocean",
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
			teardown := cleanUpProject(project.ID)
			defer teardown(t)

			cluster, err := apiRunner.CreateDOCluster(project.ID, tc.dc, rand.String(10), tc.credential, tc.version, tc.location, tc.replicas)
			if err != nil {
				t.Fatalf("can not create cluster due to error: %v", GetErrorResponse(err))
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
				t.Fatalf("cluster is not redy after %d attempts", getDOMaxAttempts)
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

		})
	}
}
