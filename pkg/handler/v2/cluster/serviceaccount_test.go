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

package cluster_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/diff"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	apiserverserviceaccount "k8s.io/apiserver/pkg/authentication/serviceaccount"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCreateClusterSAEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		saName                 string
		saNamespace            string
		body                   string
		expectedResponse       string
		httpStatus             int
		clusterToGet           string
		existingKubermaticObjs []ctrlruntimeclient.Object
		existingKubernetesObjs []ctrlruntimeclient.Object
		existingAPIUser        *apiv1.User
	}{
		{
			name:             "scenario 1: create new service account",
			saName:           "test",
			saNamespace:      "default",
			body:             `{"namespace":"default", "name":"test"}`,
			expectedResponse: `{"id":"test","name":"test","creationTimestamp":"0001-01-01T00:00:00Z","namespace":"default"}`,
			httpStatus:       http.StatusCreated,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 2: create new service account with same name in different namespace",
			saName:           "test",
			saNamespace:      "default",
			body:             `{"namespace":"default", "name":"test"}`,
			expectedResponse: `{"id":"test","name":"test","creationTimestamp":"0001-01-01T00:00:00Z","namespace":"default"}`,
			httpStatus:       http.StatusCreated,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: resources.KubeSystemNamespaceName, Name: "test"}},
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 3: create new service account should failed if already exist",
			saName:           "test",
			saNamespace:      "default",
			body:             `{"namespace":"default", "name":"test"}`,
			expectedResponse: `{"error":{"code":409,"message":"serviceaccounts \"test\" already exists"}}`,
			httpStatus:       http.StatusConflict,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test"}},
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 4: the admin can create service account",
			saName:           "test",
			saNamespace:      "default",
			body:             `{"namespace":"default", "name":"test"}`,
			expectedResponse: `{"id":"test","name":"test","creationTimestamp":"0001-01-01T00:00:00Z","namespace":"default"}`,
			httpStatus:       http.StatusCreated,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				genUser("John", "john@acme.com", true),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{},
			existingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
		},
		{
			name:             "scenario 5: user can not create service account for Bob's cluster",
			saName:           "test",
			saNamespace:      "default",
			body:             `{"name":"test"}`,
			expectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to project my-first-project-ID"}}`,
			httpStatus:       http.StatusForbidden,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				genUser("John", "john@acme.com", false),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{},
			existingAPIUser:        test.GenAPIUser("John", "john@acme.com"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			var kubermaticObj []ctrlruntimeclient.Object
			kubeObj = append(kubeObj, tc.existingKubernetesObjs...)
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/projects/%s/clusters/%s/serviceaccount", test.ProjectName, tc.clusterToGet), strings.NewReader(tc.body))
			res := httptest.NewRecorder()

			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, cs, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)

			// test k8s sa and secret has been created
			if tc.httpStatus == http.StatusOK {
				client := cs.FakeClient
				ctx := context.Background()
				sa := &corev1.ServiceAccount{}
				if err := client.Get(ctx, types.NamespacedName{Namespace: tc.saNamespace, Name: tc.saName}, sa); err != nil {
					t.Fatalf("failed to get service account: %s", err)
				}

				if sa.GetLabels()[cluster.ServiceAccountComponentKey] != cluster.ServiceAccountComponentValue {
					t.Fatalf("expect service account account to be labeled %s=%s but got labels=%v", cluster.ServiceAccountComponentKey, cluster.ServiceAccountComponentValue, sa.GetLabels())
				}

				secretList := &corev1.SecretList{}
				if err := client.List(ctx, secretList, ctrlruntimeclient.InNamespace(sa.Namespace)); err != nil {
					t.Fatalf("failed to list secret: %s", err)
				}

				secretCreated := false
				for _, secret := range secretList.Items {
					if apiserverserviceaccount.IsServiceAccountToken(&secret, sa) {
						secretCreated = true
						break
					}
				}
				if !secretCreated {
					t.Fatal("service account secret has not been created")
				}
			}
		})
	}
}

func TestDeleteClusterSAEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                        string
		saName                      string
		saNamespace                 string
		body                        string
		expectedResponse            string
		httpStatus                  int
		clusterToGet                string
		existingKubermaticObjs      []ctrlruntimeclient.Object
		existingKubernetesObjs      []ctrlruntimeclient.Object
		expectingRoleBinding        []*rbacv1.RoleBinding
		expectingClusterRoleBinding []*rbacv1.ClusterRoleBinding
		existingAPIUser             *apiv1.User
	}{
		{
			name:             "scenario 1: delete service account with no binding",
			saName:           "test",
			saNamespace:      "default",
			body:             `{"namespace":"default", "name":"test"}`,
			expectedResponse: `{}`,
			httpStatus:       http.StatusOK,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", Labels: map[string]string{cluster.ServiceAccountComponentKey: cluster.ServiceAccountComponentValue}}},
			},
			expectingRoleBinding:        []*rbacv1.RoleBinding{},
			expectingClusterRoleBinding: []*rbacv1.ClusterRoleBinding{},
			existingAPIUser:             test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 2: delete service account which is bound to one role with only this subject",
			saName:           "test",
			saNamespace:      "default",
			body:             `{"namespace":"default", "name":"test"}`,
			expectedResponse: `{}`,
			httpStatus:       http.StatusOK,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", Labels: map[string]string{cluster.ServiceAccountComponentKey: cluster.ServiceAccountComponentValue}}},
				test.GenServiceAccountRoleBinding("test", "default", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
					}),
			},
			expectingRoleBinding: []*rbacv1.RoleBinding{
				test.GenServiceAccountRoleBinding("test", "default", "role-1", []rbacv1.Subject{}),
			},
			expectingClusterRoleBinding: []*rbacv1.ClusterRoleBinding{},
			existingAPIUser:             test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 3: delete service account which is bound to one role which another SA with same name in different namespace as subjects",
			saName:           "test",
			saNamespace:      "default",
			body:             `{"namespace":"default", "name":"test"}`,
			expectedResponse: `{}`,
			httpStatus:       http.StatusOK,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", Labels: map[string]string{cluster.ServiceAccountComponentKey: cluster.ServiceAccountComponentValue}}},
				test.GenServiceAccountRoleBinding("test", "default", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "another-ns"},
					}),
			},
			expectingRoleBinding: []*rbacv1.RoleBinding{
				test.GenServiceAccountRoleBinding("test", "default", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "another-ns"},
					}),
			},
			expectingClusterRoleBinding: []*rbacv1.ClusterRoleBinding{},
			existingAPIUser:             test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 4: delete service account which is bound to one role with several subjects of different kinds",
			saName:           "test",
			saNamespace:      "default",
			body:             `{"namespace":"default", "name":"test"}`,
			expectedResponse: `{}`,
			httpStatus:       http.StatusOK,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", Labels: map[string]string{cluster.ServiceAccountComponentKey: cluster.ServiceAccountComponentValue}}},
				test.GenServiceAccountRoleBinding("test", "default", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "a-group"},
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
						{Kind: rbacv1.ServiceAccountKind, Name: "foo", Namespace: "kube-system"},
						{Kind: rbacv1.UserKind, Name: "a-user"},
					}),
			},
			expectingRoleBinding: []*rbacv1.RoleBinding{
				test.GenServiceAccountRoleBinding("test", "default", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "a-group"},
						{Kind: rbacv1.ServiceAccountKind, Name: "foo", Namespace: "kube-system"},
						{Kind: rbacv1.UserKind, Name: "a-user"},
					}),
			},
			expectingClusterRoleBinding: []*rbacv1.ClusterRoleBinding{},
			existingAPIUser:             test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 5: delete service account which is bound to several role with several subjects of different kinds",
			saName:           "test",
			saNamespace:      "default",
			body:             `{"namespace":"default", "name":"test"}`,
			expectedResponse: `{}`,
			httpStatus:       http.StatusOK,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", Labels: map[string]string{cluster.ServiceAccountComponentKey: cluster.ServiceAccountComponentValue}}},
				test.GenServiceAccountRoleBinding("test", "default", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "a-group"},
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
						{Kind: rbacv1.ServiceAccountKind, Name: "foo", Namespace: "kube-system"},
						{Kind: rbacv1.UserKind, Name: "a-user"},
					}),
				test.GenServiceAccountRoleBinding("another-binding", "default", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "a-group"},
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
					}),
				test.GenServiceAccountRoleBinding("not-bind-to-test-sa", "default", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "a-group"},
						{Kind: rbacv1.UserKind, Name: "a-user"},
						{Kind: rbacv1.ServiceAccountKind, Name: "foo", Namespace: "kube-system"},
					}),
			},
			expectingRoleBinding: []*rbacv1.RoleBinding{
				test.GenServiceAccountRoleBinding("test", "default", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "a-group"},
						{Kind: rbacv1.ServiceAccountKind, Name: "foo", Namespace: "kube-system"},
						{Kind: rbacv1.UserKind, Name: "a-user"},
					}),
				test.GenServiceAccountRoleBinding("another-binding", "default", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "a-group"},
					}),
				test.GenServiceAccountRoleBinding("not-bind-to-test-sa", "default", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "a-group"},
						{Kind: rbacv1.UserKind, Name: "a-user"},
						{Kind: rbacv1.ServiceAccountKind, Name: "foo", Namespace: "kube-system"},
					}),
			},
			expectingClusterRoleBinding: []*rbacv1.ClusterRoleBinding{},
			existingAPIUser:             test.GenDefaultAPIUser(),
		},

		// test unbind clusterRole
		{
			name:             "scenario 6: delete service account which is bound to one clusterRole with only this subject",
			saName:           "test",
			saNamespace:      "default",
			body:             `{"namespace":"default", "name":"test"}`,
			expectedResponse: `{}`,
			httpStatus:       http.StatusOK,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", Labels: map[string]string{cluster.ServiceAccountComponentKey: cluster.ServiceAccountComponentValue}}},
				test.GenServiceAccountClusterRoleBinding("test", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
					}),
			},
			expectingRoleBinding: []*rbacv1.RoleBinding{},
			expectingClusterRoleBinding: []*rbacv1.ClusterRoleBinding{
				test.GenServiceAccountClusterRoleBinding("test", "role-1", []rbacv1.Subject{}),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 7: delete service account which is bound to one ClusterRole which another SA with same name in different namespace as subjects",
			saName:           "test",
			saNamespace:      "default",
			body:             `{"namespace":"default", "name":"test"}`,
			expectedResponse: `{}`,
			httpStatus:       http.StatusOK,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", Labels: map[string]string{cluster.ServiceAccountComponentKey: cluster.ServiceAccountComponentValue}}},
				test.GenServiceAccountClusterRoleBinding("test", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "another-ns"},
					}),
			},
			expectingRoleBinding: []*rbacv1.RoleBinding{},
			expectingClusterRoleBinding: []*rbacv1.ClusterRoleBinding{
				test.GenServiceAccountClusterRoleBinding("test", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "another-ns"},
					}),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 8: delete service account which is bound to one ClusterRole with several subjects of different kinds",
			saName:           "test",
			saNamespace:      "default",
			body:             `{"namespace":"default", "name":"test"}`,
			expectedResponse: `{}`,
			httpStatus:       http.StatusOK,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", Labels: map[string]string{cluster.ServiceAccountComponentKey: cluster.ServiceAccountComponentValue}}},
				test.GenServiceAccountClusterRoleBinding("test", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "a-group"},
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
						{Kind: rbacv1.ServiceAccountKind, Name: "foo", Namespace: "kube-system"},
						{Kind: rbacv1.UserKind, Name: "a-user"},
					}),
			},
			expectingRoleBinding: []*rbacv1.RoleBinding{},
			expectingClusterRoleBinding: []*rbacv1.ClusterRoleBinding{
				test.GenServiceAccountClusterRoleBinding("test", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "a-group"},
						{Kind: rbacv1.ServiceAccountKind, Name: "foo", Namespace: "kube-system"},
						{Kind: rbacv1.UserKind, Name: "a-user"},
					}),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 9: delete service account which is bound to several clusterRole with several subjects of different kinds",
			saName:           "test",
			saNamespace:      "default",
			body:             `{"namespace":"default", "name":"test"}`,
			expectedResponse: `{}`,
			httpStatus:       http.StatusOK,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", Labels: map[string]string{cluster.ServiceAccountComponentKey: cluster.ServiceAccountComponentValue}}},
				test.GenServiceAccountClusterRoleBinding("test", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "a-group"},
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
						{Kind: rbacv1.ServiceAccountKind, Name: "foo", Namespace: "kube-system"},
						{Kind: rbacv1.UserKind, Name: "a-user"},
					}),
				test.GenServiceAccountClusterRoleBinding("another-binding", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "a-group"},
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
					}),
				test.GenServiceAccountClusterRoleBinding("not-bind-to-test-sa", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "a-group"},
						{Kind: rbacv1.UserKind, Name: "a-user"},
						{Kind: rbacv1.ServiceAccountKind, Name: "foo", Namespace: "kube-system"},
					}),
			},
			expectingRoleBinding: []*rbacv1.RoleBinding{},
			expectingClusterRoleBinding: []*rbacv1.ClusterRoleBinding{
				test.GenServiceAccountClusterRoleBinding("test", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "a-group"},
						{Kind: rbacv1.ServiceAccountKind, Name: "foo", Namespace: "kube-system"},
						{Kind: rbacv1.UserKind, Name: "a-user"},
					}),
				test.GenServiceAccountClusterRoleBinding("another-binding", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "a-group"},
					}),
				test.GenServiceAccountClusterRoleBinding("not-bind-to-test-sa", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "a-group"},
						{Kind: rbacv1.UserKind, Name: "a-user"},
						{Kind: rbacv1.ServiceAccountKind, Name: "foo", Namespace: "kube-system"},
					}),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 10: delete service account which is bound to one ClusterRole and one Role",
			saName:           "test",
			saNamespace:      "default",
			body:             `{"namespace":"default", "name":"test"}`,
			expectedResponse: `{}`,
			httpStatus:       http.StatusOK,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", Labels: map[string]string{cluster.ServiceAccountComponentKey: cluster.ServiceAccountComponentValue}}},
				test.GenServiceAccountClusterRoleBinding("test", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
						{Kind: rbacv1.GroupKind, Name: "test"},
					}),
				test.GenServiceAccountRoleBinding("test", "default", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
						{Kind: rbacv1.GroupKind, Name: "test"},
					}),
			},
			expectingRoleBinding: []*rbacv1.RoleBinding{
				test.GenServiceAccountRoleBinding("test", "default", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "test"},
					}),
			},
			expectingClusterRoleBinding: []*rbacv1.ClusterRoleBinding{
				test.GenServiceAccountClusterRoleBinding("test", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "test"},
					}),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 11: can not delete service account which is not labeled",
			saName:           "test",
			saNamespace:      "default",
			body:             `{"namespace":"default", "name":"test"}`,
			expectedResponse: `{"error":{"code":403,"message":"can not delete service account which is not labeled component=clusterServiceAccount"}}`,
			httpStatus:       http.StatusForbidden,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test"}},
				test.GenServiceAccountClusterRoleBinding("test", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
					}),
				test.GenServiceAccountRoleBinding("test", "default", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
					}),
			},
			expectingRoleBinding: []*rbacv1.RoleBinding{
				test.GenServiceAccountRoleBinding("test", "default", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
					}),
			},
			expectingClusterRoleBinding: []*rbacv1.ClusterRoleBinding{
				test.GenServiceAccountClusterRoleBinding("test", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
					}),
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 12: admin can delete service account",
			saName:           "test",
			saNamespace:      "default",
			body:             `{"namespace":"default", "name":"test"}`,
			expectedResponse: `{}`,
			httpStatus:       http.StatusOK,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				genUser("John", "john@acme.com", true),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", Labels: map[string]string{cluster.ServiceAccountComponentKey: cluster.ServiceAccountComponentValue}}},
				test.GenServiceAccountClusterRoleBinding("test", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
						{Kind: rbacv1.GroupKind, Name: "test"},
					}),
				test.GenServiceAccountRoleBinding("test", "default", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
						{Kind: rbacv1.GroupKind, Name: "test"},
					}),
			},
			expectingRoleBinding: []*rbacv1.RoleBinding{
				test.GenServiceAccountRoleBinding("test", "default", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "test"},
					}),
			},
			expectingClusterRoleBinding: []*rbacv1.ClusterRoleBinding{
				test.GenServiceAccountClusterRoleBinding("test", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.GroupKind, Name: "test"},
					}),
			},
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			name:             "scenario 13: user can not delete service account of Bob's cluster",
			saName:           "test",
			saNamespace:      "default",
			body:             `{"namespace":"default", "name":"test"}`,
			expectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to project my-first-project-ID"}}`,
			httpStatus:       http.StatusForbidden,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				genUser("John", "john@acme.com", false),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", Labels: map[string]string{cluster.ServiceAccountComponentKey: cluster.ServiceAccountComponentValue}}},
				test.GenServiceAccountClusterRoleBinding("test", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
						{Kind: rbacv1.GroupKind, Name: "test"},
					}),
				test.GenServiceAccountRoleBinding("test", "default", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
						{Kind: rbacv1.GroupKind, Name: "test"},
					}),
			},
			expectingRoleBinding: []*rbacv1.RoleBinding{
				test.GenServiceAccountRoleBinding("test", "default", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
						{Kind: rbacv1.GroupKind, Name: "test"},
					}),
			},
			expectingClusterRoleBinding: []*rbacv1.ClusterRoleBinding{
				test.GenServiceAccountClusterRoleBinding("test", "role-1",
					[]rbacv1.Subject{
						{Kind: rbacv1.ServiceAccountKind, Name: "test", Namespace: "default"},
						{Kind: rbacv1.GroupKind, Name: "test"},
					}),
			},
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			var kubermaticObj []ctrlruntimeclient.Object
			kubeObj = append(kubeObj, tc.existingKubernetesObjs...)
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v2/projects/%s/clusters/%s/serviceaccount/%s/%s", test.ProjectName, tc.clusterToGet, tc.saNamespace, tc.saName), nil)
			res := httptest.NewRecorder()

			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, cs, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
			client := cs.FakeClient
			ctx := context.Background()

			// test k8s sa has been unbinded and delete.(secret is automatically deleted by k8s controller but cannot be tested with fake client)
			if tc.httpStatus == http.StatusOK {
				sa := &corev1.ServiceAccount{}
				err := client.Get(ctx, types.NamespacedName{Namespace: tc.saNamespace, Name: tc.saName}, sa)
				if err == nil {
					t.Fatalf("service account has not been deleted")
				}
				if !apierrors.IsNotFound(err) {
					t.Fatalf("failed to check that service account has been deleted: %s", err)
				}
			}

			// test service account has been unbind to rolebinding
			for _, rolebinding := range tc.expectingRoleBinding {
				actualRoleBinding := &rbacv1.RoleBinding{}
				if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(rolebinding), actualRoleBinding); err != nil {
					t.Errorf("failed to get rolebinding '%s/%s': %s", rolebinding.Namespace, rolebinding.Name, err)
				}
				if !diff.SemanticallyEqual(rolebinding.Subjects, actualRoleBinding.Subjects) {
					t.Errorf("Subjects of rolebinding '%s/%s' differs:\n%v", rolebinding.Namespace, rolebinding.Name, diff.ObjectDiff(rolebinding.Subjects, actualRoleBinding.Subjects))
				}
			}

			// test service account has been unbind to clusterRolebinding
			for _, clusterRolebinding := range tc.expectingClusterRoleBinding {
				actualClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
				if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(clusterRolebinding), actualClusterRoleBinding); err != nil {
					t.Errorf("failed to get clusterRoleBinding '%s': %s", clusterRolebinding.Name, err)
				}
				if !diff.SemanticallyEqual(clusterRolebinding.Subjects, actualClusterRoleBinding.Subjects) {
					t.Errorf("Subjects of clusterRoleBinding '%s' differs:\n%v", clusterRolebinding.Name, diff.ObjectDiff(clusterRolebinding.Subjects, actualClusterRoleBinding.Subjects))
				}
			}
		})
	}
}

func TestListClusterSAEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		expectedResponse       string
		httpStatus             int
		clusterToGet           string
		existingKubermaticObjs []ctrlruntimeclient.Object
		existingKubernetesObjs []ctrlruntimeclient.Object
		existingAPIUser        *apiv1.User
	}{
		{
			name:             "scenario 1: list accounts (only labeled sa should be returned)",
			expectedResponse: `[{"id":"labeled-sa-1","name":"labeled-sa-1","creationTimestamp":"0001-01-01T00:00:00Z","namespace":"default"},{"id":"labeled-sa-1","name":"labeled-sa-1","creationTimestamp":"0001-01-01T00:00:00Z","namespace":"kube-system"}]`,
			httpStatus:       http.StatusOK,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: resources.KubeSystemNamespaceName, Name: "non-labeled-sa"}},
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: resources.KubeSystemNamespaceName, Name: "labeled-sa-1", Labels: map[string]string{cluster.ServiceAccountComponentKey: cluster.ServiceAccountComponentValue}}},
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "labeled-sa-1", Labels: map[string]string{cluster.ServiceAccountComponentKey: cluster.ServiceAccountComponentValue}}},
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			name:             "scenario 2: admin can list accounts (only labeled sa should be returned)",
			expectedResponse: `[{"id":"labeled-sa-1","name":"labeled-sa-1","creationTimestamp":"0001-01-01T00:00:00Z","namespace":"default"},{"id":"labeled-sa-1","name":"labeled-sa-1","creationTimestamp":"0001-01-01T00:00:00Z","namespace":"kube-system"}]`,
			httpStatus:       http.StatusOK,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				genUser("John", "john@acme.com", true),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: resources.KubeSystemNamespaceName, Name: "non-labeled-sa"}},
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: resources.KubeSystemNamespaceName, Name: "labeled-sa-1", Labels: map[string]string{cluster.ServiceAccountComponentKey: cluster.ServiceAccountComponentValue}}},
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "labeled-sa-1", Labels: map[string]string{cluster.ServiceAccountComponentKey: cluster.ServiceAccountComponentValue}}},
			},
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			name:             "scenario 3: user can non list accounts for Bob's cluster",
			expectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to project my-first-project-ID"}}`,
			httpStatus:       http.StatusForbidden,
			clusterToGet:     test.GenDefaultCluster().Name,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				genUser("John", "john@acme.com", false),
			),
			existingKubernetesObjs: []ctrlruntimeclient.Object{
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: resources.KubeSystemNamespaceName, Name: "non-labeled-sa"}},
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: resources.KubeSystemNamespaceName, Name: "labeled-sa-1", Labels: map[string]string{cluster.ServiceAccountComponentKey: cluster.ServiceAccountComponentValue}}},
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "labeled-sa-1", Labels: map[string]string{cluster.ServiceAccountComponentKey: cluster.ServiceAccountComponentValue}}},
			},
			existingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []ctrlruntimeclient.Object
			var kubeObj []ctrlruntimeclient.Object
			var kubermaticObj []ctrlruntimeclient.Object
			kubeObj = append(kubeObj, tc.existingKubernetesObjs...)
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/projects/%s/clusters/%s/serviceaccount", test.ProjectName, tc.clusterToGet), strings.NewReader(""))
			res := httptest.NewRecorder()

			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}

