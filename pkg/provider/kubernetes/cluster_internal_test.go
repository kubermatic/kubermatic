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
	"errors"
	"fmt"
	"testing"

	k8cuserclusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	openshiftuserclusterresources "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/openshift"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRevokeAdminKubeconfig(t *testing.T) {
	testCases := []struct {
		name               string
		cluster            *kubermaticv1.Cluster
		userClusterObjects []ctrlruntimeclient.Object
		verify             func(seedClient, userClusterClient ctrlruntimeclient.Client) error
	}{
		{
			name: "Openshift: ServiceAccount gets deleted",
			cluster: &kubermaticv1.Cluster{ObjectMeta: metav1.ObjectMeta{
				Name:        "cluster",
				Annotations: map[string]string{"kubermatic.io/openshift": "true"},
			}},
			userClusterObjects: []ctrlruntimeclient.Object{
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
					Namespace: metav1.NamespaceSystem,
					Name:      openshiftuserclusterresources.TokenOwnerServiceAccountName,
				}},
			},
			verify: func(_, userClusterClient ctrlruntimeclient.Client) error {
				name := types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: openshiftuserclusterresources.TokenOwnerServiceAccountName}
				if err := userClusterClient.Get(context.Background(), name, &corev1.ServiceAccount{}); !kerrors.IsNotFound(err) {
					return errors.New("expected serviceAccount to be gone but still existed")
				}
				return nil
			},
		},
		{
			name: "Kubernetes: Token gets updated",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Address: kubermaticv1.ClusterAddress{AdminToken: "123"},
			},
			verify: func(seedClient, _ ctrlruntimeclient.Client) error {
				name := types.NamespacedName{Name: "cluster"}
				cluster := &kubermaticv1.Cluster{}
				if err := seedClient.Get(context.Background(), name, cluster); err != nil {
					return fmt.Errorf("failed to fetch cluster: %v", err)
				}
				if cluster.Address.AdminToken == "123" {
					return errors.New("expected admin token to get updated, was unchanged")
				}
				return nil
			},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			seedClient := fakectrlruntimeclient.NewClientBuilder().WithObjects(tc.cluster).Build()
			userClusterClient := fakectrlruntimeclient.
				NewClientBuilder().
				WithObjects(tc.userClusterObjects...).
				Build()

			p := &ClusterProvider{
				client:                  seedClient,
				userClusterConnProvider: &fakeUserClusterConnectionProvider{client: userClusterClient},
			}

			if err := p.RevokeAdminKubeconfig(tc.cluster); err != nil {
				t.Fatalf("error calling revokeClusterAdminKubeconfig: %v", err)
			}
			if err := tc.verify(seedClient, userClusterClient); err != nil {
				t.Error(err)
			}
		})
	}
}

type fakeUserClusterConnectionProvider struct {
	client ctrlruntimeclient.Client
}

func (f *fakeUserClusterConnectionProvider) GetClient(context.Context, *kubermaticv1.Cluster, ...k8cuserclusterclient.ConfigOption) (ctrlruntimeclient.Client, error) {
	return f.client, nil
}
