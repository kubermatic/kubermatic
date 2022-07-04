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

package ipampool_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestListIPAMPools(t *testing.T) {
	testCases := []struct {
		name                  string
		existingObjects       []ctrlruntimeclient.Object
		apiUser               *apiv1.User
		expectedIPAMPools     []*apiv2.IPAMPool
		expectedHTTPStatus    int
		expectedErrorResponse []byte
	}{
		{
			name: "base case",
			existingObjects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/28",
								AllocationRange: 8,
							},
							"test-dc-2": {
								Type:             "prefix",
								PoolCIDR:         "192.168.1.0/27",
								AllocationPrefix: 28,
							},
						},
					},
				},
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-2",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-3": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/30",
								AllocationRange: 2,
							},
						},
					},
				},
			},
			apiUser:            test.GenDefaultAdminAPIUser(),
			expectedHTTPStatus: http.StatusOK,
			expectedIPAMPools: []*apiv2.IPAMPool{
				{
					Name: "test-pool-1",
					Datacenters: map[string]apiv2.IPAMPoolDatacenterSettings{
						"test-dc-1": {
							Type:            "range",
							PoolCIDR:        "192.168.1.0/28",
							AllocationRange: 8,
						},
						"test-dc-2": {
							Type:             "prefix",
							PoolCIDR:         "192.168.1.0/27",
							AllocationPrefix: 28,
						},
					},
				},
				{
					Name: "test-pool-2",
					Datacenters: map[string]apiv2.IPAMPoolDatacenterSettings{
						"test-dc-3": {
							Type:            "range",
							PoolCIDR:        "192.168.1.0/30",
							AllocationRange: 2,
						},
					},
				},
			},
		},
		{
			name: "non-admin",
			existingObjects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/28",
								AllocationRange: 8,
							},
						},
					},
				},
			},
			apiUser:               test.GenDefaultAPIUser(),
			expectedHTTPStatus:    http.StatusForbidden,
			expectedErrorResponse: []byte("{\"error\":{\"code\":403,\"message\":\"bob@acme.com doesn't have admin rights\"}}\n"),
		},
		{
			name:               "empty list",
			existingObjects:    []ctrlruntimeclient.Object{},
			apiUser:            test.GenDefaultAdminAPIUser(),
			expectedHTTPStatus: http.StatusOK,
			expectedIPAMPools:  []*apiv2.IPAMPool{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.existingObjects = append(tc.existingObjects, test.APIUserToKubermaticUser(*tc.apiUser))

			req := httptest.NewRequest("GET", "/api/v2/ipampools", strings.NewReader(""))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.apiUser, nil, tc.existingObjects, nil, hack.NewTestRouting)
			assert.NoError(t, err)

			ep.ServeHTTP(res, req)

			assert.Equal(t, tc.expectedHTTPStatus, res.Code)

			if res.Code == http.StatusOK {
				var ipamPools []*apiv2.IPAMPool
				err = json.Unmarshal(res.Body.Bytes(), &ipamPools)
				assert.NoError(t, err)

				assert.Equal(t, tc.expectedIPAMPools, ipamPools)
			} else {
				assert.Equal(t, tc.expectedErrorResponse, res.Body.Bytes())
			}
		})
	}
}

