package digitalocean

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/digitalocean/godo"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
)

const (
	tokenAnnotationKey  = "token"
	sshKeysAnnotionsKey = "ssh-keys"
)

var _ provider.CloudProvider = (*digitalocean)(nil)

type digitalocean struct{
	dcs map[string]provider.DatacenterMeta
}

// NewCloudProvider creates a new digitalocean provider.
func NewCloudProvider(dcs map[string]provider.DatacenterMeta) provider.CloudProvider {
	return &digitalocean{
		dcs: dcs,
	}
}

func (do *digitalocean) CreateAnnotations(cloud *api.CloudSpec) (map[string]string, error) {
	return map[string]string{
		// TODO(sur): change value to cloud.Digitalocean.Token, specified in the frontend by the user
		tokenAnnotationKey:  "c465373bf74b4d8eca066c71b172a5ba19ddf4c7910a9f5a7b6e39e26697c2d6",
		sshKeysAnnotionsKey: strings.Join(cloud.Digitalocean.SSHKeys, ","),
	}, nil
}

func (do *digitalocean) Cloud(annotations map[string]string) (*api.CloudSpec, error) {
	c := api.CloudSpec{
		Digitalocean: &api.DigitaloceanCloudSpec{
			SSHKeys: []string{},
		},
	}

	var ok bool
	if c.Digitalocean.Token, ok = annotations[tokenAnnotationKey]; !ok {
		return nil, errors.New("no token found")
	}

	if s, ok := annotations[sshKeysAnnotionsKey]; ok && s != "" {
		c.Digitalocean.SSHKeys = strings.Split(s, ",")
	}

	return &c, nil
}

func (do *digitalocean) CreateNodes(
	ctx context.Context,
	cluster *api.Cluster, spec *api.NodeSpec, instances int,
) ([]*api.Node, error) {
	doSpec := cluster.Spec.Cloud.GetDigitalocean()

	dc, found := do.dcs[spec.DC]
	if !found || dc.Spec.Digitalocean == nil {
		return nil, fmt.Errorf("invalid datacenter %q", spec.DC)
	}
	if spec.Digitalocean.Type != "" {
		return nil, errors.New("digitalocean node type cannot be specified on create")
	}

	// TODO(sttts): implement instances support

	t := token(doSpec.GetToken())
	client := godo.NewClient(oauth2.NewClient(ctx, t))

	dropletName := fmt.Sprintf(
		"kubermatic-%s-%s",
		cluster.Metadata.Name,
		cluster.Metadata.UID,
	)

	image := godo.DropletCreateImage{Slug: "coreos-stable"}
	createRequest := &godo.DropletCreateRequest{
		Region:            dc.Spec.Digitalocean.Region,
		Image:             image,
		Size:              spec.Digitalocean.Size,
		PrivateNetworking: true,
		SSHKeys:           dropletKeys(spec.Digitalocean.SSHKeys),
		Name:              dropletName,
	}

	droplet, _, err := client.Droplets.Create(createRequest)
	if err != nil {
		return nil, err
	}

	n := api.Node{
		Metadata: api.Metadata{
			Name: droplet.Name,
		},
		Spec: *spec,
	}
	spec.Digitalocean.Type = image.Slug

	return []*api.Node{&n}, nil
}

func (do *digitalocean) Nodes(ctx context.Context, cluster *api.Cluster) ([]*api.Node, error) {
	doSpec := cluster.Spec.Cloud.GetDigitalocean()
	t := token(doSpec.GetToken())
	client := godo.NewClient(oauth2.NewClient(ctx, t))

	ds := []godo.Droplet{}
	opt := &godo.ListOptions{}
	for {
		droplets, resp, err := client.Droplets.List(opt)
		if err != nil {
			return nil, err
		}

		// append the current page's droplets to our list
		for _, d := range droplets {
			ds = append(ds, d)
		}

		// if we are at the last page, break out the for loop
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}

		// set the page we want for the next request
		opt.Page = page + 1
	}

	nodes := make([]*api.Node, 0, len(ds))
	for _, d := range ds {
		ss := strings.Split(d.Name, "-")

		switch {
		case len(ss) != 3: // assuming %s-%s-%s format, see CreateNode
			continue
		case ss[1] != cluster.Metadata.Name:
			continue
		}

		n := &api.Node{
			Metadata: api.Metadata{
				Name: d.Name,
				UID:  ss[2],
			},
			Spec: api.NodeSpec{
				Digitalocean: &api.DigitaloceanNodeSpec{
					Type: d.Image.Slug,
					Size: d.Size.Slug,
				},
			},
		}

		nodes = append(nodes, n)
	}

	return nodes, nil
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
