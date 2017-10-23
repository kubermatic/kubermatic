package openstack

import (
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud"
	goopenstack "github.com/gophercloud/gophercloud/openstack"
	ossecuritygroups "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	osecruritygrouprules "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	"github.com/gophercloud/gophercloud/pagination"
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

func (os *openstack) Validate(*kubermaticv1.CloudSpec) error {
	panic("implement me")
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

func getAllSecurityGroups(client *gophercloud.ProviderClient) ([]ossecuritygroups.SecGroup, error) {
	netClient, err := goopenstack.NewNetworkV2(client, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}
	allGroups := []ossecuritygroups.SecGroup{}

	pager := ossecuritygroups.List(netClient, ossecuritygroups.ListOpts{})
	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		securityGroups, err := ossecuritygroups.ExtractGroups(page)
		if err != nil {
			return false, err
		}
		allGroups = append(allGroups, securityGroups...)
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return allGroups, nil
}

func validateSecurityGroupsExist(client *gophercloud.ProviderClient, securityGroups []string) error {
	existingGroups, err := getAllSecurityGroups(client)
	if err != nil {
		return err
	}

	for _, sg := range securityGroups {
		found := false
		for _, esg := range existingGroups {
			if esg.Name == sg {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("specified security group %s not found", sg)
		}
	}
	return nil
}

func deleteSecurityGroup(client *gophercloud.ProviderClient, sgName string) error {
	securityGroups, err := getAllSecurityGroups(client)
	if err != nil {
		return err
	}

	for _, sg := range securityGroups {
		if sg.Name == sgName {
			netClient, err := goopenstack.NewNetworkV2(client, gophercloud.EndpointOpts{})
			if err != nil {
				return err
			}
			res := ossecuritygroups.Delete(netClient, sg.ID)
			if res.Err != nil {
				return res.Err
			}
			if err := res.ExtractErr(); err != nil {
				return err
			}
		}
	}
	return nil
}

func createKubermaticSecurityGroup(client *gophercloud.ProviderClient, clusterName string) (*ossecuritygroups.SecGroup, error) {
	netClient, err := goopenstack.NewNetworkV2(client, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	gres := ossecuritygroups.Create(netClient, ossecuritygroups.CreateOpts{
		Name:        "kubermatic-" + clusterName,
		Description: "Contains security rules for the kubermatic worker nodes",
	})
	if gres.Err != nil {
		return nil, gres.Err
	}
	g, err := gres.Extract()
	if err != nil {
		return nil, err
	}

	rules := []osecruritygrouprules.CreateOpts{
		{
			// Allows ipv4 traffic within this group
			Direction:     osecruritygrouprules.DirIngress,
			EtherType:     osecruritygrouprules.EtherType4,
			SecGroupID:    g.ID,
			RemoteGroupID: g.ID,
		},
		{
			// Allows ipv6 traffic within this group
			Direction:     osecruritygrouprules.DirIngress,
			EtherType:     osecruritygrouprules.EtherType6,
			SecGroupID:    g.ID,
			RemoteGroupID: g.ID,
		},
		{
			// Allows ssh from external
			Direction:    osecruritygrouprules.DirIngress,
			EtherType:    osecruritygrouprules.EtherType4,
			SecGroupID:   g.ID,
			PortRangeMin: provider.DefaultSSHPort,
			PortRangeMax: provider.DefaultSSHPort,
			Protocol:     osecruritygrouprules.ProtocolTCP,
		},
	}

	for _, opts := range rules {
		rres := osecruritygrouprules.Create(netClient, opts)
		if rres.Err != nil {
			return nil, rres.Err
		}
		_, err := rres.Extract()
		if err != nil {
			return nil, err
		}
	}

	return g, nil
}

func (os *openstack) Initialize(cloud *kubermaticv1.CloudSpec, name string) (*kubermaticv1.CloudSpec, error) {
	client, err := os.getClient(cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to create a authenticated openstack client: %v", err)
	}
	if cloud.Openstack.SecurityGroups != "" {
		//User specified a security group
		securityGroups := strings.Split(cloud.Openstack.SecurityGroups, ",")
		for i, g := range securityGroups {
			securityGroups[i] = strings.TrimSpace(g)
		}
		err := validateSecurityGroupsExist(client, securityGroups)
		if err != nil {
			return nil, err
		}
		cloud.Openstack.SecurityGroups = strings.Join(securityGroups, ",")
	} else {
		g, err := createKubermaticSecurityGroup(client, name)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic security group: %v", err)
		}
		cloud.Openstack.SecurityGroups = g.Name
		cloud.Openstack.SecurityGroupCreated = true
	}

	return cloud, nil
}

func (os *openstack) CleanUp(cloud *kubermaticv1.CloudSpec) error {
	if cloud.Openstack.SecurityGroupCreated {
		client, err := os.getClient(cloud)
		if err != nil {
			return fmt.Errorf("failed to create a authenticated openstack client: %v", err)
		}

		securityGroups := strings.Split(cloud.Openstack.SecurityGroups, ",")
		for _, g := range securityGroups {
			err := deleteSecurityGroup(client, strings.TrimSpace(g))
			if err != nil {
				return err
			}
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
