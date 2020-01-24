// +build kubeadm

package api

func TestCreateBYOCluster(t *testing.T) {
	tests := []struct {
		name     string
		dc       string
		location string
		version  string
	}{
		{
			name:     "create BringYourOwn cluster",
			dc:       "prow-build-cluster",
			location: "byo-kubernetes",
			version:  "v1.15.6",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Log("Getting master token")
			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token %v", err)
			}
			t.Log("Got master token")

			apiRunner := createRunner(masterToken, t)

			t.Log("Creating project")
			project, err := apiRunner.CreateProject(rand.String(10))
			if err != nil {
				t.Fatalf("can not create project %v", err)
			}
			t.Logf("Successfully created project %q", project.ID)
			teardown := cleanUpProject(project.ID, getDOMaxAttempts)
			defer teardown(t)

			t.Log("Creating cluster")
			cluster, err := apiRunner.CreateBYOCluster(project.ID, tc.dc, tc.location, rand.String(10), tc.version)
			if err != nil {
				t.Fatalf("can not create cluster due to error: %v", err)
			}
			t.Logf("Successfully created cluster %q", cluster.ID)

			t.Logf("Waiting for cluster %q to get ready", cluster.ID)
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
			t.Logf("Cluster %q got ready", cluster.ID)

			cleanUpCluster(t, apiRunner, project.ID, tc.dc, cluster.ID)

		})
	}
}
