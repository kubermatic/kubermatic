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

package addon_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"

	"k8s.io/apimachinery/pkg/runtime"
	k8sjson "k8s.io/apimachinery/pkg/util/json"
)

func TestGetAddon(t *testing.T) {
	t.Parallel()
	creationTime := test.DefaultCreationTimestamp()
	testVariables := map[string]interface{}{"foo": "bar", "hello": "world"}

	testcases := []struct {
		Name                   string
		ProjectIDToSync        string
		ClusterIDToSync        string
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
		ExistingCluster        *kubermaticv1.Cluster
		ExistingAddons         []*kubermaticv1.Addon
		AddonToGet             string
		ExistingKubermaticObjs []runtime.Object
		ExpectedHTTPStatus     int
		ExpectedResponse       apiv1.Addon
	}{
		// scenario 1
		{
			Name:                   "scenario 1: get addon that belongs to the given cluster",
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingAddons: []*kubermaticv1.Addon{
				test.GenTestAddon("addon1", nil, test.GenDefaultCluster(), creationTime),
			},
			AddonToGet:         "addon1",
			ExpectedHTTPStatus: http.StatusOK,
			ExpectedResponse: apiv1.Addon{
				ObjectMeta: apiv1.ObjectMeta{
					ID:                "addon1",
					Name:              "addon1",
					CreationTimestamp: apiv1.NewTime(creationTime),
				},
			},
		},
		// scenario 2
		{
			Name:                   "scenario 2: get addon with variables that belongs to the given cluster",
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingAddons: []*kubermaticv1.Addon{
				test.GenTestAddon("addon1", createRawVariables(t, testVariables), test.GenDefaultCluster(), creationTime),
			},
			AddonToGet:         "addon1",
			ExpectedHTTPStatus: http.StatusOK,
			ExpectedResponse: apiv1.Addon{
				ObjectMeta: apiv1.ObjectMeta{
					ID:                "addon1",
					Name:              "addon1",
					CreationTimestamp: apiv1.NewTime(creationTime),
				},
				Spec: apiv1.AddonSpec{
					Variables: testVariables,
				},
			},
		},
		// scenario 3
		{
			Name:                   "scenario 3: try to get addon that belongs to the given cluster but isn't accessible",
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingAddons: []*kubermaticv1.Addon{
				test.GenTestAddon("addon1", nil, test.GenDefaultCluster(), creationTime),
				test.GenTestAddon("addon2", nil, test.GenDefaultCluster(), creationTime),
				test.GenTestAddon("addon3", nil, test.GenDefaultCluster(), creationTime),
			},
			AddonToGet:         "addon3",
			ExpectedHTTPStatus: http.StatusUnauthorized,
			ExpectedResponse:   apiv1.Addon{},
		},
		// scenario 4
		{
			Name:                   "scenario 4: the admin John can get addon with variables that belongs to the Bob's cluster",
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster(), genUser("John", "john@acme.com", true)),
			ExistingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
			ExistingAddons: []*kubermaticv1.Addon{
				test.GenTestAddon("addon1", createRawVariables(t, testVariables), test.GenDefaultCluster(), creationTime),
			},
			AddonToGet:         "addon1",
			ExpectedHTTPStatus: http.StatusOK,
			ExpectedResponse: apiv1.Addon{
				ObjectMeta: apiv1.ObjectMeta{
					ID:                "addon1",
					Name:              "addon1",
					CreationTimestamp: apiv1.NewTime(creationTime),
				},
				Spec: apiv1.AddonSpec{
					Variables: testVariables,
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/addons/%s", tc.ProjectIDToSync, tc.ClusterIDToSync, tc.AddonToGet), strings.NewReader(""))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			var machineObj []runtime.Object
			var kubernetesObj []runtime.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			for _, existingAddon := range tc.ExistingAddons {
				kubermaticObj = append(kubermaticObj, existingAddon)
			}
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubernetesObj, machineObj, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.ExpectedHTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatus, res.Code, res.Body.String())
			}

			if res.Code == http.StatusOK {
				bytes, err := json.Marshal(tc.ExpectedResponse)
				if err != nil {
					t.Fatalf("failed to marshall expected response %v", err)
				}

				test.CompareWithResult(t, res, string(bytes))
			}
		})
	}
}

