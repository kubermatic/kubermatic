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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newTestAlertmanagerReconciler(objects []ctrlruntimeclient.Object, handler http.Handler) (*alertmanagerReconciler, *httptest.Server) {
	fakeClient := fake.
		NewClientBuilder().
		WithObjects(objects...).
		Build()
	ts := httptest.NewServer(handler)

	alertmanagerController := newAlertmanagerController(fakeClient, kubermaticlog.Logger, ts.Client(), ts.URL)
	reconciler := alertmanagerReconciler{
		Client:                 fakeClient,
		log:                    kubermaticlog.Logger,
		recorder:               record.NewFakeRecorder(10),
		alertmanagerController: alertmanagerController,
	}
	return &reconciler, ts
}

type alertmanagerConfigStatus struct {
	clusterStatus      kubermaticv1.HealthStatus                    // Alertmanager config status in the Cluster CR
	alertmanagerStatus kubermaticv1.AlertmanagerConfigurationStatus // Alertmanager config status in the Alertmanager CR
}

// getAlertmanagerConfigStatusUp returns the needed information when the alertmanager config status is OK:
// - Cluster CR: HealthStatusUp
// - Alertmanager CR:
//   - Status True
//   - No Error message
//   - LastUpdated to now
func getAlertmanagerConfigStatusUp() alertmanagerConfigStatus {
	return alertmanagerConfigStatus{
		clusterStatus: kubermaticv1.HealthStatusUp,
		alertmanagerStatus: kubermaticv1.AlertmanagerConfigurationStatus{
			Status:      corev1.ConditionTrue,
			LastUpdated: metav1.Now(),
		},
	}
}

// getAlertmanagerConfigStatusDown returns the needed information when the alertmanager config status is KO:
// - Cluster CR: HealthStatusDown
// - Alertmanager CR:
//   - Status False
//   - Error message
//   - No LastUpdated
func getAlertmanagerConfigStatusDown() alertmanagerConfigStatus {
	return alertmanagerConfigStatus{
		clusterStatus: kubermaticv1.HealthStatusDown,
		alertmanagerStatus: kubermaticv1.AlertmanagerConfigurationStatus{
			Status:       corev1.ConditionFalse,
			ErrorMessage: "status code: 400, response body: \"error validating Alertmanager config: some explanation\"",
		},
	}
}

