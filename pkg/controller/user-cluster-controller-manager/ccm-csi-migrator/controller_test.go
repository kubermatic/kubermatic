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

package ccmcsimigrator

import (
	"context"
	"fmt"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/machine-controller/sdk/apis/cluster/common"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const defaultNamespace = "default"

func TestReconcile(t *testing.T) {
	conditionType := kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted
	testCases := []struct {
		name                     string
		clusterName              string
		machines                 []*clusterv1alpha1.Machine
		userCluster              *kubermaticv1.Cluster
		expectedClusterCondition *kubermaticv1.ClusterCondition
	}{
		{
			name:        "Machines not migrate",
			clusterName: "clusterNotToMigrate",
			machines: []*clusterv1alpha1.Machine{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "Machine",
						Namespace: defaultNamespace,
					},
				},
			},
			userCluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "clusterNotToMigrate",
				},
			},
			expectedClusterCondition: &kubermaticv1.ClusterCondition{
				Status: "False",
			},
		},
		{
			name:        "Some machines migrated",
			clusterName: "clusterToMigrate",
			machines: []*clusterv1alpha1.Machine{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "migratedMachine",
						Namespace: defaultNamespace,
						Annotations: map[string]string{
							fmt.Sprintf("%s/%s", common.KubeletFlagsGroupAnnotationPrefixV1, common.ExternalCloudProviderKubeletFlag): "true",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "notMigratedMachine",
						Namespace: defaultNamespace,
					},
				},
			},
			userCluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "clusterToMigrate",
				},
			},
			expectedClusterCondition: &kubermaticv1.ClusterCondition{
				Status: "False",
			},
		},
		{
			name:        "All machines migrated",
			clusterName: "clusterToMigrate",
			machines: []*clusterv1alpha1.Machine{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "migratedMachine1",
						Namespace: defaultNamespace,
						Annotations: map[string]string{
							fmt.Sprintf("%s/%s", common.KubeletFlagsGroupAnnotationPrefixV1, common.ExternalCloudProviderKubeletFlag): "true",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "migratedMachine2",
						Namespace: defaultNamespace,
						Annotations: map[string]string{
							fmt.Sprintf("%s/%s", common.KubeletFlagsGroupAnnotationPrefixV1, common.ExternalCloudProviderKubeletFlag): "true",
						},
					},
				},
			},
			userCluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "clusterToMigrate",
				},
			},
			expectedClusterCondition: &kubermaticv1.ClusterCondition{
				Status: "True",
			},
		},
	}

	scheme := fake.NewScheme()
	utilruntime.Must(clusterv1alpha1.AddToScheme(scheme))

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			seedClientBuilder := fake.NewClientBuilder().WithScheme(scheme)
			seedClientBuilder.WithObjects(tc.userCluster)

			userClientBuilder := fake.NewClientBuilder().WithScheme(scheme)
			for _, m := range tc.machines {
				userClientBuilder.WithObjects(m)
			}

			seedClient := seedClientBuilder.Build()
			userClient := userClientBuilder.Build()
			ctx := context.Background()
			r := &reconciler{
				log:          kubermaticlog.Logger,
				seedClient:   seedClient,
				userClient:   userClient,
				seedRecorder: events.NewFakeRecorder(10),
				versions:     kubermatic.Versions{},
				clusterName:  tc.userCluster.Name,
				clusterIsPaused: func(c context.Context) (bool, error) {
					return false, nil
				},
			}

			for range tc.machines {
				request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.clusterName}}
				if _, err := r.Reconcile(ctx, request); err != nil {
					t.Fatalf("reconciling failed: %v", err)
				}
			}

			cluster := &kubermaticv1.Cluster{}
			if err := seedClient.Get(ctx, types.NamespacedName{Name: tc.userCluster.Name}, cluster); err != nil {
				t.Fatalf("failed to get cluster: %v", err)
			}

			if cluster.Status.Conditions[conditionType].Status != tc.expectedClusterCondition.Status {
				t.Errorf("cluster doesn't have the expected status %v", tc.expectedClusterCondition.Status)
			}
		})
	}
}
