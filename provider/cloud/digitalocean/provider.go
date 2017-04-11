package digitalocean

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"github.com/digitalocean/godo"
	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/extensions"
	"github.com/kubermatic/api/provider"
	ktemplate "github.com/kubermatic/api/template"
	"github.com/kubermatic/api/uuid"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
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

func (do *digitalocean) MarshalCloudSpec(cloud *api.CloudSpec) (map[string]string, error) {
	as := map[string]string{
		tokenAnnotationKey:  cloud.Digitalocean.Token,
		sshKeysAnnotionsKey: strings.Join(cloud.Digitalocean.SSHKeys, ","),
	}
	return as, nil
}

func (do *digitalocean) UnmarshalCloudSpec(annotations map[string]string) (*api.CloudSpec, error) {
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

func node(dc string, d *godo.Droplet) (*api.Node, error) {
	publicIP, err := d.PublicIPv4()
	if err != nil {
		return nil, err
	}
	privateIP, err := d.PrivateIPv4()
	if err != nil {
		return nil, err
	}

	n := api.Node{
		Metadata: api.Metadata{
			UID:  fmt.Sprintf("%s-%d", d.Name, d.ID),
			Name: publicIP,
		},
		Status: api.NodeStatus{
			Addresses: api.NodeAddresses{
				Public:  publicIP,
				Private: privateIP,
			},
		},
		Spec: api.NodeSpec{
			DatacenterName: dc,
			Digitalocean: &api.DigitaloceanNodeSpec{
				Type: d.Image.Slug,
				Size: d.Size.Slug,
			},
		},
	}

	return &n, nil
}

func (do *digitalocean) CreateNodes(ctx context.Context, cluster *api.Cluster, spec *api.NodeSpec, instances int, keys []extensions.UserSSHKey) ([]*api.Node, error) {
	dc, found := do.dcs[spec.DatacenterName]
	if !found || dc.Spec.Digitalocean == nil {
		return nil, fmt.Errorf("invalid datacenter %q", spec.DatacenterName)
	}

	if spec.Digitalocean.Type != "" {
		return nil, errors.New("digitalocean node type cannot be specified on create")
	}

	cSpec := cluster.Spec.Cloud.GetDigitalocean()
	nSpec := spec.Digitalocean
	created := make([]*api.Node, 0, instances)

	for i := 0; i < instances; i++ {
		id := uuid.ShortUID(5)
		dropletName := fmt.Sprintf(
			"kubermatic-%s-%s",
			cluster.Metadata.Name,
			id,
		)

		glog.V(2).Infof("dropletName %q", dropletName)

		clientKC, err := cluster.CreateKeyCert(dropletName, []string{})
		if err != nil {
			return created, err
		}

		var skeys []string
		for _, k := range keys {
			skeys = append(skeys, k.PublicKey)
		}

		image := godo.DropletCreateImage{Slug: "coreos-stable"}
		data := ktemplate.Data{
			DC:                spec.DatacenterName,
			ClusterName:       cluster.Metadata.Name,
			SSHAuthorizedKeys: append(cSpec.SSHKeys, skeys...),
			EtcdURL:           cluster.Address.EtcdURL,
			APIServerURL:      cluster.Address.URL,
			Region:            dc.Spec.Digitalocean.Region,
			Name:              dropletName,
			ClientKey:         clientKC.Key.Base64(),
			ClientCert:        clientKC.Cert.Base64(),
			RootCACert:        cluster.Status.RootCA.Cert.Base64(),
			ApiserverPubSSH:   cluster.Status.ApiserverSSH,
			ApiserverToken:    cluster.Address.Token,
			FlannelCIDR:       cluster.Spec.Cloud.Network.Flannel.CIDR,
			AutoUpdate:        true,
		}

		tpl, err := template.
			New("do-cloud-config-node.yaml").
			Funcs(ktemplate.FuncMap).
			ParseFiles("template/coreos/do-cloud-config-node.yaml")

		if err != nil {
			return created, err
		}

		var buf bytes.Buffer
		if err = tpl.Execute(&buf, data); err != nil {
			return created, err
		}

		glog.V(2).Infof("---- template\n%s\n----", buf.String())

		t := token(cSpec.GetToken())
		client := godo.NewClient(oauth2.NewClient(ctx, t))

		createRequest := &godo.DropletCreateRequest{
			Region:            dc.Spec.Digitalocean.Region,
			Image:             image,
			Size:              nSpec.Size,
			PrivateNetworking: true,
			SSHKeys:           dropletKeys(nSpec.SSHKeyFingerprints),
			Name:              dropletName,
			UserData:          buf.String(),
		}

		droplet, _, err := client.Droplets.Create(createRequest)
		if err != nil {
			return created, err
		}

		n, err := node(cluster.Spec.Cloud.DatacenterName, droplet)
		if err != nil {
			return created, err
		}

		created = append(created, n)
	}

	return created, nil
}

func (do *digitalocean) InitializeCloudSpec(c *api.Cluster) error {
	return nil
}

func (do *digitalocean) DeleteNodes(ctx context.Context, c *api.Cluster, UIDs []string) error {
	doSpec := c.Spec.Cloud.GetDigitalocean()
	t := token(doSpec.GetToken())
	client := godo.NewClient(oauth2.NewClient(ctx, t))

	ids := make([]int, len(UIDs))
	for i, UID := range UIDs {
		ss := strings.Split(UID, "-")

		switch {
		case len(ss) < 4: // assuming kubermatic-%s-%s-%d format, see CreateNode and node
			return errors.New("invalid UID")
		case strings.Join(ss[1:len(ss)-2], "-") != c.Metadata.Name:
			return errors.New("cluster name mismatch")
		}

		id, err := strconv.Atoi(ss[len(ss)-1])
		if err != nil {
			return err
		}

		ids[i] = id
	}

	for _, id := range ids {
		glog.V(7).Infof("deleting %d", id)

		res, err := client.Droplets.Delete(id)
		if err != nil {
			return err
		}

		glog.V(7).Infof("digital ocean delete response %+v", res)
	}

	return nil
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
		case len(ss) < 3: // assuming kubermatic-%s-%s format, see CreateNode
			continue
		case strings.Join(ss[1:len(ss)-1], "-") != cluster.Metadata.Name:
			continue
		}

		n, err := node(cluster.Spec.Cloud.DatacenterName, &d)
		if err != nil {
			glog.Error(err)
			continue
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

func (do *digitalocean) CleanUp(c *api.Cluster) error {
	return nil
}