func TestListAddons(t *testing.T) {
	t.Parallel()
	creationTime := test.DefaultCreationTimestamp()
	cluster := test.GenDefaultCluster()
	cluster.Status.NamespaceName = fmt.Sprintf("cluster-%s", cluster.Name)
	testVariables := map[string]interface{}{"foo": "bar", "hello": "world"}

	testcases := []struct {
		Name                   string
		ExpectedResponse       []apiv1.Addon
		ExpectedHTTPStatus     int
		ExistingKubermaticObjs []runtime.Object
		ExistingAPIUser        *apiv1.User
	}{
		// scenario 1
		{
			Name: "scenario 1: gets a list of addons added to cluster",
			ExpectedResponse: []apiv1.Addon{
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "addon1",
						Name:              "addon1",
						CreationTimestamp: apiv1.NewTime(creationTime),
					},
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "addon2",
						Name:              "addon2",
						CreationTimestamp: apiv1.NewTime(creationTime),
					},
					Spec: apiv1.AddonSpec{
						Variables: testVariables,
					},
				},
			},
			ExpectedHTTPStatus: http.StatusOK,
			ExistingKubermaticObjs: []runtime.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				/*add cluster*/
				cluster,
				test.GenTestAddon("addon1", nil, cluster, creationTime),
				test.GenTestAddon("addon2", createRawVariables(t, testVariables), cluster, creationTime),
			},
			ExistingAPIUser: test.GenAPIUser("john", "john@acme.com"),
		},
		// scenario 2
		{
			Name: "scenario 2: gets a list of addons added to cluster except those that are not accessible",
			ExpectedResponse: []apiv1.Addon{
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "addon1",
						Name:              "addon1",
						CreationTimestamp: apiv1.NewTime(creationTime),
					},
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "addon2",
						Name:              "addon2",
						CreationTimestamp: apiv1.NewTime(creationTime),
					},
				},
			},
			ExpectedHTTPStatus: http.StatusOK,
			ExistingKubermaticObjs: []runtime.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				/*add cluster*/
				cluster,
				test.GenTestAddon("addon0", nil, cluster, creationTime),
				test.GenTestAddon("addon1", nil, cluster, creationTime),
				test.GenTestAddon("addon2", nil, cluster, creationTime),
				test.GenTestAddon("addon3", nil, cluster, creationTime),
				test.GenTestAddon("addon4", nil, cluster, creationTime),
			},
			ExistingAPIUser: test.GenAPIUser("john", "john@acme.com"),
		},
		// scenario 3
		{
			Name: "scenario 3: the admin Bob gets a list of addons added to any cluster",
			ExpectedResponse: []apiv1.Addon{
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "addon1",
						Name:              "addon1",
						CreationTimestamp: apiv1.NewTime(creationTime),
					},
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "addon2",
						Name:              "addon2",
						CreationTimestamp: apiv1.NewTime(creationTime),
					},
					Spec: apiv1.AddonSpec{
						Variables: testVariables,
					},
				},
			},
			ExpectedHTTPStatus: http.StatusOK,
			ExistingKubermaticObjs: []runtime.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				/*add cluster*/
				cluster,
				test.GenTestAddon("addon1", nil, cluster, creationTime),
				test.GenTestAddon("addon2", createRawVariables(t, testVariables), cluster, creationTime),
				genUser("bob", "bob@acme.com", true),
			},
			ExistingAPIUser: test.GenAPIUser("bob", "bob@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/addons", "my-first-project-ID", cluster.Name), strings.NewReader(""))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.ExpectedHTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatus, res.Code, res.Body.String())
			}

			actualAddons := test.NewAddonSliceWrapper{}
			actualAddons.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedAddons := test.NewAddonSliceWrapper(tc.ExpectedResponse)
			wrappedExpectedAddons.Sort()
			actualAddons.EqualOrDie(wrappedExpectedAddons, t)
		})
	}
}

