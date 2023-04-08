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

package mlacontroller

import (
	"context"
	"fmt"
	"testing"
	"time"

	sdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/mla-controller/grafana"
	"k8c.io/kubermatic/v3/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v3/pkg/log"
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/kubermatic/v3/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v3/pkg/util/edition"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newTestGrafanaDatasourceReconciler(objects []ctrlruntimeclient.Object) (*grafanaDatasourceReconciler, *grafana.FakeGrafana) {
	dynamicClient := ctrlruntimefakeclient.
		NewClientBuilder().
		WithObjects(objects...).
		Build()

	gClient := grafana.NewFakeClient()

	// expect that the org reconciler has created the KKP org in Grafana
	if _, err := gClient.CreateOrg(context.Background(), sdk.Org{Name: GrafanaOrganization}); err != nil {
		panic(err)
	}

	return &grafanaDatasourceReconciler{
		seedClient:   dynamicClient,
		log:          kubermaticlog.Logger,
		recorder:     record.NewFakeRecorder(10),
		versions:     kubermatic.NewFakeVersions(edition.CommunityEdition),
		mlaNamespace: "mla",
		clientProvider: func(ctx context.Context) (grafana.Client, error) {
			return gClient, nil
		},
	}, gClient
}

func TestGrafanaDatasourceReconcile(t *testing.T) {
	ctx := context.Background()
	clusterName := "testcluster"

	testCases := []struct {
		name                    string
		objects                 []ctrlruntimeclient.Object
		err                     bool
		hasResources            bool
		expectExposeAnnotations map[string]string
		assertion               func(t *testing.T, cluster *kubermaticv1.Cluster, gClient *grafana.FakeGrafana, reconciler *grafanaDatasourceReconciler, reconcileErr error)
	}{
		{
			name:         "create datasources for cluster",
			hasResources: true,
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: clusterName,
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
			assertion: func(t *testing.T, cluster *kubermaticv1.Cluster, gClient *grafana.FakeGrafana, reconciler *grafanaDatasourceReconciler, reconcileErr error) {
				if reconcileErr != nil {
					t.Fatalf("Failed to reconcile: %v", reconcileErr)
				}

				if !kubernetes.HasFinalizer(cluster, datasourceCleanupFinalizer) {
					t.Error("Expected cluster to have MLA finalizer, but does not.")
				}
			},
			// requests: []request{
			// 	{
			// 		name:     "get org by id",
			// 		request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-projectUID","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "get datasource by uid",
			// 		request: &http.Request{
			// 			Method: http.MethodGet,
			// 			URL:    &url.URL{Path: "/api/datasources/uid/alertmanager-clusterUID"},
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{StatusCode: http.StatusNotFound},
			// 	},
			// 	{
			// 		name: "create alertmanager datasource",
			// 		request: &http.Request{
			// 			Method: http.MethodPost,
			// 			URL:    &url.URL{Path: "/api/datasources"},
			// 			Body:   io.NopCloser(strings.NewReader(`{"name":"Alertmanager Super Cluster", "orgId":1,  "type":"alertmanager", "uid":"alertmanager-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":0, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource created", "id": 1}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "get datasource by uid",
			// 		request: &http.Request{
			// 			Method: http.MethodGet,
			// 			URL:    &url.URL{Path: "/api/datasources/uid/loki-clusterUID"},
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{StatusCode: http.StatusNotFound},
			// 	},
			// 	{
			// 		name: "create loki datasource",
			// 		request: &http.Request{
			// 			Method: http.MethodPost,
			// 			URL:    &url.URL{Path: "/api/datasources"},
			// 			Body:   io.NopCloser(strings.NewReader(`{"name":"Loki Super Cluster", "orgId":1,  "type":"loki", "uid":"loki-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local", "access":"proxy", "id":0, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource created", "id": 2}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "get datasource by uid",
			// 		request: &http.Request{
			// 			Method: http.MethodGet,
			// 			URL:    &url.URL{Path: "/api/datasources/uid/prometheus-clusterUID"},
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{StatusCode: http.StatusNotFound},
			// 	},
			// 	{
			// 		name: "create prometheus datasource",
			// 		request: &http.Request{
			// 			Method: http.MethodPost,
			// 			URL:    &url.URL{Path: "/api/datasources"},
			// 			Body:   io.NopCloser(strings.NewReader(`{"name":"Prometheus Super Cluster", "orgId":1,  "type":"prometheus", "uid":"prometheus-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":0, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource created", "id": 3}`)), StatusCode: http.StatusOK},
			// 	},
			// },
		},
		{
			name:         "delete datasource",
			hasResources: false,
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:              clusterName,
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
			assertion: func(t *testing.T, cluster *kubermaticv1.Cluster, gClient *grafana.FakeGrafana, reconciler *grafanaDatasourceReconciler, reconcileErr error) {
				if reconcileErr != nil {
					t.Fatalf("Failed to reconcile: %v", reconcileErr)
				}

				if kubernetes.HasFinalizer(cluster, datasourceCleanupFinalizer) {
					t.Error("Expected cluster not to have MLA finalizer, but does.")
				}
			},
			// requests: []request{
			// 	{
			// 		name:     "get org by id",
			// 		request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-projectUID","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "delete alertmanager datasource",
			// 		request: &http.Request{
			// 			Method: http.MethodDelete,
			// 			URL:    &url.URL{Path: "/api/datasources/uid/alertmanager-clusterUID"},
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "delete loki datasource",
			// 		request: &http.Request{
			// 			Method: http.MethodDelete,
			// 			URL:    &url.URL{Path: "/api/datasources/uid/loki-clusterUID"},
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "delete prometheus datasource",
			// 		request: &http.Request{
			// 			Method: http.MethodDelete,
			// 			URL:    &url.URL{Path: "/api/datasources/uid/prometheus-clusterUID"},
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
			// 	},
			// },
		},
		{
			name:         "update datasource for cluster",
			hasResources: true,
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: clusterName,
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
			assertion: func(t *testing.T, cluster *kubermaticv1.Cluster, gClient *grafana.FakeGrafana, reconciler *grafanaDatasourceReconciler, reconcileErr error) {
				if reconcileErr != nil {
					t.Fatalf("Failed to reconcile: %v", reconcileErr)
				}

				if !kubernetes.HasFinalizer(cluster, datasourceCleanupFinalizer) {
					t.Error("Expected cluster to have MLA finalizer, but does not.")
				}
			},
			// requests: []request{
			// 	{
			// 		name:     "get org by id",
			// 		request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-projectUID","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "get alertmanager datasource by uid",
			// 		request: &http.Request{
			// 			Method: http.MethodGet,
			// 			URL:    &url.URL{Path: "/api/datasources/uid/alertmanager-clusterUID"},
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"name":"Alertmanager Super Cluster", "orgId":1,  "type":"alertmanager", "uid":"alertmanager-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":1, "isDefault":false, "jsonData":null, "secureJsonData":null}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "update alertmanager datasource",
			// 		request: &http.Request{
			// 			Method: http.MethodPut,
			// 			URL:    &url.URL{Path: "/api/datasources/1"},
			// 			Body:   io.NopCloser(strings.NewReader(`{"name":"Alertmanager New Super Cluster", "orgId":1,  "type":"alertmanager", "uid":"alertmanager-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":1, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource updated", "id": 1}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "get loki datasource by uid",
			// 		request: &http.Request{
			// 			Method: http.MethodGet,
			// 			URL:    &url.URL{Path: "/api/datasources/uid/loki-clusterUID"},
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"name":"Loki Super Cluster", "orgId":1,  "type":"loki", "uid":"loki-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local", "access":"proxy", "id":2, "isDefault":false, "jsonData":null, "secureJsonData":null}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "update loki datasource",
			// 		request: &http.Request{
			// 			Method: http.MethodPut,
			// 			URL:    &url.URL{Path: "/api/datasources/2"},
			// 			Body:   io.NopCloser(strings.NewReader(`{"name":"Loki New Super Cluster", "orgId":1,  "type":"loki", "uid":"loki-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local", "access":"proxy", "id":2, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource updated", "id": 2}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "get prometheus datasource by uid",
			// 		request: &http.Request{
			// 			Method: http.MethodGet,
			// 			URL:    &url.URL{Path: "/api/datasources/uid/prometheus-clusterUID"},
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"name":"Prometheus Super Cluster", "orgId":1,  "type":"prometheus", "uid":"prometheus-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":3, "isDefault":false, "jsonData":null, "secureJsonData":null}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "update prometheus datasource",
			// 		request: &http.Request{
			// 			Method: http.MethodPut,
			// 			URL:    &url.URL{Path: "/api/datasources/3"},
			// 			Body:   io.NopCloser(strings.NewReader(`{"name":"Prometheus New Super Cluster", "orgId":1,  "type":"prometheus", "uid":"prometheus-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":3, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource updated", "id": 3}`)), StatusCode: http.StatusOK},
			// 	},
			// },
		},
		{
			name:         "Project removed before cluster",
			hasResources: false,
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: clusterName,
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
			assertion: func(t *testing.T, cluster *kubermaticv1.Cluster, gClient *grafana.FakeGrafana, reconciler *grafanaDatasourceReconciler, reconcileErr error) {
				if reconcileErr != nil {
					t.Fatalf("Failed to reconcile: %v", reconcileErr)
				}
			},
		},
		{
			name:         "MLA disabled for cluster",
			hasResources: false,
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: clusterName,
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
			assertion: func(t *testing.T, cluster *kubermaticv1.Cluster, gClient *grafana.FakeGrafana, reconciler *grafanaDatasourceReconciler, reconcileErr error) {
				if reconcileErr != nil {
					t.Fatalf("Failed to reconcile: %v", reconcileErr)
				}

				if kubernetes.HasFinalizer(cluster, datasourceCleanupFinalizer) {
					t.Error("Expected cluster not to have MLA finalizer, but does.")
				}
			},
			// requests: []request{
			// 	{
			// 		name:     "get org by id",
			// 		request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-projectUID","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "delete alertmanager datasource",
			// 		request: &http.Request{
			// 			Method: http.MethodDelete,
			// 			URL:    &url.URL{Path: "/api/datasources/uid/alertmanager-clusterUID"},
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "delete loki datasource",
			// 		request: &http.Request{
			// 			Method: http.MethodDelete,
			// 			URL:    &url.URL{Path: "/api/datasources/uid/loki-clusterUID"},
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "delete prometheus datasource",
			// 		request: &http.Request{
			// 			Method: http.MethodDelete,
			// 			URL:    &url.URL{Path: "/api/datasources/uid/prometheus-clusterUID"},
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
			// 	},
			// },
		},
		{
			name:         "MLA Logging disabled for cluster",
			hasResources: true,
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: clusterName,
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
			assertion: func(t *testing.T, cluster *kubermaticv1.Cluster, gClient *grafana.FakeGrafana, reconciler *grafanaDatasourceReconciler, reconcileErr error) {
				if reconcileErr != nil {
					t.Fatalf("Failed to reconcile: %v", reconcileErr)
				}

				if !kubernetes.HasFinalizer(cluster, datasourceCleanupFinalizer) {
					t.Error("Expected cluster to have MLA finalizer, but does not.")
				}
			},
			// requests: []request{
			// 	{
			// 		name:     "get org by id",
			// 		request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-projectUID","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "get datasource by uid",
			// 		request: &http.Request{
			// 			Method: http.MethodGet,
			// 			URL:    &url.URL{Path: "/api/datasources/uid/alertmanager-clusterUID"},
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{StatusCode: http.StatusNotFound},
			// 	},
			// 	{
			// 		name: "create alertmanager datasource",
			// 		request: &http.Request{
			// 			Method: http.MethodPost,
			// 			URL:    &url.URL{Path: "/api/datasources"},
			// 			Body:   io.NopCloser(strings.NewReader(`{"name":"Alertmanager Super Cluster", "orgId":1,  "type":"alertmanager", "uid":"alertmanager-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":0, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource created", "id": 1}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "delete loki datasource",
			// 		request: &http.Request{
			// 			Method: http.MethodDelete,
			// 			URL:    &url.URL{Path: "/api/datasources/uid/loki-clusterUID"},
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "get datasource by uid",
			// 		request: &http.Request{
			// 			Method: http.MethodGet,
			// 			URL:    &url.URL{Path: "/api/datasources/uid/prometheus-clusterUID"},
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{StatusCode: http.StatusNotFound},
			// 	},
			// 	{
			// 		name: "create prometheus datasource",
			// 		request: &http.Request{
			// 			Method: http.MethodPost,
			// 			URL:    &url.URL{Path: "/api/datasources"},
			// 			Body:   io.NopCloser(strings.NewReader(`{"name":"Prometheus Super Cluster", "orgId":1,  "type":"prometheus", "uid":"prometheus-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/api/prom", "access":"proxy", "id":0, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource created", "id": 2}`)), StatusCode: http.StatusOK},
			// 	},
			// },
		},
		{
			name:         "MLA Monitoring disabled for cluster",
			hasResources: true,
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: clusterName,
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
			assertion: func(t *testing.T, cluster *kubermaticv1.Cluster, gClient *grafana.FakeGrafana, reconciler *grafanaDatasourceReconciler, reconcileErr error) {
				if reconcileErr != nil {
					t.Fatalf("Failed to reconcile: %v", reconcileErr)
				}

				if !kubernetes.HasFinalizer(cluster, datasourceCleanupFinalizer) {
					t.Error("Expected cluster to have MLA finalizer, but does not.")
				}
			},
			// requests: []request{
			// 	{
			// 		name:     "get org by id",
			// 		request:  httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil),
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":1,"name":"projectName-projectUID","address":{"address1":"","address2":"","city":"","zipCode":"","state":"","country":""}}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "get datasource by uid",
			// 		request: &http.Request{
			// 			Method: http.MethodGet,
			// 			URL:    &url.URL{Path: "/api/datasources/uid/alertmanager-clusterUID"},
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{StatusCode: http.StatusNotFound},
			// 	},
			// 	{
			// 		name: "create alertmanager datasource",
			// 		request: &http.Request{
			// 			Method: http.MethodPost,
			// 			URL:    &url.URL{Path: "/api/datasources"},
			// 			Body:   io.NopCloser(strings.NewReader(`{"name":"Alertmanager Super Cluster", "orgId":1,  "type":"alertmanager", "uid":"alertmanager-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local/prom/api", "access":"proxy", "id":0, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource created", "id": 1}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "get datasource by uid",
			// 		request: &http.Request{
			// 			Method: http.MethodGet,
			// 			URL:    &url.URL{Path: "/api/datasources/uid/loki-clusterUID"},
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{StatusCode: http.StatusNotFound},
			// 	},
			// 	{
			// 		name: "create loki datasource",
			// 		request: &http.Request{
			// 			Method: http.MethodPost,
			// 			URL:    &url.URL{Path: "/api/datasources"},
			// 			Body:   io.NopCloser(strings.NewReader(`{"name":"Loki Super Cluster", "orgId":1,  "type":"loki", "uid":"loki-clusterUID", "url":"http://mla-gateway.cluster-clusterUID.svc.cluster.local", "access":"proxy", "id":0, "isDefault":false, "jsonData":null, "secureJsonData":null}`)),
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource created", "id": 2}`)), StatusCode: http.StatusOK},
			// 	},
			// 	{
			// 		name: "delete prometheus datasource",
			// 		request: &http.Request{
			// 			Method: http.MethodDelete,
			// 			URL:    &url.URL{Path: "/api/datasources/uid/prometheus-clusterUID"},
			// 			Header: map[string][]string{"X-Grafana-Org-Id": {"1"}},
			// 		},
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "datasource deleted"}`)), StatusCode: http.StatusOK},
			// 	},
			// },
		},
	}
	for idx := range testCases {
		tc := testCases[idx]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reconciler, gClient := newTestGrafanaDatasourceReconciler(tc.objects)

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: clusterName}}
			_, reconcileErr := reconciler.Reconcile(ctx, request)

			cluster := &kubermaticv1.Cluster{}
			if err := reconciler.seedClient.Get(ctx, request.NamespacedName, cluster); err != nil {
				t.Fatalf("Failed to get cluster: %v", err)
			}

			tc.assertion(t, cluster, gClient, reconciler, reconcileErr)

			assertResource(t, ctx, reconciler.seedClient, tc.hasResources, cluster.Status.NamespaceName, &corev1.ConfigMap{}, gatewayName)
			assertResource(t, ctx, reconciler.seedClient, tc.hasResources, cluster.Status.NamespaceName, &appsv1.Deployment{}, gatewayName)
			assertResource(t, ctx, reconciler.seedClient, tc.hasResources, cluster.Status.NamespaceName, &corev1.Service{}, gatewayName)
			assertResource(t, ctx, reconciler.seedClient, tc.hasResources, cluster.Status.NamespaceName, &corev1.Service{}, gatewayExternalName)
			assertResource(t, ctx, reconciler.seedClient, tc.hasResources, cluster.Status.NamespaceName, &corev1.Secret{}, resources.MLAGatewayCASecretName)
			assertResource(t, ctx, reconciler.seedClient, tc.hasResources, cluster.Status.NamespaceName, &corev1.Secret{}, resources.MLAGatewayCertificatesSecretName)

			if tc.hasResources {
				svc := &corev1.Service{}
				if err := reconciler.seedClient.Get(ctx, types.NamespacedName{Name: gatewayExternalName, Namespace: cluster.Status.NamespaceName}, svc); err != nil {
					t.Fatalf("Failed to get gateway Service: %v", err)
				}

				annotations := svc.GetAnnotations()
				for k, v := range tc.expectExposeAnnotations {
					svcValue := annotations[k]
					if svcValue != v {
						t.Errorf("Expected %s Service to have %s=%s annotation, but does not (actual value: %q).", gatewayExternalName, k, v, svcValue)
					}
				}
			}
		})
	}
}

func assertResource(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, expectToExist bool, namespace string, obj ctrlruntimeclient.Object, name string) {
	obj.SetName(name)
	obj.SetNamespace(namespace)

	err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(obj), obj)
	if expectToExist {
		if err != nil {
			t.Fatalf("Expected resource %v to exist, but does not.", obj)
		}
	} else {
		if !apierrors.IsNotFound(err) {
			t.Fatalf("Expected NotFound error, but got: %v", err)
		}
	}
}
