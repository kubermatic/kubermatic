package openstack

import (
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud"
	goopenstack "github.com/gophercloud/gophercloud/openstack"

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

func (os *openstack) ValidateCloudSpec(cloud *kubermaticv1.CloudSpec) error {
	client, err := os.getClient(cloud)
	if err != nil {
		return fmt.Errorf("failed to create a authenticated openstack client: %v", err)
	}

	if cloud.Openstack.SecurityGroups != "" {
		if err := validateSecurityGroupsExist(client, strings.Split(cloud.Openstack.SecurityGroups, ",")); err != nil {
			return err
		}
	}

	if cloud.Openstack.Network != "" {
		_, err := getNetworkByName(client, cloud.Openstack.Network, false)
		if err != nil {
			return fmt.Errorf("failed to get network %q: %v", cloud.Openstack.Network, err)
		}
	}

	if cloud.Openstack.FloatingIPPool != "" {
		_, err := getNetworkByName(client, cloud.Openstack.FloatingIPPool, true)
		if err != nil {
			return fmt.Errorf("failed to get floating ip pool %q: %v", cloud.Openstack.FloatingIPPool, err)
		}
	}

	return nil
}

func (os *openstack) getClient(cloud *kubermaticv1.CloudSpec) (*gophercloud.ProviderClient, error) {
	dc, found := os.dcs[cloud.DatacenterName]
	if !found || dc.Spec.Openstack == nil {
		return nil, fmt.Errorf("invalid datacenter %q", cloud.DatacenterName)
	}

	opts := gophercloud.AuthOptions{
		IdentityEndpoint: dc.Spec.Openstack.AuthURL,
		Username:         cloud.Openstack.Username,
		Password:         cloud.Openstack.Password,
		DomainName:       cloud.Openstack.Domain,
	}

	osProvider, err := goopenstack.AuthenticatedClient(opts)
	if err != nil {
		return nil, err
	}
	return osProvider, nil
}

func isInitialized(cloud *kubermaticv1.CloudSpec) bool {
	return cloud.Openstack.SecurityGroups != "" &&
		cloud.Openstack.Network != "" &&
		cloud.Openstack.FloatingIPPool != ""
}

func (os *openstack) InitializeCloudProvider(cloud *kubermaticv1.CloudSpec, name string) (*kubermaticv1.CloudSpec, error) {
	if isInitialized(cloud) {
		return nil, nil
	}

	dc, found := os.dcs[cloud.DatacenterName]
	if !found || dc.Spec.Openstack == nil {
		return nil, fmt.Errorf("invalid datacenter %q", cloud.DatacenterName)
	}

	client, err := os.getClient(cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to create a authenticated openstack client: %v", err)
	}

	if cloud.Openstack.FloatingIPPool == "" {
		extNetwork, err := getExternalNetwork(client)
		if err != nil {
			return nil, err
		}
		cloud.Openstack.FloatingIPPool = extNetwork.Name
	}

	if cloud.Openstack.SecurityGroups == "" {
		g, err := createKubermaticSecurityGroup(client, name)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic security group: %v", err)
		}
		cloud.Openstack.SecurityGroups = g.Name
		cloud.Openstack.SecurityGroupCreated = true
	}

	if cloud.Openstack.Network == "" {
		network, err := createKubermaticNetwork(client, name)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic network: %v", err)
		}
		cloud.Openstack.Network = network.Name
		cloud.Openstack.NetworkCreated = true

		subnet, err := createKubermaticSubnet(client, name, network.ID, dc.Spec.Openstack.DNSServers)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic subnet: %v", err)
		}
		cloud.Openstack.SubnetID = subnet.ID

		router, err := createKubermaticRouter(client, name, cloud.Openstack.FloatingIPPool)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic router: %v", err)
		}
		cloud.Openstack.RouterID = router.ID

		if _, err = attachSubnetToRouter(client, subnet.ID, router.ID); err != nil {
			return nil, fmt.Errorf("failed to attach subnet to router: %v", err)
		}
	}

	return cloud, nil
}

func (os *openstack) CleanUpCloudProvider(cloud *kubermaticv1.CloudSpec) error {
	client, err := os.getClient(cloud)
	if err != nil {
		return fmt.Errorf("failed to create a authenticated openstack client: %v", err)
	}

	if cloud.Openstack.SecurityGroupCreated {
		for _, g := range strings.Split(cloud.Openstack.SecurityGroups, ",") {
			if err := deleteSecurityGroup(client, strings.TrimSpace(g)); err != nil {
				return fmt.Errorf("failed to delete security group %q: %v", g, err)
			}
		}
	}

	if cloud.Openstack.NetworkCreated {

		if _, err = detachSubnetFromRouter(client, cloud.Openstack.SubnetID, cloud.Openstack.RouterID); err != nil {
			return fmt.Errorf("failed to detach subnet from router: %v", err)
		}

		if err = deleteNetworkByName(client, cloud.Openstack.Network); err != nil {
			return fmt.Errorf("failed delete network %q: %v", cloud.Openstack.Network, err)
		}

		if err = deleteRouter(client, cloud.Openstack.RouterID); err != nil {
			return fmt.Errorf("failed delete router %q: %v", cloud.Openstack.RouterID, err)
		}
	}

	return nil
}

func (os *openstack) CreateNodeClass(c *kubermaticv1.Cluster, nSpec *api.NodeSpec, keys []*kubermaticv1.UserSSHKey, version *api.MasterVersion) (*v1alpha1.NodeClass, error) {
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
