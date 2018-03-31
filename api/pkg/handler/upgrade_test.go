package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/go-test/deep"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetClusterUpgrades(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		cluster     *kubermaticv1.Cluster
		versions    map[string]*apiv1.MasterVersion
		updates     []apiv1.MasterUpdate
		wantUpdates []string
	}{
		{
			name: "upgrade available",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUsername},
				},
				Spec: kubermaticv1.ClusterSpec{MasterVersion: "1.6.0"},
			},
			wantUpdates: []string{"1.6.1", "1.7.0"},
			versions: map[string]*apiv1.MasterVersion{
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
						"k8s-version":        "v1.6.0",
						"etcd-version":       "3.2.8",
						"pod-network-bridge": "v0.1",
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
						"k8s-version":        "v1.6.1",
						"etcd-version":       "3.2.8",
						"pod-network-bridge": "v0.1",
					},
				},
				"1.7.0": {
					Name:                       "v1.7.0",
					ID:                         "1.7.0",
					Default:                    false,
					AllowedNodeVersions:        []string{"1.3.0"},
					EtcdOperatorDeploymentYaml: "etcd-dep.yaml",
					EtcdClusterYaml:            "etcd-cluster.yaml",
					ApiserverDeploymentYaml:    "apiserver-dep.yaml",
					ControllerDeploymentYaml:   "controller-manager-dep.yaml",
					SchedulerDeploymentYaml:    "scheduler-dep.yaml",
					Values: map[string]string{
						"k8s-version":        "v1.7.0",
						"etcd-version":       "3.2.8",
						"pod-network-bridge": "v0.1",
					},
				},
			},
			updates: []apiv1.MasterUpdate{
				{
					From:            "1.6.0",
					To:              "1.6.1",
					Automatic:       false,
					RollbackAllowed: false,
					Enabled:         true,
					Visible:         true,
					Promote:         true,
				},
				{
					From:            "1.6.x",
					To:              "1.7.0",
					Automatic:       false,
					RollbackAllowed: false,
					Enabled:         true,
					Visible:         true,
					Promote:         true,
				},
			},
		},
		{
			name: "no available",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUsername},
				},
				Spec: kubermaticv1.ClusterSpec{MasterVersion: "1.6.0"},
			},
			wantUpdates: []string{},
			versions: map[string]*apiv1.MasterVersion{
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
						"k8s-version":        "v1.6.0",
						"etcd-version":       "3.2.8",
						"pod-network-bridge": "v0.1",
					},
				},
			},
			updates: []apiv1.MasterUpdate{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v3/dc/us-central1/cluster/foo/upgrades", nil)
			res := httptest.NewRecorder()
			e := createTestEndpoint(getUser(testUsername, false), []runtime.Object{test.cluster}, test.versions, test.updates)
			e.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Errorf("Expected status code to be 200, got %d", res.Code)
				t.Error(res.Body.String())
				return
			}

			gotUpdates := []string{}
			err := json.Unmarshal(res.Body.Bytes(), &gotUpdates)
			if err != nil {
				t.Fatal(err)
			}

			sort.Strings(gotUpdates)
			sort.Strings(test.wantUpdates)

			if diff := deep.Equal(gotUpdates, test.wantUpdates); diff != nil {
				t.Errorf("got different upgrade response than expected. Diff: %v", diff)
			}
		})
	}
}
