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

package provider_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/vmware/govmomi/simulator"

	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/pkg/handler/test"
	"github.com/kubermatic/kubermatic/pkg/handler/test/hack"
	"github.com/kubermatic/kubermatic/pkg/provider"
)

const vSphereDatacenterName = "moon-1"

type vSphereMock struct {
	model  *simulator.Model
	server *simulator.Server
}

func (v *vSphereMock) tearUp() error {
	// ESXi model + initial set of objects (VMs, network, datastore)
	v.model = simulator.ESX()

	err := v.model.Create()
	if err != nil {
		return err
	}

	v.server = v.model.Service.NewServer()

	return nil
}

func (v *vSphereMock) tearDown() {
	v.model.Remove()
	v.server.Close()
}

func TestVsphereEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name             string
		URL              string
		ExpectedResponse string
	}{
		{
			Name:             "test networks endpoint",
			URL:              "/api/v1/providers/vsphere/networks",
			ExpectedResponse: `[{"absolutePath":"/ha-datacenter/network/VM Network","name":"VM Network","relativePath":"VM Network","type":"Network"}]`,
		},
		{
			Name:             "test folders endpoint",
			URL:              "/api/v1/providers/vsphere/folders",
			ExpectedResponse: `[{"path":"/ha-datacenter/vm"}]`,
		},
	}

	mock := &vSphereMock{}
	err := mock.tearUp()
	if err != nil {
		t.Fatalf("couldnt setup vsphere mock: %v", err)
	}

	defer mock.tearDown()

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.URL, nil)
			req.Header.Add("DatacenterName", vSphereDatacenterName)
			req.Header.Add("Username", "user")
			req.Header.Add("Password", "pass")

			apiUser := test.GetUser(test.UserEmail, test.UserID, test.UserName)

			res := httptest.NewRecorder()
			ep, _, err := test.CreateTestEndpointAndGetClients(apiUser, mock.buildVSphereDatacenter(), []runtime.Object{}, []runtime.Object{}, []runtime.Object{test.APIUserToKubermaticUser(apiUser)}, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}
			ep.ServeHTTP(res, req)

			if res.Code != http.StatusOK {
				t.Fatalf("Expected route to return code 200, got %d: %s", res.Code, res.Body.String())
			}

			compareJSON(t, res, tc.ExpectedResponse)
		})
	}

}

func (v *vSphereMock) buildVSphereDatacenter() provider.SeedsGetter {
	return func() (map[string]*kubermaticv1.Seed, error) {
		return map[string]*kubermaticv1.Seed{
			"my-seed": {
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-seed",
				},
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{
						vSphereDatacenterName: {
							Location: "Dark Side",
							Country:  "Moon States",
							Spec: kubermaticv1.DatacenterSpec{
								VSphere: &kubermaticv1.DatacenterSpecVSphere{
									Endpoint:         v.server.Server.URL,
									AllowInsecure:    true,
									DefaultDatastore: "LocalDS_0",
									Datacenter:       "ha-datacenter",
									Cluster:          "localhost.localdomain",
									RootPath:         "/ha-datacenter/vm/",
								},
							},
						},
					},
				},
			},
		}, nil
	}
}
