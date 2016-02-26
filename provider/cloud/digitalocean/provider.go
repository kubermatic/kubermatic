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
	tokenAnnotationKey  = "token"
	regionAnnotationKey = "region"
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
		// TODO(sur): change value to cloud.Digitalocean.Token, specified in the frontend by the user
		tokenAnnotationKey:  "c465373bf74b4d8eca066c71b172a5ba19ddf4c7910a9f5a7b6e39e26697c2d6",
		regionAnnotationKey: cloud.Digitalocean.Region,
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

	if c.Digitalocean.Region, ok = annotations[regionAnnotationKey]; !ok {
		return nil, errors.New("no region found")
	}

	return &c, nil
}

func (do *digitalocean) CreateNode(
	ctx context.Context,
	cluster *api.Cluster, spec *api.NodeSpec,
) (*api.Node, error) {
	doSpec := cluster.Spec.Cloud.GetDigitalocean()
	node := spec.Digitalocean

	t := token(doSpec.GetToken())
	client := godo.NewClient(oauth2.NewClient(ctx, t))

	dropletName := fmt.Sprintf(
		"%s-%s-%s",
		doSpec.Region,
		cluster.Metadata.Name,
		cluster.Metadata.UID,
	)

	createRequest := &godo.DropletCreateRequest{
		Region:            doSpec.Region,
		Image:             godo.DropletCreateImage{Slug: "coreos-stable"},
		Size:              node.Size,
		PrivateNetworking: true,
		SSHKeys:           dropletKeys(node.SSHKeys),
		Name:              dropletName,
	}

	droplet, _, err := client.Droplets.Create(createRequest)
	if err != nil {
		return nil, err
	}

	n := api.Node{}
	n.Metadata.Name = droplet.Name
	n.Spec.Digitalocean = node

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
