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
	"os"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

func getHost() string {
	host := os.Getenv("KUBERMATIC_HOST")
	if len(host) == 0 {
		fmt.Println("No KUBERMATIC_HOST env variable set.")
		os.Exit(1)
	}
	return host
}

func getScheme() string {
	scheme := os.Getenv("KUBERMATIC_SCHEME")
	if len(scheme) == 0 {
		fmt.Println("No KUBERMATIC_SCHEME env variable set.")
		os.Exit(1)
	}
	return scheme
}

func getKubernetesVersion() string {
	version := os.Getenv("VERSION_TO_TEST")
	if len(version) > 0 {
		return version
	}
	return "v1.18.8"
}

func cleanUpProject(t *testing.T, id string) {
	token, err := retrieveAdminMasterToken()
	if err != nil {
		t.Fatalf("failed to get master token: %v", err)
	}

	runner := createRunner(token, t)
	before := time.Now()

	t.Logf("Deleting project %s...", id)
	if err := runner.DeleteProject(id); err != nil {
		t.Fatalf("Failed to delete project: %v", err)
	}

	timeout := 3 * time.Minute
	t.Logf("Waiting %v for project to be gone...", timeout)

	err = wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		_, err := runner.GetProject(id)
		return err != nil, nil // return true if there *was* an error, i.e. project is gone
	})
	if err != nil {
		t.Fatalf("Failed to wait for project to be gone: %v", err)
	}

	t.Logf("Project deleted successfully after %v", time.Since(before))
}

func cleanUpCluster(t *testing.T, runner *runner, projectID, dc, clusterID string) {
	before := time.Now()

	t.Logf("Deleting cluster %s...", clusterID)
	if err := runner.DeleteCluster(projectID, dc, clusterID); err != nil {
		t.Fatalf("Failed to delete cluster: %v", err)
	}

	timeout := 3 * time.Minute
	t.Logf("Waiting %v for cluster to be gone...", timeout)

	err := wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		_, err := runner.GetCluster(projectID, dc, clusterID)
		return err != nil, nil // return true if there *was* an error, i.e. project is gone
	})
	if err != nil {
		t.Fatalf("Failed to wait for cluster to be gone: %v", err)
	}

	t.Logf("Cluster deleted successfully after %v", time.Since(before))
}

// waitFor is a convenience wrapper that makes simple, "brute force"
// waiting loops easier to write.
func waitFor(interval time.Duration, timeout time.Duration, callback func() bool) bool {
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		return callback(), nil
	})

	return err == nil
}
