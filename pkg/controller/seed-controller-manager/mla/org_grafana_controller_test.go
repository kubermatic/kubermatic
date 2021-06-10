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
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

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

type request struct {
	name     string
	request  *http.Request
	response *http.Response
}

func newTestOrgGrafanaReconciler(t *testing.T, objects []ctrlruntimeclient.Object, handler http.Handler) (*orgGrafanaReconciler, *httptest.Server) {
	dynamicClient := ctrlruntimefakeclient.
		NewClientBuilder().
		WithObjects(objects...).
		Build()
	ts := httptest.NewServer(handler)

	grafanaClient := grafanasdk.NewClient(ts.URL, "admin:admin", ts.Client())

	orgGrafanaController := newOrgGrafanaController(dynamicClient, kubermaticlog.Logger, grafanaClient)
	reconciler := orgGrafanaReconciler{
		Client:               dynamicClient,
		log:                  kubermaticlog.Logger,
		recorder:             record.NewFakeRecorder(10),
		orgGrafanaController: orgGrafanaController,
	}
	return &reconciler, ts
}

func TestOrgGrafanaReconcile(t *testing.T) {
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
			name:        "create org for project",
			requestName: "create",
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "create",
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "projectName",
					},
				},
			},
			hasFinalizer: true,
			requests: []request{
				{
					name:     "create",
					request:  httptest.NewRequest(http.MethodPost, "/api/orgs", strings.NewReader(`{"id":0,"name":"projectName-create","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "org created", "OrgID": 1}`)), StatusCode: http.StatusOK},
				},
			},
		},
		{
			name:        "create org for project with admins",
			requestName: "create",
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "create",
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "projectName",
					},
				},
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
					name:     "create",
					request:  httptest.NewRequest(http.MethodPost, "/api/orgs", strings.NewReader(`{"id":0,"name":"projectName-create","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "org created", "OrgID": 1}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "lookup user",
					request:  httptest.NewRequest(http.MethodGet, "/api/users/lookup?loginOrEmail=user@email.com", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"id":1,"email":"user@email.com","login":"admin"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get org users",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1/users", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`[]`)), StatusCode: http.StatusOK},
				},
				{
					name:     "add org user",
					request:  httptest.NewRequest(http.MethodPost, "/api/orgs/1/users", strings.NewReader(`{"loginOrEmail":"user@email.com","role":"Admin"}`)),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "User added to organization"}`)), StatusCode: http.StatusOK},
				},
			},
		},
		{
			name:        "create org for project - org already exists",
			requestName: "create",
			objects: []ctrlruntimeclient.Object{&kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "create",
					Annotations: map[string]string{grafanaOrgAnnotationKey: "1"},
				},
				Spec: kubermaticv1.ProjectSpec{
					Name: "projectName",
				},
			}},
			requests: []request{
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{StatusCode: http.StatusNotFound},
				},
				{
					name:     "create",
					request:  httptest.NewRequest(http.MethodPost, "/api/orgs", strings.NewReader(`{"id":0,"name":"projectName-create","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "name already taken"}`)), StatusCode: http.StatusConflict},
				},
				{
					name:     "get org by name",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/name/projectName-create", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-create","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
			},
			hasFinalizer: true,
			err:          false,
		},
		{
			name:        "update org for project",
			requestName: "create",
			objects: []ctrlruntimeclient.Object{&kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "create",
					Annotations: map[string]string{grafanaOrgAnnotationKey: "1"},
				},
				Spec: kubermaticv1.ProjectSpec{
					Name: "projectName",
				},
			}},
			requests: []request{
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-oldname","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "update",
					request:  httptest.NewRequest(http.MethodPut, "/api/orgs/1", strings.NewReader(`{"id":1,"name":"projectName-create","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "name already taken"}`)), StatusCode: http.StatusOK},
				},
			},
			hasFinalizer: true,
			err:          false,
		},
		{
			name:        "delete org for project",
			requestName: "delete",
			objects: []ctrlruntimeclient.Object{&kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "delete",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Annotations:       map[string]string{grafanaOrgAnnotationKey: "1"},
				},
				Spec: kubermaticv1.ProjectSpec{
					Name: "projectName",
				},
			}},
			hasFinalizer: false,
			requests: []request{
				{
					name:     "delete org by id",
					request:  httptest.NewRequest(http.MethodDelete, "/api/orgs/1", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{}`)), StatusCode: http.StatusOK},
				},
			},
		},
		{
			name:        "delete org for project without annotation",
			requestName: "delete",
			objects: []ctrlruntimeclient.Object{&kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "delete",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{mlaFinalizer},
				},
				Spec: kubermaticv1.ProjectSpec{
					Name: "projectName",
				},
			}},
			hasFinalizer: false,
			err:          false,
			requests:     []request{},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			r, assertExpectation := buildTestServer(t, tc.requests...)
			controller, server := newTestOrgGrafanaReconciler(t, tc.objects, r)
			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			_, err := controller.Reconcile(ctx, request)
			if err != nil && !tc.err {
				assert.Nil(t, err)
			}
			assert.Equal(t, tc.err, err != nil)
			project := &kubermaticv1.Project{}
			if err := controller.Get(ctx, request.NamespacedName, project); err != nil {
				t.Fatalf("unable to get project: %v", err)
			}
			assert.Equal(t, tc.hasFinalizer, kubernetes.HasFinalizer(project, mlaFinalizer))
			assertExpectation()
			server.Close()
		})
	}

}

func buildTestServer(t *testing.T, requests ...request) (http.Handler, func() bool) {
	var counter int64 = -1
	assertExpectation := func() bool {
		return assert.Equal(t, len(requests), int(counter+1))
	}
	r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := atomic.AddInt64(&counter, 1)
		if int(c) >= len(requests) {
			assert.Failf(t, "unexpected request", "%v", r)
		}
		req := requests[c]
		t.Logf("checking request: %s", req.name)
		for key := range req.request.Header {
			assert.Equal(t, req.request.Header.Get(key), r.Header.Get(key), "header %s not found", key)
		}
		assert.Equal(t, req.request.URL.Path, r.URL.Path)
		assert.Equal(t, req.request.URL.Query(), r.URL.Query())
		assert.Equal(t, req.request.Method, r.Method)
		if req.request.ContentLength > 0 {
			assert.True(t, bodyEqual(t, req.request, r))
		}
		w.WriteHeader(req.response.StatusCode)
		if req.response.Body != nil {
			defer req.response.Body.Close()
			_, err := io.Copy(w, req.response.Body)
			assert.Nil(t, err)
		}
	})
	return r, assertExpectation
}

func bodyEqual(t *testing.T, expectedRequest, request *http.Request) bool {
	defer expectedRequest.Body.Close()
	defer request.Body.Close()
	expectedBody, err := ioutil.ReadAll(expectedRequest.Body)
	assert.Nil(t, err)
	body, err := ioutil.ReadAll(request.Body)
	assert.Nil(t, err)
	if bytes.Equal(expectedBody, body) {
		return true
	}
	if jsonEqual(expectedBody, body) || yamlEqual(expectedBody, body) {
		return true
	}
	assert.Fail(t, "body not equal", cmp.Diff(string(expectedBody), string(body)))
	return false
}

func jsonEqual(expectedBody, body []byte) bool {
	expectedBodyMap := map[string]interface{}{}
	bodyMap := map[string]interface{}{}
	if err := json.Unmarshal(expectedBody, &expectedBodyMap); err != nil {
		return false
	}
	if err := json.Unmarshal(body, &bodyMap); err != nil {
		return false
	}
	return reflect.DeepEqual(expectedBodyMap, bodyMap)
}

func yamlEqual(expectedBody, body []byte) bool {
	expectedBodyMap := map[string]interface{}{}
	bodyMap := map[string]interface{}{}
	if err := yaml.Unmarshal(expectedBody, &expectedBodyMap); err != nil {
		return false
	}
	if err := yaml.Unmarshal(body, &bodyMap); err != nil {
		return false
	}
	return reflect.DeepEqual(expectedBodyMap, bodyMap)
}
