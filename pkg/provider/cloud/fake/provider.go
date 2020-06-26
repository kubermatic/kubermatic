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

package fake

import (
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

type fakeCloudProvider struct {
	nodes map[string]*apiv1.Node
}

// NewCloudProvider creates a new fake cloud provider
func NewCloudProvider() provider.CloudProvider {
	return &fakeCloudProvider{
		nodes: map[string]*apiv1.Node{},
	}
}

func (p *fakeCloudProvider) DefaultCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

func (p *fakeCloudProvider) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	return nil
}

func (p *fakeCloudProvider) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

func (p *fakeCloudProvider) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, _ provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted
func (p *fakeCloudProvider) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}