func TestCreateAddon(t *testing.T) {
	t.Parallel()
	cluster := test.GenDefaultCluster()
	cluster.Status.NamespaceName = fmt.Sprintf("cluster-%s", cluster.Name)

	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedResponse       apiv1.Addon
		ExpectedHTTPStatus     int
		ExistingKubermaticObjs []runtime.Object
		ExistingAPIUser        *apiv1.User
	}{
		// scenario 1
		{
			Name: "scenario 1: create an addon",
			Body: `{
				"name": "addon1",
				"spec": {
					"variables": null
				}
			}`,
			ExpectedResponse: apiv1.Addon{
				ObjectMeta: apiv1.ObjectMeta{
					ID:   "addon1",
					Name: "addon1",
				},
			},
			ExpectedHTTPStatus: http.StatusCreated,
			ExistingKubermaticObjs: []runtime.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				/*add cluster*/
				cluster,
			},
			ExistingAPIUser: test.GenAPIUser("john", "john@acme.com"),
		},
		// scenario 2
		{
			Name: "scenario 2: try to create an addon that wouldn't be accessible",
			Body: `{
				"name": "inaccessible",
				"spec": {
					"variables": null
				}
			}`,
			ExpectedHTTPStatus: http.StatusUnauthorized,
			ExistingKubermaticObjs: []runtime.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				/*add cluster*/
				cluster,
			},
			ExistingAPIUser: test.GenAPIUser("john", "john@acme.com"),
		},
		// scenario 3
		{
			Name: "scenario 3: the admin Bob can create addon for John's cluster",
			Body: `{
				"name": "addon1",
				"spec": {
					"variables": null
				}
			}`,
			ExpectedResponse: apiv1.Addon{
				ObjectMeta: apiv1.ObjectMeta{
					ID:   "addon1",
					Name: "addon1",
				},
			},
			ExpectedHTTPStatus: http.StatusCreated,
			ExistingKubermaticObjs: []runtime.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				genUser("bob", "bob@acme.com", true),
				/*add cluster*/
				cluster,
			},
			ExistingAPIUser: test.GenAPIUser("bob", "bob@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/addons", "my-first-project-ID", cluster.Name), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.ExpectedHTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatus, res.Code, res.Body.String())
			}

			if res.Code == http.StatusCreated {
				bytes, err := json.Marshal(tc.ExpectedResponse)
				if err != nil {
					t.Fatalf("failed to marshall expected response %v", err)
				}

				test.CompareWithResult(t, res, string(bytes))
			}
		})
	}
}

