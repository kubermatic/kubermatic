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

package gatekeeperconfig_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	configv1alpha1 "github.com/open-policy-agent/gatekeeper/apis/config/v1alpha1"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	"k8c.io/kubermatic/v2/pkg/handler/v2/gatekeeperconfig"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetConfigEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		ExpectedResponse       string
		ProjectID              string
		ClusterID              string
		HTTPStatus             int
		ExistingKubermaticObjs []runtime.Object
		ExistingGatekeeperObjs []runtime.Object
		ExistingAPIUser        *apiv1.User
	}{
		{
			Name:                   "scenario 1: get gaetkeeper config",
			ExpectedResponse:       `{"spec":{"sync":{"syncOnly":[{"version":"v1","kind":"Namespace"},{"version":"v1","kind":"Pod"}]},"validation":{"traces":[{"user":"bob","kind":{"version":"v1","kind":"Pod"}}]},"match":[{"excludedNamespaces":["default","kube-system"],"processes":["audit"]}],"readiness":{"statsEnabled":true}}}`,
			ProjectID:              test.GenDefaultProject().Name,
			ClusterID:              test.GenDefaultCluster().Name,
			HTTPStatus:             http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingGatekeeperObjs: []runtime.Object{genGatekeeperConfig()},
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		{
			Name:                   "scenario 2: fail getting non-existing gaetkeeper config",
			ExpectedResponse:       `{"error":{"code":404,"message":"configs.config.gatekeeper.sh \"config\" not found"}}`,
			ProjectID:              test.GenDefaultProject().Name,
			ClusterID:              test.GenDefaultCluster().Name,
			HTTPStatus:             http.StatusNotFound,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingGatekeeperObjs: []runtime.Object{},
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		{
			Name:             "scenario 3: user john can not get bobs gatekeeper config",
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			HTTPStatus:       http.StatusForbidden,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", false),
			),
			ExistingGatekeeperObjs: []runtime.Object{genGatekeeperConfig()},
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
		},
		{
			Name:             "scenario 4: admin john can get bobs gatekeeper config",
			ExpectedResponse: `{"spec":{"sync":{"syncOnly":[{"version":"v1","kind":"Namespace"},{"version":"v1","kind":"Pod"}]},"validation":{"traces":[{"user":"bob","kind":{"version":"v1","kind":"Pod"}}]},"match":[{"excludedNamespaces":["default","kube-system"],"processes":["audit"]}],"readiness":{"statsEnabled":true}}}`,
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			HTTPStatus:       http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
			),
			ExistingGatekeeperObjs: []runtime.Object{genGatekeeperConfig()},
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {

			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/clusters/%s/gatekeeper/config", tc.ProjectID, tc.ClusterID), strings.NewReader(""))
			res := httptest.NewRecorder()

			ep, clientsSets, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, nil, nil, tc.ExistingKubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			for _, gkObject := range tc.ExistingGatekeeperObjs {
				err = clientsSets.FakeClient.Create(context.Background(), gkObject)
				if err != nil {
					t.Fatalf("failed to create gk object %v due to %v", gkObject, err)
				}
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}
			test.CompareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func genGatekeeperConfig() *configv1alpha1.Config {
	config := &configv1alpha1.Config{}
	config.Name = gatekeeperconfig.ConfigName
	config.Namespace = gatekeeperconfig.ConfigNamespace

	config.Spec = configv1alpha1.ConfigSpec{
		Sync: configv1alpha1.Sync{
			SyncOnly: []configv1alpha1.SyncOnlyEntry{
				{
					Group:   "",
					Version: "v1",
					Kind:    "Namespace",
				},
				{
					Group:   "",
					Version: "v1",
					Kind:    "Pod",
				},
			},
		},
		Validation: configv1alpha1.Validation{
			Traces: []configv1alpha1.Trace{
				{
					User: "bob",
					Kind: configv1alpha1.GVK{
						Group:   "",
						Version: "v1",
						Kind:    "Pod",
					},
				},
			},
		},
		Match: []configv1alpha1.MatchEntry{
			{
				ExcludedNamespaces: []string{"default", "kube-system"},
				Processes:          []string{"audit"},
			},
		},
		Readiness: configv1alpha1.ReadinessSpec{
			StatsEnabled: true,
		},
	}

	return config
}
