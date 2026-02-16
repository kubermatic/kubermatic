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

package mla

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newTestUserGrafanaReconciler(t *testing.T, objects []ctrlruntimeclient.Object, handler http.Handler) (*userGrafanaReconciler, *httptest.Server) {
	dynamicClient := fake.
		NewClientBuilder().
		WithObjects(objects...).
		Build()
	ts := httptest.NewServer(handler)

	grafanaClient, err := grafanasdk.NewClient(ts.URL, "admin:admin", ts.Client())
	assert.Nil(t, err)

	userGrafanaController := newUserGrafanaController(dynamicClient, kubermaticlog.Logger, func(ctx context.Context) (*grafanasdk.Client, error) {
		return grafanaClient, nil
	}, ts.Client(), ts.URL, "X-WEBAUTH-USER")
	reconciler := userGrafanaReconciler{
		Client:                dynamicClient,
		log:                   kubermaticlog.Logger,
		recorder:              events.NewFakeRecorder(10),
		userGrafanaController: userGrafanaController,
	}
	return &reconciler, ts
}

func TestUserGrafanaReconcile(t *testing.T) {
	testCases := []struct {
		name         string
		requestName  string
		objects      []ctrlruntimeclient.Object
		handlerFunc  http.HandlerFunc
		requests     []request
		hasFinalizer bool
		err          bool
	}{
		{
			name:        "User added",
			requestName: "create",
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "create",
					},
					Spec: kubermaticv1.UserSpec{
						Email:   "user@email.com",
						IsAdmin: false,
					},
				},
			},
			hasFinalizer: true,
			requests: []request{
				{
					name:     "create OAuth user",
					request:  httptest.NewRequest(http.MethodGet, "/api/user", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1, "isGrafanaAdmin": false, "email": "user@email.com", "login": "user@email.com"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete user from default org",
					request:  httptest.NewRequest(http.MethodDelete, "/api/orgs/1/users/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message":"org user deleted"}`)), StatusCode: http.StatusOK},
				},
			},
		},
		{
			name:        "User IsAdmin updated to True",
			requestName: "update",
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "update",
					},
					Spec: kubermaticv1.UserSpec{
						Email:   "user@email.com",
						IsAdmin: true,
					},
				},
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "project1",
						Annotations: map[string]string{GrafanaOrgAnnotationKey: "1"},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "projectName1",
					},
				},
			},
			hasFinalizer: true,
			requests: []request{
				{
					name:     "create OAuth user",
					request:  httptest.NewRequest(http.MethodGet, "/api/user", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1, "isGrafanaAdmin": false, "email": "user@email.com", "login": "user@email.com"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete user from default org",
					request:  httptest.NewRequest(http.MethodDelete, "/api/orgs/1/users/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message":"org user deleted"}`)), StatusCode: http.StatusOK},
				},

				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName1-project1","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get org users",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1/users", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`[]`)), StatusCode: http.StatusOK},
				},
				{
					name:     "add org user",
					request:  httptest.NewRequest(http.MethodPost, "/api/orgs/1/users", strings.NewReader(`{"loginOrEmail":"user@email.com","role":"Editor"}`)),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "User added to organization"}`)), StatusCode: http.StatusOK},
				},

				{
					name:     "update permissions",
					request:  httptest.NewRequest(http.MethodPut, "/api/admin/users/1/permissions", io.NopCloser(strings.NewReader(`{"isGrafanaAdmin":true}`))),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "User permissions updated"}`)), StatusCode: http.StatusOK},
				},
			},
		},
		{
			name:        "User IsAdmin updated to False",
			requestName: "update",
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "update",
					},
					Spec: kubermaticv1.UserSpec{
						Email:   "user@email.com",
						IsAdmin: false,
					},
				},
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "project1",
						Annotations: map[string]string{GrafanaOrgAnnotationKey: "1"},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "projectName1",
					},
				},
				&kubermaticv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "delete",
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
						Finalizers:        []string{mlaFinalizer},
					},
					Spec: kubermaticv1.UserProjectBindingSpec{
						UserEmail: "user@email.com",
						ProjectID: "project1",
						Group:     "viewers-project1",
					},
				},
			},
			hasFinalizer: true,
			requests: []request{
				{
					name:     "create OAuth user",
					request:  httptest.NewRequest(http.MethodGet, "/api/user", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1, "isGrafanaAdmin": true, "email": "user@email.com", "login": "user@email.com"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete user from default org",
					request:  httptest.NewRequest(http.MethodDelete, "/api/orgs/1/users/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message":"org user deleted"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName1-project1","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete org user",
					request:  httptest.NewRequest(http.MethodDelete, "/api/orgs/1/users/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "User deleted"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "update permissions",
					request:  httptest.NewRequest(http.MethodPut, "/api/admin/users/1/permissions", io.NopCloser(strings.NewReader(`{"isGrafanaAdmin":false}`))),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "User permissions updated"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "lookup user",
					request:  httptest.NewRequest(http.MethodGet, "/api/users/lookup?loginOrEmail=user@email.com", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"email":"user@email.com","login":"admin"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName1-project1","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get org users",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1/users", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`[]`)), StatusCode: http.StatusOK},
				},
				{
					name:     "add org user",
					request:  httptest.NewRequest(http.MethodPost, "/api/orgs/1/users", strings.NewReader(`{"loginOrEmail":"user@email.com","role":"Viewer"}`)),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "User added to organization"}`)), StatusCode: http.StatusOK},
				},
			},
		},
		{
			name:        "User is pruned from projects it does not belong to",
			requestName: "prune",
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "prune",
					},
					Spec: kubermaticv1.UserSpec{
						Email:   "user@email.com",
						IsAdmin: false,
					},
				},
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "project1",
						Annotations: map[string]string{GrafanaOrgAnnotationKey: "1"},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "projectName1",
					},
				},
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "project2",
						Annotations: map[string]string{GrafanaOrgAnnotationKey: "2"},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "projectName2",
					},
				},
				&kubermaticv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "delete",
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
						Finalizers:        []string{mlaFinalizer},
					},
					Spec: kubermaticv1.UserProjectBindingSpec{
						UserEmail: "user@email.com",
						ProjectID: "project1",
						Group:     "viewers-project1",
					},
				},
			},
			hasFinalizer: true,
			requests: []request{
				{
					name:     "create OAuth user",
					request:  httptest.NewRequest(http.MethodGet, "/api/user", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1, "isGrafanaAdmin": true, "email": "user@email.com", "login": "user@email.com"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete user from default org",
					request:  httptest.NewRequest(http.MethodDelete, "/api/orgs/1/users/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message":"org user deleted"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName1-project1","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				}, {
					name:     "delete org user",
					request:  httptest.NewRequest(http.MethodDelete, "/api/orgs/1/users/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "User deleted"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/2", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":2,"name":"projectName2-project2","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete org user",
					request:  httptest.NewRequest(http.MethodDelete, "/api/orgs/2/users/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "User deleted"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "update permissions",
					request:  httptest.NewRequest(http.MethodPut, "/api/admin/users/1/permissions", io.NopCloser(strings.NewReader(`{"isGrafanaAdmin":true}`))),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "User permissions updated"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "lookup user",
					request:  httptest.NewRequest(http.MethodGet, "/api/users/lookup?loginOrEmail=user@email.com", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"email":"user@email.com","login":"admin"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName1-project1","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get org users",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1/users", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`[]`)), StatusCode: http.StatusOK},
				},
				{
					name:     "add org user",
					request:  httptest.NewRequest(http.MethodPost, "/api/orgs/1/users", strings.NewReader(`{"loginOrEmail":"user@email.com","role":"Viewer"}`)),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "User added to organization"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get org 2 by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/2", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":2,"name":"projectName2-project2","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete org user",
					request:  httptest.NewRequest(http.MethodDelete, "/api/orgs/2/users/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "User deleted"}`)), StatusCode: http.StatusOK},
				},
			},
		},
		{
			name:        "User delete",
			requestName: "delete",
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "delete",
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
						Finalizers:        []string{mlaFinalizer, "just-a-test-do-not-delete-thanks"},
					},
					Spec: kubermaticv1.UserSpec{
						Email:   "user@email.com",
						IsAdmin: true,
					},
				},
			},
			hasFinalizer: false,
			requests: []request{
				{
					name:     "lookup user",
					request:  httptest.NewRequest(http.MethodGet, "/api/users/lookup?loginOrEmail=user@email.com", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"email":"user@email.com","login":"admin"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete user",
					request:  httptest.NewRequest(http.MethodDelete, "/api/admin/users/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "User deleted"}`)), StatusCode: http.StatusOK},
				},
			},
		},
		{
			name:        "Add users using group project bindings",
			requestName: "group",
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "group",
					},
					Spec: kubermaticv1.UserSpec{
						Email:   "user@email.com",
						IsAdmin: false,
						Groups:  []string{"devs"},
					},
				},
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "project1",
						Annotations: map[string]string{GrafanaOrgAnnotationKey: "1"},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "projectName1",
					},
				},
				&kubermaticv1.GroupProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "group",
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
						Finalizers:        []string{mlaFinalizer},
					},
					Spec: kubermaticv1.GroupProjectBindingSpec{
						ProjectID: "project1",
						Group:     "devs",
						Role:      rbac.ViewerGroupNamePrefix,
					},
				},
			},
			hasFinalizer: true,
			requests: []request{
				{
					name:     "create OAuth user",
					request:  httptest.NewRequest(http.MethodGet, "/api/user", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1, "isGrafanaAdmin": true, "email": "user@email.com", "login": "user@email.com"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete user from default org",
					request:  httptest.NewRequest(http.MethodDelete, "/api/orgs/1/users/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message":"org user deleted"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName1-project1","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete org user",
					request:  httptest.NewRequest(http.MethodDelete, "/api/orgs/1/users/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "User deleted"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "update permissions",
					request:  httptest.NewRequest(http.MethodPut, "/api/admin/users/1/permissions", io.NopCloser(strings.NewReader(`{"isGrafanaAdmin":false}`))),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "User permissions updated"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "lookup user",
					request:  httptest.NewRequest(http.MethodGet, "/api/users/lookup?loginOrEmail=user@email.com", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"email":"user@email.com","login":"admin"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName1-project1","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get org users",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1/users", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`[]`)), StatusCode: http.StatusOK},
				},
				{
					name:     "add org user",
					request:  httptest.NewRequest(http.MethodPost, "/api/orgs/1/users", strings.NewReader(`{"loginOrEmail":"user@email.com","role":"Viewer"}`)),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "User added to organization"}`)), StatusCode: http.StatusOK},
				},
			},
		},
	}
	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			r, assertExpectation := buildTestServer(t, tc.requests...)
			controller, server := newTestUserGrafanaReconciler(t, tc.objects, r)
			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			_, err := controller.Reconcile(ctx, request)
			if err != nil && !tc.err {
				assert.Nil(t, err)
			}
			assert.Equal(t, tc.err, err != nil)
			user := &kubermaticv1.User{}
			if err := controller.Get(ctx, request.NamespacedName, user); err != nil {
				t.Fatalf("unable to get user: %v", err)
			}
			assert.Equal(t, tc.hasFinalizer, kubernetes.HasFinalizer(user, mlaFinalizer))
			assertExpectation()
			server.Close()
		})
	}
}
