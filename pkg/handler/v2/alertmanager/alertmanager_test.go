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

package alertmanager_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testAlertmanagerConfigSecretName = "test-alertmanager"
	testAlertmanagerConfig           = `
alertmanager_config: |
  global:
    smtp_smarthost: 'localhost:25'
    smtp_from: 'test@example.org'
  route:
    receiver: "test"
  receivers:
    - name: "test"
      email_configs:
      - to: 'test@example.org'
`
)

func TestGetEndpoint(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name                      string
		ProjectID                 string
		ClusterID                 string
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAlertmanager      *kubermaticv1.Alertmanager
		ExistingConfigSecret      *corev1.Secret
		ExistingAPIUser           *apiv1.User
		ExpectedResponse          apiv2.Alertmanager
		ExpectedHTTPStatus        int
	}{
		{
			Name:      "scenario 1: get alertmanager that belongs to the given cluster",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAlertmanager: test.GenAlertmanager(test.GenDefaultCluster().Status.NamespaceName,
				testAlertmanagerConfigSecretName),
			ExistingConfigSecret: test.GenAlertmanagerConfigSecret(testAlertmanagerConfigSecretName,
				test.GenDefaultCluster().Status.NamespaceName,
				[]byte(testAlertmanagerConfig)),
			ExistingAPIUser:    test.GenDefaultAPIUser(),
			ExpectedHTTPStatus: http.StatusOK,
			ExpectedResponse: apiv2.Alertmanager{
				Spec: apiv2.AlertmanagerSpec{
					Config: []byte(testAlertmanagerConfig),
				},
			},
		},
		{
			Name:      "scenario 2: try to get alertmanager but alertmanager is not found",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser:    test.GenDefaultAPIUser(),
			ExpectedHTTPStatus: http.StatusNotFound,
		},
		{
			Name:      "scenario 3: try to get alertmanager but alertmanager config secret is not found",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAlertmanager: test.GenAlertmanager(test.GenDefaultCluster().Status.NamespaceName,
				testAlertmanagerConfigSecretName),
			ExistingAPIUser:    test.GenDefaultAPIUser(),
			ExpectedHTTPStatus: http.StatusNotFound,
		},
		{
			Name:      "scenario 4: user john can not get alertmanager that belongs to bob's cluster",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", false),
			),
			ExistingAlertmanager: test.GenAlertmanager(test.GenDefaultCluster().Status.NamespaceName,
				testAlertmanagerConfigSecretName),
			ExistingConfigSecret: test.GenAlertmanagerConfigSecret(testAlertmanagerConfigSecretName,
				test.GenDefaultCluster().Status.NamespaceName,
				[]byte(testAlertmanagerConfig)),
			ExistingAPIUser:    test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatus: http.StatusForbidden,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, requestURL(tc.ProjectID, tc.ClusterID), nil)
			resp := httptest.NewRecorder()
			var kubermaticObjs []ctrlruntimeclient.Object
			kubermaticObjs = append(kubermaticObjs, tc.ExistingKubermaticObjects...)
			if tc.ExistingAlertmanager != nil {
				kubermaticObjs = append(kubermaticObjs, tc.ExistingAlertmanager)
			}
			var kubernetesObjs []ctrlruntimeclient.Object
			if tc.ExistingConfigSecret != nil {
				kubernetesObjs = append(kubernetesObjs, tc.ExistingConfigSecret)
			}

			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, kubernetesObjs, kubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}
			ep.ServeHTTP(resp, req)

			if resp.Code != tc.ExpectedHTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatus, resp.Code, resp.Body.String())
			}
			if resp.Code == http.StatusOK {
				bytes, err := json.Marshal(tc.ExpectedResponse)
				if err != nil {
					t.Fatalf("failed to marshall expected response %v", err)
				}

				test.CompareWithResult(t, resp, string(bytes))
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
		Body                      string
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAlertmanager      *kubermaticv1.Alertmanager
		ExistingConfigSecret      *corev1.Secret
		ExistingAPIUser           *apiv1.User
		ExpectedResponse          apiv2.Alertmanager
		ExpectedHTTPStatus        int
	}{
		{
			Name:      "scenario 1: create alertmanager",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			Body: fmt.Sprintf(`{"spec":{"config":"%s"}}`, base64.StdEncoding.EncodeToString([]byte(testAlertmanagerConfig))),
			ExistingAlertmanager: test.GenAlertmanager(test.GenDefaultCluster().Status.NamespaceName,
				testAlertmanagerConfigSecretName),
			ExistingConfigSecret: test.GenAlertmanagerConfigSecret(testAlertmanagerConfigSecretName,
				test.GenDefaultCluster().Status.NamespaceName,
				[]byte("test")),
			ExistingAPIUser:    test.GenDefaultAPIUser(),
			ExpectedHTTPStatus: http.StatusCreated,
			ExpectedResponse: apiv2.Alertmanager{
				Spec: apiv2.AlertmanagerSpec{
					Config: []byte(testAlertmanagerConfig),
				},
			},
		},
		{
			Name:      "scenario 2: create alertmanager with invalid request",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			Body: fmt.Sprintf(`{"spec":{"config":"%s"}}`, base64.StdEncoding.EncodeToString([]byte("bad-request"))),
			ExistingAlertmanager: test.GenAlertmanager(test.GenDefaultCluster().Status.NamespaceName,
				testAlertmanagerConfigSecretName),
			ExistingConfigSecret: test.GenAlertmanagerConfigSecret(testAlertmanagerConfigSecretName,
				test.GenDefaultCluster().Status.NamespaceName,
				[]byte("test")),
			ExistingAPIUser:    test.GenDefaultAPIUser(),
			ExpectedHTTPStatus: http.StatusBadRequest,
		},
		{
			Name:      "scenario 3: alertmanager is not found",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			Body:               fmt.Sprintf(`{"spec":{"config":"%s"}}`, base64.StdEncoding.EncodeToString([]byte(testAlertmanagerConfig))),
			ExistingAPIUser:    test.GenDefaultAPIUser(),
			ExpectedHTTPStatus: http.StatusInternalServerError,
		},
		{
			Name:      "scenario 4: alertmanager config secret is not found",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAlertmanager: test.GenAlertmanager(test.GenDefaultCluster().Status.NamespaceName,
				testAlertmanagerConfigSecretName),
			Body:               fmt.Sprintf(`{"spec":{"config":"%s"}}`, base64.StdEncoding.EncodeToString([]byte(testAlertmanagerConfig))),
			ExistingAPIUser:    test.GenDefaultAPIUser(),
			ExpectedHTTPStatus: http.StatusInternalServerError,
		},
		{
			Name:      "scenario 5: user john can not get alertmanager that belongs to bob's cluster",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", false),
			),
			ExistingAlertmanager: test.GenAlertmanager(test.GenDefaultCluster().Status.NamespaceName,
				testAlertmanagerConfigSecretName),
			ExistingConfigSecret: test.GenAlertmanagerConfigSecret(testAlertmanagerConfigSecretName,
				test.GenDefaultCluster().Status.NamespaceName,
				[]byte("test")),
			Body:               fmt.Sprintf(`{"spec":{"config":"%s"}}`, base64.StdEncoding.EncodeToString([]byte(testAlertmanagerConfig))),
			ExistingAPIUser:    test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatus: http.StatusForbidden,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, requestURL(tc.ProjectID, tc.ClusterID), strings.NewReader(tc.Body))
			resp := httptest.NewRecorder()
			var kubermaticObjs []ctrlruntimeclient.Object
			kubermaticObjs = append(kubermaticObjs, tc.ExistingKubermaticObjects...)
			if tc.ExistingAlertmanager != nil {
				kubermaticObjs = append(kubermaticObjs, tc.ExistingAlertmanager)
			}
			var kubernetesObjs []ctrlruntimeclient.Object
			if tc.ExistingConfigSecret != nil {
				kubernetesObjs = append(kubernetesObjs, tc.ExistingConfigSecret)
			}

			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, kubernetesObjs, kubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}
			ep.ServeHTTP(resp, req)

			if resp.Code != tc.ExpectedHTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatus, resp.Code, resp.Body.String())
			}
			if resp.Code == http.StatusCreated {
				bytes, err := json.Marshal(tc.ExpectedResponse)
				if err != nil {
					t.Fatalf("failed to marshall expected response %v", err)
				}

				test.CompareWithResult(t, resp, string(bytes))
			}
		})
	}
}