func TestGetIPAMPool(t *testing.T) {
	testCases := []struct {
		name                  string
		existingObjects       []ctrlruntimeclient.Object
		apiUser               *apiv1.User
		ipamPoolName          string
		expectedIPAMPool      *apiv2.IPAMPool
		expectedHTTPStatus    int
		expectedErrorResponse []byte
	}{
		{
			name: "base case",
			existingObjects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/28",
								AllocationRange: 8,
							},
							"test-dc-2": {
								Type:             "prefix",
								PoolCIDR:         "192.168.1.0/27",
								AllocationPrefix: 28,
							},
						},
					},
				},
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-2",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-3": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/30",
								AllocationRange: 2,
							},
						},
					},
				},
			},
			apiUser:            test.GenDefaultAdminAPIUser(),
			ipamPoolName:       "test-pool-1",
			expectedHTTPStatus: http.StatusOK,
			expectedIPAMPool: &apiv2.IPAMPool{
				Name: "test-pool-1",
				Datacenters: map[string]apiv2.IPAMPoolDatacenterSettings{
					"test-dc-1": {
						Type:            "range",
						PoolCIDR:        "192.168.1.0/28",
						AllocationRange: 8,
					},
					"test-dc-2": {
						Type:             "prefix",
						PoolCIDR:         "192.168.1.0/27",
						AllocationPrefix: 28,
					},
				},
			},
		},
		{
			name: "non-admin",
			existingObjects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/28",
								AllocationRange: 8,
							},
						},
					},
				},
			},
			apiUser:               test.GenDefaultAPIUser(),
			ipamPoolName:          "test-pool-1",
			expectedHTTPStatus:    http.StatusForbidden,
			expectedErrorResponse: []byte("{\"error\":{\"code\":403,\"message\":\"bob@acme.com doesn't have admin rights\"}}\n"),
		},
		{
			name:                  "not found",
			existingObjects:       []ctrlruntimeclient.Object{},
			apiUser:               test.GenDefaultAdminAPIUser(),
			ipamPoolName:          "test-pool-1",
			expectedHTTPStatus:    http.StatusNotFound,
			expectedErrorResponse: []byte("{\"error\":{\"code\":404,\"message\":\"IPAMPool \\\"test-pool-1\\\" not found\"}}\n"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.existingObjects = append(tc.existingObjects, test.APIUserToKubermaticUser(*tc.apiUser))

			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/ipampools/%s", tc.ipamPoolName), strings.NewReader(""))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.apiUser, nil, tc.existingObjects, nil, hack.NewTestRouting)
			assert.NoError(t, err)

			ep.ServeHTTP(res, req)

			assert.Equal(t, tc.expectedHTTPStatus, res.Code)

			if res.Code == http.StatusOK {
				ipamPool := &apiv2.IPAMPool{}
				err = json.Unmarshal(res.Body.Bytes(), ipamPool)
				assert.NoError(t, err)

				assert.Equal(t, tc.expectedIPAMPool, ipamPool)
			} else {
				assert.Equal(t, tc.expectedErrorResponse, res.Body.Bytes())
			}
		})
	}
}

