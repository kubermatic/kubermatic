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

package cluster_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetMasterKubeconfig(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		ExpectedResponseString string
		ExpectedActions        int
		ProjectToGet           string
		ClusterToGet           string
		HTTPStatus             int
		ExistingAPIUser        apiv1.User
		ExistingKubermaticObjs []ctrlruntimeclient.Object
		ExistingObjects        []ctrlruntimeclient.Object
	}{
		{
			Name:         "scenario 1: owner gets master kubeconfig",
			HTTPStatus:   http.StatusOK,
			ProjectToGet: "foo-ID",
			ClusterToGet: "cluster-foo",
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				test.GenTestSeed(),
				/*add projects*/
				test.GenProject("foo", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("foo-ID", "john@acme.com", "owners"),

				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenCluster("cluster-foo", "cluster-foo", "foo-ID", test.DefaultCreationTimestamp()),
			},
			ExistingObjects: []ctrlruntimeclient.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "cluster-cluster-foo",
						Name:      "admin-kubeconfig",
					},
					Data: map[string][]byte{
						"kubeconfig": []byte(test.GenerateTestKubeconfig("cluster-foo", test.IDToken)),
					},
				},
			},
			ExistingAPIUser:        *test.GenAPIUser("john", "john@acme.com"),
			ExpectedResponseString: genToken(test.IDToken),
		},
		{
			Name:         "scenario 2: viewer gets viewer kubeconfig",
			HTTPStatus:   http.StatusOK,
			ProjectToGet: "foo-ID",
			ClusterToGet: "cluster-foo",
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				test.GenTestSeed(),
				/*add projects*/
				test.GenProject("foo", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("foo-ID", "john@acme.com", "viewers"),

				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenCluster("cluster-foo", "cluster-foo", "foo-ID", test.DefaultCreationTimestamp()),
			},
			ExistingObjects: []ctrlruntimeclient.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "cluster-cluster-foo",
						Name:      "viewer-kubeconfig",
					},
					Data: map[string][]byte{
						"kubeconfig": []byte(test.GenerateTestKubeconfig("cluster-foo", test.IDViewerToken)),
					},
				},
			},
			ExistingAPIUser:        *test.GenAPIUser("john", "john@acme.com"),
			ExpectedResponseString: genToken(test.IDViewerToken),
		},
		{
			Name:         "scenario 3: the admin gets master kubeconfig for any cluster",
			HTTPStatus:   http.StatusOK,
			ProjectToGet: "foo-ID",
			ClusterToGet: "cluster-foo",
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				test.GenTestSeed(),
				/*add projects*/
				test.GenProject("foo", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("foo-ID", "john@acme.com", "owners"),

				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				genUser("bob", "bob@acme.com", true),
				test.GenCluster("cluster-foo", "cluster-foo", "foo-ID", test.DefaultCreationTimestamp()),
			},
			ExistingObjects: []ctrlruntimeclient.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "cluster-cluster-foo",
						Name:      "admin-kubeconfig",
					},
					Data: map[string][]byte{
						"kubeconfig": []byte(test.GenerateTestKubeconfig("cluster-foo", test.IDToken)),
					},
				},
			},
			ExistingAPIUser:        *test.GenAPIUser("bob", "bob@acme.com"),
			ExpectedResponseString: genToken(test.IDToken),
		},
		{
			Name:         "scenario 4: the user Bob can not get John's kubeconfig",
			HTTPStatus:   http.StatusForbidden,
			ProjectToGet: "foo-ID",
			ClusterToGet: "cluster-foo",
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				test.GenTestSeed(),
				/*add projects*/
				test.GenProject("foo", kubermaticapiv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("foo-ID", "john@acme.com", "owners"),

				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				genUser("bob", "bob@acme.com", false),
				test.GenCluster("cluster-foo", "cluster-foo", "foo-ID", test.DefaultCreationTimestamp()),
			},
			ExistingObjects: []ctrlruntimeclient.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "cluster-cluster-foo",
						Name:      "admin-kubeconfig",
					},
					Data: map[string][]byte{
						"kubeconfig": []byte(test.GenerateTestKubeconfig("cluster-foo", test.IDToken)),
					},
				},
			},
			ExistingAPIUser:        *test.GenAPIUser("bob", "bob@acme.com"),
			ExpectedResponseString: `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't belong to the given project = foo-ID"}}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/clusters/%s/kubeconfig", tc.ProjectToGet, tc.ClusterToGet), nil)
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(tc.ExistingAPIUser, nil, tc.ExistingObjects, []ctrlruntimeclient.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.ExpectedResponseString)
		})
	}

}

func genToken(tokenID string) string {
	return fmt.Sprintf(`apiVersion: v1
clusters:
- cluster:
    server: test.fake.io
  name: cluster-foo
contexts:
- context:
    cluster: cluster-foo
    user: default
  name: default
current-context: default
kind: Config
preferences: {}
users:
- name: default
  user:
    token: %s`, tokenID)
}
