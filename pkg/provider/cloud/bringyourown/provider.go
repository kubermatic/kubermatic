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

package bringyourown

import (
	"context"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
)

type bringyourown struct{}

// NewCloudProvider creates a new bringyourown provider.
func NewCloudProvider() provider.CloudProvider {
	return &bringyourown{}
}

var _ provider.ReconcilingCloudProvider = &bringyourown{}

func (b *bringyourown) DefaultCloudSpec(_ context.Context, _ *kubermaticv1.ClusterSpec) error {
	return nil
}

func (*bringyourown) ClusterNeedsReconciling(cluster *kubermaticv1.Cluster) bool {
	return false
}

func (b *bringyourown) ValidateCloudSpec(_ context.Context, _ kubermaticv1.CloudSpec) error {
	return nil
}

func (b *bringyourown) InitializeCloudProvider(_ context.Context, cluster *kubermaticv1.Cluster, _ provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

func (b *bringyourown) ReconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

func (b *bringyourown) CleanUpCloudProvider(_ context.Context, cluster *kubermaticv1.Cluster, _ provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted.
func (b *bringyourown) ValidateCloudSpecUpdate(_ context.Context, _ kubermaticv1.CloudSpec, _ kubermaticv1.CloudSpec) error {
	return nil
}
