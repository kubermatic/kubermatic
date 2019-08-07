package hetzner

import (
	"context"

	"github.com/hetznercloud/hcloud-go/hcloud"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

type hetzner struct {
}

// NewCloudProvider creates a new hetzner provider.
func NewCloudProvider() provider.CloudProvider {
	return &hetzner{}
}

// DefaultCloudSpec
func (h *hetzner) DefaultCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

// ValidateCloudSpec
func (h *hetzner) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	client := hcloud.NewClient(hcloud.WithToken(spec.Hetzner.Token))
	_, _, err := client.ServerType.List(context.Background(), hcloud.ServerTypeListOpts{})
	return err
}

// InitializeCloudProvider
func (h *hetzner) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, secretKeySelector provider.SecretKeySelectorValueFunc) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

// CleanUpCloudProvider
func (h *hetzner) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, _ provider.ClusterUpdater, _ provider.SecretKeySelectorValueFunc) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted
func (h *hetzner) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}
