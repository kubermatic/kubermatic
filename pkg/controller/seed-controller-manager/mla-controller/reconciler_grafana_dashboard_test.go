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
	"net/http"
	"testing"
	"time"

	"go.uber.org/zap"

	grafanasdk "github.com/kubermatic/grafanasdk"
	sdk "github.com/kubermatic/grafanasdk"
	"k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/mla-controller/grafana"
	"k8c.io/kubermatic/v3/pkg/kubernetes"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newTestGrafanaDashboardReconciler(objects []ctrlruntimeclient.Object) (*grafanaDashboardReconciler, *grafana.FakeGrafana) {
	dynamicClient := ctrlruntimefakeclient.
		NewClientBuilder().
		WithObjects(objects...).
		Build()

	gClient := grafana.NewFakeClient()

	// expect that the org reconciler has created the KKP org in Grafana
	if _, err := gClient.CreateOrg(context.Background(), sdk.Org{Name: GrafanaOrganization}); err != nil {
		panic(err)
	}

	return &grafanaDashboardReconciler{
		seedClient:   dynamicClient,
		log:          zap.NewNop().Sugar(),
		recorder:     record.NewFakeRecorder(10),
		mlaNamespace: "mla",
		clientProvider: func(ctx context.Context) (grafana.Client, error) {
			return gClient, nil
		},
	}, gClient
}

func TestGrafanaDashboardReconcile(t *testing.T) {
	ctx := context.Background()
	configMapName := grafanaDashboardsConfigmapNamePrefix + "-defaults"

	var board struct {
		Dashboard grafanasdk.Board `json:"dashboard"`
		FolderID  int              `json:"folderId"`
		Overwrite bool             `json:"overwrite"`
	}

	board.Overwrite = true
	board.Dashboard.Title = "dashboard"
	board.Dashboard.UID = "unique"

	testCases := []struct {
		name        string
		objects     []ctrlruntimeclient.Object
		handlerFunc http.HandlerFunc
		assertion   func(t *testing.T, configMap *corev1.ConfigMap, gClient *grafana.FakeGrafana, reconciler *grafanaDashboardReconciler, reconcileErr error)
	}{
		{
			name: "add configmap with dashboards",
			objects: []ctrlruntimeclient.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      configMapName,
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
			assertion: func(t *testing.T, configMap *corev1.ConfigMap, gClient *grafana.FakeGrafana, reconciler *grafanaDashboardReconciler, reconcileErr error) {
				if reconcileErr != nil {
					t.Fatalf("Failed to reconcile: %v", reconcileErr)
				}

				if !kubernetes.HasFinalizer(configMap, mlaFinalizer) {
					t.Error("Expected ConfigMap to have MLA finalizer, but does not.")
				}
			},
			// requests: []request{
			// 	{
			// 		name:     "set dashboard",
			// 		request:  httptest.NewRequest(http.MethodPost, "/api/dashboards/db", strings.NewReader(string(boardData))),
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "dashboard set"}`)), StatusCode: http.StatusOK},
			// 	},
			// },
		},
		{
			name: "add configmap with dashboards, but project not ready yep",
			objects: []ctrlruntimeclient.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      configMapName,
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
			assertion: func(t *testing.T, configMap *corev1.ConfigMap, gClient *grafana.FakeGrafana, reconciler *grafanaDashboardReconciler, reconcileErr error) {
				if reconcileErr != nil {
					t.Fatalf("Failed to reconcile: %v", reconcileErr)
				}

				if !kubernetes.HasFinalizer(configMap, mlaFinalizer) {
					t.Error("Expected ConfigMap to have MLA finalizer, but does not.")
				}
			},
		},
		{
			name: "delete configmap with dashboards",
			objects: []ctrlruntimeclient.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:              configMapName,
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
			assertion: func(t *testing.T, configMap *corev1.ConfigMap, gClient *grafana.FakeGrafana, reconciler *grafanaDashboardReconciler, reconcileErr error) {
				if reconcileErr != nil {
					t.Fatalf("Failed to reconcile: %v", reconcileErr)
				}

				if kubernetes.HasFinalizer(configMap, mlaFinalizer) {
					t.Error("Expected ConfigMap not to have MLA finalizer, but does.")
				}
			},
			// requests: []request{
			// 	{
			// 		name:     "delete dashboard",
			// 		request:  httptest.NewRequest(http.MethodDelete, "/api/dashboards/uid/"+"unique", nil),
			// 		response: &http.Response{Body: io.NopCloser(strings.NewReader(`{"message": "Dashboard dashboard deleted"}`)), StatusCode: http.StatusOK},
			// 	},
			// },
		},
	}

	for idx := range testCases {
		tc := testCases[idx]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reconciler, gClient := newTestGrafanaDashboardReconciler(tc.objects)

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: configMapName, Namespace: "mla"}}
			_, reconcileErr := reconciler.Reconcile(ctx, request)

			configMap := &corev1.ConfigMap{}
			if err := reconciler.seedClient.Get(ctx, request.NamespacedName, configMap); err != nil {
				t.Fatalf("unable to get configMap: %v", err)
			}

			tc.assertion(t, configMap, gClient, reconciler, reconcileErr)
		})
	}
}
