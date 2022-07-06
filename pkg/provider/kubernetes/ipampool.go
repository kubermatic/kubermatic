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

// PrivilegedIPAMPoolProvider struct that holds required components of the PrivilegedIPAMPoolProvider.
type PrivilegedIPAMPoolProvider struct {
	privilegedClient ctrlruntimeclient.Client
}

var _ provider.PrivilegedIPAMPoolProvider = &PrivilegedIPAMPoolProvider{}

// NewPrivilegedIPAMPoolProvider returns a new PrivilegedIPAMPoolProvider.
func NewPrivilegedIPAMPoolProvider(privilegedClient ctrlruntimeclient.Client) *PrivilegedIPAMPoolProvider {
	return &PrivilegedIPAMPoolProvider{
		privilegedClient: privilegedClient,
	}
}

// ListUnsecured lists available IPAM pools.
func (p *PrivilegedIPAMPoolProvider) ListUnsecured(ctx context.Context) (*kubermaticv1.IPAMPoolList, error) {
	ipamPoolList := &kubermaticv1.IPAMPoolList{}
	if err := p.privilegedClient.List(ctx, ipamPoolList); err != nil {
		return nil, err
	}
	return ipamPoolList, nil
}

// GetUnsecured gets IPAM pool by name.
func (p *PrivilegedIPAMPoolProvider) GetUnsecured(ctx context.Context, ipamPoolName string) (*kubermaticv1.IPAMPool, error) {
	ipamPool := &kubermaticv1.IPAMPool{}
	if err := p.privilegedClient.Get(ctx, types.NamespacedName{Name: ipamPoolName}, ipamPool); err != nil {
		return nil, err
	}
	return ipamPool, nil
}

// DeleteUnsecured deletes IPAM pool by name.
func (p *PrivilegedIPAMPoolProvider) DeleteUnsecured(ctx context.Context, ipamPoolName string) error {
	ipamPool, err := p.GetUnsecured(ctx, ipamPoolName)
	if err != nil {
		return err
	}

	if err := p.privilegedClient.Delete(ctx, ipamPool); err != nil {
		return err
	}

	return nil
}

// CreateUnsecured creates a IPAM pool.
func (p *PrivilegedIPAMPoolProvider) CreateUnsecured(ctx context.Context, ipamPool *kubermaticv1.IPAMPool) error {
	return p.privilegedClient.Create(ctx, ipamPool)
}

// PatchUnsecured patches a IPAM pool.
func (p *PrivilegedIPAMPoolProvider) PatchUnsecured(ctx context.Context, oldIPAMPool *kubermaticv1.IPAMPool, newIPAMPool *kubermaticv1.IPAMPool) error {
	return p.privilegedClient.Patch(ctx, newIPAMPool, ctrlruntimeclient.MergeFrom(oldIPAMPool))
}
