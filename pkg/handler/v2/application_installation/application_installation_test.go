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

package applicationinstallation_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	semverlib "github.com/Masterminds/semver/v3"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	app1TargetNamespace = "app-namespace-1"
	app2TargetNamespace = "app-namespace-2"
)

func TestListApplicationInstallations(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                      string
		ProjectID                 string
		ClusterID                 string
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
		ExpectedResponse          []apiv2.ApplicationInstallationListItem
		ExpectedHTTPStatusCode    int
	}{
		{
			Name:      "scenario 1: list all applicationinstallations that belong to the given cluster in different namespaces",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenApplicationInstallation("app1", test.GenDefaultCluster().Name, app1TargetNamespace),
				test.GenApplicationInstallation("app2", test.GenDefaultCluster().Name, app2TargetNamespace),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse: []apiv2.ApplicationInstallationListItem{
				{
					Name: "app1",
					Spec: &apiv2.ApplicationInstallationListItemSpec{
						Namespace: apiv2.NamespaceSpec{
							Name:   app1TargetNamespace,
							Create: true,
						},
						ApplicationRef: appskubermaticv1.ApplicationRef{
							Name: "sample-app",
							Version: appskubermaticv1.Version{
								Version: *semverlib.MustParse("v1.0.0"),
							},
						},
					},
					Status: &apiv2.ApplicationInstallationListItemStatus{},
				},
				{
					Name: "app2",
					Spec: &apiv2.ApplicationInstallationListItemSpec{
						Namespace: apiv2.NamespaceSpec{
							Name:   app2TargetNamespace,
							Create: true,
						},
						ApplicationRef: appskubermaticv1.ApplicationRef{
							Name: "sample-app",
							Version: appskubermaticv1.Version{
								Version: *semverlib.MustParse("v1.0.0"),
							},
						},
					},
					Status: &apiv2.ApplicationInstallationListItemStatus{},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			requestURL := fmt.Sprintf("/api/v2/projects/%s/clusters/%s/applicationinstallations", tc.ProjectID, tc.ClusterID)
			req := httptest.NewRequest(http.MethodGet, requestURL, strings.NewReader(""))
			res := httptest.NewRecorder()

			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, nil, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to: %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.ExpectedHTTPStatusCode {
				t.Errorf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatusCode, res.Code, res.Body.String())
				return
			}

			actualApplicationInstallations := test.NewApplicationInstallationWrapper{}
			actualApplicationInstallations.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedApplicationInstallations := test.NewApplicationInstallationWrapper(tc.ExpectedResponse)
			wrappedExpectedApplicationInstallations.Sort()

			actualApplicationInstallations.EqualOrDie(wrappedExpectedApplicationInstallations, t)
		})
	}
}

func TestCreateApplicationInstallation(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                      string
		ProjectID                 string
		ClusterID                 string
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
		ApplicationInstallation   *apiv2.ApplicationInstallation
		ExpectedResponse          *apiv2.ApplicationInstallation
		ExpectedHTTPStatusCode    int
		ExpectedNamespaces        []string
	}{
		{
			Name:      "create ApplicationInstallation that matches spec",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser:         test.GenDefaultAPIUser(),
			ApplicationInstallation: test.GenApiApplicationInstallation("app1", test.GenDefaultCluster().Name, app1TargetNamespace),
			ExpectedHTTPStatusCode:  http.StatusCreated,
			ExpectedNamespaces:      []string{app1TargetNamespace},
			ExpectedResponse: &apiv2.ApplicationInstallation{
				ObjectMeta: apiv1.ObjectMeta{
					Name: "app1",
				},
				Namespace: app1TargetNamespace,
				Spec: &apiv2.ApplicationInstallationSpec{
					Namespace: apiv2.NamespaceSpec{
						Name:   app1TargetNamespace,
						Create: true,
					},
					ApplicationRef: appskubermaticv1.ApplicationRef{
						Name: "sample-app",
						Version: appskubermaticv1.Version{
							Version: *semverlib.MustParse("v1.0.0"),
						},
					},
				},
				Status: &apiv2.ApplicationInstallationStatus{},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			requestURL := fmt.Sprintf("/api/v2/projects/%s/clusters/%s/applicationinstallations", tc.ProjectID, tc.ClusterID)
			body, err := json.Marshal(tc.ApplicationInstallation)
			if err != nil {
				t.Fatalf("failed to marshal ApplicationInstallation: %v", err)
			}
			req := httptest.NewRequest(http.MethodPost, requestURL, bytes.NewBuffer(body))
			res := httptest.NewRecorder()

			ep, cl, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, nil, nil, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to: %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.ExpectedHTTPStatusCode {
				t.Errorf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatusCode, res.Code, res.Body.String())
				return
			}

			if res.Code == http.StatusCreated {
				b, err := json.Marshal(tc.ExpectedResponse)
				if err != nil {
					t.Fatalf("failed to marshal expected response: %v", err)
				}
				test.CompareWithResult(t, res, string(b))

				for _, nsname := range tc.ExpectedNamespaces {
					if err := cl.FakeClient.Get(context.Background(), types.NamespacedName{Name: nsname}, &corev1.Namespace{}); err != nil {
						t.Errorf("Could not get expected namespace %q: %v", nsname, err)
					}
				}
			}
		})
	}
}

