package kubernetes

import (
	"context"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// SeedProvider struct that holds required components in order seeds
type SeedProvider struct {
	clientPrivileged ctrlruntimeclient.Client
}

func NewSeedProvider(client ctrlruntimeclient.Client) *SeedProvider {
	return &SeedProvider{
		clientPrivileged: client,
	}
}

func (p *SeedProvider) UpdateUnsecured(seed *kubermaticv1.Seed) (*kubermaticv1.Seed, error) {
	if err := p.clientPrivileged.Update(context.Background(), seed); err != nil {
		return nil, err
	}
	return seed, nil
}
