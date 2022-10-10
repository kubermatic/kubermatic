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
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
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
				test.GenProject("foo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
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
			ExpectedResponseString: genToken("default", test.IDToken),
		},
		{
			Name:         "scenario 2: viewer gets viewer kubeconfig",
			HTTPStatus:   http.StatusOK,
			ProjectToGet: "foo-ID",
			ClusterToGet: "cluster-foo",
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				test.GenTestSeed(),
				/*add projects*/
				test.GenProject("foo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
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
			ExpectedResponseString: genToken("default", test.IDViewerToken),
		},
		{
			Name:         "scenario 3: the admin gets master kubeconfig for any cluster",
			HTTPStatus:   http.StatusOK,
			ProjectToGet: "foo-ID",
			ClusterToGet: "cluster-foo",
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				test.GenTestSeed(),
				/*add projects*/
				test.GenProject("foo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
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
			ExpectedResponseString: genToken("default", test.IDToken),
		},
		{
			Name:         "scenario 4: the user Bob can not get John's kubeconfig",
			HTTPStatus:   http.StatusForbidden,
			ProjectToGet: "foo-ID",
			ClusterToGet: "cluster-foo",
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				test.GenTestSeed(),
				/*add projects*/
				test.GenProject("foo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
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
			ExpectedResponseString: `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't belong to project foo-ID"}}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/projects/%s/clusters/%s/kubeconfig", tc.ProjectToGet, tc.ClusterToGet), nil)
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(tc.ExistingAPIUser, nil, tc.ExistingObjects, []ctrlruntimeclient.Object{}, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.ExpectedResponseString)
		})
	}
}

func TestGetClusterSAKubeconfig(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                    string
		ExpectedResponseString  string
		ExpectedActions         int
		ProjectToGet            string
		ClusterToGet            string
		ServiceAccountName      string
		ServiceAccountNamespace string
		HTTPStatus              int
		ExistingAPIUser         apiv1.User
		ExistingKubermaticObjs  []ctrlruntimeclient.Object
		ExistingObjects         []ctrlruntimeclient.Object
	}{
		{
			Name:                    "scenario 1: can get sa kubeconfig",
			HTTPStatus:              http.StatusOK,
			ProjectToGet:            "foo-ID",
			ClusterToGet:            "cluster-foo",
			ServiceAccountName:      "test",
			ServiceAccountNamespace: "default",
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				test.GenTestSeed(),
				/*add projects*/
				test.GenProject("foo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("foo-ID", "john@acme.com", "owners"),

				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenCluster("cluster-foo", "cluster-foo", "foo-ID", test.DefaultCreationTimestamp()),
			},
			ExistingObjects: []ctrlruntimeclient.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: "cluster-cluster-foo", Name: "admin-kubeconfig"},
					Data:       map[string][]byte{"kubeconfig": []byte(test.GenerateTestKubeconfig("cluster-foo", test.IDToken))},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", UID: "someUID"},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-sa-token", Annotations: map[string]string{corev1.ServiceAccountNameKey: "test", corev1.ServiceAccountUIDKey: "someUID"}},
					Type:       corev1.SecretTypeServiceAccountToken,
					Data:       map[string][]byte{corev1.ServiceAccountTokenKey: []byte("fake-sa-token")},
				},
			},
			ExistingAPIUser:        *test.GenAPIUser("john", "john@acme.com"),
			ExpectedResponseString: genToken("sa-test", "fake-sa-token"),
		},
		{
			Name:                    "scenario 2: error is returned if service account has no secret (may happen in k8s >= 1.24 if secret is not annoated)",
			HTTPStatus:              http.StatusInternalServerError,
			ProjectToGet:            "foo-ID",
			ClusterToGet:            "cluster-foo",
			ServiceAccountName:      "test",
			ServiceAccountNamespace: "default",
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				test.GenTestSeed(),
				/*add projects*/
				test.GenProject("foo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
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
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test"},
				},
			},
			ExistingAPIUser:        *test.GenAPIUser("john", "john@acme.com"),
			ExpectedResponseString: `{"error":{"code":500,"message":"service account has no secret"}}`,
		},
		{
			Name:                    "scenario 3: error is returned if service account's secret has no key token",
			HTTPStatus:              http.StatusInternalServerError,
			ProjectToGet:            "foo-ID",
			ClusterToGet:            "cluster-foo",
			ServiceAccountName:      "test",
			ServiceAccountNamespace: "default",
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				test.GenTestSeed(),
				/*add projects*/
				test.GenProject("foo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("foo-ID", "john@acme.com", "owners"),

				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenCluster("cluster-foo", "cluster-foo", "foo-ID", test.DefaultCreationTimestamp()),
			},
			ExistingObjects: []ctrlruntimeclient.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: "cluster-cluster-foo", Name: "admin-kubeconfig"},
					Data:       map[string][]byte{"kubeconfig": []byte(test.GenerateTestKubeconfig("cluster-foo", test.IDToken))},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", UID: "someUID"},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-sa-token", Annotations: map[string]string{corev1.ServiceAccountNameKey: "test", corev1.ServiceAccountUIDKey: "someUID"}},
					Type:       corev1.SecretTypeServiceAccountToken,
					Data:       map[string][]byte{"no-token-key": []byte("fake-sa-token")},
				},
			},
			ExistingAPIUser:        *test.GenAPIUser("john", "john@acme.com"),
			ExpectedResponseString: `{"error":{"code":500,"message":"no token defined in the service account's secret"}}`,
		},
		{
			Name:                    "scenario 4: error is returned if service account's secret is not annoatated with account UID",
			HTTPStatus:              http.StatusInternalServerError,
			ProjectToGet:            "foo-ID",
			ClusterToGet:            "cluster-foo",
			ServiceAccountName:      "test",
			ServiceAccountNamespace: "default",
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				test.GenTestSeed(),
				/*add projects*/
				test.GenProject("foo", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("foo-ID", "john@acme.com", "owners"),

				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				test.GenCluster("cluster-foo", "cluster-foo", "foo-ID", test.DefaultCreationTimestamp()),
			},
			ExistingObjects: []ctrlruntimeclient.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: "cluster-cluster-foo", Name: "admin-kubeconfig"},
					Data:       map[string][]byte{"kubeconfig": []byte(test.GenerateTestKubeconfig("cluster-foo", test.IDToken))},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", UID: "someUID"},
					Secrets:    []corev1.ObjectReference{{Name: "test-token-kgl2b"}},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-sa-token", Annotations: map[string]string{corev1.ServiceAccountNameKey: "test", corev1.ServiceAccountUIDKey: "anotherID"}},
					Type:       corev1.SecretTypeServiceAccountToken,
					Data:       map[string][]byte{"no-token-key": []byte("fake-sa-token")},
				},
			},
			ExistingAPIUser:        *test.GenAPIUser("john", "john@acme.com"),
			ExpectedResponseString: `{"error":{"code":500,"message":"service account has no secret"}}`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/projects/%s/clusters/%s/serviceaccount/%s/%s/kubeconfig", tc.ProjectToGet, tc.ClusterToGet, tc.ServiceAccountNamespace, tc.ServiceAccountName), nil)
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(tc.ExistingAPIUser, nil, tc.ExistingObjects, []ctrlruntimeclient.Object{}, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.ExpectedResponseString)
		})
	}
}

func genToken(userName string, tokenID string) string {
	return fmt.Sprintf(`apiVersion: v1
clusters:
- cluster:
    server: test.fake.io
  name: cluster-foo
contexts:
- context:
    cluster: cluster-foo
    user: %s
  name: default
current-context: default
kind: Config
preferences: {}
users:
- name: %s
  user:
    token: %s`, userName, userName, tokenID)
}
