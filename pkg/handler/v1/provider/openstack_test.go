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
	"bytes"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
)

var (
	openstackMux    *http.ServeMux
	openstackServer *httptest.Server
)

const tokenID = "cbc36478b0bd8e67e89469c7749d4127"
const datacenterName = "ap-northeast-1"
const region = "RegionOne"

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
		OpenstackURL        string
		JSONResponse        string
		ExpectedQueryParams map[string]string
	}{
		{
			OpenstackURL: "/",
			JSONResponse: "{}",
		},
		{
			OpenstackURL: "/v2.0/security-groups",
			JSONResponse: GetSecurityGroups,
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
			OpenstackURL:        "/v2.0/subnets",
			ExpectedQueryParams: map[string]string{"network_id": "foo"},
			JSONResponse:        GetSubnets,
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
		User:    test.TestOSuserName,
		Pass:    test.TestOSuserPass,
		Domain:  test.TestOSdomain,
		Region:  region,
	}

	for _, service := range openstackService {
		expectedPath := service.OpenstackURL
		expectedQueryParams := service.ExpectedQueryParams
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
			if r.URL.Path != expectedPath {
				t.Fatalf("Unexpected call: %s %s", r.Method, r.URL)
			}

			for expectedKey, expectedValue := range expectedQueryParams {
				queryValue := r.URL.Query().Get(expectedKey)
				if (expectedValue != "") != (queryValue != "") {
					t.Fatalf("Wrong value for query param %s: expected %s, got: %s", expectedKey, expectedValue, queryValue)
				}
			}

			w.Header().Add("Content-Type", "application/json")
			if r.Method == "POST" {
				w.WriteHeader(201)
			} else {
				w.WriteHeader(200)
			}

			_, err := w.Write(buf.Bytes())
			if err != nil {
				t.Fatalf("failed to write rendered template to HTTP response: %v", err)
			}
		})
	}

}

func TeardownOpenstackServer() {
	openstackServer.Close()
}

func buildOpenstackDatacenter() provider.SeedsGetter {
	return func() (map[string]*kubermaticv1.Seed, error) {
		return map[string]*kubermaticv1.Seed{
			"my-seed": {
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-seed",
				},
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{
						datacenterName: {
							Location: "ap-northeast",
							Country:  "JP",
							Spec: kubermaticv1.DatacenterSpec{
								Openstack: &kubermaticv1.DatacenterSpecOpenstack{
									Region:  region,
									AuthURL: openstackServer.URL + "/v3/",
								},
							},
						},
					},
				},
			},
		}, nil
	}
}

func TestOpenstackEndpoints(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name              string
		URL               string
		QueryParams       map[string]string
		Credential        string
		Credentials       []runtime.Object
		OpenstackURL      string
		OpenstackResponse string
		ExpectedResponse  string
	}{
		{
			Name: "test tenants endpoint",
			URL:  "/api/v1/providers/openstack/securitygroups",
			ExpectedResponse: `[
				{"id": "85cc3048-abc3-43cc-89b3-377341426ac5", "name": "default"}
			]`,
		},
		{
			Name: "test tenants endpoint",
			URL:  "/api/v1/providers/openstack/tenants",
			ExpectedResponse: `[
				{"id":"456788", "name": "a project name"},
				{"id":"456789", "name": "another domain"}
			]`,
		},
		{
			Name:       "test tenants endpoint with predefined credentials",
			Credential: test.TestFakeCredential,
			Credentials: []runtime.Object{
				test.GenDefaultPreset(),
			},
			URL: "/api/v1/providers/openstack/tenants",
			ExpectedResponse: `[
				{"id":"456788", "name": "a project name"},
				{"id":"456789", "name": "another domain"}
			]`,
		},
		{
			Name:        "test subnets endpoint",
			URL:         "/api/v1/providers/openstack/subnets",
			QueryParams: map[string]string{"network_id": "foo"},
			ExpectedResponse: `[
				{"id": "08eae331-0402-425a-923c-34f7cfe39c1b", "name": "private-subnet"},
				{"id": "54d6f61d-db07-451c-9ab3-b9609b6b6f0b", "name": "my_subnet"}
			]`,
		},
		{
			Name:        "test subnets endpoint with predefined credentials",
			Credential:  test.TestFakeCredential,
			URL:         "/api/v1/providers/openstack/subnets",
			QueryParams: map[string]string{"network_id": "foo"},
			Credentials: []runtime.Object{
				test.GenDefaultPreset(),
			},
			ExpectedResponse: `[
				{"id": "08eae331-0402-425a-923c-34f7cfe39c1b", "name": "private-subnet"},
				{"id": "54d6f61d-db07-451c-9ab3-b9609b6b6f0b", "name": "my_subnet"}
			]`,
		},
		{
			Name: "test networks endpoint",
			URL:  "/api/v1/providers/openstack/networks",
			ExpectedResponse: `[
				{"id": "71c1e68c-171a-4aa2-aca5-50ea153a3718", "name": "net2", "external": false}
			]`,
		},
		{
			Name:       "test networks endpoint with predefined credentials",
			Credential: test.TestFakeCredential,
			URL:        "/api/v1/providers/openstack/networks",
			Credentials: []runtime.Object{
				test.GenDefaultPreset(),
			},
			ExpectedResponse: `[
				{"id": "71c1e68c-171a-4aa2-aca5-50ea153a3718", "name": "net2", "external": false}
			]`,
		},
		{
			Name: "test sizes endpoint",
			URL:  "/api/v1/providers/openstack/sizes",
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
		{
			Name:       "test sizes endpoint with predefined credentials",
			Credential: test.TestFakeCredential,
			Credentials: []runtime.Object{
				test.GenDefaultPreset(),
			},
			URL: "/api/v1/providers/openstack/sizes",
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
			if tc.QueryParams != nil {
				q := req.URL.Query()
				for k, v := range tc.QueryParams {
					q.Add(k, v)
				}
				req.URL.RawQuery = q.Encode()
			}

			req.Header.Add("DatacenterName", datacenterName)
			if len(tc.Credential) > 0 {
				req.Header.Add("Credential", test.TestFakeCredential)
			} else {
				req.Header.Add("Username", test.TestOSuserName)
				req.Header.Add("Password", test.TestOSuserPass)
				req.Header.Add("Domain", test.TestOSdomain)
			}

			apiUser := test.GenDefaultAPIUser()

			res := httptest.NewRecorder()
			credentials := []runtime.Object{}
			if tc.Credentials != nil {
				credentials = tc.Credentials
			}
			router, _, err := test.CreateTestEndpointAndGetClients(*apiUser, buildOpenstackDatacenter(), []runtime.Object{}, credentials, []runtime.Object{test.APIUserToKubermaticUser(*apiUser)}, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v\n", err)
			}

			router.ServeHTTP(res, req)
			compareJSON(t, res, tc.ExpectedResponse)
		})
	}
}

