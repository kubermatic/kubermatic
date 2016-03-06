package digitalocean

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/digitalocean/godo"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"k8s.io/kubernetes/pkg/util"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	ktemplate "github.com/kubermatic/api/template"
)

const (
	tokenAnnotationKey  = "token"
	sshKeysAnnotionsKey = "ssh-keys"
)

var _ provider.CloudProvider = (*digitalocean)(nil)

type digitalocean struct {
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
		tokenAnnotationKey:  cloud.Digitalocean.Token,
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
	dc, found := do.dcs[spec.DC]
	if !found || dc.Spec.Digitalocean == nil {
		return nil, fmt.Errorf("invalid datacenter %q", spec.DC)
	}

	if spec.Digitalocean.Type != "" {
		return nil, errors.New("digitalocean node type cannot be specified on create")
	}

	cSpec := cluster.Spec.Cloud.GetDigitalocean()
	nSpec := spec.Digitalocean

	id := string(util.NewUUID())
	dropletName := fmt.Sprintf(
		"kubermatic-%s-%s",
		cluster.Metadata.Name,
		id,
	)

	glog.V(2).Infof("dropletName %q", dropletName)

	image := godo.DropletCreateImage{Slug: "coreos-stable"}

	data := ktemplate.Data{
		SSHAuthorizedKeys: cSpec.SSHKeys,
		EtcdURL:           cluster.Address.EtcdURL,
		APIServerURL:      cluster.Address.URL,
		KubeletToken:      cluster.Address.Token,
		Region:            dc.Spec.Digitalocean.Region,
		Name:              dropletName,
	}

	tpl, err := template.
		New("cloud-config-node.yaml").
		Funcs(ktemplate.FuncMap).
		ParseFiles("template/coreos/cloud-config-node.yaml")

	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err = tpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	glog.V(2).Infof("---- template\n%s\n----", buf.String())

	t := token(cSpec.GetToken())
	client := godo.NewClient(oauth2.NewClient(ctx, t))

	createRequest := &godo.DropletCreateRequest{
		Region:            dc.Spec.Digitalocean.Region,
		Image:             image,
		Size:              nSpec.Size,
		PrivateNetworking: true,
		SSHKeys:           dropletKeys(nSpec.SSHKeys),
		Name:              dropletName,
		UserData:          buf.String(),
	}

	droplet, _, err := client.Droplets.Create(createRequest)
	if err != nil {
		return nil, err
	}

	n := api.Node{
		Metadata: api.Metadata{
			UID:  id,
			Name: droplet.Name,
			User: cluster.Metadata.User,
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
		ss := strings.SplitN(d.Name, "-", 3)

		switch {
		case len(ss) != 3: // assuming kubermatic-%s-%s format, see CreateNode
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
