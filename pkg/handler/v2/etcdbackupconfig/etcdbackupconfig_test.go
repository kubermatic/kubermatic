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

package etcdbackupconfig_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	modifiedSchedule = "2 10 * * *"
)

func TestCreateEndpoint(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name                      string
		ProjectID                 string
		ClusterID                 string
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
		EtcdBackupConfig          *apiv2.EtcdBackupConfig
		ExpectedHTTPStatusCode    int
		ExpectedResponse          *apiv2.EtcdBackupConfig
	}{
		{
			Name:      "create etcd backup config in the given cluster",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			EtcdBackupConfig:       test.GenAPIEtcdBackupConfig("test-ebc", test.GenDefaultCluster().Name),
			ExpectedHTTPStatusCode: http.StatusCreated,
			ExpectedResponse:       test.GenAPIEtcdBackupConfig("test-ebc", test.GenDefaultCluster().Name),
		},
		{
			Name:      "user john cannot create etcd backup config that belongs to bob's cluster",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", false),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			EtcdBackupConfig:       test.GenAPIEtcdBackupConfig("test-ebc", test.GenDefaultCluster().Name),
			ExpectedHTTPStatusCode: http.StatusForbidden,
			ExpectedResponse:       nil,
		},
		{
			Name:      "admin user john can create etcd backup config that belongs to bob's cluster",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			EtcdBackupConfig:       test.GenAPIEtcdBackupConfig("test-ebc", test.GenDefaultCluster().Name),
			ExpectedHTTPStatusCode: http.StatusCreated,
			ExpectedResponse:       test.GenAPIEtcdBackupConfig("test-ebc", test.GenDefaultCluster().Name),
		},
		{
			Name:      "validation fails when schedule is set and keep is not",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
			EtcdBackupConfig: func() *apiv2.EtcdBackupConfig {
				ebc := test.GenAPIEtcdBackupConfig("test-ebc", test.GenDefaultCluster().Name)
				ebc.Spec.Keep = nil
				return ebc
			}(),
			ExpectedHTTPStatusCode: http.StatusBadRequest,
			ExpectedResponse:       nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			requestURL := fmt.Sprintf("/api/v2/projects/%s/clusters/%s/etcdbackupconfigs", tc.ProjectID, tc.ClusterID)
			body, err := json.Marshal(tc.EtcdBackupConfig)
			if err != nil {
				t.Fatalf("failed marshalling etcdbackupconfig: %v", err)
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

func TestGetEndpoint(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name                      string
		EtcdBackupConfigName      string
		ProjectID                 string
		ClusterID                 string
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
		ExpectedResponse          *apiv2.EtcdBackupConfig
		ExpectedHTTPStatusCode    int
	}{
		{
			Name:                 "get etcdbackupconfig that belongs to the given cluster",
			EtcdBackupConfigName: "test-1",
			ProjectID:            test.GenDefaultProject().Name,
			ClusterID:            test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenEtcdBackupConfig("test-1", test.GenDefaultCluster(), test.GenDefaultProject().Name),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse:       test.GenAPIEtcdBackupConfig("test-1", test.GenDefaultCluster().Name),
		},
		{
			Name:                 "get etcdbackupconfig which doesn't exist",
			EtcdBackupConfigName: "test-1",
			ProjectID:            test.GenDefaultProject().Name,
			ClusterID:            test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusNotFound,
		},
		{
			Name:                 "user john cannot get etcdbackupconfig that belongs to bob's cluster",
			EtcdBackupConfigName: "test-1",
			ProjectID:            test.GenDefaultProject().Name,
			ClusterID:            test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", false),
				test.GenEtcdBackupConfig("test-1", test.GenDefaultCluster(), test.GenDefaultProject().Name),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatusCode: http.StatusForbidden,
		},
		{
			Name:                 "admin user john can get etcdbackupconfig that belongs to bob's cluster",
			EtcdBackupConfigName: "test-1",
			ProjectID:            test.GenDefaultProject().Name,
			ClusterID:            test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
				test.GenEtcdBackupConfig("test-1", test.GenDefaultCluster(), test.GenDefaultProject().Name),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse:       test.GenAPIEtcdBackupConfig("test-1", test.GenDefaultCluster().Name),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			requestURL := fmt.Sprintf("/api/v2/projects/%s/clusters/%s/etcdbackupconfigs/%s", tc.ProjectID, tc.ClusterID, tc.EtcdBackupConfigName)
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
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
		ExpectedResponse          []*apiv2.EtcdBackupConfig
		ExpectedHTTPStatusCode    int
	}{
		{
			Name:      "list etcdbackupconfigs that belongs to the given cluster",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenEtcdBackupConfig("test-1", test.GenDefaultCluster(), test.GenDefaultProject().Name),
				test.GenEtcdBackupConfig("test-2", test.GenDefaultCluster(), test.GenDefaultProject().Name),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse: []*apiv2.EtcdBackupConfig{
				test.GenAPIEtcdBackupConfig("test-1", test.GenDefaultCluster().Name),
				test.GenAPIEtcdBackupConfig("test-2", test.GenDefaultCluster().Name),
			},
		},
		{
			Name:      "user john cannot list etcdbackupconfigs that belongs to bob's cluster",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", false),
				test.GenEtcdBackupConfig("test-1", test.GenDefaultCluster(), test.GenDefaultProject().Name),
				test.GenEtcdBackupConfig("test-2", test.GenDefaultCluster(), test.GenDefaultProject().Name),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatusCode: http.StatusForbidden,
		},
		{
			Name:      "admin user john can get etcdbackupconfig that belongs to bob's cluster",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
				test.GenEtcdBackupConfig("test-1", test.GenDefaultCluster(), test.GenDefaultProject().Name),
				test.GenEtcdBackupConfig("test-2", test.GenDefaultCluster(), test.GenDefaultProject().Name),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse: []*apiv2.EtcdBackupConfig{
				test.GenAPIEtcdBackupConfig("test-1", test.GenDefaultCluster().Name),
				test.GenAPIEtcdBackupConfig("test-2", test.GenDefaultCluster().Name),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			requestURL := fmt.Sprintf("/api/v2/projects/%s/clusters/%s/etcdbackupconfigs", tc.ProjectID, tc.ClusterID)
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
				etcdBackupConfigs := test.NewEtcdBackupConfigSliceWrapper{}
				etcdBackupConfigs.DecodeOrDie(resp.Body, t).Sort()

				expectedEtcdBackupConfigs := test.NewEtcdBackupConfigSliceWrapper(tc.ExpectedResponse)
				expectedEtcdBackupConfigs.Sort()

				etcdBackupConfigs.EqualOrDie(expectedEtcdBackupConfigs, t)
			}

		})
	}
}

func TestDeleteEndpoint(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name                      string
		EtcdBackupConfigName      string
		ProjectID                 string
		ClusterID                 string
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
		ExpectedHTTPStatusCode    int
	}{
		{
			Name:                 "delete etcdbackupconfig that belongs to the given cluster",
			EtcdBackupConfigName: "test-1",
			ProjectID:            test.GenDefaultProject().Name,
			ClusterID:            test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenEtcdBackupConfig("test-1", test.GenDefaultCluster(), test.GenDefaultProject().Name),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusOK,
		},
		{
			Name:                 "delete etcdbackupconfig which doesn't exist",
			EtcdBackupConfigName: "test-1",
			ProjectID:            test.GenDefaultProject().Name,
			ClusterID:            test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusNotFound,
		},
		{
			Name:                 "user john cannot delete etcdbackupconfig that belongs to bob's cluster",
			EtcdBackupConfigName: "test-1",
			ProjectID:            test.GenDefaultProject().Name,
			ClusterID:            test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", false),
				test.GenEtcdBackupConfig("test-1", test.GenDefaultCluster(), test.GenDefaultProject().Name),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatusCode: http.StatusForbidden,
		},
		{
			Name:                 "admin user john can delete etcdbackupconfig that belongs to bob's cluster",
			EtcdBackupConfigName: "test-1",
			ProjectID:            test.GenDefaultProject().Name,
			ClusterID:            test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
				test.GenEtcdBackupConfig("test-1", test.GenDefaultCluster(), test.GenDefaultProject().Name),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatusCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			requestURL := fmt.Sprintf("/api/v2/projects/%s/clusters/%s/etcdbackupconfigs/%s", tc.ProjectID, tc.ClusterID, tc.EtcdBackupConfigName)
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

func TestPatchEndpoint(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name                      string
		EtcdBackupConfigName      string
		PatchSpec                 *apiv2.EtcdBackupConfigSpec
		ProjectID                 string
		ClusterID                 string
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
		ExpectedResponse          *apiv2.EtcdBackupConfig
		ExpectedHTTPStatusCode    int
	}{
		{
			Name:                 "patch etcdbackupconfig",
			EtcdBackupConfigName: "test-1",
			PatchSpec: func() *apiv2.EtcdBackupConfigSpec {
				spec := test.GenAPIEtcdBackupConfig("test-1", test.GenDefaultCluster().Name).Spec
				spec.Schedule = modifiedSchedule
				return &spec
			}(),
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenEtcdBackupConfig("test-1", test.GenDefaultCluster(), test.GenDefaultProject().Name),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse: func() *apiv2.EtcdBackupConfig {
				ebc := test.GenAPIEtcdBackupConfig("test-1", test.GenDefaultCluster().Name)
				ebc.Spec.Schedule = modifiedSchedule
				return ebc
			}(),
		},
		{
			Name:                 "patch etcdbackupconfig which doesn't exist",
			EtcdBackupConfigName: "test-1",
			PatchSpec: func() *apiv2.EtcdBackupConfigSpec {
				spec := test.GenAPIEtcdBackupConfig("test-1", test.GenDefaultCluster().Name).Spec
				spec.Schedule = modifiedSchedule
				return &spec
			}(),
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusNotFound,
		},
		{
			Name:                 "user john cannot patch etcdbackupconfig that belongs to bob's cluster",
			EtcdBackupConfigName: "test-1",
			PatchSpec: func() *apiv2.EtcdBackupConfigSpec {
				spec := test.GenAPIEtcdBackupConfig("test-1", test.GenDefaultCluster().Name).Spec
				spec.Schedule = modifiedSchedule
				return &spec
			}(),
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", false),
				test.GenEtcdBackupConfig("test-1", test.GenDefaultCluster(), test.GenDefaultProject().Name),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatusCode: http.StatusForbidden,
		},
		{
			Name:                 "admin user john can patch etcdbackupconfig that belongs to bob's cluster",
			EtcdBackupConfigName: "test-1",
			PatchSpec: func() *apiv2.EtcdBackupConfigSpec {
				spec := test.GenAPIEtcdBackupConfig("test-1", test.GenDefaultCluster().Name).Spec
				spec.Schedule = modifiedSchedule
				return &spec
			}(),
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
				test.GenEtcdBackupConfig("test-1", test.GenDefaultCluster(), test.GenDefaultProject().Name),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse: func() *apiv2.EtcdBackupConfig {
				ebc := test.GenAPIEtcdBackupConfig("test-1", test.GenDefaultCluster().Name)
				ebc.Spec.Schedule = modifiedSchedule
				return ebc
			}(),
		},
		{
			Name:                 "patch etcdbackupconfig validation fail",
			EtcdBackupConfigName: "test-1",
			PatchSpec: func() *apiv2.EtcdBackupConfigSpec {
				spec := test.GenAPIEtcdBackupConfig("test-1", test.GenDefaultCluster().Name).Spec
				spec.Schedule = modifiedSchedule
				spec.Keep = nil
				return &spec
			}(),
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenEtcdBackupConfig("test-1", test.GenDefaultCluster(), test.GenDefaultProject().Name),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			requestURL := fmt.Sprintf("/api/v2/projects/%s/clusters/%s/etcdbackupconfigs/%s", tc.ProjectID, tc.ClusterID, tc.EtcdBackupConfigName)
			body, err := json.Marshal(tc.PatchSpec)
			if err != nil {
				t.Fatalf("failed marshalling etcdbackupconfig: %v", err)
			}
			req := httptest.NewRequest(http.MethodPatch, requestURL, bytes.NewBuffer(body))
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

func TestProjectListEndpoint(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name                      string
		ProjectID                 string
		Type                      string
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
		ExpectedResponse          []*apiv2.EtcdBackupConfig
		ExpectedHTTPStatusCode    int
	}{
		{
			Name:      "list etcdbackupconfigs that belongs to the given project",
			ProjectID: test.GenDefaultProject().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenEtcdBackupConfig("test-1", test.GenDefaultCluster(), test.GenDefaultProject().Name),
				test.GenEtcdBackupConfig("test-2", test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)), test.GenDefaultProject().Name),
				test.GenEtcdBackupConfig("test-3", test.GenDefaultCluster(), "some-different-project"),
				func() *kubermaticv1.EtcdBackupConfig {
					ebc := test.GenEtcdBackupConfig("test-4", test.GenDefaultCluster(), test.GenDefaultProject().Name)
					ebc.Spec.Schedule = ""
					return ebc
				}(),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse: []*apiv2.EtcdBackupConfig{
				test.GenAPIEtcdBackupConfig("test-1", test.GenDefaultCluster().Name),
				test.GenAPIEtcdBackupConfig("test-2", "clusterAbcID"),
				func() *apiv2.EtcdBackupConfig {
					ebc := test.GenAPIEtcdBackupConfig("test-4", test.GenDefaultCluster().Name)
					ebc.Spec.Schedule = ""
					return ebc
				}(),
			},
		},
		{
			Name:      "user john cannot list etcdbackupconfigs from a project he does not belong to",
			ProjectID: test.GenDefaultProject().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", false),
				test.GenEtcdBackupConfig("test-1", test.GenDefaultCluster(), test.GenDefaultProject().Name),
				test.GenEtcdBackupConfig("test-2", test.GenDefaultCluster(), test.GenDefaultProject().Name),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatusCode: http.StatusForbidden,
		},
		{
			Name:      "admin user john can get etcdbackupconfig that belongs to bob's project",
			ProjectID: test.GenDefaultProject().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
				test.GenEtcdBackupConfig("test-1", test.GenDefaultCluster(), test.GenDefaultProject().Name),
				test.GenEtcdBackupConfig("test-2", test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)), test.GenDefaultProject().Name),
				test.GenEtcdBackupConfig("test-3", test.GenDefaultCluster(), "some-different-project"),
				func() *kubermaticv1.EtcdBackupConfig {
					ebc := test.GenEtcdBackupConfig("test-4", test.GenDefaultCluster(), test.GenDefaultProject().Name)
					ebc.Spec.Schedule = ""
					return ebc
				}(),
			),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse: []*apiv2.EtcdBackupConfig{
				test.GenAPIEtcdBackupConfig("test-1", test.GenDefaultCluster().Name),
				test.GenAPIEtcdBackupConfig("test-2", "clusterAbcID"),
				func() *apiv2.EtcdBackupConfig {
					ebc := test.GenAPIEtcdBackupConfig("test-4", test.GenDefaultCluster().Name)
					ebc.Spec.Schedule = ""
					return ebc
				}(),
			},
		},
		{
			Name:      "filter by type snapshot",
			ProjectID: test.GenDefaultProject().Name,
			Type:      "snapshot",
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenEtcdBackupConfig("test-1", test.GenDefaultCluster(), test.GenDefaultProject().Name),
				test.GenEtcdBackupConfig("test-2", test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)), test.GenDefaultProject().Name),
				test.GenEtcdBackupConfig("test-3", test.GenDefaultCluster(), "some-different-project"),
				func() *kubermaticv1.EtcdBackupConfig {
					ebc := test.GenEtcdBackupConfig("test-4", test.GenDefaultCluster(), test.GenDefaultProject().Name)
					ebc.Spec.Schedule = ""
					return ebc
				}(),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse: []*apiv2.EtcdBackupConfig{
				func() *apiv2.EtcdBackupConfig {
					ebc := test.GenAPIEtcdBackupConfig("test-4", test.GenDefaultCluster().Name)
					ebc.Spec.Schedule = ""
					return ebc
				}(),
			},
		},
		{
			Name:      "filter by type automatic",
			ProjectID: test.GenDefaultProject().Name,
			Type:      "automatic",
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenEtcdBackupConfig("test-1", test.GenDefaultCluster(), test.GenDefaultProject().Name),
				test.GenEtcdBackupConfig("test-2", test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)), test.GenDefaultProject().Name),
				test.GenEtcdBackupConfig("test-3", test.GenDefaultCluster(), "some-different-project"),
				func() *kubermaticv1.EtcdBackupConfig {
					ebc := test.GenEtcdBackupConfig("test-4", test.GenDefaultCluster(), test.GenDefaultProject().Name)
					ebc.Spec.Schedule = ""
					return ebc
				}(),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse: []*apiv2.EtcdBackupConfig{
				test.GenAPIEtcdBackupConfig("test-1", test.GenDefaultCluster().Name),
				test.GenAPIEtcdBackupConfig("test-2", "clusterAbcID"),
			},
		},
		{
			Name:      "fail filtering by unknown type",
			Type:      "unknown",
			ProjectID: test.GenDefaultProject().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", false),
				test.GenEtcdBackupConfig("test-1", test.GenDefaultCluster(), test.GenDefaultProject().Name),
				test.GenEtcdBackupConfig("test-2", test.GenDefaultCluster(), test.GenDefaultProject().Name),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			requestURL := fmt.Sprintf("/api/v2/projects/%s/etcdbackupconfigs?type=%s", tc.ProjectID, tc.Type)
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
				etcdBackupConfigs := test.NewEtcdBackupConfigSliceWrapper{}
				etcdBackupConfigs.DecodeOrDie(resp.Body, t).Sort()

				expectedEtcdBackupConfigs := test.NewEtcdBackupConfigSliceWrapper(tc.ExpectedResponse)
				expectedEtcdBackupConfigs.Sort()

				etcdBackupConfigs.EqualOrDie(expectedEtcdBackupConfigs, t)
			}

		})
	}
}
