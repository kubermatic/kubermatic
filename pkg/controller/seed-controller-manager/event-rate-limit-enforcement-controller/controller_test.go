/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package eventratelimitenforcement

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	workerSelector, err := workerlabel.LabelSelector("")
	if err != nil {
		t.Fatalf("failed to build worker-name selector: %v", err)
	}

	defaultConfig := &kubermaticv1.EventRateLimitConfig{
		Server: &kubermaticv1.EventRateLimitConfigItem{
			QPS:       50,
			Burst:     100,
			CacheSize: 2000,
		},
	}

	testCases := []struct {
		name                         string
		cluster                      *kubermaticv1.Cluster
		config                       *kubermaticv1.KubermaticConfiguration
		expectedUseEventRateLimit    bool
		expectedEventRateLimitConfig *kubermaticv1.EventRateLimitConfig
	}{
		{
			name:                         "enforce EventRateLimit on cluster without it",
			cluster:                      genCluster(false, nil, false),
			config:                       genConfig(true, defaultConfig),
			expectedUseEventRateLimit:    true,
			expectedEventRateLimitConfig: defaultConfig,
		},
		{
			name:                         "skip enforcement when cluster is paused",
			cluster:                      genCluster(false, nil, true),
			config:                       genConfig(true, defaultConfig),
			expectedUseEventRateLimit:    false,
			expectedEventRateLimitConfig: nil,
		},
		{
			name:                         "no update when cluster already matches enforced state",
			cluster:                      genCluster(true, defaultConfig, false),
			config:                       genConfig(true, defaultConfig),
			expectedUseEventRateLimit:    true,
			expectedEventRateLimitConfig: defaultConfig,
		},
		{
			name:                         "no-op when enforcement is not enabled",
			cluster:                      genCluster(false, nil, false),
			config:                       genConfig(false, defaultConfig),
			expectedUseEventRateLimit:    false,
			expectedEventRateLimitConfig: nil,
		},
		{
			name:                         "no-op when EventRateLimit config is nil",
			cluster:                      genCluster(false, nil, false),
			config:                       genConfigNilEventRateLimit(),
			expectedUseEventRateLimit:    false,
			expectedEventRateLimitConfig: nil,
		},
		{
			name:                         "no-op when Enforced field is nil",
			cluster:                      genCluster(false, nil, false),
			config:                       genConfigEnforcedNil(defaultConfig),
			expectedUseEventRateLimit:    false,
			expectedEventRateLimitConfig: nil,
		},
		{
			name:                         "skip enforcement when cluster is being deleted",
			cluster:                      genDeletedCluster(),
			config:                       genConfig(true, defaultConfig),
			expectedUseEventRateLimit:    false,
			expectedEventRateLimitConfig: nil,
		},
		{
			name:                         "enforce enables plugin even without default config",
			cluster:                      genCluster(false, nil, false),
			config:                       genConfig(true, nil),
			expectedUseEventRateLimit:    true,
			expectedEventRateLimitConfig: nil,
		},
		{
			name: "enforce overwrites existing cluster config with default config",
			cluster: genCluster(true, &kubermaticv1.EventRateLimitConfig{
				Server: &kubermaticv1.EventRateLimitConfigItem{
					QPS:       10,
					Burst:     20,
					CacheSize: 500,
				},
			}, false),
			config:                       genConfig(true, defaultConfig),
			expectedUseEventRateLimit:    true,
			expectedEventRateLimitConfig: defaultConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			seedClient := fake.
				NewClientBuilder().
				WithObjects(tc.cluster).
				Build()

			configGetter := func(_ context.Context) (*kubermaticv1.KubermaticConfiguration, error) {
				return tc.config, nil
			}

			r := &reconciler{
				log:                     kubermaticlog.Logger,
				workerNameLabelSelector: workerSelector,
				recorder:                &events.FakeRecorder{},
				configGetter:            configGetter,
				seedClient:              seedClient,
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.cluster.Name}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			updatedCluster := &kubermaticv1.Cluster{}
			if err := seedClient.Get(ctx, types.NamespacedName{Name: tc.cluster.Name}, updatedCluster); err != nil {
				t.Fatalf("failed to get cluster: %v", err)
			}

			if updatedCluster.Spec.UseEventRateLimitAdmissionPlugin != tc.expectedUseEventRateLimit {
				t.Errorf("UseEventRateLimitAdmissionPlugin: got %v, want %v",
					updatedCluster.Spec.UseEventRateLimitAdmissionPlugin, tc.expectedUseEventRateLimit)
			}

			if !diff.SemanticallyEqual(tc.expectedEventRateLimitConfig, updatedCluster.Spec.EventRateLimitConfig) {
				t.Fatalf("EventRateLimitConfig mismatch:\n%v", diff.ObjectDiff(tc.expectedEventRateLimitConfig, updatedCluster.Spec.EventRateLimitConfig))
			}
		})
	}
}

func genCluster(useEventRateLimit bool, eventRateLimitConfig *kubermaticv1.EventRateLimitConfig, paused bool) *kubermaticv1.Cluster {
	cluster := generator.GenDefaultCluster()
	cluster.Spec.UseEventRateLimitAdmissionPlugin = useEventRateLimit
	cluster.Spec.EventRateLimitConfig = eventRateLimitConfig
	cluster.Spec.Pause = paused

	return cluster
}

func genConfig(enforced bool, defaultConfig *kubermaticv1.EventRateLimitConfig) *kubermaticv1.KubermaticConfiguration {
	return &kubermaticv1.KubermaticConfiguration{
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
				AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
					EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
						Enforced:      ptr.To(enforced),
						DefaultConfig: defaultConfig,
					},
				},
			},
		},
	}
}

func genConfigNilEventRateLimit() *kubermaticv1.KubermaticConfiguration {
	return &kubermaticv1.KubermaticConfiguration{}
}

func genConfigEnforcedNil(defaultConfig *kubermaticv1.EventRateLimitConfig) *kubermaticv1.KubermaticConfiguration {
	return &kubermaticv1.KubermaticConfiguration{
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
				AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
					EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
						DefaultConfig: defaultConfig,
					},
				},
			},
		},
	}
}

func genDeletedCluster() *kubermaticv1.Cluster {
	cluster := generator.GenDefaultCluster()
	now := metav1.Now()
	cluster.DeletionTimestamp = &now
	cluster.Finalizers = []string{"test-finalizer"}
	return cluster
}
