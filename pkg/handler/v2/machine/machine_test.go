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

package machine_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestCreateMachineDeployment(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedResponse       string
		ProjectID              string
		ClusterID              string
		HTTPStatus             int
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
		ExistingCluster        *kubermaticv1.Cluster
		ExistingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			Name:                   "scenario 1: create a machine deployment that match the given spec",
			Body:                   `{"spec":{"replicas":1,"template":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":[]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}}}}}`,
			ExpectedResponse:       `{"id":"%s","name":"%s","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"replicas":1,"template":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":["kubernetes","kubernetes-cluster-defClusterID","system-cluster-defClusterID","system-project-my-first-project-ID"]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"versions":{"kubelet":"9.9.9"},"labels":{"system/cluster":"defClusterID","system/project":"my-first-project-ID"}},"paused":false,"dynamicConfig":false},"status":{}}`,
			HTTPStatus:             http.StatusCreated,
			ProjectID:              test.GenDefaultProject().Name,
			ClusterID:              test.GenDefaultCluster().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genTestCluster(true)),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},

		// scenario 2
		{
			Name:                   "scenario 2: cluster components are not ready",
			Body:                   `{"spec":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":[]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}}}}`,
			ExpectedResponse:       `{"error":{"code":503,"message":"Cluster components are not ready yet"}}`,
			HTTPStatus:             http.StatusServiceUnavailable,
			ProjectID:              test.GenDefaultProject().Name,
			ClusterID:              test.GenDefaultCluster().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genTestCluster(false)),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},

		// scenario 3
		{
			Name:                   "scenario 3: kubelet version is too old",
			Body:                   `{"spec":{"replicas":1,"template":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":[]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"versions":{"kubelet":"9.6.0"}}}}`,
			ExpectedResponse:       `{"error":{"code":400,"message":"node deployment validation failed: kubelet version 9.6.0 is not compatible with control plane version 9.9.9"}}`,
			HTTPStatus:             http.StatusBadRequest,
			ProjectID:              test.GenDefaultProject().Name,
			ClusterID:              test.GenDefaultCluster().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genTestCluster(true)),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},

		// scenario 4
		{
			Name:                   "scenario 4: kubelet version is too new",
			Body:                   `{"spec":{"replicas":1,"template":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":[]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"versions":{"kubelet":"9.10.0"}}}}`,
			ExpectedResponse:       `{"error":{"code":400,"message":"node deployment validation failed: kubelet version 9.10.0 is not compatible with control plane version 9.9.9"}}`,
			HTTPStatus:             http.StatusBadRequest,
			ProjectID:              test.GenDefaultProject().Name,
			ClusterID:              test.GenDefaultCluster().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genTestCluster(true)),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},

		// scenario 5
		{
			Name:                   "scenario 5: set taints",
			Body:                   `{"spec":{"replicas":1,"template":{"taints": [{"key":"foo","value":"bar","effect":"NoExecute"}],"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":[]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"versions":{"kubelet":"9.9.9"}}}}`,
			ExpectedResponse:       `{"id":"%s","name":"%s","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"replicas":1,"template":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":["kubernetes","kubernetes-cluster-defClusterID","system-cluster-defClusterID","system-project-my-first-project-ID"]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"versions":{"kubelet":"9.9.9"},"labels":{"system/cluster":"defClusterID","system/project":"my-first-project-ID"},"taints":[{"key":"foo","value":"bar","effect":"NoExecute"}]},"paused":false,"dynamicConfig":false},"status":{}}`,
			HTTPStatus:             http.StatusCreated,
			ProjectID:              test.GenDefaultProject().Name,
			ClusterID:              test.GenDefaultCluster().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genTestCluster(true)),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},

		// scenario 6
		{
			Name:                   "scenario 6: invalid taint",
			Body:                   `{"spec":{"replicas":1,"template":{"taints": [{"key":"foo","value":"bar","effect":"BAD_EFFECT"}],"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":[]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"versions":{"kubelet":"9.9.9"}}}}`,
			ExpectedResponse:       `{"error":{"code":400,"message":"node deployment validation failed: taint effect 'BAD_EFFECT' not allowed. Allowed: NoExecute, NoSchedule, PreferNoSchedule"}}`,
			HTTPStatus:             http.StatusBadRequest,
			ProjectID:              test.GenDefaultProject().Name,
			ClusterID:              test.GenDefaultCluster().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genTestCluster(true)),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},

		// scenario 7
		{
			Name:                   "scenario 7: create a machine deployment with dynamic config",
			Body:                   `{"spec":{"replicas":1,"dynamicConfig":true,"template":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":[]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}}}}}`,
			ExpectedResponse:       `{"id":"%s","name":"%s","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"replicas":1,"template":{"cloud":{"digitalocean":{"size":"s-1vcpu-1gb","backups":false,"ipv6":false,"monitoring":false,"tags":["kubernetes","kubernetes-cluster-defClusterID","system-cluster-defClusterID","system-project-my-first-project-ID"]}},"operatingSystem":{"ubuntu":{"distUpgradeOnBoot":false}},"versions":{"kubelet":"9.9.9"},"labels":{"system/cluster":"defClusterID","system/project":"my-first-project-ID"}},"paused":false,"dynamicConfig":true},"status":{}}`,
			HTTPStatus:             http.StatusCreated,
			ProjectID:              test.GenDefaultProject().Name,
			ClusterID:              test.GenDefaultCluster().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(genTestCluster(true)),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v2/projects/%s/clusters/%s/machinedeployments", tc.ProjectID, tc.ClusterID), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			// Since Node Deployment's ID, name and match labels are automatically generated by the system just rewrite them.
			nd := &apiv1.NodeDeployment{}
			var expectedResponse string
			err = json.Unmarshal(res.Body.Bytes(), nd)
			if err != nil {
				t.Fatal(err)
			}
			if tc.HTTPStatus > 399 {
				expectedResponse = tc.ExpectedResponse
			} else {
				expectedResponse = fmt.Sprintf(tc.ExpectedResponse, nd.Name, nd.Name)
			}

			test.CompareWithResult(t, res, expectedResponse)
		})
	}
}

func genTestCluster(isControllerReady bool) *kubermaticv1.Cluster {
	controllerStatus := kubermaticv1.HealthStatusDown
	if isControllerReady {
		controllerStatus = kubermaticv1.HealthStatusUp
	}
	cluster := test.GenDefaultCluster()
	cluster.Status = kubermaticv1.ClusterStatus{
		ExtendedHealth: kubermaticv1.ExtendedClusterHealth{
			Apiserver:                    kubermaticv1.HealthStatusUp,
			Scheduler:                    kubermaticv1.HealthStatusUp,
			Controller:                   controllerStatus,
			MachineController:            kubermaticv1.HealthStatusUp,
			Etcd:                         kubermaticv1.HealthStatusUp,
			CloudProviderInfrastructure:  kubermaticv1.HealthStatusUp,
			UserClusterControllerManager: kubermaticv1.HealthStatusUp,
		},
	}
	cluster.Spec.Cloud = kubermaticv1.CloudSpec{
		DatacenterName: "regular-do1",
	}
	return cluster
}
