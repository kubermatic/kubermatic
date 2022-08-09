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
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	"k8c.io/kubermatic/v2/pkg/provider"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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

func setupOpenstackServer(t *testing.T) {
	openstackMux = http.NewServeMux()
	openstackServer = httptest.NewServer(openstackMux)

	openstackService := []struct {
		OpenstackURL        string
		JSONResponse        string
		ExpectedQueryParams map[string]string
	}{
		{
			OpenstackURL: "/v3/auth/tokens",
			JSONResponse: PostTokens,
		},
		{
			OpenstackURL: "/v2.0/subnetpools",
			JSONResponse: GetSubnetPools,
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
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
			} else {
				w.WriteHeader(http.StatusOK)
			}

			_, err := w.Write(buf.Bytes())
			if err != nil {
				t.Fatalf("failed to write rendered template to HTTP response: %v", err)
			}
		})
	}
}

func teardownOpenstackServer() {
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
		Name             string
		URL              string
		QueryParams      map[string]string
		Credential       string
		Credentials      []ctrlruntimeclient.Object
		ExpectedResponse string
	}{
		{
			Name: "test subnet pools endpoint",
			URL:  "/api/v2/providers/openstack/subnetpools",
			ExpectedResponse: `[
				{
				   "id":"03f761e6-eee0-43fc-a921-8acf64c14988",
				   "name":"my-subnet-pool-ipv6",
				   "ipVersion":6,
				   "isDefault":false,
				   "prefixes":[
						"2001:db8:0:2::/64",
						"2001:db8::/63"
					]
				},
				{
					"id":"f49a1319-423a-4ee6-ba54-1d95a4f6cc68",
					"name":"my-subnet-pool-ipv4",
					"ipVersion":4,
					"isDefault":false,
					"prefixes":[
						"10.10.0.0/21",
						"192.168.0.0/16"
					]
				 }
			 ]`,
		},
	}

	setupOpenstackServer(t)
	defer teardownOpenstackServer()

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.URL, strings.NewReader(""))
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
			credentials := []ctrlruntimeclient.Object{}
			if tc.Credentials != nil {
				credentials = tc.Credentials
			}
			router, _, err := test.CreateTestEndpointAndGetClients(*apiUser, buildOpenstackDatacenter(), []ctrlruntimeclient.Object{}, credentials, []ctrlruntimeclient.Object{test.APIUserToKubermaticUser(*apiUser)}, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			router.ServeHTTP(res, req)
			compareJSON(t, res, tc.ExpectedResponse)
		})
	}
}
