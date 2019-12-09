// +build create

package e2e

import (
	"testing"
	"time"

	v1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"

	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
)

const getMaxAttempts = 24

func TestCreateClusterRoleBinding(t *testing.T) {
	tests := []struct {
		name              string
		dc                string
		location          string
		version           string
		credential        string
		replicas          int32
		expectedRoleNames []string
	}{
		{
			name:              "create role binding",
			dc:                "prow-build-cluster",
			location:          "do-fra1",
			version:           "v1.15.6",
			credential:        "loodse",
			replicas:          1,
			expectedRoleNames: []string{"namespace-admin", "namespace-editor", "namespace-viewer"},
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
			teardown := cleanUpProject(project.ID, getMaxAttempts)
			defer teardown(t)

			cluster, err := apiRunner.CreateDOCluster(project.ID, tc.dc, rand.String(10), tc.credential, tc.version, tc.location, tc.replicas)
			if err != nil {
				t.Fatalf("can not create cluster due to error: %v", GetErrorResponse(err))
			}

			var clusterReady bool
			for attempt := 1; attempt <= getMaxAttempts; attempt++ {
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
				t.Fatalf("cluster not ready after %d attempts", getMaxAttempts)
			}

			roleNameList := []v1.RoleName{}
			// wait for controller
			for attempt := 1; attempt <= getMaxAttempts; attempt++ {
				roleNameList, err = apiRunner.GetUserClusterRoles(project.ID, tc.dc, cluster.ID)
				if err != nil {
					t.Fatalf("can not get user cluster roles due to error: %v", err)
				}

				if len(roleNameList) == len(tc.expectedRoleNames) {
					break
				}
				time.Sleep(2 * time.Second)
			}

			if len(roleNameList) != len(tc.expectedRoleNames) {
				t.Fatalf("expectd length list is different then returned")
			}

			roleNames := []string{}
			for _, roleName := range roleNameList {
				roleNames = append(roleNames, roleName.Name)
			}
			namesSet := sets.NewString(tc.expectedRoleNames...)
			if !namesSet.HasAll(roleNames...) {
				t.Fatalf("expects roles %v, got %v", tc.expectedRoleNames, roleNames)
			}

			for _, roleName := range roleNameList {
				binding, err := apiRunner.BindUserToRole(project.ID, tc.dc, cluster.ID, roleName.Name, "default", "test@example.com")
				if err != nil {
					t.Fatalf("can not create binding due to error: %v", err)
				}
				if binding.RoleRefName != roleName.Name {
					t.Fatalf("expected binding RoleRefName %s got %s", roleName.Name, binding.RoleRefName)
				}
			}

			cleanUpCluster(t, apiRunner, project.ID, tc.dc, cluster.ID)

		})
	}
}
