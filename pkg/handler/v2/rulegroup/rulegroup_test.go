/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package rulegroup_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetEndpoint(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name                      string
		RuleGroupName             string
		ProjectID                 string
		ClusterID                 string
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
		ExpectedResponse          *apiv2.RuleGroup
		ExpectedHTTPStatusCode    int
	}{
		{
			Name:          "get rule group that belongs to the given cluster",
			RuleGroupName: "test-rule-group",
			ProjectID:     test.GenDefaultProject().Name,
			ClusterID:     test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenRuleGroup("test-rule-group", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse:       test.GenAPIRuleGroup("test-rule-group", v1.RuleGroupTypeMetrics),
		},
		{
			Name:          "get rule group which doesn't exist",
			RuleGroupName: "test-rule-group",
			ProjectID:     test.GenDefaultProject().Name,
			ClusterID:     test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusNotFound,
		},
		{
			Name:          "user john cannot get rule group that belongs to bob's cluster",
			RuleGroupName: "test-rule-group",
			ProjectID:     test.GenDefaultProject().Name,
			ClusterID:     test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", false),
				test.GenRuleGroup("test-rule-group", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatusCode: http.StatusForbidden,
		},
		{
			Name:          "admin user john can get rule group that belongs to bob's cluster",
			RuleGroupName: "test-rule-group",
			ProjectID:     test.GenDefaultProject().Name,
			ClusterID:     test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
				test.GenRuleGroup("test-rule-group", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse:       test.GenAPIRuleGroup("test-rule-group", v1.RuleGroupTypeMetrics),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			requestURL := fmt.Sprintf("/api/v2/projects/%s/clusters/%s/rulegroups/%s", tc.ProjectID, tc.ClusterID, tc.RuleGroupName)
			req := httptest.NewRequest(http.MethodGet, requestURL, nil)
			resp := httptest.NewRecorder()

			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, nil, tc.ExistingKubermaticObjects, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to: %v", err)
			}
			ep.ServeHTTP(resp, req)

			if resp.Code != tc.ExpectedHTTPStatusCode {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatusCode, resp.Code, resp.Body.String())
			}
			if resp.Code == http.StatusOK {
				b, err := json.Marshal(tc.ExpectedResponse)
				if err != nil {
					t.Fatalf("failed to marshal expected response: %v", err)
				}

				test.CompareWithResult(t, resp, string(b))
			}

		})
	}
}

