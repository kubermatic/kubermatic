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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newTestDatasourceGrafanaReconciler(t *testing.T, objects []ctrlruntimeclient.Object, handler http.Handler) (*datasourceGrafanaReconciler, *httptest.Server) {
	dynamicClient := ctrlruntimefakeclient.
		NewClientBuilder().
		WithObjects(objects...).
		Build()
	ts := httptest.NewServer(handler)

	grafanaClient := grafanasdk.NewClient(ts.URL, "admin:admin", ts.Client())

	reconciler := datasourceGrafanaReconciler{
		Client:        dynamicClient,
		grafanaClient: grafanaClient,
		log:           kubermaticlog.Logger,
		recorder:      record.NewFakeRecorder(10),
	}
	return &reconciler, ts
}

func TestDatasourceGrafanaReconcile(t *testing.T) {
	testCases := []struct {
		name         string
		requestName  string
		objects      []ctrlruntimeclient.Object
		handlerFunc  http.HandlerFunc
		requests     []request
		err          bool
		hasFinalizer bool
		hasResources bool
	}{
		{
			name:         "create datasource for cluster",
			requestName:  "clusterUID",
			hasFinalizer: true,
			hasResources: true,
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "projectUID",
						Annotations: map[string]string{grafanaOrgAnnotationKey: "1"},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "projectName",
					},
				},
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: "projectUID"},
						Name:   "clusterUID",
					},
					Spec: kubermaticv1.ClusterSpec{
						HumanReadableName: "Super Cluster",
						MLA:               &kubermaticv1.MLASettings{LoggingEnabled: true, MonitoringEnabled: true},
					},
					Status: kubermaticv1.ClusterStatus{NamespaceName: "cluster-clusterUID"},
				},
			},
			requests: []request{
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-projectUID","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get datasource by uid",
					request:  httptest.NewRequest(http.MethodGet, "/api/datasources/uid/loki-clusterUID", nil),
					response: &http.Response{StatusCode: http.StatusNotFound},
				},
				{
					name:     "create loki datasource",
					request:  httptest.NewRequest(http.MethodPost, "/api/datasources", strings.NewReader(`{"name":"Loki Super Cluster", "orgId":1,  "type":"loki", "uid":"loki-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local", "access":"proxy", "id":0, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "datasource created", "id": 1}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get datasource by uid",
					request:  httptest.NewRequest(http.MethodGet, "/api/datasources/uid/prometheus-clusterUID", nil),
					response: &http.Response{StatusCode: http.StatusNotFound},
				},
				{
					name:     "create prometheus datasource",
					request:  httptest.NewRequest(http.MethodPost, "/api/datasources", strings.NewReader(`{"name":"Prometheus Super Cluster", "orgId":1,  "type":"prometheus", "uid":"prometheus-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":0, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "datasource created", "id": 2}`)), StatusCode: http.StatusOK},
				},
			},
		},
		{
			name:         "delete datasource",
			requestName:  "clusterUID",
			hasFinalizer: false,
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "projectUID",
						Annotations: map[string]string{grafanaOrgAnnotationKey: "1"},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "projectName",
					},
				},
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Labels:            map[string]string{kubermaticv1.ProjectIDLabelKey: "projectUID"},
						Name:              "clusterUID",
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
					},
					Spec: kubermaticv1.ClusterSpec{
						HumanReadableName: "Super Cluster",
						MLA:               &kubermaticv1.MLASettings{LoggingEnabled: true, MonitoringEnabled: true},
					},
					Status: kubermaticv1.ClusterStatus{NamespaceName: "cluster-clusterUID"},
				},
			},
			requests: []request{
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-projectUID","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "switch user context",
					request:  httptest.NewRequest(http.MethodPost, "/api/user/using/1", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message":"context switched"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete loki datasource",
					request:  httptest.NewRequest(http.MethodDelete, "/api/datasources/uid/loki-clusterUID", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete prometheus datasource",
					request:  httptest.NewRequest(http.MethodDelete, "/api/datasources/uid/prometheus-clusterUID", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
				},
			},
		},
		{
			name:         "update datasource for cluster",
			requestName:  "clusterUID",
			hasFinalizer: true,
			hasResources: true,
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "projectUID",
						Annotations: map[string]string{grafanaOrgAnnotationKey: "1"},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "projectName",
					},
				},
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: "projectUID"},
						Name:   "clusterUID",
					},
					Spec: kubermaticv1.ClusterSpec{
						HumanReadableName: "New Super Cluster",
						MLA:               &kubermaticv1.MLASettings{LoggingEnabled: true, MonitoringEnabled: true},
					},
					Status: kubermaticv1.ClusterStatus{NamespaceName: "cluster-clusterUID"},
				},
			},
			requests: []request{
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-projectUID","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get loki datasource by uid",
					request:  httptest.NewRequest(http.MethodGet, "/api/datasources/uid/loki-clusterUID", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"name":"Loki Super Cluster", "orgId":1,  "type":"loki", "uid":"loki-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local", "access":"proxy", "id":1, "isDefault":false, "jsonData":null, "secureJsonData":null}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "update loki datasource",
					request:  httptest.NewRequest(http.MethodPut, "/api/datasources/1", strings.NewReader(`{"name":"Loki New Super Cluster", "orgId":1,  "type":"loki", "uid":"loki-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local", "access":"proxy", "id":1, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "datasource updated", "id": 1}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get prometheus datasource by uid",
					request:  httptest.NewRequest(http.MethodGet, "/api/datasources/uid/prometheus-clusterUID", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"name":"Prometheus Super Cluster", "orgId":1,  "type":"prometheus", "uid":"prometheus-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":2, "isDefault":false, "jsonData":null, "secureJsonData":null}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "update prometheus datasource",
					request:  httptest.NewRequest(http.MethodPut, "/api/datasources/2", strings.NewReader(`{"name":"Prometheus New Super Cluster", "orgId":1,  "type":"prometheus", "uid":"prometheus-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":2, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "datasource updated", "id": 2}`)), StatusCode: http.StatusOK},
				},
			},
		},
		{
			name:         "MLA disabled for cluster",
			requestName:  "clusterUID",
			hasFinalizer: false,
			hasResources: false,
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "projectUID",
						Annotations: map[string]string{grafanaOrgAnnotationKey: "1"},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "projectName",
					},
				},
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: "projectUID"},
						Name:   "clusterUID",
					},
					Spec: kubermaticv1.ClusterSpec{
						HumanReadableName: "Super Cluster",
					},
					Status: kubermaticv1.ClusterStatus{NamespaceName: "cluster-clusterUID"},
				},
			},
			requests: []request{
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-projectUID","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "switch user context",
					request:  httptest.NewRequest(http.MethodPost, "/api/user/using/1", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message":"context switched"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete loki datasource",
					request:  httptest.NewRequest(http.MethodDelete, "/api/datasources/uid/loki-clusterUID", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete prometheus datasource",
					request:  httptest.NewRequest(http.MethodDelete, "/api/datasources/uid/prometheus-clusterUID", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
				},
			},
		},
		{
			name:         "MLA Logging disabled for cluster",
			requestName:  "clusterUID",
			hasFinalizer: true,
			hasResources: true,
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "projectUID",
						Annotations: map[string]string{grafanaOrgAnnotationKey: "1"},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "projectName",
					},
				},
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: "projectUID"},
						Name:   "clusterUID",
					},
					Spec: kubermaticv1.ClusterSpec{
						HumanReadableName: "Super Cluster",
						MLA: &kubermaticv1.MLASettings{
							MonitoringEnabled: true,
							LoggingEnabled:    false,
						},
					},
					Status: kubermaticv1.ClusterStatus{NamespaceName: "cluster-clusterUID"},
				},
			},
			requests: []request{
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-projectUID","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "switch user context",
					request:  httptest.NewRequest(http.MethodPost, "/api/user/using/1", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message":"context switched"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete loki datasource",
					request:  httptest.NewRequest(http.MethodDelete, "/api/datasources/uid/loki-clusterUID", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get datasource by uid",
					request:  httptest.NewRequest(http.MethodGet, "/api/datasources/uid/prometheus-clusterUID", nil),
					response: &http.Response{StatusCode: http.StatusNotFound},
				},
				{
					name:     "create prometheus datasource",
					request:  httptest.NewRequest(http.MethodPost, "/api/datasources", strings.NewReader(`{"name":"Prometheus Super Cluster", "orgId":1,  "type":"prometheus", "uid":"prometheus-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":0, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "datasource created", "id": 2}`)), StatusCode: http.StatusOK},
				},
			},
		},
		{
			name:         "MLA Monitoring disabled for cluster",
			requestName:  "clusterUID",
			hasFinalizer: true,
			hasResources: true,
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "projectUID",
						Annotations: map[string]string{grafanaOrgAnnotationKey: "1"},
					},
					Spec: kubermaticv1.ProjectSpec{
						Name: "projectName",
					},
				},
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: "projectUID"},
						Name:   "clusterUID",
					},
					Spec: kubermaticv1.ClusterSpec{
						HumanReadableName: "Super Cluster",
						MLA: &kubermaticv1.MLASettings{
							MonitoringEnabled: false,
							LoggingEnabled:    true,
						},
					},
					Status: kubermaticv1.ClusterStatus{NamespaceName: "cluster-clusterUID"},
				},
			},
			requests: []request{
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-projectUID","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "get datasource by uid",
					request:  httptest.NewRequest(http.MethodGet, "/api/datasources/uid/loki-clusterUID", nil),
					response: &http.Response{StatusCode: http.StatusNotFound},
				},
				{
					name:     "create loki datasource",
					request:  httptest.NewRequest(http.MethodPost, "/api/datasources", strings.NewReader(`{"name":"Loki Super Cluster", "orgId":1,  "type":"loki", "uid":"loki-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local", "access":"proxy", "id":0, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "datasource created", "id": 1}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "switch user context",
					request:  httptest.NewRequest(http.MethodPost, "/api/user/using/1", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message":"context switched"}`)), StatusCode: http.StatusOK},
				},
				{
					name:     "delete prometheus datasource",
					request:  httptest.NewRequest(http.MethodDelete, "/api/datasources/uid/prometheus-clusterUID", nil),
					response: &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
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
			controller, server := newTestDatasourceGrafanaReconciler(t, tc.objects, r)
			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			_, err := controller.Reconcile(ctx, request)
			if err != nil && !tc.err {
				assert.Nil(t, err)
			}
			assert.Equal(t, tc.err, err != nil)
			cluster := &kubermaticv1.Cluster{}
			if err := controller.Get(ctx, request.NamespacedName, cluster); err != nil {
				t.Fatalf("unable to get cluster: %v", err)
			}
			assert.Equal(t, tc.hasFinalizer, kubernetes.HasFinalizer(cluster, mlaFinalizer))
			if tc.hasResources {
				cm := &corev1.ConfigMap{}
				request := reconcile.Request{NamespacedName: types.NamespacedName{Name: "mla-gateway", Namespace: cluster.Status.NamespaceName}}
				if err := controller.Get(ctx, request.NamespacedName, cm); err != nil {
					t.Fatalf("unable to get configmap: %v", err)
				}
				dp := &appsv1.Deployment{}
				request = reconcile.Request{NamespacedName: types.NamespacedName{Name: "mla-gateway", Namespace: cluster.Status.NamespaceName}}
				if err := controller.Get(ctx, request.NamespacedName, dp); err != nil {
					t.Fatalf("unable to get deployment: %v", err)
				}

				svc := &corev1.Service{}
				request = reconcile.Request{NamespacedName: types.NamespacedName{Name: "mla-gateway-alert", Namespace: cluster.Status.NamespaceName}}
				if err := controller.Get(ctx, request.NamespacedName, svc); err != nil {
					t.Fatalf("unable to get alert service: %v", err)
				}

				request = reconcile.Request{NamespacedName: types.NamespacedName{Name: "mla-gateway-ext", Namespace: cluster.Status.NamespaceName}}
				if err := controller.Get(ctx, request.NamespacedName, svc); err != nil {
					t.Fatalf("unable to get external service: %v", err)
				}

			}
			assertExpectation()
			server.Close()
		})
	}
}