func TestMeetsOpentackNodeSizeRequirement(t *testing.T) {
	tests := []struct {
		name                string
		apiSize             apiv1.OpenstackSize
		nodeSizeRequirement kubermaticv1.OpenstackNodeSizeRequirements
		meetsRequirement    bool
	}{
		{
			name: "not enough memory",
			apiSize: apiv1.OpenstackSize{
				Memory: 2048,
				VCPUs:  2,
			},
			nodeSizeRequirement: kubermaticv1.OpenstackNodeSizeRequirements{
				MinimumMemory: 4096,
				MinimumVCPUs:  1,
			},
			meetsRequirement: false,
		},
		{
			name: "not enough cpu",
			apiSize: apiv1.OpenstackSize{
				Memory: 2048,
				VCPUs:  2,
			},
			nodeSizeRequirement: kubermaticv1.OpenstackNodeSizeRequirements{
				MinimumMemory: 1024,
				MinimumVCPUs:  4,
			},
			meetsRequirement: false,
		},
		{
			name: "meets requirements",
			apiSize: apiv1.OpenstackSize{
				Memory: 2048,
				VCPUs:  2,
			},
			nodeSizeRequirement: kubermaticv1.OpenstackNodeSizeRequirements{
				MinimumMemory: 1024,
				MinimumVCPUs:  1,
			},
			meetsRequirement: true,
		},
		{
			name: "no requirements",
			apiSize: apiv1.OpenstackSize{
				Memory: 2048,
				VCPUs:  2,
			},
			nodeSizeRequirement: kubermaticv1.OpenstackNodeSizeRequirements{},
			meetsRequirement:    true,
		},
		{
			name: "required cpu equals size",
			apiSize: apiv1.OpenstackSize{
				VCPUs: 2,
			},
			nodeSizeRequirement: kubermaticv1.OpenstackNodeSizeRequirements{
				MinimumVCPUs: 2,
			},
			meetsRequirement: true,
		},
		{
			name: "required memory equals size",
			apiSize: apiv1.OpenstackSize{
				Memory: 2,
			},
			nodeSizeRequirement: kubermaticv1.OpenstackNodeSizeRequirements{
				MinimumMemory: 2,
			},
			meetsRequirement: true,
		},
	}
	for _, testToRun := range tests {
		t.Run(testToRun.name, func(t *testing.T) {
			if providercommon.MeetsOpenstackNodeSizeRequirement(testToRun.apiSize, testToRun.nodeSizeRequirement) != testToRun.meetsRequirement {
				t.Errorf("expected result to be %v, but got %v", testToRun.meetsRequirement, !testToRun.meetsRequirement)
			}
		})
	}
}

func compareJSON(t *testing.T, res *httptest.ResponseRecorder, expectedResponseString string) {
	t.Helper()
	var actualResponse interface{}
	var expectedResponse interface{}

	// var err error
	bBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read response body")
	}
	err = json.Unmarshal(bBytes, &actualResponse)
	if err != nil {
		t.Fatalf("Error marshaling string 1 :: %s", err.Error())
	}
	err = json.Unmarshal([]byte(expectedResponseString), &expectedResponse)
	if err != nil {
		t.Fatalf("Error marshaling string 2 :: %s", err.Error())
	}
	if !equality.Semantic.DeepEqual(actualResponse, expectedResponse) {
		t.Fatalf("Objects are different: %v", diff.ObjectDiff(actualResponse, expectedResponse))
	}
}
