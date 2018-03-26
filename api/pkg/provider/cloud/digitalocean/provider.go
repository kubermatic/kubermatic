package digitalocean

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/uuid"
	"golang.org/x/oauth2"
)

const (
	tplPath = "/opt/template/nodes/digitalocean.yaml"
)

type digitalocean struct {
	dcs map[string]provider.DatacenterMeta
}

// NewCloudProvider creates a new digitalocean provider.
func NewCloudProvider(dcs map[string]provider.DatacenterMeta) provider.CloudProvider {
	return &digitalocean{
		dcs: dcs,
	}
}

func (do *digitalocean) ValidateCloudSpec(cloud *kubermaticv1.CloudSpec) error {
	static := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cloud.Digitalocean.Token})
	client := godo.NewClient(oauth2.NewClient(context.Background(), static))

	_, _, err := client.Regions.List(context.Background(), nil)
	return err
}

func (do *digitalocean) InitializeCloudProvider(cloud *kubermaticv1.CloudSpec, name string) (*kubermaticv1.CloudSpec, error) {
	return nil, nil
}

func (do *digitalocean) CleanUpCloudProvider(*kubermaticv1.CloudSpec) error {
	return nil
}

func (do *digitalocean) NodeClassName(nSpec *apiv1.NodeSpec) string {
	return fmt.Sprintf("kubermatic-%s-%s-%s", "coreos", nSpec.Digitalocean.Size, uuid.ShortUID(5))
}

func (do *digitalocean) ValidateNodeSpec(cloudSpec *kubermaticv1.CloudSpec, nodeSpec *apiv1.NodeSpec) error {
	return nil
}
