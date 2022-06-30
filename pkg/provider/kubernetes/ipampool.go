/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8s.io/apimachinery/pkg/types"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// IPAMPoolProvider struct that holds required components of the IPAMPoolProvider.
type IPAMPoolProvider struct {
	client ctrlruntimeclient.Client
}

var _ provider.IPAMPoolProvider = &IPAMPoolProvider{}

// NewIPAMPoolProvider returns a new IPAMPoolProvider.
func NewIPAMPoolProvider(client ctrlruntimeclient.Client) *IPAMPoolProvider {
	return &IPAMPoolProvider{
		client: client,
	}
}

// List available IPAM pools.
func (p *IPAMPoolProvider) List(ctx context.Context) (*kubermaticv1.IPAMPoolList, error) {
	ipamPoolList := &kubermaticv1.IPAMPoolList{}
	if err := p.client.List(ctx, ipamPoolList); err != nil {
		return nil, err
	}
	return ipamPoolList, nil
}

// Get IPAM pool by name.
func (p *IPAMPoolProvider) Get(ctx context.Context, ipamPoolName string) (*kubermaticv1.IPAMPool, error) {
	ipamPool := &kubermaticv1.IPAMPool{}
	if err := p.client.Get(ctx, types.NamespacedName{Name: ipamPoolName}, ipamPool); err != nil {
		return nil, err
	}
	return ipamPool, nil
}

// Delete deletes IPAM pool by name.
func (p *IPAMPoolProvider) Delete(ctx context.Context, ipamPoolName string) error {
	ipamPool, err := p.Get(ctx, ipamPoolName)
	if err != nil {
		return err
	}

	if err := p.client.Delete(ctx, ipamPool); err != nil {
		return err
	}

	return nil
}

// Create creates a IPAM pool.
func (p *IPAMPoolProvider) Create(ctx context.Context, ipamPool *kubermaticv1.IPAMPool) error {
	return p.client.Create(ctx, ipamPool)
}
