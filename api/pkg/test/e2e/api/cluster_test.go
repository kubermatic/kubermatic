// +build cloud

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
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

// runOIDCProxy runs the OIDC proxy. It is non-blocking. It does
// so by shelling out which is not pretty, but better than the previous
// approach of forking in a script and having no way of making the test
// fail of the OIDC failed
func runOIDCProxy(t *testing.T, cancel <-chan struct{}) error {
	gopathRaw, err := exec.Command("go", "env", "GOPATH").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get gopath: %v", err)
	}
	goPathSanitized := strings.Replace(string(gopathRaw), "\n", "", -1)
	oidcProxyDir := fmt.Sprintf("%s/src/github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/oidc-proxy-client", goPathSanitized)

	makePath, err := exec.LookPath("make")
	if err != nil {
		return fmt.Errorf("failed to look up path for `make`: %v", err)
	}

	oidProxyCommand := &exec.Cmd{
		Path: makePath,
		Args: []string{"run"},
		Dir:  oidcProxyDir,
	}

	errChan := make(chan error, 1)
	go func() {
		if out, err := oidProxyCommand.CombinedOutput(); err != nil {
			errChan <- fmt.Errorf("failed to run oidc proxy. Output:\n%s\nError: %v", string(out), err)
		}
	}()

	select {
	case err := <-errChan:
		t.Fatalf("oidc proxy failed: %v", err)
	case <-cancel:
		return nil
	}

	return nil
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

	cancel := make(chan struct{}, 1)
	if err := runOIDCProxy(t, cancel); err != nil {
		t.Fatalf("failed to start oidc proxy: %v", err)
	}
	defer func() {
		cancel <- struct{}{}
	}()

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
			teardown := cleanUpProject(project.ID, 1)
			defer teardown(t)

			cluster, err := apiRunner.CreateAWSCluster(project.ID, tc.dc, rand.String(10), getSecretAccessKey(), getAccessKeyID(), getKubernetesVersion(), tc.location, tc.availabilityZone, tc.replicas)
			if err != nil {
				t.Fatalf("can not create cluster due to error: %v", GetErrorResponse(err))
			}

			var clusterReady bool
			for attempt := 1; attempt <= getAWSMaxAttempts; attempt++ {
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
				if err := apiRunner.PrintClusterEvents(project.ID, tc.dc, cluster.ID); err != nil {
					t.Errorf("failed to print cluster events: %v", err)
				}
				t.Fatalf("cluster is not ready after %d attempts", getAWSMaxAttempts)
			}

			var ndReady bool
			for attempt := 1; attempt <= getAWSMaxAttempts; attempt++ {
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
				t.Fatalf("node deployment is not redy after %d attempts", getAWSMaxAttempts)
			}

			var replicasReady bool
			for attempt := 1; attempt <= getAWSMaxAttempts; attempt++ {
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
