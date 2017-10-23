package digitalocean

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	"github.com/kubermatic/kubermatic/api"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/template"
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

func (do *digitalocean) Validate(cloud *kubermaticv1.CloudSpec) error {
	client := godo.NewClient(oauth2.NewClient(context.Background(), token(cloud.Digitalocean.Token)))
	_, _, err := client.Account.Get(context.Background())
	return err
}

func (do *digitalocean) Initialize(cloud *kubermaticv1.CloudSpec, name string) (*kubermaticv1.CloudSpec, error) {
	return cloud, nil
}

func (do *digitalocean) CleanUp(*kubermaticv1.CloudSpec) error {
	return nil
}

func (do *digitalocean) CreateNodeClass(c *kubermaticv1.Cluster, nSpec *api.NodeSpec, keys []*kubermaticv1.UserSSHKey, version *api.MasterVersion) (*v1alpha1.NodeClass, error) {
	dc, found := do.dcs[c.Spec.Cloud.DatacenterName]
	if !found || dc.Spec.Digitalocean == nil {
		return nil, fmt.Errorf("invalid datacenter %q", c.Spec.Cloud.DatacenterName)
	}

	nc, err := resources.LoadNodeClassFile(tplPath, do.GetNodeClassName(nSpec), c, nSpec, dc, keys, version)
	if err != nil {
		return nil, fmt.Errorf("could not load nodeclass: %v", err)
	}

	client, err := c.GetNodesetClient()
	if err != nil {
		return nil, fmt.Errorf("could not get nodeclass client: %v", err)
	}

	cnc, err := client.NodesetV1alpha1().NodeClasses().Create(nc)
	if err != nil {
		return nil, fmt.Errorf("could not create nodeclass: %v", err)
	}

	return cnc, nil
}

func (do *digitalocean) GetNodeClassName(nSpec *api.NodeSpec) string {
	return fmt.Sprintf("kubermatic-%s-%s-%s", "coreos", nSpec.Digitalocean.Size, uuid.ShortUID(5))
}

var _ oauth2.TokenSource = (*token)(nil)

type token string

func (t token) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: string(t)}, nil
}