func TestDeleteApplication(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                                 string
		ProjectID                            string
		ClusterID                            string
		ApplicationInstallationName          string
		ApplicationInstallationNS            string
		ExistingKubermaticObjects            []ctrlruntimeclient.Object
		ExistingAPIUser                      *apiv1.User
		ExpectedApplicationinstallationCount int
		ExpectedHTTPStatusCode               int
	}{
		{
			Name:                        "delete an ApplicationInstallation that belongs to the given cluster",
			ProjectID:                   test.GenDefaultProject().Name,
			ClusterID:                   test.GenDefaultCluster().Name,
			ApplicationInstallationName: "app1",
			ApplicationInstallationNS:   app1TargetNamespace,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenApplicationInstallation("app1", test.GenDefaultCluster().Name, app1TargetNamespace),
				test.GenApplicationInstallation("app2", test.GenDefaultCluster().Name, app2TargetNamespace),
			),
			ExistingAPIUser:                      test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode:               http.StatusOK,
			ExpectedApplicationinstallationCount: 1,
		},
		{
			Name:                        "try to delete an ApplicationInstallation that does not exist",
			ProjectID:                   test.GenDefaultProject().Name,
			ClusterID:                   test.GenDefaultCluster().Name,
			ApplicationInstallationName: "does-not-exist",
			ApplicationInstallationNS:   app1TargetNamespace,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenApplicationInstallation("app1", test.GenDefaultCluster().Name, app1TargetNamespace),
			),
			ExistingAPIUser:                      test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode:               http.StatusNotFound,
			ExpectedApplicationinstallationCount: 1,
		},
		{
			Name:                        "John cannot delete Bob's ApplicationInstallation",
			ProjectID:                   test.GenDefaultProject().Name,
			ClusterID:                   test.GenDefaultCluster().Name,
			ApplicationInstallationName: "app1",
			ApplicationInstallationNS:   app1TargetNamespace,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenApplicationInstallation("app1", test.GenDefaultCluster().Name, app1TargetNamespace),
			),
			ExistingAPIUser:                      test.GenAPIUser("John", "john@acme.com"),
			ExpectedHTTPStatusCode:               http.StatusForbidden,
			ExpectedApplicationinstallationCount: 1,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			requestURL := fmt.Sprintf("/api/v2/projects/%s/clusters/%s/applicationinstallations/%s/%s", tc.ProjectID, tc.ClusterID, tc.ApplicationInstallationNS, tc.ApplicationInstallationName)
			req := httptest.NewRequest(http.MethodDelete, requestURL, nil)
			res := httptest.NewRecorder()

			ep, clients, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, nil, nil, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to: %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.ExpectedHTTPStatusCode {
				t.Errorf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatusCode, res.Code, res.Body.String())
				return
			}

			appInstalls := &appskubermaticv1.ApplicationInstallationList{}
			if err := clients.FakeClient.List(context.Background(), appInstalls); err != nil {
				t.Fatalf("failed to list MachineDeployments: %v", err)
			}

			if appInstalls := len(appInstalls.Items); tc.ExpectedApplicationinstallationCount != appInstalls {
				t.Errorf("Expected %d  ApplicationInstallations but got %d", tc.ExpectedApplicationinstallationCount, appInstalls)
			}
		})
	}
}

