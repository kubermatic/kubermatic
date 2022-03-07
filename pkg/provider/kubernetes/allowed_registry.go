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

package kubernetes

import (
	"context"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// PrivilegedAllowedRegistryProvider struct that holds required components in order manage allowed registries.
type PrivilegedAllowedRegistryProvider struct {
	clientPrivileged ctrlruntimeclient.Client
}

var _ provider.PrivilegedAllowedRegistryProvider = &PrivilegedAllowedRegistryProvider{}

// NewAllowedRegistryProvider returns a allowed registry provider.
func NewAllowedRegistryPrivilegedProvider(client ctrlruntimeclient.Client) (*PrivilegedAllowedRegistryProvider, error) {
	return &PrivilegedAllowedRegistryProvider{
		clientPrivileged: client,
	}, nil
}

// CreateUnsecured creates a allowed registry.
func (p *PrivilegedAllowedRegistryProvider) CreateUnsecured(ctx context.Context, wr *kubermaticv1.AllowedRegistry) (*kubermaticv1.AllowedRegistry, error) {
	if err := p.clientPrivileged.Create(ctx, wr); err != nil {
		return nil, err
	}

	return wr, nil
}

// GetUnsecured gets a allowed registry.
func (p *PrivilegedAllowedRegistryProvider) GetUnsecured(ctx context.Context, name string) (*kubermaticv1.AllowedRegistry, error) {
	wr := &kubermaticv1.AllowedRegistry{}
	err := p.clientPrivileged.Get(ctx, types.NamespacedName{Name: name}, wr)
	return wr, err
}

// ListUnsecured lists a allowed registries.
func (p *PrivilegedAllowedRegistryProvider) ListUnsecured(ctx context.Context) (*kubermaticv1.AllowedRegistryList, error) {
	wrList := &kubermaticv1.AllowedRegistryList{}
	err := p.clientPrivileged.List(ctx, wrList)
	return wrList, err
}

// UpdateUnsecured updates the allowed registry.
func (p *PrivilegedAllowedRegistryProvider) UpdateUnsecured(ctx context.Context, ar *kubermaticv1.AllowedRegistry) (*kubermaticv1.AllowedRegistry, error) {
	if err := p.clientPrivileged.Update(ctx, ar); err != nil {
		return nil, err
	}

	return ar, nil
}

// DeleteUnsecured deletes a allowed registry.
func (p *PrivilegedAllowedRegistryProvider) DeleteUnsecured(ctx context.Context, name string) error {
	wr := &kubermaticv1.AllowedRegistry{}
	wr.Name = name
	return p.clientPrivileged.Delete(ctx, wr)
}
