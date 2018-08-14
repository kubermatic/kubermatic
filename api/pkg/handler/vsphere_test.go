package handler

import (
	"flag"
	"net/http"
	"net/http/httptest"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/vmware/govmomi/simulator"
)

type vSphereMock struct {
	model  *simulator.Model
	server *simulator.Server
}

func (v *vSphereMock) tearUp() error {
	// Yeah this is sucky, but api doesnt offer any other method without getting too low level.
	f := flag.Lookup("httptest.serve")
	err := f.Value.Set("127.0.0.1:8989")
	if err != nil {
		return err
	}

	// ESXi model + initial set of objects (VMs, network, datastore)
	v.model = simulator.ESX()

	err = v.model.Create()
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

	req := httptest.NewRequest("GET", "/api/v1/vsphere/networks", nil)
	req.Header.Add("DatacenterName", "moon-1")
	req.Header.Add("Username", "user")
	req.Header.Add("Password", "pass")

	apiUser := getUser(testUserEmail, testUserID, testUserName, false)

	res := httptest.NewRecorder()
	ep, err := createTestEndpoint(apiUser, []runtime.Object{}, []runtime.Object{apiUserToKubermaticUser(apiUser)}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}
	ep.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Expected route to return code 200, got %d: %s", res.Code, res.Body.String())
	}

	compareWithResult(t, res, `[{"name":"VM Network"}]`)
}