func TestCreateIPAMPool(t *testing.T) {
	testCases := []struct {
		name               string
		existingObjects    []ctrlruntimeclient.Object
		apiUser            *apiv1.User
		ipamPool           *apiv2.IPAMPool
		expectedHTTPStatus int
		expectedResponse   []byte
	}{
		{
			name:            "base case",
			existingObjects: []ctrlruntimeclient.Object{},
			apiUser:         test.GenDefaultAdminAPIUser(),
			ipamPool: &apiv2.IPAMPool{
				Name: "test-pool-1",
				Datacenters: map[string]apiv2.IPAMPoolDatacenterSettings{
					"test-dc-2": {
						Type:            "range",
						PoolCIDR:        "192.168.1.0/28",
						AllocationRange: 8,
					},
					"test-dc-3": {
						Type:             "prefix",
						PoolCIDR:         "192.168.1.0/27",
						AllocationPrefix: 28,
					},
				},
			},
			expectedHTTPStatus: http.StatusOK,
			expectedResponse:   []byte("{}"),
		},
		{
			name:            "non-admin",
			existingObjects: []ctrlruntimeclient.Object{},
			apiUser:         test.GenDefaultAPIUser(),
			ipamPool: &apiv2.IPAMPool{
				Name: "test-pool-1",
				Datacenters: map[string]apiv2.IPAMPoolDatacenterSettings{
					"test-dc-2": {
						Type:            "range",
						PoolCIDR:        "192.168.1.0/28",
						AllocationRange: 8,
					},
					"test-dc-3": {
						Type:             "prefix",
						PoolCIDR:         "192.168.1.0/27",
						AllocationPrefix: 28,
					},
				},
			},
			expectedHTTPStatus: http.StatusForbidden,
			expectedResponse:   []byte("{\"error\":{\"code\":403,\"message\":\"bob@acme.com doesn't have admin rights\"}}\n"),
		},
		{
			name: "already exists",
			existingObjects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/28",
								AllocationRange: 8,
							},
						},
					},
				},
			},
			apiUser: test.GenDefaultAdminAPIUser(),
			ipamPool: &apiv2.IPAMPool{
				Name: "test-pool-1",
				Datacenters: map[string]apiv2.IPAMPoolDatacenterSettings{
					"test-dc-2": {
						Type:            "range",
						PoolCIDR:        "192.168.1.0/28",
						AllocationRange: 8,
					},
					"test-dc-3": {
						Type:             "prefix",
						PoolCIDR:         "192.168.1.0/27",
						AllocationPrefix: 28,
					},
				},
			},
			expectedHTTPStatus: http.StatusConflict,
			expectedResponse:   []byte("{\"error\":{\"code\":409,\"message\":\"IPAMPool \\\"test-pool-1\\\" already exists\"}}\n"),
		},
		{
			name:               "invalid: empty name",
			existingObjects:    []ctrlruntimeclient.Object{},
			apiUser:            test.GenDefaultAdminAPIUser(),
			ipamPool:           &apiv2.IPAMPool{},
			expectedHTTPStatus: http.StatusBadRequest,
			expectedResponse:   []byte("{\"error\":{\"code\":400,\"message\":\"missing attribute \\\"name\\\"\"}}\n"),
		},
		{
			name:            "invalid: missing datacenters",
			existingObjects: []ctrlruntimeclient.Object{},
			apiUser:         test.GenDefaultAdminAPIUser(),
			ipamPool: &apiv2.IPAMPool{
				Name: "test-pool-1",
			},
			expectedHTTPStatus: http.StatusBadRequest,
			expectedResponse:   []byte("{\"error\":{\"code\":400,\"message\":\"missing or empty attribute \\\"datacenters\\\"\"}}\n"),
		},
		{
			name:            "invalid: empty datacenters",
			existingObjects: []ctrlruntimeclient.Object{},
			apiUser:         test.GenDefaultAdminAPIUser(),
			ipamPool: &apiv2.IPAMPool{
				Name:        "test-pool-1",
				Datacenters: map[string]apiv2.IPAMPoolDatacenterSettings{},
			},
			expectedHTTPStatus: http.StatusBadRequest,
			expectedResponse:   []byte("{\"error\":{\"code\":400,\"message\":\"missing or empty attribute \\\"datacenters\\\"\"}}\n"),
		},
		{
			name:            "invalid: missing dc pool cidr",
			existingObjects: []ctrlruntimeclient.Object{},
			apiUser:         test.GenDefaultAdminAPIUser(),
			ipamPool: &apiv2.IPAMPool{
				Name: "test-pool-1",
				Datacenters: map[string]apiv2.IPAMPoolDatacenterSettings{
					"test-dc-1": {},
				},
			},
			expectedHTTPStatus: http.StatusBadRequest,
			expectedResponse:   []byte("{\"error\":{\"code\":400,\"message\":\"missing attribute \\\"poolCidr\\\" for datacenter test-dc-1\"}}\n"),
		},
		{
			name:            "invalid: missing dc pool type",
			existingObjects: []ctrlruntimeclient.Object{},
			apiUser:         test.GenDefaultAdminAPIUser(),
			ipamPool: &apiv2.IPAMPool{
				Name: "test-pool-1",
				Datacenters: map[string]apiv2.IPAMPoolDatacenterSettings{
					"test-dc-1": {
						PoolCIDR: "192.168.1.0/28",
					},
				},
			},
			expectedHTTPStatus: http.StatusBadRequest,
			expectedResponse:   []byte("{\"error\":{\"code\":400,\"message\":\"missing attribute \\\"type\\\" for datacenter test-dc-1\"}}\n"),
		},
		{
			name:            "invalid: missing dc pool allocation prefix",
			existingObjects: []ctrlruntimeclient.Object{},
			apiUser:         test.GenDefaultAdminAPIUser(),
			ipamPool: &apiv2.IPAMPool{
				Name: "test-pool-1",
				Datacenters: map[string]apiv2.IPAMPoolDatacenterSettings{
					"test-dc-1": {
						PoolCIDR: "192.168.1.0/28",
						Type:     "prefix",
					},
				},
			},
			expectedHTTPStatus: http.StatusBadRequest,
			expectedResponse:   []byte("{\"error\":{\"code\":400,\"message\":\"missing attribute \\\"allocationPrefix\\\" for datacenter test-dc-1\"}}\n"),
		},
		{
			name:            "invalid: missing dc pool allocation range",
			existingObjects: []ctrlruntimeclient.Object{},
			apiUser:         test.GenDefaultAdminAPIUser(),
			ipamPool: &apiv2.IPAMPool{
				Name: "test-pool-1",
				Datacenters: map[string]apiv2.IPAMPoolDatacenterSettings{
					"test-dc-1": {
						PoolCIDR: "192.168.1.0/28",
						Type:     "range",
					},
				},
			},
			expectedHTTPStatus: http.StatusBadRequest,
			expectedResponse:   []byte("{\"error\":{\"code\":400,\"message\":\"missing attribute \\\"allocationRange\\\" for datacenter test-dc-1\"}}\n"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.existingObjects = append(tc.existingObjects, test.APIUserToKubermaticUser(*tc.apiUser))

			reqBody, err := json.Marshal(tc.ipamPool)
			assert.NoError(t, err)

			req := httptest.NewRequest("POST", "/api/v2/ipampools", bytes.NewReader(reqBody))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.apiUser, nil, tc.existingObjects, nil, hack.NewTestRouting)
			assert.NoError(t, err)

			ep.ServeHTTP(res, req)

			assert.Equal(t, tc.expectedHTTPStatus, res.Code)

			assert.Equal(t, tc.expectedResponse, res.Body.Bytes())
		})
	}
}

