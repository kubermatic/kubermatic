// +build cloud

package e2e

import (
	"os"
	"testing"
	"time"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/rand"
)

const getAWSMaxAttempts = 12

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
	}
}

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
		name     string
		dc       string
		location string
		replicas int32
	}{
		{
			name:     "create cluster on AWS",
			dc:       "europe-west3-c-1",
			location: "aws-eu-central-1a",
			replicas: 1,
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
			teardown := cleanUpProject(project.ID)
			defer teardown(t)

			cluster, err := apiRunner.CreateAWSCluster(project.ID, tc.dc, rand.String(10), getSecretAccessKey(), getAccessKeyID(), getKubernetesVersion(), tc.location, tc.replicas)
			if err != nil {
				t.Fatalf("can not create cluster due to error: %v", err)
			}

			var clusterReady bool
			for attempt := 1; attempt <= getAWSMaxAttempts; attempt++ {
				healthStatus, err := apiRunner.GetClusterHealthStatus(project.ID, tc.dc, cluster.ID)
				if err != nil {
					t.Fatalf("can not get health status %v", err)
				}

				if isHealthyCluster(healthStatus) {
					clusterReady = true
					break
				}
				time.Sleep(30 * time.Second)
			}

			if !clusterReady {
				t.Fatalf("cluster is not redy after %d attempts", getAWSMaxAttempts)
			}

			var ndReady bool
			for attempt := 1; attempt <= getAWSMaxAttempts; attempt++ {
				ndList, err := apiRunner.GetClusterNodeDeployment(project.ID, tc.dc, cluster.ID)
				if err != nil {
					t.Fatalf("can not get node deployments %v", err)
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

			var replicasReady bool
			for attempt := 1; attempt <= getAWSMaxAttempts; attempt++ {
				ndList, err := apiRunner.GetClusterNodeDeployment(project.ID, tc.dc, cluster.ID)
				if err != nil {
					t.Fatalf("can not get node deployments %v", err)
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

func isHealthyCluster(healthStatus *apiv1.ClusterHealth) bool {
	if healthStatus.UserClusterControllerManager == kubermaticv1.HealthStatusUp && healthStatus.Scheduler == kubermaticv1.HealthStatusUp &&
		healthStatus.MachineController == kubermaticv1.HealthStatusUp && healthStatus.Etcd == kubermaticv1.HealthStatusUp &&
		healthStatus.Controller == kubermaticv1.HealthStatusUp && healthStatus.Apiserver == kubermaticv1.HealthStatusUp {
		return true
	}
	return false
}
