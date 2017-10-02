package openstack

import (
	"fmt"

	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	"github.com/kubermatic/kubermatic/api"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/template"
	"github.com/kubermatic/kubermatic/api/pkg/uuid"
)

const (
	tplPath = "/opt/template/nodes/openstack.yaml"
)

type openstack struct {
	dcs map[string]provider.DatacenterMeta
}

// NewCloudProvider creates a new digitalocean provider.
func NewCloudProvider(dcs map[string]provider.DatacenterMeta) provider.CloudProvider {
	return &openstack{
		dcs: dcs,
	}
}

func (os *openstack) Initialize(cloud *kubermaticv1.CloudSpec, name string) (*kubermaticv1.CloudSpec, error) {
	return cloud, nil
}

func (os *openstack) CleanUp(*kubermaticv1.CloudSpec) error {
	return nil
}

func (os *openstack) CreateNodeClass(c *kubermaticv1.Cluster, nSpec *api.NodeSpec, keys []kubermaticv1.UserSSHKey, version *api.MasterVersion) (*v1alpha1.NodeClass, error) {
	dc, found := os.dcs[c.Spec.Cloud.DatacenterName]
	if !found || dc.Spec.Openstack == nil {
		return nil, fmt.Errorf("invalid datacenter %q", c.Spec.Cloud.DatacenterName)
	}

	nc, err := resources.LoadNodeClassFile(tplPath, os.GetNodeClassName(nSpec), c, nSpec, dc, keys, version)
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

func (os *openstack) GetNodeClassName(nSpec *api.NodeSpec) string {
	return fmt.Sprintf("kubermatic-%s", uuid.ShortUID(5))
}