func TestCreateClusterSAReqValidate(t *testing.T) {
	tests := []struct {
		name             string
		projectReq       common.ProjectReq
		body             apiv2.ClusterServiceAccount
		expectedErrorMsg string
	}{
		{
			name:       "valid req",
			projectReq: common.ProjectReq{ProjectID: "projectID"},
			body: apiv2.ClusterServiceAccount{
				Namespace:  "default",
				ObjectMeta: apiv1.ObjectMeta{Name: "test"},
			},
			expectedErrorMsg: "",
		},
		{
			name:       "invalid: missing projectID",
			projectReq: common.ProjectReq{},
			body: apiv2.ClusterServiceAccount{
				Namespace:  "default",
				ObjectMeta: apiv1.ObjectMeta{Name: "test"},
			},
			expectedErrorMsg: "the project ID cannot be empty",
		},
		{
			name:       "invalid: missing namespace",
			projectReq: common.ProjectReq{ProjectID: "projectID"},
			body: apiv2.ClusterServiceAccount{
				Namespace:  "",
				ObjectMeta: apiv1.ObjectMeta{Name: "test"},
			},
			expectedErrorMsg: "namespace must be set",
		},
		{
			name:       "invalid: missing name",
			projectReq: common.ProjectReq{ProjectID: "projectID"},
			body: apiv2.ClusterServiceAccount{
				Namespace:  "default",
				ObjectMeta: apiv1.ObjectMeta{Name: ""},
			},
			expectedErrorMsg: "name must be set",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := cluster.CreateClusterSAReq{
				ProjectReq: tt.projectReq,
				Body:       tt.body,
			}

			err := req.Validate()

			if len(tt.expectedErrorMsg) > 0 {
				if err == nil {
					t.Fatalf("expect error '%s' but got nil", tt.expectedErrorMsg)
				}
				if tt.expectedErrorMsg != err.Error() {
					t.Errorf("expected error '%s' got '%s'", tt.expectedErrorMsg, err.Error())
				}
			} else if err != nil {
				t.Fatalf("expect no error but got error=%s", err)
			}
		})
	}
}