func TestDeleteIPAMPool(t *testing.T) {
	testCases := []struct {
		name               string
		existingObjects    []ctrlruntimeclient.Object
		apiUser            *apiv1.User
		ipamPoolName       string
		expectedHTTPStatus int
		expectedResponse   []byte
	}{
		{
			name: "base case",
			existingObjects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/28",
								AllocationRange: 8,
							},
						},
					},
				},
			},
			apiUser:            test.GenDefaultAdminAPIUser(),
			ipamPoolName:       "test-pool-1",
			expectedHTTPStatus: http.StatusOK,
			expectedResponse:   []byte("{}"),
		},
		{
			name: "non-admin",
			existingObjects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/28",
								AllocationRange: 8,
							},
						},
					},
				},
			},
			apiUser:            test.GenDefaultAPIUser(),
			ipamPoolName:       "test-pool-1",
			expectedHTTPStatus: http.StatusForbidden,
			expectedResponse:   []byte("{\"error\":{\"code\":403,\"message\":\"bob@acme.com doesn't have admin rights\"}}\n"),
		},
		{
			name:               "not found",
			existingObjects:    []ctrlruntimeclient.Object{},
			apiUser:            test.GenDefaultAdminAPIUser(),
			ipamPoolName:       "test-pool-1",
			expectedHTTPStatus: http.StatusNotFound,
			expectedResponse:   []byte("{\"error\":{\"code\":404,\"message\":\"IPAMPool \\\"test-pool-1\\\" not found\"}}\n"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.existingObjects = append(tc.existingObjects, test.APIUserToKubermaticUser(*tc.apiUser))

			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v2/ipampools/%s", tc.ipamPoolName), strings.NewReader(""))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.apiUser, nil, tc.existingObjects, nil, hack.NewTestRouting)
			assert.NoError(t, err)

			ep.ServeHTTP(res, req)

			assert.Equal(t, tc.expectedHTTPStatus, res.Code)

			assert.Equal(t, tc.expectedResponse, res.Body.Bytes())
		})
	}
}

