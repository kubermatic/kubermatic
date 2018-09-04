package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/vmware/govmomi/simulator"
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

func TestVsphereNetworksEndpoint(t *testing.T) {
	t.Parallel()

	mock := &vSphereMock{}
	err := mock.tearUp()
	if err != nil {
		t.Fatalf("couldnt setup vsphere mock: %v", err)
	}

	defer mock.tearDown()

	req := httptest.NewRequest("GET", "/api/v1/providers/vsphere/networks", nil)
	req.Header.Add("DatacenterName", vSphereDatacenterName)
	req.Header.Add("Username", "user")
	req.Header.Add("Password", "pass")

	apiUser := getUser(testUserEmail, testUserID, testUserName, false)

	res := httptest.NewRecorder()
	ep, _, err := createTestEndpointAndGetClients(apiUser, mock.buildVSphereDatacenterMeta(), []runtime.Object{}, []runtime.Object{}, []runtime.Object{apiUserToKubermaticUser(apiUser)}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}
	ep.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Expected route to return code 200, got %d: %s", res.Code, res.Body.String())
	}

	compareWithResult(t, res, `[{"name":"VM Network"}]`)
}

func (v *vSphereMock) buildVSphereDatacenterMeta() map[string]provider.DatacenterMeta {
	return map[string]provider.DatacenterMeta{
		vSphereDatacenterName: {
			Location: "Dark Side",
			Seed:     "us-central1",
			Country:  "Moon States",
			Spec: provider.DatacenterSpec{
				VSphere: &provider.VSphereSpec{
					Endpoint:      v.server.Server.URL,
					AllowInsecure: true,
					Datastore:     "LocalDS_0",
					Datacenter:    "ha-datacenter",
					Cluster:       "localhost.localdomain",
					RootPath:      "/ha-datacenter/vm/",
				},
			},
		},
	}
}
