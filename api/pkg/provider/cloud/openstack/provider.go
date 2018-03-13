package openstack

import (
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud"
	goopenstack "github.com/gophercloud/gophercloud/openstack"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"

	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/template"
	"github.com/kubermatic/kubermatic/api/pkg/uuid"
)

const (
	tplPath = "/opt/template/nodes/openstack.yaml"
)

// Impl is a struct that implements CloudProvider interface
type Impl struct {
	dcs map[string]provider.DatacenterMeta
}

var _ provider.CloudProvider = &Impl{}

// NewCloudProvider creates a new openstack provider.
func NewCloudProvider(dcs map[string]provider.DatacenterMeta) *Impl {
	return &Impl{
		dcs: dcs,
	}
}

// ValidateCloudSpec validates the given CloudSpec
func (os *Impl) ValidateCloudSpec(cloud *kubermaticv1.CloudSpec) error {
	netClient, err := os.getNetClient(cloud)
	if err != nil {
		return fmt.Errorf("failed to create a authenticated openstack client: %v", err)
	}

	if cloud.Openstack.SecurityGroups != "" {
		if err := validateSecurityGroupsExist(netClient, strings.Split(cloud.Openstack.SecurityGroups, ",")); err != nil {
			return err
		}
	}

	if cloud.Openstack.Network != "" {
		_, err := getNetworkByName(netClient, cloud.Openstack.Network, false)
		if err != nil {
			return fmt.Errorf("failed to get network %q: %v", cloud.Openstack.Network, err)
		}
	}

	if cloud.Openstack.FloatingIPPool != "" {
		_, err := getNetworkByName(netClient, cloud.Openstack.FloatingIPPool, true)
		if err != nil {
			return fmt.Errorf("failed to get floating ip pool %q: %v", cloud.Openstack.FloatingIPPool, err)
		}
	}

	return nil
}

// InitializeCloudProvider initializes a cluster, in particular
// creates security group and network configuration
func (os *Impl) InitializeCloudProvider(cloud *kubermaticv1.CloudSpec, name string) (*kubermaticv1.CloudSpec, error) {
	if isInitialized(cloud) {
		return nil, nil
	}

	dc, found := os.dcs[cloud.DatacenterName]
	if !found || dc.Spec.Openstack == nil {
		return nil, fmt.Errorf("invalid datacenter %q", cloud.DatacenterName)
	}

	netClient, err := os.getNetClient(cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to create a authenticated openstack client: %v", err)
	}

	if cloud.Openstack.FloatingIPPool == "" {
		extNetwork, err := getExternalNetwork(netClient)
		if err != nil {
			return nil, err
		}
		cloud.Openstack.FloatingIPPool = extNetwork.Name
	}

	if cloud.Openstack.SecurityGroups == "" {
		g, err := createKubermaticSecurityGroup(netClient, name)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic security group: %v", err)
		}
		cloud.Openstack.SecurityGroups = g.Name
		cloud.Openstack.SecurityGroupCreated = true
	}

	if cloud.Openstack.Network == "" {
		network, err := createKubermaticNetwork(netClient, name)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic network: %v", err)
		}
		cloud.Openstack.Network = network.Name
		cloud.Openstack.NetworkCreated = true

		subnet, err := createKubermaticSubnet(netClient, name, network.ID, dc.Spec.Openstack.DNSServers)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic subnet: %v", err)
		}
		cloud.Openstack.SubnetID = subnet.ID

		router, err := createKubermaticRouter(netClient, name, cloud.Openstack.FloatingIPPool)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic router: %v", err)
		}
		cloud.Openstack.RouterID = router.ID

		if _, err = attachSubnetToRouter(netClient, subnet.ID, router.ID); err != nil {
			return nil, fmt.Errorf("failed to attach subnet to router: %v", err)
		}
	}

	return cloud, nil
}

// CleanUpCloudProvider does the clean-up in particular:
// removes security group and network configuration
func (os *Impl) CleanUpCloudProvider(cloud *kubermaticv1.CloudSpec) error {
	netClient, err := os.getNetClient(cloud)
	if err != nil {
		return fmt.Errorf("failed to create a authenticated openstack client: %v", err)
	}

	if cloud.Openstack.SecurityGroupCreated {
		for _, g := range strings.Split(cloud.Openstack.SecurityGroups, ",") {
			if err := deleteSecurityGroup(netClient, strings.TrimSpace(g)); err != nil {
				return fmt.Errorf("failed to delete security group %q: %v", g, err)
			}
		}
	}

	if cloud.Openstack.NetworkCreated {

		if _, err = detachSubnetFromRouter(netClient, cloud.Openstack.SubnetID, cloud.Openstack.RouterID); err != nil {
			return fmt.Errorf("failed to detach subnet from router: %v", err)
		}

		if err = deleteNetworkByName(netClient, cloud.Openstack.Network); err != nil {
			return fmt.Errorf("failed delete network %q: %v", cloud.Openstack.Network, err)
		}

		if err = deleteRouter(netClient, cloud.Openstack.RouterID); err != nil {
			return fmt.Errorf("failed delete router %q: %v", cloud.Openstack.RouterID, err)
		}
	}

	return nil
}