func TestPatchIPAMPool(t *testing.T) {
	testCases := []struct {
		name               string
		existingObjects    []ctrlruntimeclient.Object
		apiUser            *apiv1.User
		ipamPool           *apiv2.IPAMPool
		expectedHTTPStatus int
		expectedResponse   []byte
	}{
		{
			name: "base case",
			existingObjects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/28",
								AllocationRange: 8,
							},
						},
					},
				},
			},
			apiUser: test.GenDefaultAdminAPIUser(),
			ipamPool: &apiv2.IPAMPool{
				Name: "test-pool-1",
				Datacenters: map[string]apiv2.IPAMPoolDatacenterSettings{
					"test-dc-1": {
						Type:            "range",
						PoolCIDR:        "192.168.1.0/28",
						AllocationRange: 8,
					},
					"test-dc-2": {
						Type:             "prefix",
						PoolCIDR:         "192.168.1.0/27",
						AllocationPrefix: 28,
					},
				},
			},
			expectedHTTPStatus: http.StatusOK,
			expectedResponse:   []byte("{}"),
		},
		{
			name: "non-admin",
			existingObjects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/28",
								AllocationRange: 8,
							},
						},
					},
				},
			},
			apiUser: test.GenDefaultAPIUser(),
			ipamPool: &apiv2.IPAMPool{
				Name: "test-pool-1",
				Datacenters: map[string]apiv2.IPAMPoolDatacenterSettings{
					"test-dc-1": {
						Type:            "range",
						PoolCIDR:        "192.168.1.0/28",
						AllocationRange: 8,
					},
					"test-dc-2": {
						Type:             "prefix",
						PoolCIDR:         "192.168.1.0/27",
						AllocationPrefix: 28,
					},
				},
			},
			expectedHTTPStatus: http.StatusForbidden,
			expectedResponse:   []byte("{\"error\":{\"code\":403,\"message\":\"bob@acme.com doesn't have admin rights\"}}\n"),
		},
		{
			name:            "not found",
			existingObjects: []ctrlruntimeclient.Object{},
			apiUser:         test.GenDefaultAdminAPIUser(),
			ipamPool: &apiv2.IPAMPool{
				Name: "test-pool-1",
				Datacenters: map[string]apiv2.IPAMPoolDatacenterSettings{
					"test-dc-1": {
						Type:            "range",
						PoolCIDR:        "192.168.1.0/28",
						AllocationRange: 8,
					},
				},
			},
			expectedHTTPStatus: http.StatusNotFound,
			expectedResponse:   []byte("{\"error\":{\"code\":404,\"message\":\"IPAMPool \\\"test-pool-1\\\" not found\"}}\n"),
		},
		{
			name:            "invalid: missing datacenters",
			existingObjects: []ctrlruntimeclient.Object{},
			apiUser:         test.GenDefaultAdminAPIUser(),
			ipamPool: &apiv2.IPAMPool{
				Name: "test-pool-1",
			},
			expectedHTTPStatus: http.StatusBadRequest,
			expectedResponse:   []byte("{\"error\":{\"code\":400,\"message\":\"missing or empty attribute \\\"datacenters\\\"\"}}\n"),
		},
		{
			name:            "invalid: empty datacenters",
			existingObjects: []ctrlruntimeclient.Object{},
			apiUser:         test.GenDefaultAdminAPIUser(),
			ipamPool: &apiv2.IPAMPool{
				Name:        "test-pool-1",
				Datacenters: map[string]apiv2.IPAMPoolDatacenterSettings{},
			},
			expectedHTTPStatus: http.StatusBadRequest,
			expectedResponse:   []byte("{\"error\":{\"code\":400,\"message\":\"missing or empty attribute \\\"datacenters\\\"\"}}\n"),
		},
		{
			name:            "invalid: missing dc pool cidr",
			existingObjects: []ctrlruntimeclient.Object{},
			apiUser:         test.GenDefaultAdminAPIUser(),
			ipamPool: &apiv2.IPAMPool{
				Name: "test-pool-1",
				Datacenters: map[string]apiv2.IPAMPoolDatacenterSettings{
					"test-dc-1": {},
				},
			},
			expectedHTTPStatus: http.StatusBadRequest,
			expectedResponse:   []byte("{\"error\":{\"code\":400,\"message\":\"missing attribute \\\"poolCidr\\\" for datacenter test-dc-1\"}}\n"),
		},
		{
			name:            "invalid: missing dc pool type",
			existingObjects: []ctrlruntimeclient.Object{},
			apiUser:         test.GenDefaultAdminAPIUser(),
			ipamPool: &apiv2.IPAMPool{
				Name: "test-pool-1",
				Datacenters: map[string]apiv2.IPAMPoolDatacenterSettings{
					"test-dc-1": {
						PoolCIDR: "192.168.1.0/28",
					},
				},
			},
			expectedHTTPStatus: http.StatusBadRequest,
			expectedResponse:   []byte("{\"error\":{\"code\":400,\"message\":\"missing attribute \\\"type\\\" for datacenter test-dc-1\"}}\n"),
		},
		{
			name:            "invalid: missing dc pool allocation prefix",
			existingObjects: []ctrlruntimeclient.Object{},
			apiUser:         test.GenDefaultAdminAPIUser(),
			ipamPool: &apiv2.IPAMPool{
				Name: "test-pool-1",
				Datacenters: map[string]apiv2.IPAMPoolDatacenterSettings{
					"test-dc-1": {
						PoolCIDR: "192.168.1.0/28",
						Type:     "prefix",
					},
				},
			},
			expectedHTTPStatus: http.StatusBadRequest,
			expectedResponse:   []byte("{\"error\":{\"code\":400,\"message\":\"missing attribute \\\"allocationPrefix\\\" for datacenter test-dc-1\"}}\n"),
		},
		{
			name:            "invalid: missing dc pool allocation range",
			existingObjects: []ctrlruntimeclient.Object{},
			apiUser:         test.GenDefaultAdminAPIUser(),
			ipamPool: &apiv2.IPAMPool{
				Name: "test-pool-1",
				Datacenters: map[string]apiv2.IPAMPoolDatacenterSettings{
					"test-dc-1": {
						PoolCIDR: "192.168.1.0/28",
						Type:     "range",
					},
				},
			},
			expectedHTTPStatus: http.StatusBadRequest,
			expectedResponse:   []byte("{\"error\":{\"code\":400,\"message\":\"missing attribute \\\"allocationRange\\\" for datacenter test-dc-1\"}}\n"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.existingObjects = append(tc.existingObjects, test.APIUserToKubermaticUser(*tc.apiUser))

			reqBody, err := json.Marshal(tc.ipamPool)
			assert.NoError(t, err)

			req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v2/ipampools/%s", tc.ipamPool.Name), bytes.NewReader(reqBody))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.apiUser, nil, tc.existingObjects, nil, hack.NewTestRouting)
			assert.NoError(t, err)

			ep.ServeHTTP(res, req)

			assert.Equal(t, tc.expectedHTTPStatus, res.Code)

			assert.Equal(t, tc.expectedResponse, res.Body.Bytes())
		})
	}
}
