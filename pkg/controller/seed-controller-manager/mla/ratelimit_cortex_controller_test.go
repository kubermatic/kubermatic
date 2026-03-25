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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newTestRatelimitCortexReconciler(t *testing.T, objects []ctrlruntimeclient.Object) *ratelimitCortexReconciler {
	dynamicClient := fake.
		NewClientBuilder().
		WithObjects(objects...).
		Build()
	ratelimitCortexController := newRatelimitCortexController(dynamicClient, kubermaticlog.Logger, "mla")
	reconciler := ratelimitCortexReconciler{
		Client:                    dynamicClient,
		log:                       kubermaticlog.Logger,
		recorder:                  events.NewFakeRecorder(10),
		ratelimitCortexController: ratelimitCortexController,
	}
	return &reconciler
}

func TestRatelimitCortexReconcile(t *testing.T) {
	oldTenantOverride := TenantOverride{
		IngestionRate:      ptr.To[int32](1),
		MaxSeriesPerMetric: ptr.To[int32](1),
		MaxSeriesPerQuery:  ptr.To[int32](1),
		MaxSamplesPerQuery: ptr.To[int32](1),
		IngestionBurstSize: ptr.To[int32](1),
		MaxSeriesTotal:     ptr.To[int32](1),
	}
	oldRatelimitConfig := Overrides{Overrides: map[string]TenantOverride{"old": oldTenantOverride}}
	data, err := yaml.Marshal(oldRatelimitConfig)
	assert.Nil(t, err)
	oldRatelimitConfigData := string(data)
	testCases := []struct {
		name              string
		request           types.NamespacedName
		objects           []ctrlruntimeclient.Object
		expectedOverrides Overrides
		hasFinalizer      bool
		err               bool
	}{
		{
			name: "create MLAAdmin settings",
			request: types.NamespacedName{
				Namespace: "cluster-123",
				Name:      resources.MLAAdminSettingsName,
			},
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.MLAAdminSetting{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resources.MLAAdminSettingsName,
						Namespace: "cluster-123",
					},
					Spec: kubermaticv1.MLAAdminSettingSpec{
						ClusterName: "123",
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      RuntimeConfigMap,
						Namespace: "mla",
					},
					Data: map[string]string{RuntimeConfigFileName: "overrides: {}"},
				},
			},
			expectedOverrides: Overrides{
				Overrides: map[string]TenantOverride{
					"123": {},
				},
			},
			hasFinalizer: true,
			err:          false,
		},
		{
			name: "create MLAAdmin settings with values",
			request: types.NamespacedName{
				Namespace: "cluster-123",
				Name:      resources.MLAAdminSettingsName,
			},
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.MLAAdminSetting{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resources.MLAAdminSettingsName,
						Namespace: "cluster-123",
					},
					Spec: kubermaticv1.MLAAdminSettingSpec{
						ClusterName: "123",
						MonitoringRateLimits: &kubermaticv1.MonitoringRateLimitSettings{
							IngestionRate:      1,
							IngestionBurstSize: 2,
							MaxSeriesPerMetric: 3,
							MaxSeriesTotal:     4,
							MaxSamplesPerQuery: 5,
							MaxSeriesPerQuery:  6,
						},
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      RuntimeConfigMap,
						Namespace: "mla",
					},
					Data: map[string]string{RuntimeConfigFileName: "overrides: {}"},
				},
			},
			expectedOverrides: Overrides{
				Overrides: map[string]TenantOverride{
					"123": {
						IngestionRate:      ptr.To[int32](1),
						IngestionBurstSize: ptr.To[int32](2),
						MaxSeriesPerMetric: ptr.To[int32](3),
						MaxSeriesTotal:     ptr.To[int32](4),
						MaxSamplesPerQuery: ptr.To[int32](5),
						MaxSeriesPerQuery:  ptr.To[int32](6),
					},
				},
			},
			hasFinalizer: true,
			err:          false,
		},
		{
			name: "create MLAAdmin settings append",
			request: types.NamespacedName{
				Namespace: "cluster-123",
				Name:      resources.MLAAdminSettingsName,
			},
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.MLAAdminSetting{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resources.MLAAdminSettingsName,
						Namespace: "cluster-123",
					},
					Spec: kubermaticv1.MLAAdminSettingSpec{
						ClusterName: "123",
						MonitoringRateLimits: &kubermaticv1.MonitoringRateLimitSettings{
							IngestionRate:      1,
							IngestionBurstSize: 2,
							MaxSeriesPerMetric: 3,
							MaxSeriesTotal:     4,
							MaxSamplesPerQuery: 5,
							MaxSeriesPerQuery:  6,
						},
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      RuntimeConfigMap,
						Namespace: "mla",
					},
					Data: map[string]string{RuntimeConfigFileName: oldRatelimitConfigData},
				},
			},
			expectedOverrides: Overrides{
				Overrides: map[string]TenantOverride{
					"123": {
						IngestionRate:      ptr.To[int32](1),
						IngestionBurstSize: ptr.To[int32](2),
						MaxSeriesPerMetric: ptr.To[int32](3),
						MaxSeriesTotal:     ptr.To[int32](4),
						MaxSamplesPerQuery: ptr.To[int32](5),
						MaxSeriesPerQuery:  ptr.To[int32](6),
					},
					"old": oldTenantOverride,
				},
			},
			hasFinalizer: true,
			err:          false,
		},
		{
			name: "MLAAdmin settings delete values",
			request: types.NamespacedName{
				Namespace: "cluster-old",
				Name:      resources.MLAAdminSettingsName,
			},
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.MLAAdminSetting{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resources.MLAAdminSettingsName,
						Namespace: "cluster-old",
					},
					Spec: kubermaticv1.MLAAdminSettingSpec{
						ClusterName: "old",
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      RuntimeConfigMap,
						Namespace: "mla",
					},
					Data: map[string]string{RuntimeConfigFileName: oldRatelimitConfigData},
				},
			},
			expectedOverrides: Overrides{
				Overrides: map[string]TenantOverride{
					"old": {
						IngestionRate:      nil,
						IngestionBurstSize: nil,
						MaxSeriesPerMetric: nil,
						MaxSeriesTotal:     nil,
						MaxSamplesPerQuery: nil,
						MaxSeriesPerQuery:  nil,
					},
				},
			},
			hasFinalizer: true,
			err:          false,
		},
		{
			name: "delete MLAAdmin settings",
			request: types.NamespacedName{
				Namespace: "cluster-old",
				Name:      resources.MLAAdminSettingsName,
			},
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.MLAAdminSetting{
					ObjectMeta: metav1.ObjectMeta{
						Name:              resources.MLAAdminSettingsName,
						Namespace:         "cluster-old",
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
						Finalizers:        []string{mlaFinalizer, "do-not-remove"},
					},
					Spec: kubermaticv1.MLAAdminSettingSpec{
						ClusterName: "old",
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      RuntimeConfigMap,
						Namespace: "mla",
					},
					Data: map[string]string{RuntimeConfigFileName: oldRatelimitConfigData},
				},
			},
			expectedOverrides: Overrides{
				Overrides: map[string]TenantOverride{},
			},
			hasFinalizer: false,
			err:          false,
		},
	}
	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			controller := newTestRatelimitCortexReconciler(t, tc.objects)
			request := reconcile.Request{NamespacedName: tc.request}
			_, err := controller.Reconcile(ctx, request)
			if err != nil && !tc.err {
				assert.Nil(t, err)
			}
			assert.Equal(t, tc.err, err != nil)
			configMap := &corev1.ConfigMap{}
			if err := controller.Get(ctx, types.NamespacedName{Namespace: "mla", Name: RuntimeConfigMap}, configMap); err != nil {
				t.Fatalf("unable to get configMap: %v", err)
			}
			actualOverrides := &Overrides{}
			decoder := yaml.NewDecoder(strings.NewReader(configMap.Data[RuntimeConfigFileName]))
			decoder.KnownFields(true)
			err = decoder.Decode(actualOverrides)
			assert.Nil(t, err)
			assert.Equal(t, tc.expectedOverrides, *actualOverrides)
			mlaAdminSetting := &kubermaticv1.MLAAdminSetting{}
			if err := controller.Get(ctx, tc.request, mlaAdminSetting); err != nil {
				t.Fatalf("unable to get mlaAdminSetting: %v", err)
			}
			assert.Equal(t, tc.hasFinalizer, kubernetes.HasFinalizer(mlaAdminSetting, mlaFinalizer))
		})
	}
}
