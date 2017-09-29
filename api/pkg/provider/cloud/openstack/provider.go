package openstack

import (
	"errors"
	"fmt"

	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/template"
	"github.com/kubermatic/kubermatic/api/pkg/uuid"
)

const (
	tplPath = "/opt/template/nodes/openstack.yaml"

	usernameAnnotationKey       = "username"
	passwordAnnotationKey       = "password"
	tenantAnnotationKey         = "tenant"
	domainAnnotationKey         = "domain"
	networkAnnotationKey        = "network"
	floatingIPPoolAnnotationKey = "floating-ip-pool"
	securityGroupsAnnotationKey = "security-groups"
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

func (os *openstack) Initialize(cloud *api.CloudSpec, name string) (*api.CloudSpec, error) {
	return cloud, nil
}

func (os *openstack) CleanUp(*api.CloudSpec) error {
	return nil
}

func (os *openstack) MarshalCloudSpec(cloud *api.CloudSpec) (map[string]string, error) {
	as := map[string]string{
		usernameAnnotationKey:       cloud.Openstack.Username,
		passwordAnnotationKey:       cloud.Openstack.Password,
		tenantAnnotationKey:         cloud.Openstack.Tenant,
		domainAnnotationKey:         cloud.Openstack.Domain,
		networkAnnotationKey:        cloud.Openstack.Network,
		floatingIPPoolAnnotationKey: cloud.Openstack.FloatingIPPool,
		securityGroupsAnnotationKey: cloud.Openstack.SecurityGroups,
	}
	return as, nil
}

func (os *openstack) UnmarshalCloudSpec(annotations map[string]string) (*api.CloudSpec, error) {
	c := api.CloudSpec{Openstack: &api.OpenstackCloudSpec{}}

	var ok bool
	if c.Openstack.Username, ok = annotations[usernameAnnotationKey]; !ok {
		return nil, errors.New("no username found")
	}

	if c.Openstack.Password, ok = annotations[passwordAnnotationKey]; !ok {
		return nil, errors.New("no password found")
	}

	if c.Openstack.Tenant, ok = annotations[tenantAnnotationKey]; !ok {
		return nil, errors.New("no project-name found")
	}

	if c.Openstack.Domain, ok = annotations[domainAnnotationKey]; !ok {
		return nil, errors.New("no domain found")
	}

	if c.Openstack.Network, ok = annotations[networkAnnotationKey]; !ok {
		return nil, errors.New("no subnet id found")
	}

	if c.Openstack.SecurityGroups, ok = annotations[securityGroupsAnnotationKey]; !ok {
		return nil, errors.New("no security groups found")
	}

	if c.Openstack.FloatingIPPool, ok = annotations[floatingIPPoolAnnotationKey]; !ok {
		return nil, errors.New("no floating ip pool found")
	}

	return &c, nil
}

func (os *openstack) CreateNodeClass(c *api.Cluster, nSpec *api.NodeSpec, keys []v1.UserSSHKey, version *api.MasterVersion) (*v1alpha1.NodeClass, error) {
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
