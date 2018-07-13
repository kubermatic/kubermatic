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

func SetupOpenstackServer(t *testing.T) {
	openstackMux = http.NewServeMux()
	openstackServer = httptest.NewServer(openstackMux)

	openstackService := []struct {
		OpenstackURL string
		JSONResponse string
	}{
		{
			OpenstackURL: "/",
			JSONResponse: "{}",
		},
		{
			OpenstackURL: "/v3/auth/tokens",
			JSONResponse: PostTokens,
		},
		{
			OpenstackURL: "/v3/users/" + tokenID + "/projects",
			JSONResponse: GetUserProjects,
		},
		{
			OpenstackURL: "/v2.0/subnets",
			JSONResponse: GetSubnets,
		},
		{
			OpenstackURL: "/v2.0/networks",
			JSONResponse: GetNetworks,
		},
		{
			OpenstackURL: "/flavors/detail",
			JSONResponse: GetFlaivorsDetail,
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
		path := service.OpenstackURL
		tmpl, err := template.New("test").Parse(service.JSONResponse)
		if err != nil {
			t.Fatal(err)
		}
		buf := bytes.NewBuffer(nil)
		err = tmpl.Execute(buf, data)
		if err != nil {
			t.Fatal(err)
		}
		openstackMux.HandleFunc(service.OpenstackURL, func(w http.ResponseWriter, r *http.Request) {
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
			Name: "test tenants endpoint",
			URL:  "/api/v1/openstack/tenants",
			ExpectedResponse: `[
				{"id":"456788", "name": "a project name"},
				{"id":"456789", "name": "another domain"}
			]`,
		},
		{
			Name: "test subnets endpoint",
			URL:  "/api/v1/openstack/subnets",
			ExpectedResponse: `[
				{"id": "08eae331-0402-425a-923c-34f7cfe39c1b", "name": "private-subnet"},
				{"id": "54d6f61d-db07-451c-9ab3-b9609b6b6f0b", "name": "my_subnet"}
			]`,
		},
		{
			Name: "test networks endpoint",
			URL:  "/api/v1/openstack/networks",
			ExpectedResponse: `[
				{"id": "396f12f8-521e-4b91-8e21-2e003500433a", "name": "net3"},
				{"id": "71c1e68c-171a-4aa2-aca5-50ea153a3718", "name": "net2"}
			]`,
		},
		{
			Name: "test sizes endpoint",
			URL:  "/api/v1/openstack/sizes",
			ExpectedResponse: `[
				{
					"disk":40, "isPublic":true, "memory":4096, "region":"RegionOne", "slug":"m1.medium", "swap":0, "vcpus":2
				},
				{
					"disk":80, "isPublic":true, "memory":8192, "region":"RegionOne", "slug":"m1.large", "swap":0, "vcpus":4
				},
				{
					"disk":1, "isPublic":true, "memory":512, "region":"RegionOne", "slug":"m1.tiny.specs", "swap":0, "vcpus":1
				}
			]`,
		},
	}

	SetupOpenstackServer(t)
	defer TeardownOpenstackServer()

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {

			req := httptest.NewRequest("GET", tc.URL, strings.NewReader(""))
			req.Header.Add("DatacenterName", datacenterName)
			req.Header.Add("Username", userName)
			req.Header.Add("Password", userPass)
			req.Header.Add("Domain", domain)

			res := httptest.NewRecorder()
			router, _, err := createTestEndpointAndGetClients(getUser(testUsername, false), buildOpenstackDatacenterMeta(), []runtime.Object{}, []runtime.Object{}, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v\n", err)
			}

			router.ServeHTTP(res, req)
			compareJSON(t, res, tc.ExpectedResponse)
		})
	}
}