// CreateNodeClass creates a node class
func (os *Impl) CreateNodeClass(c *kubermaticv1.Cluster, nSpec *apiv1.NodeSpec, keys []*kubermaticv1.UserSSHKey, version *apiv1.MasterVersion) (*v1alpha1.NodeClass, error) {
	dc, found := os.dcs[c.Spec.Cloud.DatacenterName]
	if !found || dc.Spec.Openstack == nil {
		return nil, fmt.Errorf("invalid datacenter %q", c.Spec.Cloud.DatacenterName)
	}

	nc, err := resources.LoadNodeClassFile(tplPath, os.NodeClassName(nSpec), c, nSpec, dc, keys, version)
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

// NodeClassName generates a node class name
func (os *Impl) NodeClassName(nSpec *apiv1.NodeSpec) string {
	return fmt.Sprintf("kubermatic-%s", uuid.ShortUID(5))
}

// ValidateNodeSpec not implemented yet!
func (os *Impl) ValidateNodeSpec(cloudSpec *kubermaticv1.CloudSpec, nodeSpec *apiv1.NodeSpec) error {
	return nil
}

// GetFlavors lists available flavors for the given CloudSpec.DatacenterName and OpenstackSpec.Region
func (os *Impl) GetFlavors(cloud *kubermaticv1.CloudSpec) ([]apiv1.OpenstackSize, error) {
	authClient, err := os.getAuthClient(cloud)
	if err != nil {
		return nil, err
	}
	dc, found := os.dcs[cloud.DatacenterName]
	if !found || dc.Spec.Openstack == nil {
		return nil, fmt.Errorf("invalid datacenter %q", cloud.DatacenterName)
	}
	flavors, err := getFlavors(authClient, dc.Spec.Openstack.Region)
	if err != nil {
		return nil, err
	}

	apiSizes := []apiv1.OpenstackSize{}
	for _, flavor := range flavors {
		apiSize := apiv1.OpenstackSize{
			Slug:     flavor.Name,
			Memory:   flavor.RAM,
			VCPUs:    flavor.VCPUs,
			Disk:     flavor.Disk,
			Swap:     flavor.Swap,
			Region:   dc.Spec.Openstack.Region,
			IsPublic: flavor.IsPublic,
		}
		apiSizes = append(apiSizes, apiSize)
	}
	return apiSizes, nil
}

func (os *Impl) getAuthClient(cloud *kubermaticv1.CloudSpec) (*gophercloud.ProviderClient, error) {
	dc, found := os.dcs[cloud.DatacenterName]
	if !found || dc.Spec.Openstack == nil {
		return nil, fmt.Errorf("invalid datacenter %q", cloud.DatacenterName)
	}

	opts := gophercloud.AuthOptions{
		IdentityEndpoint: dc.Spec.Openstack.AuthURL,
		Username:         cloud.Openstack.Username,
		Password:         cloud.Openstack.Password,
		DomainName:       cloud.Openstack.Domain,
		TenantName:       cloud.Openstack.Tenant,
	}

	client, err := goopenstack.AuthenticatedClient(opts)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (os *Impl) getNetClient(cloud *kubermaticv1.CloudSpec) (*gophercloud.ServiceClient, error) {
	authClient, err := os.getAuthClient(cloud)
	if err != nil {
		return nil, err
	}

	dc, found := os.dcs[cloud.DatacenterName]
	if !found || dc.Spec.Openstack == nil {
		return nil, fmt.Errorf("invalid datacenter %q", cloud.DatacenterName)
	}

	return goopenstack.NewNetworkV2(authClient, gophercloud.EndpointOpts{Region: dc.Spec.Openstack.Region})
}

func isInitialized(cloud *kubermaticv1.CloudSpec) bool {
	return cloud.Openstack.SecurityGroups != "" &&
		cloud.Openstack.Network != "" &&
		cloud.Openstack.FloatingIPPool != ""
}
