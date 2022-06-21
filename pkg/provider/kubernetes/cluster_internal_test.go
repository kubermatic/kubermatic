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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	k8cuserclusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
			name: "Kubernetes: Token gets updated",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Status: kubermaticv1.ClusterStatus{
					Address: kubermaticv1.ClusterAddress{AdminToken: "123"},
				},
			},
			verify: func(seedClient, _ ctrlruntimeclient.Client) error {
				name := types.NamespacedName{Name: "cluster"}
				cluster := &kubermaticv1.Cluster{}
				if err := seedClient.Get(context.Background(), name, cluster); err != nil {
					return fmt.Errorf("failed to fetch cluster: %w", err)
				}
				if cluster.Status.Address.AdminToken == "123" {
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

			versions := kubermatic.NewFakeVersions()
			seed := &kubermaticv1.Seed{}
			seed.SetKubermaticVersion(versions)

			p := &ClusterProvider{
				client:                  seedClient,
				userClusterConnProvider: &fakeUserClusterConnectionProvider{client: userClusterClient},
				seed:                    seed,
				versions:                versions,
			}

			if err := p.RevokeAdminKubeconfig(context.Background(), tc.cluster); err != nil {
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

func (f *fakeUserClusterConnectionProvider) GetK8sClient(_ context.Context, _ *kubermaticv1.Cluster, _ ...k8cuserclusterclient.ConfigOption) (kubernetes.Interface, error) {
	return nil, nil
}

func (f *fakeUserClusterConnectionProvider) GetClientConfig(_ context.Context, _ *kubermaticv1.Cluster, _ ...k8cuserclusterclient.ConfigOption) (*rest.Config, error) {
	return nil, nil
}

func (f *fakeUserClusterConnectionProvider) GetClient(context.Context, *kubermaticv1.Cluster, ...k8cuserclusterclient.ConfigOption) (ctrlruntimeclient.Client, error) {
	return f.client, nil
}