func TestDeleteEndpoint(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name                      string
		ProjectID                 string
		ClusterID                 string
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAlertmanager      *kubermaticv1.Alertmanager
		ExistingConfigSecret      *corev1.Secret
		ExistingAPIUser           *apiv1.User
		ExpectedHTTPStatus        int
	}{
		{
			Name:      "scenario 1: delete alertmanager that belongs to the given cluster",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAlertmanager: test.GenAlertmanager(test.GenDefaultCluster().Status.NamespaceName,
				testAlertmanagerConfigSecretName),
			ExistingConfigSecret: test.GenAlertmanagerConfigSecret(testAlertmanagerConfigSecretName,
				test.GenDefaultCluster().Status.NamespaceName,
				[]byte(testAlertmanagerConfig)),
			ExistingAPIUser:    test.GenDefaultAPIUser(),
			ExpectedHTTPStatus: http.StatusOK,
		},
		{
			Name:      "scenario 2: try to delete alertmanager but alertmanager is not found",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser:    test.GenDefaultAPIUser(),
			ExpectedHTTPStatus: http.StatusInternalServerError,
		},
		{
			Name:      "scenario 3: try to delete alertmanager but alertmanager config secret is not found",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAlertmanager: test.GenAlertmanager(test.GenDefaultCluster().Status.NamespaceName,
				testAlertmanagerConfigSecretName),
			ExistingAPIUser:    test.GenDefaultAPIUser(),
			ExpectedHTTPStatus: http.StatusNotFound,
		},
		{
			Name:      "scenario 4: user john can not delete alertmanager that belongs to bob's cluster",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", false),
			),
			ExistingAlertmanager: test.GenAlertmanager(test.GenDefaultCluster().Status.NamespaceName,
				testAlertmanagerConfigSecretName),
			ExistingConfigSecret: test.GenAlertmanagerConfigSecret(testAlertmanagerConfigSecretName,
				test.GenDefaultCluster().Status.NamespaceName,
				[]byte(testAlertmanagerConfig)),
			ExistingAPIUser:    test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatus: http.StatusForbidden,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, requestURL(tc.ProjectID, tc.ClusterID), nil)
			resp := httptest.NewRecorder()
			var kubermaticObjs []ctrlruntimeclient.Object
			kubermaticObjs = append(kubermaticObjs, tc.ExistingKubermaticObjects...)
			if tc.ExistingAlertmanager != nil {
				kubermaticObjs = append(kubermaticObjs, tc.ExistingAlertmanager)
			}
			var kubernetesObjs []ctrlruntimeclient.Object
			if tc.ExistingConfigSecret != nil {
				kubernetesObjs = append(kubernetesObjs, tc.ExistingConfigSecret)
			}

			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, kubernetesObjs, kubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}
			ep.ServeHTTP(resp, req)

			if resp.Code != tc.ExpectedHTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatus, resp.Code, resp.Body.String())
			}
		})
	}
}

func requestURL(projectID, clusterID string) string {
	return fmt.Sprintf("/api/v2/projects/%s/clusters/%s/alertmanager/config", projectID, clusterID)
}
