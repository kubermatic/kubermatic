package otc

import (
	"errors"
	"fmt"
	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/extensions"
	"github.com/kubermatic/kubermatic/api/provider"
	"github.com/kubermatic/kubermatic/api/provider/template"
	"github.com/kubermatic/kubermatic/api/uuid"
)

const (
	tplPath = "/opt/template/nodes/otc.yaml"

	usernameAnnotationKey       = "username"
	passwordAnnotationKey       = "password"
	projectAnnotationKey        = "project"
	domainAnnotationKey         = "domain"
	subnetIDAnnotationKey       = "subnet-id"
	securityGroupsAnnotationKey = "security-groups"
)

type otc struct {
	dcs map[string]provider.DatacenterMeta
}

// NewCloudProvider creates a new digitalocean provider.
func NewCloudProvider(dcs map[string]provider.DatacenterMeta) provider.CloudProvider {
	return &otc{
		dcs: dcs,
	}
}

func (os *otc) Initialize(cloud *api.CloudSpec, name string) (*api.CloudSpec, error) {
	return cloud, nil
}

func (os *otc) CleanUp(*api.CloudSpec) error {
	return nil
}

func (os *otc) MarshalCloudSpec(cloud *api.CloudSpec) (map[string]string, error) {
	as := map[string]string{
		usernameAnnotationKey:       cloud.OTC.Username,
		passwordAnnotationKey:       cloud.OTC.Password,
		projectAnnotationKey:        cloud.OTC.Project,
		domainAnnotationKey:         cloud.OTC.Domain,
		subnetIDAnnotationKey:       cloud.OTC.SubnetID,
		securityGroupsAnnotationKey: cloud.OTC.SecurityGroups,
	}
	return as, nil
}

func (os *otc) UnmarshalCloudSpec(annotations map[string]string) (*api.CloudSpec, error) {
	c := api.CloudSpec{OTC: &api.OTCCloudSpec{}}

	var ok bool
	if c.OTC.Username, ok = annotations[usernameAnnotationKey]; !ok {
		return nil, errors.New("no username found")
	}

	if c.OTC.Password, ok = annotations[passwordAnnotationKey]; !ok {
		return nil, errors.New("no password found")
	}

	if c.OTC.Project, ok = annotations[projectAnnotationKey]; !ok {
		return nil, errors.New("no project-name found")
	}

	if c.OTC.Domain, ok = annotations[domainAnnotationKey]; !ok {
		return nil, errors.New("no domain found")
	}

	if c.OTC.SubnetID, ok = annotations[subnetIDAnnotationKey]; !ok {
		return nil, errors.New("no subnet id found")
	}

	if c.OTC.SecurityGroups, ok = annotations[securityGroupsAnnotationKey]; !ok {
		return nil, errors.New("no security groups found")
	}

	return &c, nil
}

func (os *otc) CreateNodeClass(c *api.Cluster, nSpec *api.NodeSpec, keys []extensions.UserSSHKey) (*v1alpha1.NodeClass, error) {
	dc, found := os.dcs[c.Spec.Cloud.DatacenterName]
	if !found || dc.Spec.OTC == nil {
		return nil, fmt.Errorf("invalid datacenter %q", c.Spec.Cloud.DatacenterName)
	}

	nc, err := resources.LoadNodeClassFile(tplPath, os.GetNodeClassName(nSpec), c, nSpec, dc, keys)
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

func (os *otc) GetNodeClassName(nSpec *api.NodeSpec) string {
	return fmt.Sprintf("kubermatic-%s", uuid.ShortUID(5))
}
