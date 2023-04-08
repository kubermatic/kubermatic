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
	"testing"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/mla-controller/grafana"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newTestGrafanaOrgReconciler(objects []ctrlruntimeclient.Object) (*grafanaOrgReconciler, *grafana.FakeGrafana) {
	dynamicClient := ctrlruntimefakeclient.
		NewClientBuilder().
		WithObjects(objects...).
		Build()

	gClient := grafana.NewFakeClient()

	return &grafanaOrgReconciler{
		seedClient:   dynamicClient,
		log:          zap.NewNop().Sugar(),
		recorder:     record.NewFakeRecorder(999),
		mlaNamespace: "mla",
		clientProvider: func(ctx context.Context) (grafana.Client, error) {
			return gClient, nil
		},
	}, gClient
}

func TestGrafanaOrgReconcile(t *testing.T) {
	clusterName := "testcluster"
	ctx := context.Background()

	testCases := []struct {
		name      string
		objects   []ctrlruntimeclient.Object
		assertion func(t *testing.T, gClient *grafana.FakeGrafana, reconciler *grafanaOrgReconciler, reconcileErr error)
	}{
		{
			name: "create org for cluster",
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: clusterName,
					},
				},
			},
			assertion: func(t *testing.T, gClient *grafana.FakeGrafana, reconciler *grafanaOrgReconciler, reconcileErr error) {
				if reconcileErr != nil {
					t.Fatalf("Failed to reconcile: %v", reconcileErr)
				}

				org, err := gClient.GetOrgByOrgName(ctx, GrafanaOrganization)
				if err != nil {
					t.Fatalf("Failed to get Grafana org: %v", err)
				}
				if org.ID == 0 {
					t.Fatal("Org should have a positive ID.")
				}
				if org.Name != GrafanaOrganization {
					t.Fatalf("Org should be called %q, but is called %q.", GrafanaOrganization, org.Name)
				}
			},
		},
		{
			name: "create org with dashboards",
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: clusterName,
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      grafanaDashboardsConfigmapNamePrefix + "-defaults",
						Namespace: "mla",
					},
					Data: map[string]string{"first": `{"title": "dashboard", "uid":"unique"}`},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "wrong-prefix-defaults",
						Namespace: "mla",
					},
					Data: map[string]string{"second": "not dashboard data"},
				},
			},
			assertion: func(t *testing.T, gClient *grafana.FakeGrafana, reconciler *grafanaOrgReconciler, reconcileErr error) {
				if reconcileErr != nil {
					t.Fatalf("Failed to reconcile: %v", reconcileErr)
				}

				org, err := gClient.GetOrgByOrgName(ctx, GrafanaOrganization)
				if err != nil {
					t.Fatalf("Failed to get Grafana org: %v", err)
				}

				orgDashboards, ok := gClient.Database.Dashboards[org.ID]
				if !ok {
					t.Fatal("Organization has no dashboards.")
				}

				dashboard, ok := orgDashboards["unique"]
				if !ok {
					t.Fatal("unique dashboard does not exist.")
				}

				if dashboard.Title != "dashboard" {
					t.Fatalf("Expected 'dashboard' as the title, but got %q.", dashboard.Title)
				}
			},
		},
		{
			name: "create org - org already exists",
			objects: []ctrlruntimeclient.Object{&kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterName,
				},
			}},
			assertion: func(t *testing.T, gClient *grafana.FakeGrafana, reconciler *grafanaOrgReconciler, reconcileErr error) {
				if reconcileErr != nil {
					t.Fatalf("Failed to reconcile: %v", reconcileErr)
				}

				// reconcile again
				request := reconcile.Request{NamespacedName: types.NamespacedName{Name: clusterName}}
				_, reconcileErr = reconciler.Reconcile(ctx, request)

				if reconcileErr != nil {
					t.Fatalf("Failed to reconcile a second time: %v", reconcileErr)
				}
			},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reconciler, gClient := newTestGrafanaOrgReconciler(tc.objects)

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: clusterName}}
			_, reconcileErr := reconciler.Reconcile(ctx, request)

			tc.assertion(t, gClient, reconciler, reconcileErr)
		})
	}
}
