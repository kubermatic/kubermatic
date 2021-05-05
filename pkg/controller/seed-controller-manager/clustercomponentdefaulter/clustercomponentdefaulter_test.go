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

package clustercomponentdefaulter

import (
	"context"
	"testing"

	"github.com/go-test/deep"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilpointer "k8s.io/utils/pointer"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const clusterName = "test-cluster"

func exampleCluster(settings *kubermaticv1.ComponentSettings) *kubermaticv1.Cluster {
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: clusterName},
		Spec:       kubermaticv1.ClusterSpec{ComponentsOverride: *settings},
	}
	return cluster
}

func exampleSettings(reconciling *bool, replicas bool) *kubermaticv1.ComponentSettings {
	settings := &kubermaticv1.ComponentSettings{
		Apiserver: kubermaticv1.APIServerSettings{EndpointReconcilingDisabled: reconciling},
	}
	if replicas {
		settings.Apiserver.Replicas = utilpointer.Int32Ptr(1)
		settings.Scheduler.Replicas = utilpointer.Int32Ptr(2)
	}
	return settings
}

func TestReconciliation(t *testing.T) {
	testCases := []struct {
		name     string
		cluster  *kubermaticv1.Cluster
		override *kubermaticv1.ComponentSettings
		verify   func(input, override, reconciled *kubermaticv1.ComponentSettings) []string
	}{
		{
			name:     "Defaulting without EndpointReconcilingDisabled",
			cluster:  exampleCluster(exampleSettings(nil, true)),
			override: exampleSettings(nil, false),
			verify: func(input, override, reconciled *kubermaticv1.ComponentSettings) []string {
				return deep.Equal(reconciled, input)
			},
		},
		{
			name:     "Defaulting with EndpointReconcilingDisabled: true",
			cluster:  exampleCluster(exampleSettings(nil, true)),
			override: exampleSettings(utilpointer.BoolPtr(true), true),
			verify: func(input, override, reconciled *kubermaticv1.ComponentSettings) []string {
				return deep.Equal(reconciled, override)
			},
		},
		{
			name:     "Defaulting with EndpointReconcilingDisabled: false",
			cluster:  exampleCluster(exampleSettings(nil, true)),
			override: exampleSettings(utilpointer.BoolPtr(false), true),
			verify: func(input, override, reconciled *kubermaticv1.ComponentSettings) []string {
				return deep.Equal(reconciled, override)
			},
		},
		{
			name:     "No override when EndpointReconcilingDisabled is specified in cluster",
			cluster:  exampleCluster(exampleSettings(utilpointer.BoolPtr(false), true)),
			override: exampleSettings(utilpointer.BoolPtr(true), false),
			verify: func(input, override, reconciled *kubermaticv1.ComponentSettings) []string {
				return deep.Equal(reconciled, input)
			},
		},
	}

	logger := zap.NewExample().Sugar()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			cluster := tc.cluster.DeepCopy()
			client := fakectrlruntimeclient.
				NewClientBuilder().
				WithObjects(cluster).
				Build()
			r := &Reconciler{client: client, log: logger, defaults: *tc.override}
			if err := r.reconcile(ctx, logger, cluster); err != nil {
				t.Fatalf("failed to reconcile cluster: %v", err)
			}
			reconciledCluster := &kubermaticv1.Cluster{}
			if err := r.client.Get(ctx, types.NamespacedName{Name: clusterName}, reconciledCluster); err != nil {
				t.Fatalf("failed to get reconciledCluster: %v", err)
			}
			if diff := tc.verify(&tc.cluster.Spec.ComponentsOverride, tc.override, &reconciledCluster.Spec.ComponentsOverride); diff != nil {
				t.Fatalf("unexpected difference in cluster after reconciliation: %v", diff)
			}
		})
	}
}
