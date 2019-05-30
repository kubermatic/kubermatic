package provider_test

import (
	"github.com/kubermatic/kubermatic/api/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"
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

	apiUser := test.GetUser(test.UserEmail, test.UserID, test.UserName, false)

	res := httptest.NewRecorder()
	ep, _, err := test.CreateTestEndpointAndGetClients(apiUser, mock.buildVSphereDatacenterMeta(), []runtime.Object{}, []runtime.Object{}, []runtime.Object{test.APIUserToKubermaticUser(apiUser)}, nil, nil, hack.NewTestRouting)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}
	ep.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("Expected route to return code 200, got %d: %s", res.Code, res.Body.String())
	}

	test.CompareWithResult(t, res, `[{"name":"VM Network"}]`)
}

func TestVsphereCredentialEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name             string
		credentials      []credentials.VSphereCredentials
		httpStatus       int
		expectedResponse string
	}{
		{
			name:             "test no credentials for VSphere",
			httpStatus:       http.StatusOK,
			expectedResponse: "{}",
		},
		{
			name: "test list of credential names for VSphere",
			credentials: []credentials.VSphereCredentials{
				{Name: "first"},
				{Name: "second"},
			},
			httpStatus:       http.StatusOK,
			expectedResponse: `{"names":["first","second"]}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			req := httptest.NewRequest("GET", "/api/v1/providers/vsphere/credentials", strings.NewReader(""))

			credentialsManager := credentials.New()
			cred := credentialsManager.GetCredentials()
			cred.VSphere = tc.credentials

			res := httptest.NewRecorder()
			router, err := test.CreateCredentialTestEndpoint(credentialsManager, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v\n", err)
			}
			router.ServeHTTP(res, req)

			// validate
			assert.Equal(t, tc.httpStatus, res.Code)

			if res.Code == http.StatusOK {
				compareJSON(t, res, tc.expectedResponse)
			}
		})
	}
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
