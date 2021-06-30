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
