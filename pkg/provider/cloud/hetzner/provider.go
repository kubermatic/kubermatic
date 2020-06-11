package hetzner

import (
	"context"
	"errors"

	"github.com/hetznercloud/hcloud-go/hcloud"
	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/pkg/provider"
	"github.com/kubermatic/kubermatic/pkg/resources"
)

type hetzner struct {
	secretKeySelector provider.SecretKeySelectorValueFunc
}

// NewCloudProvider creates a new hetzner provider.
func NewCloudProvider(secretKeyGetter provider.SecretKeySelectorValueFunc) provider.CloudProvider {
	return &hetzner{
		secretKeySelector: secretKeyGetter,
	}
}

// DefaultCloudSpec
func (h *hetzner) DefaultCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

// ValidateCloudSpec
func (h *hetzner) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	hetznerToken, err := GetCredentialsForCluster(spec, h.secretKeySelector)
	if err != nil {
		return err
	}

	client := hcloud.NewClient(hcloud.WithToken(hetznerToken))
	_, _, err = client.ServerType.List(context.Background(), hcloud.ServerTypeListOpts{})
	return err
}

// InitializeCloudProvider
func (h *hetzner) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

// CleanUpCloudProvider
func (h *hetzner) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, _ provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted
func (h *hetzner) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error
func GetCredentialsForCluster(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (hetznerToken string, err error) {
	hetznerToken = cloud.Hetzner.Token

	if hetznerToken == "" {
		if cloud.Hetzner.CredentialsReference == nil {
			return "", errors.New("no credentials provided")
		}
		hetznerToken, err = secretKeySelector(cloud.Hetzner.CredentialsReference, resources.HetznerToken)
		if err != nil {
			return "", err
		}
	}

	return hetznerToken, nil
}
