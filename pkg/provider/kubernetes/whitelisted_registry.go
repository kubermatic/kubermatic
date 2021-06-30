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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// WhitelistedRegistryProvider struct that holds required components in order manage whitelisted registries
type PrivilegedWhitelistedRegistryProvider struct {
	clientPrivileged ctrlruntimeclient.Client
}

// NewWhitelistedRegistryProvider returns a whitelisted registry provider
func NewWhitelistedRegistryPrivilegedProvider(client ctrlruntimeclient.Client) (*PrivilegedWhitelistedRegistryProvider, error) {
	return &PrivilegedWhitelistedRegistryProvider{
		clientPrivileged: client,
	}, nil
}

// Create creates a whitelisted registry
func (p *PrivilegedWhitelistedRegistryProvider) Create(wr *kubermaticv1.WhitelistedRegistry) (*kubermaticv1.WhitelistedRegistry, error) {

	if err := p.clientPrivileged.Create(context.Background(), wr); err != nil {
		return nil, err
	}

	return wr, nil
}
