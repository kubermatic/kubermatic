package kubernetes

import (
	"context"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// AddonConfigProvider struct that holds required components of the AddonConfigProvider
type AddonConfigProvider struct {
	client ctrlruntimeclient.Client
}

// NewAddonConfigProvider returns a new AddonConfigProvider
func NewAddonConfigProvider(client ctrlruntimeclient.Client) *AddonConfigProvider {
	return &AddonConfigProvider{
		client: client,
	}
}

// Get addon configuration
func (p *AddonConfigProvider) Get(addonName string) (*kubermaticv1.AddonConfig, error) {
	addonConfig := &kubermaticv1.AddonConfig{}
	if err := p.client.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: addonName}, addonConfig); err != nil {
		return nil, err
	}
	return addonConfig, nil
}

// List available addon configurations
func (p *AddonConfigProvider) List() (*kubermaticv1.AddonConfigList, error) {
	addonConfigList := &kubermaticv1.AddonConfigList{}
	if err := p.client.List(context.Background(), addonConfigList); err != nil {
		return nil, err
	}
	return addonConfigList, nil
}
