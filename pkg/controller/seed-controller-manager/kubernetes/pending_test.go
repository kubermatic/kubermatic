/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package kubernetes

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnsureEtcdLauncherFeatureFlag(t *testing.T) {
	tests := []struct {
		name                 string
		clusterFeatures      map[string]bool
		seedEtcdLauncher     bool
		expectedEtcdLauncher bool
	}{
		{
			name:                 "Seed feature gate enabled, cluster has no feature flag",
			clusterFeatures:      nil, // no features set
			seedEtcdLauncher:     true,
			expectedEtcdLauncher: true,
		},
		{
			name: "Seed feature gate enabled, cluster explicitly set to false",
			clusterFeatures: map[string]bool{
				kubermaticv1.ClusterFeatureEtcdLauncher: false,
			},
			seedEtcdLauncher:     true,
			expectedEtcdLauncher: false,
		},
		{
			name: "Seed feature gate disabled, cluster explicitly set to true",
			clusterFeatures: map[string]bool{
				kubermaticv1.ClusterFeatureEtcdLauncher: true,
			},
			seedEtcdLauncher:     false,
			expectedEtcdLauncher: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cluster := &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster",
					Labels: map[string]string{
						kubermaticv1.ProjectIDLabelKey: "project",
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					Features: test.clusterFeatures,
				},
			}
			r := &Reconciler{
				Client: fake.NewClientBuilder().WithObjects(cluster).Build(),
				features: Features{
					EtcdLauncher: test.seedEtcdLauncher,
				},
			}
			if err := r.ensureEtcdLauncherFeatureFlag(context.Background(), cluster); err != nil {
				t.Fatal(err)
			}
			if cluster.Spec.Features != nil && cluster.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher] != test.expectedEtcdLauncher {
				t.Errorf("expected clsuter flag to be %v , got %v instead", test.expectedEtcdLauncher, cluster.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher])
			}
		})
	}
}
