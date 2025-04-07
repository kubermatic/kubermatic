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

package usersshkeysynchronizer

import (
	"context"
	"reflect"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestUserSSHKeysClusterRemove(t *testing.T) {
	deletionTimestamp := metav1.Now()
	testCases := []struct {
		name               string
		reconciler         *Reconciler
		request            reconcile.Request
		expectedUserSSHKey kubermaticv1.UserSSHKey
	}{
		{
			name: "Test cleanup cluster ids in UserSSHKey on cluster deletion",
			reconciler: &Reconciler{
				log: kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				masterClient: fake.NewClientBuilder().WithObjects(
					&kubermaticv1.UserSSHKey{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test_user_ssh_keys",
							Namespace: "test_namespace",
						},
						Spec: kubermaticv1.SSHKeySpec{
							Clusters: []string{
								"test_cluster_1",
								"test_cluster_2",
							},
						},
					},
				).Build(),
				seedClients: map[string]ctrlruntimeclient.Client{
					"seed_test": fake.NewClientBuilder().WithObjects(
						&kubermaticv1.Cluster{
							ObjectMeta: metav1.ObjectMeta{
								Name:              "test_cluster_1",
								DeletionTimestamp: &deletionTimestamp,
								Finalizers:        []string{"dummy"},
							},
							Status: kubermaticv1.ClusterStatus{
								NamespaceName: "cluster-test_cluster_1",
							},
						},
					).Build(),
				},
			},
			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test_cluster_1", // cluster name
					Namespace: "seed_test",      // seed name
				},
			},
			expectedUserSSHKey: kubermaticv1.UserSSHKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test_user_ssh_keys",
				},
				Spec: kubermaticv1.SSHKeySpec{
					Clusters: []string{
						"test_cluster_2",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			if _, err := tc.reconciler.Reconcile(ctx, tc.request); err != nil {
				t.Fatalf("failed reconciling test: %v", err)
			}

			userSSHKey := &kubermaticv1.UserSSHKey{}
			identifier := types.NamespacedName{Namespace: "test_namespace", Name: "test_user_ssh_keys"}
			if err := tc.reconciler.masterClient.Get(ctx, identifier, userSSHKey); err != nil {
				t.Fatalf("failed to get usersshkey: %v", err)
			}

			if !reflect.DeepEqual(userSSHKey.Spec.Clusters, tc.expectedUserSSHKey.Spec.Clusters) {
				t.Fatalf("usersshkey clusters and expected clusters don't match: want: %v, got: %v",
					tc.expectedUserSSHKey.Spec.Clusters, userSSHKey.Spec.Clusters)
			}
		})
	}
}