func TestAlertmanagerReconcile(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name                             string
		requestName                      string
		objects                          []ctrlruntimeclient.Object
		requests                         []request
		expectedErr                      bool
		hasFinalizer                     bool
		hasResources                     bool
		expectedAlertmanagerConfigStatus alertmanagerConfigStatus
	}{
		{
			name:        "create default alertmanager configuration when no alertmanager is created",
			requestName: "test",
			objects: []ctrlruntimeclient.Object{
				generateCluster("test", true, false, false),
			},
			requests: []request{
				{
					name:     "get",
					request:  httptest.NewRequest(http.MethodGet, AlertmanagerConfigEndpoint, nil),
					response: &http.Response{StatusCode: http.StatusNotFound},
				},
				{
					name: "post",
					request: httptest.NewRequest(http.MethodPost,
						AlertmanagerConfigEndpoint,
						bytes.NewBuffer([]byte(resources.DefaultAlertmanagerConfig))),
					response: &http.Response{StatusCode: http.StatusCreated},
				},
			},
			hasFinalizer:                     true,
			hasResources:                     true,
			expectedAlertmanagerConfigStatus: getAlertmanagerConfigStatusUp(),
		},
		{
			name:        "create alertmanager configuration based on the config secret failure, check health status",
			requestName: "test",
			objects: []ctrlruntimeclient.Object{
				generateCluster("test", false, true, false),
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config-secret",
						Namespace: "cluster-test",
					},
					Data: map[string][]byte{
						resources.AlertmanagerConfigSecretKey: []byte(generateAlertmanagerConfig("test-user")),
					},
				},
				&kubermaticv1.Alertmanager{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resources.AlertmanagerName,
						Namespace: "cluster-test",
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
					name:     "get",
					request:  httptest.NewRequest(http.MethodGet, AlertmanagerConfigEndpoint, nil),
					response: &http.Response{StatusCode: http.StatusNotFound},
				},
				{
					name: "post",
					request: httptest.NewRequest(http.MethodPost,
						AlertmanagerConfigEndpoint,
						bytes.NewBuffer([]byte(generateAlertmanagerConfig("test-user")))),
					response: &http.Response{
						StatusCode: http.StatusBadRequest, // any StatusCode != http.StatusCreated
						Body:       io.NopCloser(strings.NewReader(`"error validating Alertmanager config: some explanation"`))},
				},
			},
			hasFinalizer:                     true,
			hasResources:                     true,
			expectedErr:                      true,
			expectedAlertmanagerConfigStatus: getAlertmanagerConfigStatusDown(),
		},
		{
			name:        "clean up alertmanager configuration when mla is disabled",
			requestName: "test",
			objects: []ctrlruntimeclient.Object{
				generateCluster("test", false, false, false),
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config-secret",
						Namespace: "cluster-test",
					},
					Data: map[string][]byte{
						resources.AlertmanagerConfigSecretKey: []byte(generateAlertmanagerConfig("test-user")),
					},
				},
				&kubermaticv1.Alertmanager{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resources.AlertmanagerName,
						Namespace: "cluster-test",
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
					name: "delete",
					request: httptest.NewRequest(http.MethodDelete,
						AlertmanagerConfigEndpoint,
						nil),
					response: &http.Response{StatusCode: http.StatusOK},
				},
			},
			hasFinalizer: false,
			hasResources: false,
		},
		{
			name:        "clean up alertmanager configuration when mla is disabled - delete configuration failed",
			requestName: "test",
			objects: []ctrlruntimeclient.Object{
				generateCluster("test", false, false, false),
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config-secret",
						Namespace: "cluster-test",
					},
					Data: map[string][]byte{
						resources.AlertmanagerConfigSecretKey: []byte(generateAlertmanagerConfig("test-user")),
					},
				},
				&kubermaticv1.Alertmanager{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resources.AlertmanagerName,
						Namespace: "cluster-test",
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
					name: "delete",
					request: httptest.NewRequest(http.MethodDelete,
						AlertmanagerConfigEndpoint,
						nil),
					response: &http.Response{
						StatusCode: http.StatusBadRequest, // any StatusCode != http.StatusOK
						Body:       io.NopCloser(strings.NewReader(`"error validating Alertmanager config: some explanation"`))},
				},
			},
			hasFinalizer:                     false,
			hasResources:                     true,
			expectedErr:                      true,
			expectedAlertmanagerConfigStatus: getAlertmanagerConfigStatusDown(),
		},
		{
			name:        "clean up alertmanager configuration when cluster is removed",
			requestName: "test",
			objects: []ctrlruntimeclient.Object{
				generateCluster("test", false, true, true),
			},
			requests: []request{
				{
					name: "delete",
					request: httptest.NewRequest(http.MethodDelete,
						AlertmanagerConfigEndpoint,
						nil),
					response: &http.Response{StatusCode: http.StatusOK},
				},
			},
			hasFinalizer: false,
			hasResources: false,
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.name, func(t *testing.T) {
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
				Name:      resources.AlertmanagerName,
				Namespace: cluster.Status.NamespaceName,
			}, alertmanager)
			if testcase.hasResources {
				assert.Nil(t, err)
				secret := &corev1.Secret{}
				err = reconciler.Get(ctx, types.NamespacedName{
					Name:      alertmanager.Spec.ConfigSecret.Name,
					Namespace: cluster.Status.NamespaceName,
				}, secret)
				assert.Nil(t, err)
				assert.Equal(t, testcase.expectedAlertmanagerConfigStatus.clusterStatus, *cluster.Status.ExtendedHealth.AlertmanagerConfig)
				assert.Equal(t, testcase.expectedAlertmanagerConfigStatus.alertmanagerStatus.Status, alertmanager.Status.ConfigStatus.Status)
				assert.Equal(t, testcase.expectedAlertmanagerConfigStatus.alertmanagerStatus.ErrorMessage, alertmanager.Status.ConfigStatus.ErrorMessage)
				if testcase.expectedAlertmanagerConfigStatus.alertmanagerStatus.Status == corev1.ConditionTrue {
					assert.False(t, alertmanager.Status.ConfigStatus.LastUpdated.IsZero())
				}
			} else {
				assert.True(t, apierrors.IsNotFound(err))
				secretList := &corev1.SecretList{}
				err = reconciler.List(ctx, secretList, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName))
				assert.Nil(t, err)
				assert.Len(t, secretList.Items, 0)
				// No alertmanager config status any more in Cluster CR
				assert.Nil(t, cluster.Status.ExtendedHealth.AlertmanagerConfig)
			}
			assertExpectation()
			server.Close()
		})
	}
}

func generateCluster(name string, monitoringEnabled, loggingEnabled, deleted bool) *kubermaticv1.Cluster {
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.ClusterSpec{
			HumanReadableName: name,
			MLA: &kubermaticv1.MLASettings{
				MonitoringEnabled: monitoringEnabled,
				LoggingEnabled:    loggingEnabled,
			},
		},
		Status: kubermaticv1.ClusterStatus{NamespaceName: fmt.Sprintf("cluster-%s", name)},
	}
	if deleted {
		deleteTime := metav1.NewTime(time.Now())
		cluster.DeletionTimestamp = &deleteTime

		// to validate objects after reconciling, we want to prevent the fakeclient
		// from deleting them once all finalizers are gone; we achieve this by
		// attached a dummy finalizer
		kubernetes.AddFinalizer(cluster, "just-a-test-do-not-delete-thanks")
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