func TestGetApplication(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                        string
		ProjectID                   string
		ClusterID                   string
		ApplicationInstallationName string
		ApplicationInstallationNS   string
		ExistingKubermaticObjects   []ctrlruntimeclient.Object
		ExistingAPIUser             *apiv1.User
		ExpectedResponse            *apiv2.ApplicationInstallation
		ExpectedHTTPStatusCode      int
	}{
		{
			Name:                        "get ApplicationInstallation that belongs to the given cluster",
			ProjectID:                   test.GenDefaultProject().Name,
			ClusterID:                   test.GenDefaultCluster().Name,
			ApplicationInstallationName: "app1",
			ApplicationInstallationNS:   app1TargetNamespace,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenApplicationInstallation("app1", test.GenDefaultCluster().Name, app1TargetNamespace),
			),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode: http.StatusOK,
			ExpectedResponse: &apiv2.ApplicationInstallation{
				ObjectMeta: apiv1.ObjectMeta{
					Name: "app1",
				},
				Namespace: app1TargetNamespace,
				Spec: &apiv2.ApplicationInstallationSpec{
					Namespace: apiv2.NamespaceSpec{
						Name:   app1TargetNamespace,
						Create: true,
					},
					ApplicationRef: appskubermaticv1.ApplicationRef{
						Name: "sample-app",
						Version: appskubermaticv1.Version{
							Version: *semverlib.MustParse("v1.0.0"),
						},
					},
				},
				Status: &apiv2.ApplicationInstallationStatus{},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			requestURL := fmt.Sprintf("/api/v2/projects/%s/clusters/%s/applicationinstallations/%s/%s", tc.ProjectID, tc.ClusterID, tc.ApplicationInstallationNS, tc.ApplicationInstallationName)
			req := httptest.NewRequest(http.MethodGet, requestURL, nil)
			res := httptest.NewRecorder()

			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, nil, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to: %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.ExpectedHTTPStatusCode {
				t.Errorf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatusCode, res.Code, res.Body.String())
				return
			}

			if res.Code == http.StatusOK {
				b, err := json.Marshal(tc.ExpectedResponse)
				if err != nil {
					t.Fatalf("failed to marshal expected response: %v", err)
				}
				test.CompareWithResult(t, res, string(b))
			}
		})
	}
}

func TestUpdateApplicationInstallation(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                        string
		ProjectID                   string
		ClusterID                   string
		ApplicationInstallationName string
		ApplicationInstallationNS   string
		ExistingKubermaticObjects   []ctrlruntimeclient.Object
		ExistingAPIUser             *apiv1.User
		ApplicationInstallation     *apiv2.ApplicationInstallation
		ExpectedResponse            *apiv2.ApplicationInstallation
		ExpectedHTTPStatusCode      int
	}{
		{
			Name:                        "update an existing ApplicationInstallation",
			ProjectID:                   test.GenDefaultProject().Name,
			ClusterID:                   test.GenDefaultCluster().Name,
			ApplicationInstallationName: "app1",
			ApplicationInstallationNS:   app1TargetNamespace,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenApplicationInstallation("app1", test.GenDefaultCluster().Name, app1TargetNamespace),
			),
			ExistingAPIUser:         test.GenDefaultAPIUser(),
			ApplicationInstallation: test.GenApiApplicationInstallation("app1", test.GenDefaultCluster().Name, app1TargetNamespace),
			ExpectedHTTPStatusCode:  http.StatusOK,
			ExpectedResponse: &apiv2.ApplicationInstallation{
				ObjectMeta: apiv1.ObjectMeta{
					Name: "app1",
				},
				Namespace: app1TargetNamespace,
				Spec: &apiv2.ApplicationInstallationSpec{
					Namespace: apiv2.NamespaceSpec{
						Name:   app1TargetNamespace,
						Create: true,
					},
					ApplicationRef: appskubermaticv1.ApplicationRef{
						Name: "sample-app",
						Version: appskubermaticv1.Version{
							Version: *semverlib.MustParse("v1.0.0"),
						},
					},
					Values: *test.CreateRawVariables(t, map[string]interface{}{"key": "val"}),
				},
				Status: &apiv2.ApplicationInstallationStatus{},
			},
		},
		{
			Name:                        "try to update an ApplicationInstallation that does not exist",
			ProjectID:                   test.GenDefaultProject().Name,
			ClusterID:                   test.GenDefaultCluster().Name,
			ApplicationInstallationName: "app1",
			ApplicationInstallationNS:   app1TargetNamespace,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser:         test.GenDefaultAPIUser(),
			ApplicationInstallation: test.GenApiApplicationInstallation("app1", test.GenDefaultCluster().Name, app1TargetNamespace),
			ExpectedHTTPStatusCode:  http.StatusNotFound,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			requestURL := fmt.Sprintf("/api/v2/projects/%s/clusters/%s/applicationinstallations/%s/%s", tc.ProjectID, tc.ClusterID, tc.ApplicationInstallationNS, tc.ApplicationInstallationName)
			tc.ApplicationInstallation.Spec.Values = *test.CreateRawVariables(t, map[string]interface{}{"key": "val"})
			body, err := json.Marshal(tc.ApplicationInstallation)
			if err != nil {
				t.Fatalf("failed to marshal ApplicationInstallation: %v", err)
			}
			req := httptest.NewRequest(http.MethodPut, requestURL, bytes.NewBuffer(body))
			res := httptest.NewRecorder()

			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, nil, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to: %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.ExpectedHTTPStatusCode {
				t.Errorf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatusCode, res.Code, res.Body.String())
				return
			}

			if res.Code == http.StatusOK {
				b, err := json.Marshal(tc.ExpectedResponse)
				if err != nil {
					t.Fatalf("failed to marshal expected response: %v", err)
				}
				test.CompareWithResult(t, res, string(b))
			}
		})
	}
}
