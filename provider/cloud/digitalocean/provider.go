package digitalocean

import (
	"errors"
	"fmt"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/digitalocean/godo"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
)

const (
	tokenAnnotationKey = "token"
)

var _ provider.CloudProvider = (*digitalocean)(nil)

type digitalocean struct{}

// NewCloudProvider creates a new digitalocean provider.
func NewCloudProvider() provider.CloudProvider {
	return &digitalocean{}
}

func (do *digitalocean) Name() string {
	return provider.DigitaloceanCloudProvider
}

func (do *digitalocean) CreateAnnotations(cloud *api.CloudSpec) (map[string]string, error) {
	return map[string]string{
		tokenAnnotationKey: cloud.Digitalocean.Token,
	}, nil
}

func (do *digitalocean) Cloud(annotations map[string]string) (*api.CloudSpec, error) {
	c := api.CloudSpec{
		Digitalocean: &api.DigitaloceanCloudSpec{},
	}

	var ok bool
	if c.Digitalocean.Token, ok = annotations[tokenAnnotationKey]; !ok {
		return nil, errors.New("no token found")
	}

	return &c, nil
}

func (do *digitalocean) CreateNode(
	ctx context.Context,
	cluster *api.Cluster,
	spec *api.NodeSpec,
) (*api.Node, error) {
	doSpec := cluster.Spec.Cloud.GetDigitalocean()

	t := token(doSpec.GetToken())
	client := godo.NewClient(oauth2.NewClient(ctx, t))

	dropletName := fmt.Sprintf(
		"%s-%s-%s",
		doSpec.DC,
		cluster.Metadata.Name,
		cluster.Metadata.UID,
	)

	createRequest := &godo.DropletCreateRequest{
		Name:   dropletName,
		Region: doSpec.DC,
		Size:   "512mb",
		Image: godo.DropletCreateImage{
			Slug: "coreos-stable",
		},
		SSHKeys: dropletKeys(doSpec.SSHKeys),
	}

	droplet, _, err := client.Droplets.Create(createRequest)
	if err != nil {
		return nil, err
	}

	n := api.Node{}
	n.Metadata.Name = droplet.Name
	n.Spec.OS = droplet.Image.Name

	return &n, nil
}

func (do *digitalocean) Nodes(ctx context.Context, cluster *api.Cluster) ([]*api.Node, error) {
	panic("not implemented")
}

var _ oauth2.TokenSource = (*token)(nil)

type token string

func (t token) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: string(t)}, nil
}

func dropletKeys(keys []string) []godo.DropletCreateSSHKey {
	dropletKeys := make([]godo.DropletCreateSSHKey, len(keys))

	for i, key := range keys {
		dropletKeys[i] = godo.DropletCreateSSHKey{
			Fingerprint: key,
		}
	}

	return dropletKeys
}