func TestListEndpoint(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name                      string
		ProjectID                 string
		ClusterID                 string
		QueryParams               map[string]string
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
		ExpectedResponse          []*apiv2.RuleGroup
		ExpectedHTTPStatusCode    int
	}{
		{
			Name:      "list all rule groups that belong to the given cluster",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenRuleGroup("test-1", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-2", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-3", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse: []*apiv2.RuleGroup{
				test.GenAPIRuleGroup("test-1", v1.RuleGroupTypeMetrics),
				test.GenAPIRuleGroup("test-2", v1.RuleGroupTypeMetrics),
				test.GenAPIRuleGroup("test-3", v1.RuleGroupTypeMetrics),
			},
		},
		{
			Name:      "list rule groups when there is no rule groups",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse:       []*apiv2.RuleGroup{},
		},
		{
			Name:        "list all rule groups that belong to the given cluster with empty query parameters",
			ProjectID:   test.GenDefaultProject().Name,
			ClusterID:   test.GenDefaultCluster().Name,
			QueryParams: map[string]string{},
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenRuleGroup("test-1", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-2", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-3", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse: []*apiv2.RuleGroup{
				test.GenAPIRuleGroup("test-1", v1.RuleGroupTypeMetrics),
				test.GenAPIRuleGroup("test-2", v1.RuleGroupTypeMetrics),
				test.GenAPIRuleGroup("test-3", v1.RuleGroupTypeMetrics),
			},
		},
		{
			Name:        "list rule groups with type Metrics that belong to the given cluster",
			ProjectID:   test.GenDefaultProject().Name,
			ClusterID:   test.GenDefaultCluster().Name,
			QueryParams: map[string]string{"type": "Metrics"},
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenRuleGroup("test-1", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-2", test.GenDefaultCluster().Name, "FakeType"),
				test.GenRuleGroup("test-3", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-4", test.GenDefaultCluster().Name, v1.RuleGroupTypeLogs),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse: []*apiv2.RuleGroup{
				test.GenAPIRuleGroup("test-1", v1.RuleGroupTypeMetrics),
				test.GenAPIRuleGroup("test-3", v1.RuleGroupTypeMetrics),
			},
		},
		{
			Name:        "list rule groups with type Logs that belong to the given cluster",
			ProjectID:   test.GenDefaultProject().Name,
			ClusterID:   test.GenDefaultCluster().Name,
			QueryParams: map[string]string{"type": "Logs"},
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenRuleGroup("test-1", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-2", test.GenDefaultCluster().Name, "FakeType"),
				test.GenRuleGroup("test-3", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-4", test.GenDefaultCluster().Name, v1.RuleGroupTypeLogs),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse: []*apiv2.RuleGroup{
				test.GenAPIRuleGroup("test-4", v1.RuleGroupTypeLogs),
			},
		},
		{
			Name:        "list rule groups with invalid type",
			ProjectID:   test.GenDefaultProject().Name,
			ClusterID:   test.GenDefaultCluster().Name,
			QueryParams: map[string]string{"type": "FakeType"},
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenRuleGroup("test-1", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-2", test.GenDefaultCluster().Name, "FakeType"),
				test.GenRuleGroup("test-3", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusBadRequest,
		},
		{
			Name:      "user john cannot list rule groups that belong to bob's cluster",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", false),
				test.GenRuleGroup("test-1", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-2", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-3", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatusCode: http.StatusForbidden,
		},
		{
			Name:      "admin user john can list rule groups that belong to bob's cluster",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
				test.GenRuleGroup("test-1", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-2", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-3", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse: []*apiv2.RuleGroup{
				test.GenAPIRuleGroup("test-1", v1.RuleGroupTypeMetrics),
				test.GenAPIRuleGroup("test-2", v1.RuleGroupTypeMetrics),
				test.GenAPIRuleGroup("test-3", v1.RuleGroupTypeMetrics),
			},
		},
		{
			Name:        "admin user john can list rule groups with type Metrics that belong to bob's cluster",
			ProjectID:   test.GenDefaultProject().Name,
			ClusterID:   test.GenDefaultCluster().Name,
			QueryParams: map[string]string{"type": "Metrics"},
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
				test.GenRuleGroup("test-1", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-2", test.GenDefaultCluster().Name, "FakeType"),
				test.GenRuleGroup("test-3", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse: []*apiv2.RuleGroup{
				test.GenAPIRuleGroup("test-1", v1.RuleGroupTypeMetrics),
				test.GenAPIRuleGroup("test-3", v1.RuleGroupTypeMetrics),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			requestURL := fmt.Sprintf("/api/v2/projects/%s/clusters/%s/rulegroups", tc.ProjectID, tc.ClusterID)
			req := httptest.NewRequest(http.MethodGet, requestURL, nil)
			if tc.QueryParams != nil {
				q := req.URL.Query()
				for k, v := range tc.QueryParams {
					q.Add(k, v)
				}
				req.URL.RawQuery = q.Encode()
			}
			resp := httptest.NewRecorder()

			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, nil, tc.ExistingKubermaticObjects, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to: %v", err)
			}
			ep.ServeHTTP(resp, req)

			if resp.Code != tc.ExpectedHTTPStatusCode {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatusCode, resp.Code, resp.Body.String())
			}
			if resp.Code == http.StatusOK {
				ruleGroups := test.NewRuleGroupSliceWrapper{}
				ruleGroups.DecodeOrDie(resp.Body, t).Sort()

				expectedRuleGroups := test.NewRuleGroupSliceWrapper(tc.ExpectedResponse)
				expectedRuleGroups.Sort()

				ruleGroups.EqualOrDie(expectedRuleGroups, t)
			}
		})
	}
}

func TestCreateEndpoint(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name                      string
		ProjectID                 string
		ClusterID                 string
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
		RuleGroup                 *apiv2.RuleGroup
		ExpectedHTTPStatusCode    int
		ExpectedResponse          *apiv2.RuleGroup
	}{
		{
			Name:      "create rule group in the given cluster",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			RuleGroup:              test.GenAPIRuleGroup("test-rule-group", v1.RuleGroupTypeMetrics),
			ExpectedHTTPStatusCode: http.StatusCreated,
			ExpectedResponse:       test.GenAPIRuleGroup("test-rule-group", v1.RuleGroupTypeMetrics),
		},
		{
			Name:      "cannot create rule group in the given cluster because it already exists",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenRuleGroup("test-rule-group", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			RuleGroup:              test.GenAPIRuleGroup("test-rule-group", v1.RuleGroupTypeMetrics),
			ExpectedHTTPStatusCode: http.StatusConflict,
		},
		{
			Name:      "cannot create rule group in the given cluster because the name in data is empty",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			RuleGroup:              test.GenAPIRuleGroup("", v1.RuleGroupTypeMetrics),
			ExpectedHTTPStatusCode: http.StatusBadRequest,
		},
		{
			Name:      "cannot create rule group in the given cluster because the in data cannot be unmarshalled into yaml",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
			RuleGroup: &apiv2.RuleGroup{
				Data: []byte("fake data"),
				Type: v1.RuleGroupTypeMetrics,
			},
			ExpectedHTTPStatusCode: http.StatusBadRequest,
		},
		{
			Name:      "user john cannot get rule group that belongs to bob's cluster",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", false),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			RuleGroup:              test.GenAPIRuleGroup("test-rule-group", v1.RuleGroupTypeMetrics),
			ExpectedHTTPStatusCode: http.StatusForbidden,
			ExpectedResponse:       test.GenAPIRuleGroup("test-rule-group", v1.RuleGroupTypeMetrics),
		},
		{
			Name:      "admin user john can get rule group that belongs to bob's cluster",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			RuleGroup:              test.GenAPIRuleGroup("test-rule-group", v1.RuleGroupTypeMetrics),
			ExpectedHTTPStatusCode: http.StatusCreated,
			ExpectedResponse:       test.GenAPIRuleGroup("test-rule-group", v1.RuleGroupTypeMetrics),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			requestURL := fmt.Sprintf("/api/v2/projects/%s/clusters/%s/rulegroups", tc.ProjectID, tc.ClusterID)
			body, err := json.Marshal(tc.RuleGroup)
			if err != nil {
				t.Fatalf("failed to marshalling rule group: %v", err)
			}
			req := httptest.NewRequest(http.MethodPost, requestURL, bytes.NewBuffer(body))
			resp := httptest.NewRecorder()

			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, nil, tc.ExistingKubermaticObjects, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to: %v", err)
			}
			ep.ServeHTTP(resp, req)

			if resp.Code != tc.ExpectedHTTPStatusCode {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatusCode, resp.Code, resp.Body.String())
			}
			if resp.Code == http.StatusCreated {
				b, err := json.Marshal(tc.ExpectedResponse)
				if err != nil {
					t.Fatalf("failed to marshal expected response %v", err)
				}
				test.CompareWithResult(t, resp, string(b))
			}
		})
	}
}

func TestUpdateEndpoint(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name                      string
		RuleGroupName             string
		ProjectID                 string
		ClusterID                 string
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
		RuleGroup                 *apiv2.RuleGroup
		ExpectedHTTPStatusCode    int
		ExpectedResponse          *apiv2.RuleGroup
	}{
		{
			Name:          "update rule group in the given cluster",
			RuleGroupName: "test-rule-group",
			ProjectID:     test.GenDefaultProject().Name,
			ClusterID:     test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenRuleGroup("test-rule-group", test.GenDefaultCluster().Name, "UpdateThisType"),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			RuleGroup:              test.GenAPIRuleGroup("test-rule-group", v1.RuleGroupTypeMetrics),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse:       test.GenAPIRuleGroup("test-rule-group", v1.RuleGroupTypeMetrics),
		},
		{
			Name:          "cannot update rule group in the given cluster because it doesn't exists",
			RuleGroupName: "test-rule-group",
			ProjectID:     test.GenDefaultProject().Name,
			ClusterID:     test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			RuleGroup:              test.GenAPIRuleGroup("test-rule-group", v1.RuleGroupTypeMetrics),
			ExpectedHTTPStatusCode: http.StatusNotFound,
		},
		{
			Name:          "cannot update rule group name in the data",
			RuleGroupName: "test-rule-group",
			ProjectID:     test.GenDefaultProject().Name,
			ClusterID:     test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenRuleGroup("test-rule-group", test.GenDefaultCluster().Name, "UpdateThisType"),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			RuleGroup:              test.GenAPIRuleGroup("test-rule-group-2", v1.RuleGroupTypeMetrics),
			ExpectedHTTPStatusCode: http.StatusBadRequest,
		},
		{
			Name:          "cannot update rule group in the given cluster because the in data cannot be unmarshalled into yaml",
			RuleGroupName: "test-rule-group",
			ProjectID:     test.GenDefaultProject().Name,
			ClusterID:     test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenRuleGroup("test-rule-group", test.GenDefaultCluster().Name, "UpdateThisType"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
			RuleGroup: &apiv2.RuleGroup{
				Data: []byte("fake data"),
				Type: v1.RuleGroupTypeMetrics,
			},
			ExpectedHTTPStatusCode: http.StatusBadRequest,
		},
		{
			Name:          "user john cannot update rule group that belongs to bob's cluster",
			RuleGroupName: "test-rule-group",
			ProjectID:     test.GenDefaultProject().Name,
			ClusterID:     test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", false),
				test.GenRuleGroup("test-rule-group", test.GenDefaultCluster().Name, "UpdateThisType"),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			RuleGroup:              test.GenAPIRuleGroup("test-rule-group", v1.RuleGroupTypeMetrics),
			ExpectedHTTPStatusCode: http.StatusForbidden,
			ExpectedResponse:       test.GenAPIRuleGroup("test-rule-group", v1.RuleGroupTypeMetrics),
		},
		{
			Name:          "admin user john can update rule group that belongs to bob's cluster",
			RuleGroupName: "test-rule-group",
			ProjectID:     test.GenDefaultProject().Name,
			ClusterID:     test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
				test.GenRuleGroup("test-rule-group", test.GenDefaultCluster().Name, "UpdateThisType"),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			RuleGroup:              test.GenAPIRuleGroup("test-rule-group", v1.RuleGroupTypeMetrics),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse:       test.GenAPIRuleGroup("test-rule-group", v1.RuleGroupTypeMetrics),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			requestURL := fmt.Sprintf("/api/v2/projects/%s/clusters/%s/rulegroups/%s", tc.ProjectID, tc.ClusterID, tc.RuleGroupName)
			body, err := json.Marshal(tc.RuleGroup)
			if err != nil {
				t.Fatalf("failed to marshalling rule group: %v", err)
			}
			req := httptest.NewRequest(http.MethodPut, requestURL, bytes.NewBuffer(body))
			resp := httptest.NewRecorder()

			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, nil, tc.ExistingKubermaticObjects, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to: %v", err)
			}
			ep.ServeHTTP(resp, req)

			if resp.Code != tc.ExpectedHTTPStatusCode {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatusCode, resp.Code, resp.Body.String())
			}
			if resp.Code == http.StatusCreated {
				b, err := json.Marshal(tc.ExpectedResponse)
				if err != nil {
					t.Fatalf("failed to marshal expected response %v", err)
				}
				test.CompareWithResult(t, resp, string(b))
			}
		})
	}
}

func TestDeleteEndpoint(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name                      string
		RuleGroupName             string
		ProjectID                 string
		ClusterID                 string
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
		ExpectedHTTPStatusCode    int
	}{
		{
			Name:          "delete rule group that belongs to the given cluster",
			RuleGroupName: "test-rule-group",
			ProjectID:     test.GenDefaultProject().Name,
			ClusterID:     test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenRuleGroup("test-rule-group", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusOK,
		},
		{
			Name:          "delete rule group which doesn't exist",
			RuleGroupName: "test-rule-group",
			ProjectID:     test.GenDefaultProject().Name,
			ClusterID:     test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusNotFound,
		},
		{
			Name:          "user john cannot delete rule group that belongs to bob's cluster",
			RuleGroupName: "test-rule-group",
			ProjectID:     test.GenDefaultProject().Name,
			ClusterID:     test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", false),
				test.GenRuleGroup("test-rule-group", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatusCode: http.StatusForbidden,
		},
		{
			Name:          "admin user john can delete rule group that belongs to bob's cluster",
			RuleGroupName: "test-rule-group",
			ProjectID:     test.GenDefaultProject().Name,
			ClusterID:     test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
				test.GenRuleGroup("test-rule-group", test.GenDefaultCluster().Name, v1.RuleGroupTypeMetrics),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatusCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			requestURL := fmt.Sprintf("/api/v2/projects/%s/clusters/%s/rulegroups/%s", tc.ProjectID, tc.ClusterID, tc.RuleGroupName)
			req := httptest.NewRequest(http.MethodDelete, requestURL, nil)
			resp := httptest.NewRecorder()

			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, nil, tc.ExistingKubermaticObjects, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to: %v", err)
			}
			ep.ServeHTTP(resp, req)

			if resp.Code != tc.ExpectedHTTPStatusCode {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatusCode, resp.Code, resp.Body.String())
			}
		})
	}
}
