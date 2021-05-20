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
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"id":1, "isGrafanaAdmin": false, "email": "user@email.com", "login": "user@email.com"}`)), StatusCode: http.StatusOK},
				},
			},
		},
		{
			name:        "User IsAdmin updated",
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
			},
			hasFinalizer: true,
			requests: []request{
				{
					name:     "create OAuth user",
					request:  httptest.NewRequest(http.MethodGet, "/api/user", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"id":1, "isGrafanaAdmin": false, "email": "user@email.com", "login": "user@email.com"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "update permissions",
					request:  httptest.NewRequest(http.MethodPut, "/api/admin/users/1/permissions", ioutil.NopCloser(strings.NewReader(`{"isGrafanaAdmin":true}`))),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "User permissions updated"}`)), StatusCode: http.StatusOK},
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
						Finalizers:        []string{mlaFinalizer},
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
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"id":1,"email":"user@email.com","login":"admin"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete user",
					request:  httptest.NewRequest(http.MethodDelete, "/api/admin/users/1", nil),
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
