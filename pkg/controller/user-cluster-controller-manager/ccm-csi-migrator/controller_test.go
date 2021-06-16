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

package ccmcsimigrator

import (
	"context"
	"fmt"
	"github.com/kubermatic/machine-controller/pkg/apis/cluster/common"
	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"

	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const defaultNamespace = "default"

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name                     string
		clusterName              string
		machines                 []*v1alpha1.Machine
		userCluster              *kubermaticv1.Cluster
		expectedClusterCondition *kubermaticv1.ClusterCondition
	}{
		{
			name:        "Machines not migrate",
			clusterName: "clusterNotToMigrate",
			machines: []*v1alpha1.Machine{
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
				Type:   kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted,
				Status: "False",
			},
		},
		{
			name:        "Some machines migrated",
			clusterName: "clusterToMigrate",
			machines: []*v1alpha1.Machine{
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
				Type:   kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted,
				Status: "False",
			},
		},
		{
			name:        "All machines migrated",
			clusterName: "clusterToMigrate",
			machines: []*v1alpha1.Machine{
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
				Type:   kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted,
				Status: "True",
			},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			scheme := runtime.NewScheme()
			_ = kubermaticv1.AddToScheme(scheme)
			_ = v1alpha1.AddToScheme(scheme)

			seedClientBuilder := fakectrlruntimeclient.NewClientBuilder().WithScheme(scheme)
			seedClientBuilder.WithObjects(tc.userCluster)

			userClientBuilder := fakectrlruntimeclient.NewClientBuilder().WithScheme(scheme)
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
				seedRecorder: record.NewFakeRecorder(10),
				versions:     kubermatic.Versions{},
				clusterName:  tc.userCluster.Name,
			}

			for _ = range tc.machines {
				request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.clusterName}}
				if _, err := r.Reconcile(ctx, request); err != nil {
					t.Fatalf("reconciling failed: %v", err)
				}
			}

			cluster := &kubermaticv1.Cluster{}
			if err := seedClient.Get(ctx, types.NamespacedName{Name: tc.userCluster.Name}, cluster); err != nil {
				t.Fatalf("failed to get cluster: %v", err)
			}

			if !helper.ClusterConditionHasStatus(cluster, tc.expectedClusterCondition.Type, tc.expectedClusterCondition.Status) {
				t.Errorf("cluster doesn't have the expected status %v", tc.expectedClusterCondition.Status)
			}
		})
	}
}
