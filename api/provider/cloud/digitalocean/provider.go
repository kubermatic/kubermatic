package digitalocean

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/extensions"
	"github.com/kubermatic/kubermatic/api/provider"
	"github.com/kubermatic/kubermatic/api/provider/template"
	"github.com/kubermatic/kubermatic/api/uuid"
)

const (
	tokenAnnotationKey   = "token"
	sshKeysAnnotationKey = "ssh-keys"
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

func (do *digitalocean) Initialize(cloud *api.CloudSpec, name string) (*api.CloudSpec, error) {
	return cloud, nil
}

func (do *digitalocean) CleanUp(*api.CloudSpec) error {
	return nil
}

func (do *digitalocean) MarshalCloudSpec(cloud *api.CloudSpec) (map[string]string, error) {
	as := map[string]string{
		tokenAnnotationKey:   cloud.Digitalocean.Token,
		sshKeysAnnotationKey: strings.Join(cloud.Digitalocean.SSHKeys, ","),
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

	if s, ok := annotations[sshKeysAnnotationKey]; ok && s != "" {
		c.Digitalocean.SSHKeys = strings.Split(s, ",")
	}

	return &c, nil
}

func (do *digitalocean) CreateNodeClass(c *api.Cluster, nSpec *api.NodeSpec, keys []extensions.UserSSHKey, version *api.MasterVersion) (*v1alpha1.NodeClass, error) {
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
