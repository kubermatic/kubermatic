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

package usersshkeyssynchronizer

import (
	"context"
	"reflect"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
				ctx: context.Background(),
				log: kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				client: fake.NewFakeClient(
					&kubermaticv1.UserSSHKeyList{
						Items: []kubermaticv1.UserSSHKey{
							{
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
						},
					},
				),
				seedClients: map[string]client.Client{
					"seed_test": fake.NewFakeClient(
						&kubermaticv1.Cluster{
							ObjectMeta: metav1.ObjectMeta{
								DeletionTimestamp: &deletionTimestamp,
								Name:              "test_cluster_1",
								Namespace:         "test_namespace",
							},
						}),
				},
			},
			request: reconcile.Request{
				NamespacedName: types.NamespacedName{Namespace: "seed_test"},
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
			if _, err := tc.reconciler.Reconcile(tc.request); err != nil {
				t.Fatalf("failed reconciling test: %v", err)
			}

			userSSHKey := &kubermaticv1.UserSSHKey{}
			if err := tc.reconciler.client.Get(context.TODO(),
				types.NamespacedName{Namespace: "test_namespace", Name: "test_user_ssh_keys"},
				userSSHKey); err != nil {
				t.Fatalf("failed to get usersshkey: %v", err)
			}

			if !reflect.DeepEqual(userSSHKey.Spec.Clusters, tc.expectedUserSSHKey.Spec.Clusters) {
				t.Fatalf("usersshkey clusters and expected clusters don't match: want: %v, got: %v",
					tc.expectedUserSSHKey, userSSHKey.Spec.Clusters)
			}
		})
	}
}
