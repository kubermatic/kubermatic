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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	semverlib "github.com/Masterminds/semver/v3"
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
		ExpectedResponse          []*apiv2.ApplicationInstallation
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
			ExpectedResponse: []*apiv2.ApplicationInstallation{
				{
					ObjectMeta: apiv1.ObjectMeta{
						Name: "app1",
					},
					Namespace: app1TargetNamespace,
					Spec: &appskubermaticv1.ApplicationInstallationSpec{
						Namespace: appskubermaticv1.NamespaceSpec{
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
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						Name: "app2",
					},
					Namespace: app2TargetNamespace,
					Spec: &appskubermaticv1.ApplicationInstallationSpec{
						Namespace: appskubermaticv1.NamespaceSpec{
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
