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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newTestDatasourceGrafanaReconciler(t *testing.T, objects []ctrlruntimeclient.Object, handler http.Handler) (*datasourceGrafanaReconciler, *httptest.Server) {
	dynamicClient := fake.
		NewClientBuilder().
		WithObjects(objects...).
		Build()
	ts := httptest.NewServer(handler)

	grafanaClient, err := grafanasdk.NewClient(ts.URL, "admin:admin", ts.Client())
	if err != nil {
		t.Fatalf("unable to initialize grafana client: %v", err)
	}

	datasourceGrafanaController := newDatasourceGrafanaController(dynamicClient, func(ctx context.Context) (*grafanasdk.Client, error) {
		return grafanaClient, nil
	}, "mla", kubermaticlog.Logger, "")
	reconciler := datasourceGrafanaReconciler{
		Client:                      dynamicClient,
		log:                         kubermaticlog.Logger,
		recorder:                    record.NewFakeRecorder(10),
		datasourceGrafanaController: datasourceGrafanaController,
	}
	return &reconciler, ts
}

func TestDatasourceGrafanaReconcile(t *testing.T) {
	testCases := []struct {
		name                    string
		requestName             string
		objects                 []ctrlruntimeclient.Object
		handlerFunc             http.HandlerFunc
		requests                []request
		err                     bool
		hasFinalizer            bool
		hasResources            bool
		expectExposeAnnotations map[string]string
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
						Annotations: map[string]string{GrafanaOrgAnnotationKey: "1"},
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
						ExposeStrategy:    kubermaticv1.ExposeStrategyNodePort,
					},
					Status: kubermaticv1.ClusterStatus{
						NamespaceName: "cluster-clusterUID",
						Address: kubermaticv1.ClusterAddress{
							ExternalName: "abcd.test.kubermatic.io",
						},
					},
				},
			},
			expectExposeAnnotations: map[string]string{
				nodeportproxy.DefaultExposeAnnotationKey: nodeportproxy.NodePortType.String(),
			},
			requests: []request{
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-projectUID","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name: "get datasource by uid",
					request: &http.Request{
						Method: http.MethodGet,
						URL:    &url.URL{Path: "/api/datasources/uid/alertmanager-clusterUID"},
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{StatusCode: http.StatusNotFound},
				},
				{
					name: "create alertmanager datasource",
					request: &http.Request{
						Method: http.MethodPost,
						URL:    &url.URL{Path: "/api/datasources"},
						Body:   io.NopCloser(strings.NewReader(`{"name":"Alertmanager Super Cluster", "orgId":1,  "type":"alertmanager", "uid":"alertmanager-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":0, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource created", "id": 1}`)), StatusCode: http.StatusOK},
				},
				{
					name: "get datasource by uid",
					request: &http.Request{
						Method: http.MethodGet,
						URL:    &url.URL{Path: "/api/datasources/uid/loki-clusterUID"},
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{StatusCode: http.StatusNotFound},
				},
				{
					name: "create loki datasource",
					request: &http.Request{
						Method: http.MethodPost,
						URL:    &url.URL{Path: "/api/datasources"},
						Body:   io.NopCloser(strings.NewReader(`{"name":"Loki Super Cluster", "orgId":1,  "type":"loki", "uid":"loki-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local", "access":"proxy", "id":0, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource created", "id": 2}`)), StatusCode: http.StatusOK},
				},
				{
					name: "get datasource by uid",
					request: &http.Request{
						Method: http.MethodGet,
						URL:    &url.URL{Path: "/api/datasources/uid/prometheus-clusterUID"},
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{StatusCode: http.StatusNotFound},
				},
				{
					name: "create prometheus datasource",
					request: &http.Request{
						Method: http.MethodPost,
						URL:    &url.URL{Path: "/api/datasources"},
						Body:   io.NopCloser(strings.NewReader(`{"name":"Prometheus Super Cluster", "orgId":1,  "type":"prometheus", "uid":"prometheus-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":0, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource created", "id": 3}`)), StatusCode: http.StatusOK},
				},
			},
		},
		{
			name:         "delete datasource",
			requestName:  "clusterUID",
			hasFinalizer: false,
			hasResources: false,
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "projectUID",
						Annotations: map[string]string{GrafanaOrgAnnotationKey: "1"},
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
						Finalizers:        []string{"just-a-test-do-not-delete-thanks"},
					},
					Spec: kubermaticv1.ClusterSpec{
						HumanReadableName: "Super Cluster",
						MLA:               &kubermaticv1.MLASettings{LoggingEnabled: true, MonitoringEnabled: true},
						ExposeStrategy:    kubermaticv1.ExposeStrategyNodePort,
					},
					Status: kubermaticv1.ClusterStatus{
						NamespaceName: "cluster-clusterUID",
						Address: kubermaticv1.ClusterAddress{
							ExternalName: "abcd.test.kubermatic.io",
						},
					},
				},
			},
			requests: []request{
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-projectUID","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name: "delete alertmanager datasource",
					request: &http.Request{
						Method: http.MethodDelete,
						URL:    &url.URL{Path: "/api/datasources/uid/alertmanager-clusterUID"},
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
				},
				{
					name: "delete loki datasource",
					request: &http.Request{
						Method: http.MethodDelete,
						URL:    &url.URL{Path: "/api/datasources/uid/loki-clusterUID"},
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
				},
				{
					name: "delete prometheus datasource",
					request: &http.Request{
						Method: http.MethodDelete,
						URL:    &url.URL{Path: "/api/datasources/uid/prometheus-clusterUID"},
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
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
						Annotations: map[string]string{GrafanaOrgAnnotationKey: "1"},
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
						ExposeStrategy:    kubermaticv1.ExposeStrategyLoadBalancer,
					},
					Status: kubermaticv1.ClusterStatus{
						NamespaceName: "cluster-clusterUID",
						Address: kubermaticv1.ClusterAddress{
							ExternalName: "abcd.test.kubermatic.io",
						},
					},
				},
			},
			expectExposeAnnotations: map[string]string{
				nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey: "true",
			},
			requests: []request{
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-projectUID","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name: "get alertmanager datasource by uid",
					request: &http.Request{
						Method: http.MethodGet,
						URL:    &url.URL{Path: "/api/datasources/uid/alertmanager-clusterUID"},
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"name":"Alertmanager Super Cluster", "orgId":1,  "type":"alertmanager", "uid":"alertmanager-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":1, "isDefault":false, "jsonData":null, "secureJsonData":null}`)), StatusCode: http.StatusOK},
				},
				{
					name: "update alertmanager datasource",
					request: &http.Request{
						Method: http.MethodPut,
						URL:    &url.URL{Path: "/api/datasources/1"},
						Body:   io.NopCloser(strings.NewReader(`{"name":"Alertmanager New Super Cluster", "orgId":1,  "type":"alertmanager", "uid":"alertmanager-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":1, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource updated", "id": 1}`)), StatusCode: http.StatusOK},
				},
				{
					name: "get loki datasource by uid",
					request: &http.Request{
						Method: http.MethodGet,
						URL:    &url.URL{Path: "/api/datasources/uid/loki-clusterUID"},
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"name":"Loki Super Cluster", "orgId":1,  "type":"loki", "uid":"loki-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local", "access":"proxy", "id":2, "isDefault":false, "jsonData":null, "secureJsonData":null}`)), StatusCode: http.StatusOK},
				},
				{
					name: "update loki datasource",
					request: &http.Request{
						Method: http.MethodPut,
						URL:    &url.URL{Path: "/api/datasources/2"},
						Body:   io.NopCloser(strings.NewReader(`{"name":"Loki New Super Cluster", "orgId":1,  "type":"loki", "uid":"loki-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local", "access":"proxy", "id":2, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource updated", "id": 2}`)), StatusCode: http.StatusOK},
				},
				{
					name: "get prometheus datasource by uid",
					request: &http.Request{
						Method: http.MethodGet,
						URL:    &url.URL{Path: "/api/datasources/uid/prometheus-clusterUID"},
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"name":"Prometheus Super Cluster", "orgId":1,  "type":"prometheus", "uid":"prometheus-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":3, "isDefault":false, "jsonData":null, "secureJsonData":null}`)), StatusCode: http.StatusOK},
				},
				{
					name: "update prometheus datasource",
					request: &http.Request{
						Method: http.MethodPut,
						URL:    &url.URL{Path: "/api/datasources/3"},
						Body:   io.NopCloser(strings.NewReader(`{"name":"Prometheus New Super Cluster", "orgId":1,  "type":"prometheus", "uid":"prometheus-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":3, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource updated", "id": 3}`)), StatusCode: http.StatusOK},
				},
			},
		},
		{
			name:         "Project removed before cluster",
			requestName:  "clusterUID",
			hasFinalizer: false,
			hasResources: false,
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: "projectUID"},
						Name:   "clusterUID",
					},
					Spec: kubermaticv1.ClusterSpec{
						HumanReadableName: "Super Cluster",
					},
					Status: kubermaticv1.ClusterStatus{
						NamespaceName: "cluster-clusterUID",
						Address: kubermaticv1.ClusterAddress{
							ExternalName: "abcd.test.kubermatic.io",
						},
					},
				},
			},
			requests: []request{},
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
						Annotations: map[string]string{GrafanaOrgAnnotationKey: "1"},
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
					Status: kubermaticv1.ClusterStatus{
						NamespaceName: "cluster-clusterUID",
						Address: kubermaticv1.ClusterAddress{
							ExternalName: "abcd.test.kubermatic.io",
						},
					},
				},
			},
			requests: []request{
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-projectUID","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name: "delete alertmanager datasource",
					request: &http.Request{
						Method: http.MethodDelete,
						URL:    &url.URL{Path: "/api/datasources/uid/alertmanager-clusterUID"},
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
				},
				{
					name: "delete loki datasource",
					request: &http.Request{
						Method: http.MethodDelete,
						URL:    &url.URL{Path: "/api/datasources/uid/loki-clusterUID"},
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
				},
				{
					name: "delete prometheus datasource",
					request: &http.Request{
						Method: http.MethodDelete,
						URL:    &url.URL{Path: "/api/datasources/uid/prometheus-clusterUID"},
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
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
						Annotations: map[string]string{GrafanaOrgAnnotationKey: "1"},
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
						ExposeStrategy: kubermaticv1.ExposeStrategyTunneling,
					},
					Status: kubermaticv1.ClusterStatus{
						NamespaceName: "cluster-clusterUID",
						Address: kubermaticv1.ClusterAddress{
							ExternalName: "abcd.test.kubermatic.io",
						},
					},
				},
			},
			expectExposeAnnotations: map[string]string{
				nodeportproxy.DefaultExposeAnnotationKey:   nodeportproxy.SNIType.String(),
				nodeportproxy.PortHostMappingAnnotationKey: fmt.Sprintf(`{%q: %q}`, extPortName, resources.MLAGatewaySNIPrefix+"abcd.test.kubermatic.io"),
			},
			requests: []request{
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-projectUID","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name: "get datasource by uid",
					request: &http.Request{
						Method: http.MethodGet,
						URL:    &url.URL{Path: "/api/datasources/uid/alertmanager-clusterUID"},
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{StatusCode: http.StatusNotFound},
				},
				{
					name: "create alertmanager datasource",
					request: &http.Request{
						Method: http.MethodPost,
						URL:    &url.URL{Path: "/api/datasources"},
						Body:   io.NopCloser(strings.NewReader(`{"name":"Alertmanager Super Cluster", "orgId":1,  "type":"alertmanager", "uid":"alertmanager-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":0, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource created", "id": 1}`)), StatusCode: http.StatusOK},
				},
				{
					name: "delete loki datasource",
					request: &http.Request{
						Method: http.MethodDelete,
						URL:    &url.URL{Path: "/api/datasources/uid/loki-clusterUID"},
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
				},
				{
					name: "get datasource by uid",
					request: &http.Request{
						Method: http.MethodGet,
						URL:    &url.URL{Path: "/api/datasources/uid/prometheus-clusterUID"},
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{StatusCode: http.StatusNotFound},
				},
				{
					name: "create prometheus datasource",
					request: &http.Request{
						Method: http.MethodPost,
						URL:    &url.URL{Path: "/api/datasources"},
						Body:   io.NopCloser(strings.NewReader(`{"name":"Prometheus Super Cluster", "orgId":1,  "type":"prometheus", "uid":"prometheus-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":0, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource created", "id": 2}`)), StatusCode: http.StatusOK},
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
						Annotations: map[string]string{GrafanaOrgAnnotationKey: "1"},
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
						ExposeStrategy: kubermaticv1.ExposeStrategyNodePort,
					},
					Status: kubermaticv1.ClusterStatus{
						NamespaceName: "cluster-clusterUID",
						Address: kubermaticv1.ClusterAddress{
							ExternalName: "abcd.test.kubermatic.io",
						},
					},
				},
			},
			expectExposeAnnotations: map[string]string{
				nodeportproxy.DefaultExposeAnnotationKey: nodeportproxy.NodePortType.String(),
			},
			requests: []request{
				{
					name:     "get org by id",
					request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-projectUID","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
				},
				{
					name: "get datasource by uid",
					request: &http.Request{
						Method: http.MethodGet,
						URL:    &url.URL{Path: "/api/datasources/uid/alertmanager-clusterUID"},
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{StatusCode: http.StatusNotFound},
				},
				{
					name: "create alertmanager datasource",
					request: &http.Request{
						Method: http.MethodPost,
						URL:    &url.URL{Path: "/api/datasources"},
						Body:   io.NopCloser(strings.NewReader(`{"name":"Alertmanager Super Cluster", "orgId":1,  "type":"alertmanager", "uid":"alertmanager-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/prom/api", "access":"proxy", "id":0, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource created", "id": 1}`)), StatusCode: http.StatusOK},
				},
				{
					name: "get datasource by uid",
					request: &http.Request{
						Method: http.MethodGet,
						URL:    &url.URL{Path: "/api/datasources/uid/loki-clusterUID"},
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{StatusCode: http.StatusNotFound},
				},
				{
					name: "create loki datasource",
					request: &http.Request{
						Method: http.MethodPost,
						URL:    &url.URL{Path: "/api/datasources"},
						Body:   io.NopCloser(strings.NewReader(`{"name":"Loki Super Cluster", "orgId":1,  "type":"loki", "uid":"loki-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local", "access":"proxy", "id":0, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource created", "id": 2}`)), StatusCode: http.StatusOK},
				},
				{
					name: "delete prometheus datasource",
					request: &http.Request{
						Method: http.MethodDelete,
						URL:    &url.URL{Path: "/api/datasources/uid/prometheus-clusterUID"},
						Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
					},
					response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
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

			cm := &corev1.ConfigMap{}
			request = reconcile.Request{NamespacedName: types.NamespacedName{Name: gatewayName, Namespace: cluster.Status.NamespaceName}}
			err = controller.Get(ctx, request.NamespacedName, cm)
			if tc.hasResources {
				assert.Nil(t, err)
			} else {
				assert.True(t, apierrors.IsNotFound(err))
			}

			dp := &appsv1.Deployment{}
			request = reconcile.Request{NamespacedName: types.NamespacedName{Name: gatewayName, Namespace: cluster.Status.NamespaceName}}
			err = controller.Get(ctx, request.NamespacedName, dp)
			if tc.hasResources {
				assert.Nil(t, err)
			} else {
				assert.True(t, apierrors.IsNotFound(err))
			}

			svc := &corev1.Service{}
			request = reconcile.Request{NamespacedName: types.NamespacedName{Name: gatewayName, Namespace: cluster.Status.NamespaceName}}
			err = controller.Get(ctx, request.NamespacedName, svc)
			if tc.hasResources {
				assert.Nil(t, err)
			} else {
				assert.True(t, apierrors.IsNotFound(err))
			}
			request = reconcile.Request{NamespacedName: types.NamespacedName{Name: gatewayExternalName, Namespace: cluster.Status.NamespaceName}}
			err = controller.Get(ctx, request.NamespacedName, svc)
			if tc.hasResources {
				assert.Nil(t, err)
				assert.EqualValues(t, tc.expectExposeAnnotations, svc.Annotations)
			} else {
				assert.True(t, apierrors.IsNotFound(err))
			}

			secret := &corev1.Secret{}
			request = reconcile.Request{NamespacedName: types.NamespacedName{Name: resources.MLAGatewayCASecretName, Namespace: cluster.Status.NamespaceName}}
			err = controller.Get(ctx, request.NamespacedName, secret)
			if tc.hasResources {
				assert.Nil(t, err)
			} else {
				assert.True(t, apierrors.IsNotFound(err))
			}
			request = reconcile.Request{NamespacedName: types.NamespacedName{Name: resources.MLAGatewayCertificatesSecretName, Namespace: cluster.Status.NamespaceName}}
			err = controller.Get(ctx, request.NamespacedName, secret)
			if tc.hasResources {
				assert.Nil(t, err)
			} else {
				assert.True(t, apierrors.IsNotFound(err))
			}

			assertExpectation()
			server.Close()
		})
	}
}
