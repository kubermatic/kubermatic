/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/test/diff"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNutanixUsername = "test"
	testNutanixPassword = "test"
	testNutanixDC1      = "nutanix-dc1"
	testNutanixDC2      = "nutanix-dc2"
)

type mockClientImpl struct {
	clusterList apiv1.NutanixClusterList
	projectList apiv1.NutanixProjectList
}

func TestNutanixClustersEndpoint(t *testing.T) {
	testcases := []struct {
		name             string
		dc               string
		password         string
		credential       string
		location         string
		httpStatus       int
		expectedResponse string
	}{
		{
			name:             "test unauthorized access",
			dc:               testNutanixDC1,
			httpStatus:       http.StatusBadRequest,
			expectedResponse: "",
		},
		{
			name:             "test invalid credential reference",
			dc:               testNutanixDC1,
			credential:       "invalid",
			httpStatus:       http.StatusInternalServerError,
			expectedResponse: "",
		},
		{
			name:       "test cluster list in dc 'nutanix-dc1'",
			dc:         testNutanixDC1,
			password:   testNutanixPassword,
			httpStatus: http.StatusOK,
			expectedResponse: `[
                {"name": "dc1-cluster"}
            ]`,
		},
		{
			name:       "test cluster list in dc 'nutanix-dc2'",
			dc:         testNutanixDC2,
			password:   testNutanixPassword,
			httpStatus: http.StatusOK,
			expectedResponse: `[
                {"name": "dc2-cluster1"},
                {"name": "dc2-cluster2"}
            ]`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/providers/nutanix/%s/clusters", tc.dc), strings.NewReader(""))

			req.Header.Add("NutanixUsername", testNutanixUsername)
			req.Header.Add("NutanixPassword", tc.password)
			req.Header.Add("Credential", tc.credential)

			providercommon.NewNutanixClient = mockNutanixClient

			apiUser := test.GetUser(test.UserEmail, test.UserID, test.UserName)

			res := httptest.NewRecorder()
			router, _, err := test.CreateTestEndpointAndGetClients(apiUser, buildNutanixDatacenterMeta(), []ctrlruntimeclient.Object{}, []ctrlruntimeclient.Object{}, []ctrlruntimeclient.Object{test.APIUserToKubermaticUser(apiUser)}, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
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

func TestNutanixProjectsEndpoint(t *testing.T) {
	testcases := []struct {
		name             string
		dc               string
		password         string
		credential       string
		location         string
		httpStatus       int
		expectedResponse string
	}{
		{
			name:             "test unauthorized access",
			dc:               testNutanixDC1,
			httpStatus:       http.StatusBadRequest,
			expectedResponse: "",
		},
		{
			name:             "test invalid credential reference",
			dc:               testNutanixDC1,
			credential:       "invalid",
			httpStatus:       http.StatusInternalServerError,
			expectedResponse: "",
		},
		{
			name:       "test project list in dc 'nutanix-dc1'",
			dc:         testNutanixDC1,
			password:   testNutanixPassword,
			httpStatus: http.StatusOK,
			expectedResponse: `[
                {"name": "dc1-project"}
            ]`,
		},
		{
			name:       "test project list in dc 'nutanix-dc2'",
			dc:         testNutanixDC2,
			password:   testNutanixPassword,
			httpStatus: http.StatusOK,
			expectedResponse: `[
                {"name": "dc2-project1"},
                {"name": "dc2-project2"}
            ]`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/providers/nutanix/%s/projects", tc.dc), strings.NewReader(""))

			req.Header.Add("NutanixUsername", testNutanixUsername)
			req.Header.Add("NutanixPassword", tc.password)
			req.Header.Add("Credential", tc.credential)

			providercommon.NewNutanixClient = mockNutanixClient

			apiUser := test.GetUser(test.UserEmail, test.UserID, test.UserName)

			res := httptest.NewRecorder()
			router, _, err := test.CreateTestEndpointAndGetClients(apiUser, buildNutanixDatacenterMeta(), []ctrlruntimeclient.Object{}, []ctrlruntimeclient.Object{}, []ctrlruntimeclient.Object{test.APIUserToKubermaticUser(apiUser)}, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
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

func buildNutanixDatacenterMeta() provider.SeedsGetter {
	return func() (map[string]*kubermaticv1.Seed, error) {
		return map[string]*kubermaticv1.Seed{
			"my-seed": {
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-seed",
				},
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{
						testNutanixDC1: {
							Location: "Hamburg",
							Country:  "DE",
							Spec: kubermaticv1.DatacenterSpec{
								Nutanix: &kubermaticv1.DatacenterSpecNutanix{
									Endpoint: "127.0.0.1",
								},
							},
						},
						testNutanixDC2: {
							Location: "Hamburg",
							Country:  "DE",
							Spec: kubermaticv1.DatacenterSpec{
								Nutanix: &kubermaticv1.DatacenterSpecNutanix{
									Endpoint: "127.0.0.2",
								},
							},
						},
					},
				},
			},
		}, nil
	}
}

func mockNutanixClient(dc *kubermaticv1.DatacenterSpecNutanix, creds *providercommon.NutanixCredentials) providercommon.NutanixClientSet {
	var (
		clusterList apiv1.NutanixClusterList
		projectList apiv1.NutanixProjectList
	)

	if dc.Endpoint == "127.0.0.1" {
		clusterList = apiv1.NutanixClusterList{
			{
				Name: "dc1-cluster",
			},
		}
		projectList = apiv1.NutanixProjectList{
			{
				Name: "dc1-project",
			},
		}
	} else if dc.Endpoint == "127.0.0.2" {
		clusterList = apiv1.NutanixClusterList{
			{
				Name: "dc2-cluster1",
			},
			{
				Name: "dc2-cluster2",
			},
		}
		projectList = apiv1.NutanixProjectList{
			{
				Name: "dc2-project1",
			},
			{
				Name: "dc2-project2",
			},
		}
	}

	return &mockClientImpl{
		clusterList: clusterList,
		projectList: projectList,
	}
}

func (m *mockClientImpl) ListNutanixClusters(ctx context.Context) (apiv1.NutanixClusterList, error) {
	return m.clusterList, nil
}

func (m *mockClientImpl) ListNutanixProjects(ctx context.Context) (apiv1.NutanixProjectList, error) {
	return m.projectList, nil
}

func (m *mockClientImpl) ListNutanixSubnets(ctx context.Context, clusterName, projectName string) (apiv1.NutanixSubnetList, error) {
	return nil, nil
}

func (m *mockClientImpl) ListNutanixCategories(ctx context.Context) (apiv1.NutanixCategoryList, error) {
	return nil, nil
}

func (m *mockClientImpl) ListNutanixCategoryValues(ctx context.Context, categoryName string) (apiv1.NutanixCategoryValueList, error) {
	return nil, nil
}

func compareJSON(t *testing.T, res *httptest.ResponseRecorder, expectedResponseString string) {
	t.Helper()
	var actualResponse interface{}
	var expectedResponse interface{}

	err := json.Unmarshal(res.Body.Bytes(), &actualResponse)
	if err != nil {
		t.Fatalf("Error marshaling string 1 :: %s", err.Error())
	}
	err = json.Unmarshal([]byte(expectedResponseString), &expectedResponse)
	if err != nil {
		t.Fatalf("Error marshaling string 2 :: %s", err.Error())
	}

	if !diff.SemanticallyEqual(expectedResponse, actualResponse) {
		t.Fatalf("Objects are different:\n%v", diff.ObjectDiff(expectedResponse, actualResponse))
	}
}
