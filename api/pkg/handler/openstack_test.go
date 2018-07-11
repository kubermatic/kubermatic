package handler

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	openstackMux    *http.ServeMux
	openstackServer *httptest.Server
)

const tokenID = "cbc36478b0bd8e67e89469c7749d4127"
const datacenterName = "ap-northeast-1"
const userName = "OSuser"
const userPass = "OSpass"
const region = "RegionOne"
const domain = "OSdomain"

type ServerTemplateData struct {
	URL     string
	TokenID string
	User    string
	Pass    string
	Domain  string
	Region  string
}

func SetupOpenstackServer(t *testing.T, url, resp string) {
	openstackMux = http.NewServeMux()
	openstackServer = httptest.NewServer(openstackMux)

	openstackService := []struct {
		Name         string
		Path         string
		JSONResponse string
	}{
		{
			Path:         "/",
			JSONResponse: "{}",
		},
		{
			Path:         url,
			JSONResponse: resp,
		},
		{
			Path:         "/v3/auth/tokens",
			JSONResponse: PostTokens,
		},
	}

	data := ServerTemplateData{
		URL:     openstackServer.URL,
		TokenID: tokenID,
		User:    userName,
		Pass:    userPass,
		Domain:  domain,
		Region:  region,
	}

	for _, service := range openstackService {
		path := service.Path
		tmpl, err := template.New("test").Parse(service.JSONResponse)
		if err != nil {
			t.Fatal(err)
		}
		buf := bytes.NewBuffer(nil)
		err = tmpl.Execute(buf, data)
		if err != nil {
			t.Fatal(err)
		}
		openstackMux.HandleFunc(service.Path, func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf(" >>> %s %s\n", r.Method, r.URL)
			if r.URL.String() != path {
				t.Fatalf("Unexpected call: %s %s", r.Method, r.URL)
			}

			w.Header().Add("Content-Type", "application/json")
			if r.Method == "POST" {
				w.WriteHeader(201)
			} else {
				w.WriteHeader(200)
			}

			fmt.Fprintf(w, buf.String())
		})
	}

}

func TeardownOpenstackServer() {
	openstackServer.Close()
}

func buildOpenstackDatacenterMeta() map[string]provider.DatacenterMeta {
	return map[string]provider.DatacenterMeta{
		datacenterName: {
			Location: "ap-northeast",
			Country:  "JP",
			Private:  false,
			IsSeed:   true,
			Spec: provider.DatacenterSpec{
				Openstack: &provider.OpenstackSpec{
					Region:  region,
					AuthURL: openstackServer.URL + "/v3/",
				},
			},
		},
	}
}

func TestOpenstackEndpoint(t *testing.T) {

	testcases := []struct {
		Name              string
		URL               string
		OpenstackURL      string
		OpenstackResponse string
		ExpectedResponse  string
	}{
		{
			Name:              "test tenants endpoint",
			URL:               "/api/v1/openstack/tenants",
			OpenstackURL:      "/v3/users/" + tokenID + "/projects",
			OpenstackResponse: GetUserProjects,
			ExpectedResponse:  ExpectedTenants,
		},
		{
			Name:              "test subnets endpoint",
			URL:               "/api/v1/openstack/subnets",
			OpenstackURL:      "/v2.0/subnets",
			OpenstackResponse: GetSubnets,
			ExpectedResponse:  ExpectedSubnets,
		},
		{
			Name:              "test networks endpoint",
			URL:               "/api/v1/openstack/networks",
			OpenstackURL:      "/v2.0/networks",
			OpenstackResponse: GetNetworks,
			ExpectedResponse:  ExpectedNetworks,
		},
		{
			Name:              "test sizes endpoint",
			URL:               "/api/v1/openstack/sizes",
			OpenstackURL:      "/flavors/detail",
			OpenstackResponse: GetFlaivorsDetail,
			ExpectedResponse:  ExpectedSizes,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			SetupOpenstackServer(t, tc.OpenstackURL, tc.OpenstackResponse)
			defer TeardownOpenstackServer()

			req := httptest.NewRequest("GET", tc.URL, strings.NewReader(""))
			req.Header.Add("DatacenterName", datacenterName)
			req.Header.Add("Username", userName)
			req.Header.Add("Password", userPass)
			req.Header.Add("Domain", domain)

			res := httptest.NewRecorder()
			ep, err := createTestEndpointForDC(getUser(testUsername, false), buildOpenstackDatacenterMeta(), []runtime.Object{}, []runtime.Object{}, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v\n", err)
			}

			ep.ServeHTTP(res, req)
			compareJSON(t, res, tc.ExpectedResponse)
		})
	}
}
