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
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newTestDashboardGrafanaReconciler(t *testing.T, objects []ctrlruntimeclient.Object, handler http.Handler) (*dashboardGrafanaReconciler, *httptest.Server) {
	dynamicClient := fake.
		NewClientBuilder().
		WithObjects(objects...).
		Build()
	ts := httptest.NewServer(handler)

	grafanaClient, err := grafanasdk.NewClient(ts.URL, "admin:admin", ts.Client())
	assert.Nil(t, err)

	dashboardGrafanaController := newDashboardGrafanaController(dynamicClient, kubermaticlog.Logger, "mla", func(ctx context.Context) (*grafanasdk.Client, error) {
		return grafanaClient, nil
	})
	reconciler := dashboardGrafanaReconciler{
		Client:                     dynamicClient,
		log:                        kubermaticlog.Logger,
		recorder:                   record.NewFakeRecorder(10),
		dashboardGrafanaController: dashboardGrafanaController,
	}
	return &reconciler, ts
}

func TestDashboardGrafanaReconcile(t *testing.T) {
	var board struct {
		Dashboard grafanasdk.Board `json:"dashboard"`
		FolderID  int              `json:"folderId"`
		Overwrite bool             `json:"overwrite"`
	}

	board.Overwrite = true
	board.Dashboard.Title = "dashboard"
	board.Dashboard.UID = "unique"
	boardData, err := json.Marshal(board)
	assert.Nil(t, err)
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
			name:        "add configmap with dashboards",
			requestName: grafanaDashboardsConfigmapNamePrefix + "-defaults",
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "create",
						Annotations: map[string]string{GrafanaOrgAnnotationKey: "1"},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "projectName",
					},
				},

				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      grafanaDashboardsConfigmapNamePrefix + "-defaults",
						Namespace: "mla",
					},
					Data: map[string]string{"first": `{"title": "dashboard", "uid": "unique"}`},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "wrong-prefix-defaults",
						Namespace: "mla",
					},
					Data: map[string]string{"second": "not dashboard data"},
				},
			},
			hasFinalizer: true,
			requests: []request{
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-create","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "set dashboard",
					request:  httptest.NewRequest(http.MethodPost, "/api/dashboards/db", strings.NewReader(string(boardData))),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "dashboard set"}`)), StatusCode: http.StatusOK},
				},
			},
		},
		{
			name:        "add configmap with dashboards, but project not ready yep",
			requestName: grafanaDashboardsConfigmapNamePrefix + "-defaults",
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "create",
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "projectName",
					},
				},

				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      grafanaDashboardsConfigmapNamePrefix + "-defaults",
						Namespace: "mla",
					},
					Data: map[string]string{"first": `{"title": "dashboard", "uid": "unique"}`},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "wrong-prefix-defaults",
						Namespace: "mla",
					},
					Data: map[string]string{"second": "not dashboard data"},
				},
			},
			hasFinalizer: true,
			requests:     []request{},
		},
		{
			name:        "delete configmap with dashboards",
			requestName: grafanaDashboardsConfigmapNamePrefix + "-defaults",
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "create",
						Annotations: map[string]string{GrafanaOrgAnnotationKey: "1"},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "projectName",
					},
				},
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "skip",
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "skipProjectName",
					},
				},

				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:              grafanaDashboardsConfigmapNamePrefix + "-defaults",
						Namespace:         "mla",
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
						Finalizers:        []string{mlaFinalizer, "do-not-remove"},
					},
					Data: map[string]string{"first": `{"title": "dashboard", "uid": "unique"}`},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "wrong-prefix-defaults",
						Namespace: "mla",
					},
					Data: map[string]string{"second": "not dashboard data"},
				},
			},
			hasFinalizer: false,
			requests: []request{
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-create","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete dashboard",
					request:  httptest.NewRequest(http.MethodDelete, "/api/dashboards/uid/"+"unique", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "Dashboard dashboard deleted"}`)), StatusCode: http.StatusOK},
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
			controller, server := newTestDashboardGrafanaReconciler(t, tc.objects, r)
			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName, Namespace: "mla"}}
			_, err := controller.Reconcile(ctx, request)
			if err != nil && !tc.err {
				assert.Nil(t, err)
			}
			assert.Equal(t, tc.err, err != nil)
			configMap := &corev1.ConfigMap{}
			if err := controller.Get(ctx, request.NamespacedName, configMap); err != nil {
				t.Fatalf("unable to get configMap: %v", err)
			}
			assert.Equal(t, tc.hasFinalizer, kubernetes.HasFinalizer(configMap, mlaFinalizer))
			assertExpectation()
			server.Close()
		})
	}
}
