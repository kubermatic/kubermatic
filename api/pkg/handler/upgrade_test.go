package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"

	"github.com/kubermatic/kubermatic/api"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetClusterUpgrades(t *testing.T) {
	tests := []struct {
		name        string
		cluster     *kubermaticv1.Cluster
		versions    map[string]*api.MasterVersion
		updates     []api.MasterUpdate
		wantUpdates []string
	}{
		{
			name: "upgrade available",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: kubermaticv1.ClusterSpec{MasterVersion: "1.6.0"},
			},
			wantUpdates: []string{"1.6.1"},
			versions: map[string]*api.MasterVersion{
				"1.6.0": {
					Name:                       "v1.6.0",
					ID:                         "1.6.0",
					Default:                    false,
					AllowedNodeVersions:        []string{"1.3.0"},
					EtcdOperatorDeploymentYaml: "etcd-dep.yaml",
					EtcdClusterYaml:            "etcd-cluster.yaml",
					ApiserverDeploymentYaml:    "apiserver-dep.yaml",
					ControllerDeploymentYaml:   "controller-manager-dep.yaml",
					SchedulerDeploymentYaml:    "scheduler-dep.yaml",
					Values: map[string]string{
						"k8s-version":  "v1.6.0",
						"etcd-version": "3.2.8",
					},
				},
				"1.6.1": {
					Name:                       "v1.6.1",
					ID:                         "1.6.1",
					Default:                    false,
					AllowedNodeVersions:        []string{"1.3.0"},
					EtcdOperatorDeploymentYaml: "etcd-dep.yaml",
					EtcdClusterYaml:            "etcd-cluster.yaml",
					ApiserverDeploymentYaml:    "apiserver-dep.yaml",
					ControllerDeploymentYaml:   "controller-manager-dep.yaml",
					SchedulerDeploymentYaml:    "scheduler-dep.yaml",
					Values: map[string]string{
						"k8s-version":  "v1.6.1",
						"etcd-version": "3.2.8",
					},
				},
			},
			updates: []api.MasterUpdate{
				{
					From:            "1.6.0",
					To:              "1.6.1",
					Automatic:       false,
					RollbackAllowed: false,
					Enabled:         true,
					Visible:         true,
					Promote:         true,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/cluster/foo/upgrades", nil)
			res := httptest.NewRecorder()
			e := createTestEndpoint(getUser(false), []runtime.Object{test.cluster}, test.versions, test.updates)
			e.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Errorf("Expected status code to be 200, got %d", res.Code)
				t.Error(res.Body.String())
				return
			}

			gotUpdates := []string{}
			json.Unmarshal(res.Body.Bytes(), &gotUpdates)

			sort.Strings(gotUpdates)
			sort.Strings(test.wantUpdates)

			if !reflect.DeepEqual(gotUpdates, test.wantUpdates) || !reflect.DeepEqual(test.wantUpdates, gotUpdates) {
				t.Errorf("unexpected upgrades response. Want: %v Got %v", test.wantUpdates, gotUpdates)
			}
		})
	}
}
