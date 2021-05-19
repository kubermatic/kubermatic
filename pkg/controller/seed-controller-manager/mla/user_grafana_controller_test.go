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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newTestUserGrafanaReconciler(t *testing.T, objects []ctrlruntimeclient.Object, handler http.Handler) (*userGrafanaReconciler, *httptest.Server) {
	dynamicClient := ctrlruntimefakeclient.
		NewClientBuilder().
		WithObjects(objects...).
		Build()
	ts := httptest.NewServer(handler)

	grafanaClient := grafanasdk.NewClient(ts.URL, "admin:admin", ts.Client())

	reconciler := userGrafanaReconciler{
		Client:        dynamicClient,
		grafanaClient: grafanaClient,
		log:           kubermaticlog.Logger,
		recorder:      record.NewFakeRecorder(10),
		grafanaHeader: "X-WEBAUTH-USER",
		grafanaURL:    ts.URL,
		httpClient:    ts.Client(),
		mlaEnabled:    true,
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
			name:        "UserProjectBinding added",
			requestName: "create",
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "create",
					},
					Spec: kubermaticv1.UserProjectBindingSpec{
						UserEmail: "user@email.com",
						ProjectID: "projectID",
						Group:     "owners-projectID",
					},
				},
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "projectID",
						Annotations: map[string]string{grafanaOrgAnnotationKey: "1"},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "projectName",
					},
				},
			},
			hasFinalizer: true,
			requests: []request{
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"id":1,"name":"projectName","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get org users",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1/users", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`[]`)), StatusCode: http.StatusOK},
				},
				{
					name:     "create OAuth user",
					request:  httptest.NewRequest(http.MethodGet, "/api/user", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"id":1}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete user from default org",
					request:  httptest.NewRequest(http.MethodDelete, "/api/orgs/1/users/1", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message":"org user deleted"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "add org user",
					request:  httptest.NewRequest(http.MethodPost, "/api/orgs/1/users", strings.NewReader(`{"loginOrEmail":"user@email.com","role":"Editor"}`)),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "User added to organization"}`)), StatusCode: http.StatusOK},
				},
			},
		},
		{
			name:        "UserProjectBinding role updated",
			requestName: "update",
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "update",
					},
					Spec: kubermaticv1.UserProjectBindingSpec{
						UserEmail: "user@email.com",
						ProjectID: "projectID",
						Group:     "owners-projectID",
					},
				},
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "projectID",
						Annotations: map[string]string{grafanaOrgAnnotationKey: "1"},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "projectName",
					},
				},
			},
			hasFinalizer: true,
			requests: []request{
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"id":1,"name":"projectName","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get org users",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1/users", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`[{"orgId":1,"userId":1,"email":"user@email.com","login":"admin","role":"Viewer"}]`)), StatusCode: http.StatusOK},
				},
				{
					name:     "update org user",
					request:  httptest.NewRequest(http.MethodPatch, "/api/orgs/1/users/1", strings.NewReader(`{"loginOrEmail":"user@email.com","role":"Editor"}`)),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "User updated"}`)), StatusCode: http.StatusOK},
				},
			},
		},
		{
			name:        "UserProjectBinding role delete",
			requestName: "delete",
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "delete",
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
					},
					Spec: kubermaticv1.UserProjectBindingSpec{
						UserEmail: "user@email.com",
						ProjectID: "projectID",
						Group:     "owners-projectID",
					},
				},
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "projectID",
						Annotations: map[string]string{grafanaOrgAnnotationKey: "1"},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "projectName",
					},
				},
			},
			hasFinalizer: false,
			requests: []request{
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"id":1,"name":"projectName","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get org users",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1/users", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`[{"orgId":1,"userId":1,"email":"user@email.com","login":"admin","role":"Viewer"}]`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete org user",
					request:  httptest.NewRequest(http.MethodDelete, "/api/orgs/1/users/1", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "User deleted"}`)), StatusCode: http.StatusOK},
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
			upb := &kubermaticv1.UserProjectBinding{}
			if err := controller.Get(ctx, request.NamespacedName, upb); err != nil {
				t.Fatalf("unable to get upb: %v", err)
			}
			assert.Equal(t, tc.hasFinalizer, kubernetes.HasFinalizer(upb, mlaFinalizer))
			assertExpectation()
			server.Close()
		})
	}
}
