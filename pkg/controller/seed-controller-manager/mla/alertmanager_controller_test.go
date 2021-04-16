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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var testScheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(testScheme))
	utilruntime.Must(kubermaticv1.AddToScheme(testScheme))
}

func newTestAlertmanagerReconciler(objects []ctrlruntimeclient.Object, handler http.Handler) (*alertmanagerReconciler, *httptest.Server) {
	fakeClient := ctrlruntimefakeclient.
		NewClientBuilder().
		WithObjects(objects...).
		WithScheme(testScheme).
		Build()
	ts := httptest.NewServer(handler)

	reconciler := alertmanagerReconciler{
		Client:              fakeClient,
		httpClient:          ts.Client(),
		log:                 kubermaticlog.Logger,
		recorder:            record.NewFakeRecorder(10),
		mlaGatewayURLGetter: newFakeMLAGatewayURLGetter(*ts),
	}
	return &reconciler, ts
}

func TestAlertmanagerReconcile(t *testing.T) {
	testCases := []struct {
		name         string
		requestName  string
		objects      []ctrlruntimeclient.Object
		requests     []request
		expectedErr  bool
		hasFinalizer bool
		cleanup      bool
	}{
		{
			name:        "create default alertmanager configuration when no config secret is created",
			requestName: "cluster-1",
			objects: []ctrlruntimeclient.Object{
				generateCluster("cluster-1", true, false),
			},
			requests: []request{
				{
					name: "cluster-1",
					request: httptest.NewRequest(http.MethodPost,
						alertmanagerConfigEndpoint,
						bytes.NewBuffer([]byte(defaultConfig))),
					response: &http.Response{StatusCode: http.StatusCreated},
				},
			},
			hasFinalizer: true,
		},
		{
			name:        "update alertmanager configuration based on the config secret",
			requestName: "cluster-2",
			objects: []ctrlruntimeclient.Object{
				generateCluster("cluster-2", true, false),
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config-secret",
						Namespace: "cluster-cluster-2",
					},
					Data: map[string][]byte{
						alertmanagerConfigSecretKey: []byte(generateAlertmanagerConfig("test-user")),
					},
				},
				&kubermaticv1.Alertmanager{
					ObjectMeta: metav1.ObjectMeta{
						Name:      alertmanagerName,
						Namespace: "cluster-cluster-2",
					},
					Spec: kubermaticv1.AlertmanagerSpec{
						ConfigSecret: corev1.LocalObjectReference{
							Name: "config-secret",
						},
					},
				},
			},
			requests: []request{
				{
					name: "update",
					request: httptest.NewRequest(http.MethodPost,
						alertmanagerConfigEndpoint,
						bytes.NewBuffer([]byte(generateAlertmanagerConfig("test-user")))),
					response: &http.Response{StatusCode: http.StatusCreated},
				},
			},
			hasFinalizer: true,
		},
		{
			name:        "clean up alertmanager configuration when monitoring is disabled",
			requestName: "cluster-3",
			objects: []ctrlruntimeclient.Object{
				generateCluster("cluster-3", false, false),
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config-secret",
						Namespace: "cluster-cluster-3",
					},
					Data: map[string][]byte{
						alertmanagerConfigSecretKey: []byte(generateAlertmanagerConfig("test-user")),
					},
				},
				&kubermaticv1.Alertmanager{
					ObjectMeta: metav1.ObjectMeta{
						Name:      alertmanagerName,
						Namespace: "cluster-cluster-3",
					},
					Spec: kubermaticv1.AlertmanagerSpec{
						ConfigSecret: corev1.LocalObjectReference{
							Name: "config-secret",
						},
					},
				},
			},
			requests: []request{
				{
					name: "clean-up",
					request: httptest.NewRequest(http.MethodDelete,
						alertmanagerConfigEndpoint,
						nil),
					response: &http.Response{StatusCode: http.StatusOK},
				},
			},
			hasFinalizer: false,
			cleanup:      true,
		},
		{
			name:        "clean up alertmanager configuration when cluster is removed",
			requestName: "cluster-4",
			objects: []ctrlruntimeclient.Object{
				generateCluster("cluster-4", false, true),
			},
			requests: []request{
				{
					name: "clean-up",
					request: httptest.NewRequest(http.MethodDelete,
						alertmanagerConfigEndpoint,
						nil),
					response: &http.Response{StatusCode: http.StatusOK},
				},
			},
			hasFinalizer: false,
			cleanup:      true,
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			r, assertExpectation := buildTestServer(t, testcase.requests...)
			reconciler, server := newTestAlertmanagerReconciler(testcase.objects, r)
			request := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: testcase.requestName,
				},
			}
			_, err := reconciler.Reconcile(ctx, request)
			assert.Equal(t, testcase.expectedErr, err != nil)
			cluster := &kubermaticv1.Cluster{}
			if err := reconciler.Get(ctx, request.NamespacedName, cluster); err != nil {
				t.Fatalf("unable to get cluster: %v", err)
			}
			assert.Equal(t, testcase.hasFinalizer, kubernetes.HasFinalizer(cluster, alertmanagerFinalizer))

			alertmanager := &kubermaticv1.Alertmanager{}
			err = reconciler.Get(ctx, types.NamespacedName{
				Name:      alertmanagerName,
				Namespace: cluster.Status.NamespaceName,
			}, alertmanager)
			if testcase.cleanup {
				assert.True(t, errors.IsNotFound(err))
				secret := &corev1.Secret{}
				err = reconciler.Get(ctx, types.NamespacedName{
					Name:      "config-secret",
					Namespace: cluster.Status.NamespaceName,
				}, secret)
				assert.True(t, errors.IsNotFound(err))
			} else {
				assert.Nil(t, err)
			}
			assertExpectation()
			server.Close()
		})
	}
}

func generateCluster(name string, monitoringEnabled, deleted bool) *kubermaticv1.Cluster {
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.ClusterSpec{
			HumanReadableName: name,
			MLA:               &kubermaticv1.MLASettings{MonitoringEnabled: monitoringEnabled},
		},
		Status: kubermaticv1.ClusterStatus{NamespaceName: fmt.Sprintf("cluster-%s", name)},
	}
	if deleted {
		deleteTime := metav1.NewTime(time.Now())
		cluster.DeletionTimestamp = &deleteTime
	}
	return cluster
}

func generateAlertmanagerConfig(name string) string {
	return fmt.Sprintf(`
alertmanager_config: |
  global:
    smtp_smarthost: 'localhost:25'
    smtp_from: '%s@example.org'
  route:
    receiver: "test"
  receivers:
    - name: "test"
      email_configs:
      - to: '%s@example.org'
`, name, name)
}

type fakeMLAGatewayURLGetter struct {
	httptest.Server
}

func newFakeMLAGatewayURLGetter(server httptest.Server) *fakeMLAGatewayURLGetter {
	return &fakeMLAGatewayURLGetter{
		Server: server,
	}
}

func (f *fakeMLAGatewayURLGetter) mlaGatewayURL(cluster *kubermaticv1.Cluster) string {
	return f.URL
}
