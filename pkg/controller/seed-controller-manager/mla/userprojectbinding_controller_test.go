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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	grafanasdk "github.com/kubermatic/grafanasdk"
	"github.com/stretchr/testify/assert"
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

func newTestUserProjectBindingReconciler(t *testing.T, objects []ctrlruntimeclient.Object, handler http.Handler) (*userProjectBindingReconciler, *httptest.Server) {
	dynamicClient := ctrlruntimefakeclient.
		NewClientBuilder().
		WithObjects(objects...).
		Build()
	ts := httptest.NewServer(handler)

	grafanaClient := grafanasdk.NewClient(ts.URL, "admin:admin", ts.Client())

	reconciler := userProjectBindingReconciler{
		Client:        dynamicClient,
		grafanaClient: grafanaClient,
		log:           kubermaticlog.Logger,
		recorder:      record.NewFakeRecorder(10),
		grafanaHeader: "X-WEBAUTH-USER",
		grafanaURL:    ts.URL,
		httpClient:    ts.Client(),
	}
	return &reconciler, ts
}

func TestUserProjectBindingReconcile(t *testing.T) {
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
					name:     "delete user from default org",
					code:     200,
					method:   "DELETE",
					path:     "/api/orgs/1/users/1",
					response: `{"message":"org user deleted"}`,
				},
				{
					name:     "add org user",
					code:     200,
					method:   "POST",
					path:     "/api/orgs/1/users",
					response: `{"message": "User added to organization"}`,
					body:     `{"loginOrEmail":"user@email.com","role":"Admin"}`,
				},
				{
					name:     "get org users",
					code:     200,
					method:   "GET",
					path:     "/api/orgs/1/users",
					response: `[]`,
				},
				{
					name:     "get org by id",
					code:     200,
					method:   "GET",
					path:     "/api/orgs/1",
					response: `{"id":1,"name":"projectName","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`,
				},
				{
					name:     "create OAuth user",
					code:     200,
					method:   "GET",
					path:     "/api/user",
					response: `{"id":1}`,
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
					name:     "get org users",
					code:     200,
					method:   "GET",
					path:     "/api/orgs/1/users",
					response: `[{"orgId":1,"userId":1,"email":"user@email.com","login":"admin","role":"Viewer"}]`,
				},
				{
					name:     "get org by id",
					code:     200,
					method:   "GET",
					path:     "/api/orgs/1",
					response: `{"id":1,"name":"projectName","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`,
				},
				{
					name:     "update org user",
					code:     200,
					method:   "PATCH",
					path:     "/api/orgs/1/users/1",
					response: `{"message": "User updated"}`,
					body:     `{"loginOrEmail":"user@email.com","role":"Admin"}`,
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
					name:     "get org users",
					code:     200,
					method:   "GET",
					path:     "/api/orgs/1/users",
					response: `[{"orgId":1,"userId":1,"email":"user@email.com","login":"admin","role":"Viewer"}]`,
				},
				{
					name:     "get org by id",
					code:     200,
					method:   "GET",
					path:     "/api/orgs/1",
					response: `{"id":1,"name":"projectName","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`,
				},
				{
					name:     "delete org user",
					code:     200,
					method:   "DELETE",
					path:     "/api/orgs/1/users/1",
					response: `{"message": "User deleted"}`,
				},
			},
		},
	}
	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			r, assertExpectation := buildTestServer(t, tc.requests)
			controller, server := newTestUserProjectBindingReconciler(t, tc.objects, r)
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