func TestCreatePatchGetAddon(t *testing.T) {
	t.Parallel()
	cluster := test.GenDefaultCluster()
	cluster.Status.NamespaceName = fmt.Sprintf("cluster-%s", cluster.Name)

	testcases := []struct {
		Name                     string
		CreateBody               string
		PatchBody                string
		ExpectedGetResponse      apiv1.Addon
		ExpectedCreateHTTPStatus int
		AddonToPatch             string
		ExpectedPatchHTTPStatus  int
		AddonToGet               string
		ExpectedGetHTTPStatus    int
		ExistingKubermaticObjs   []runtime.Object
		ExistingAPIUser          *apiv1.User
	}{
		// scenario 1
		{
			Name: "scenario 1: create, patch, get an addon",
			CreateBody: `{
				"name": "addon1",
				"spec": {
					"variables": null
				}
			}`,
			ExpectedCreateHTTPStatus: http.StatusCreated,
			AddonToPatch:             "addon1",
			PatchBody: `{
				"name": "addon1",
				"spec": {
					"variables": {"foo": "bar"}
				}
			}`,
			ExpectedPatchHTTPStatus: http.StatusOK,
			AddonToGet:              "addon1",
			ExpectedGetHTTPStatus:   http.StatusOK,
			ExpectedGetResponse: apiv1.Addon{
				ObjectMeta: apiv1.ObjectMeta{
					ID:   "addon1",
					Name: "addon1",
				},
				Spec: apiv1.AddonSpec{
					Variables: map[string]interface{}{"foo": "bar"},
				},
			},
			ExistingKubermaticObjs: []runtime.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				/*add cluster*/
				cluster,
			},
			ExistingAPIUser: test.GenAPIUser("john", "john@acme.com"),
		},
		// scenario 2
		{
			Name: "scenario 2: the admin Bob can patch, get an addon for Jonh cluster",
			CreateBody: `{
				"name": "addon1",
				"spec": {
					"variables": null
				}
			}`,
			ExpectedCreateHTTPStatus: http.StatusCreated,
			AddonToPatch:             "addon1",
			PatchBody: `{
				"name": "addon1",
				"spec": {
					"variables": {"foo": "bar"}
				}
			}`,
			ExpectedPatchHTTPStatus: http.StatusOK,
			AddonToGet:              "addon1",
			ExpectedGetHTTPStatus:   http.StatusOK,
			ExpectedGetResponse: apiv1.Addon{
				ObjectMeta: apiv1.ObjectMeta{
					ID:   "addon1",
					Name: "addon1",
				},
				Spec: apiv1.AddonSpec{
					Variables: map[string]interface{}{"foo": "bar"},
				},
			},
			ExistingKubermaticObjs: []runtime.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				genUser("bob", "bob@acme.com", true),
				/*add cluster*/
				cluster,
			},
			ExistingAPIUser: test.GenAPIUser("bob", "bob@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/addons", "my-first-project-ID", cluster.Name), strings.NewReader(tc.CreateBody))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.ExpectedCreateHTTPStatus {
				t.Fatalf("Expected CREATE HTTP status code %d, got %d: %s", tc.ExpectedCreateHTTPStatus, res.Code, res.Body.String())
			}

			if res.Code != http.StatusCreated {
				return
			}

			req = httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/addons/%s", "my-first-project-ID", cluster.Name, tc.AddonToPatch), strings.NewReader(tc.PatchBody))
			res = httptest.NewRecorder()

			ep.ServeHTTP(res, req)

			if res.Code != tc.ExpectedPatchHTTPStatus {
				t.Fatalf("Expected PATCH HTTP status code %d, got %d: %s", tc.ExpectedPatchHTTPStatus, res.Code, res.Body.String())
			}

			if res.Code != http.StatusOK {
				return
			}

			req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/addons/%s", "my-first-project-ID", cluster.Name, tc.AddonToGet), strings.NewReader(""))
			res = httptest.NewRecorder()

			if res.Code != tc.ExpectedGetHTTPStatus {
				t.Fatalf("Expected GET HTTP status code %d, got %d: %s", tc.ExpectedGetHTTPStatus, res.Code, res.Body.String())
			}

			ep.ServeHTTP(res, req)

			if res.Code != http.StatusOK {
				return
			}

			bytes, err := json.Marshal(tc.ExpectedGetResponse)
			if err != nil {
				t.Fatalf("failed to marshall expected response %v", err)
			}
			test.CompareWithResult(t, res, string(bytes))
		})
	}
}

func createRawVariables(t *testing.T, in map[string]interface{}) *runtime.RawExtension {
	result := &runtime.RawExtension{}
	raw, err := k8sjson.Marshal(in)
	if err != nil {
		t.Fatalf("failed to marshal external Variables: %v", err)
	}
	result.Raw = raw
	return result
}

func genUser(name, email string, isAdmin bool) *kubermaticv1.User {
	user := test.GenUser("", name, email)
	user.Spec.IsAdmin = isAdmin
	return user
}
