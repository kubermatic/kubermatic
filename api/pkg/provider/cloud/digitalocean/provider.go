package digitalocean

import (
	"context"

	"github.com/digitalocean/godo"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"golang.org/x/oauth2"
)

type digitalocean struct{}

// NewCloudProvider creates a new digitalocean provider.
func NewCloudProvider() provider.CloudProvider {
	return &digitalocean{}
}

func (do *digitalocean) ValidateCloudSpec(spec *kubermaticv1.CloudSpec) error {
	static := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: spec.Digitalocean.Token})
	client := godo.NewClient(oauth2.NewClient(context.Background(), static))

	_, _, err := client.Regions.List(context.Background(), nil)
	return err
}

func (do *digitalocean) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

func (do *digitalocean) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}
