package provider_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/vmware/govmomi/simulator"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"
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

	apiUser := test.GetUser(test.UserEmail, test.UserID, test.UserName, false)

	res := httptest.NewRecorder()
	ep, _, err := test.CreateTestEndpointAndGetClients(apiUser, mock.buildVSphereDatacenter(), []runtime.Object{}, []runtime.Object{}, []runtime.Object{test.APIUserToKubermaticUser(apiUser)}, nil, nil, hack.NewTestRouting)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}
	ep.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Expected route to return code 200, got %d: %s", res.Code, res.Body.String())
	}

	test.CompareWithResult(t, res, `[{"name":"VM Network"}]`)
}

func (v *vSphereMock) buildVSphereDatacenter() map[string]*kubermaticv1.SeedDatacenter {
	return map[string]*kubermaticv1.SeedDatacenter{
		"my-seed": &kubermaticv1.SeedDatacenter{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-seed",
			},
			Spec: kubermaticv1.SeedDatacenterSpec{
				NodeLocations: map[string]kubermaticv1.NodeLocation{
					vSphereDatacenterName: {
						Location: "Dark Side",
						Country:  "Moon States",
						Spec: kubermaticv1.DatacenterSpec{
							VSphere: &kubermaticv1.DatacenterSpecVSphere{
								Endpoint:      v.server.Server.URL,
								AllowInsecure: true,
								Datastore:     "LocalDS_0",
								Datacenter:    "ha-datacenter",
								Cluster:       "localhost.localdomain",
								RootPath:      "/ha-datacenter/vm/",
							},
						},
					},
				},
			},
		},
	}
}
