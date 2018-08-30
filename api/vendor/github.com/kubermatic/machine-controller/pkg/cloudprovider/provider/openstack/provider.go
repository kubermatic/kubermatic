package openstack

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/golang/glog"

	"github.com/gophercloud/gophercloud"
	goopenstack "github.com/gophercloud/gophercloud/openstack"
	osextendedstatus "github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/extendedstatus"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	osservers "github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/pagination"

	"github.com/kubermatic/machine-controller/pkg/cloudprovider/cloud"
	cloudprovidererrors "github.com/kubermatic/machine-controller/pkg/cloudprovider/errors"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/instance"
	"github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
)

type provider struct {
	configVarResolver *providerconfig.ConfigVarResolver
}

// New returns a openstack provider
func New(configVarResolver *providerconfig.ConfigVarResolver) cloud.Provider {
	return &provider{configVarResolver: configVarResolver}
}

type RawConfig struct {
	// Auth details
	IdentityEndpoint providerconfig.ConfigVarString `json:"identityEndpoint"`
	Username         providerconfig.ConfigVarString `json:"username"`
	Password         providerconfig.ConfigVarString `json:"password"`
	DomainName       providerconfig.ConfigVarString `json:"domainName"`
	TenantName       providerconfig.ConfigVarString `json:"tenantName"`
	TokenID          providerconfig.ConfigVarString `json:"tokenId"`

	// Machine details
	Image            providerconfig.ConfigVarString   `json:"image"`
	Flavor           providerconfig.ConfigVarString   `json:"flavor"`
	SecurityGroups   []providerconfig.ConfigVarString `json:"securityGroups"`
	Network          providerconfig.ConfigVarString   `json:"network"`
	Subnet           providerconfig.ConfigVarString   `json:"subnet"`
	FloatingIPPool   providerconfig.ConfigVarString   `json:"floatingIpPool"`
	AvailabilityZone providerconfig.ConfigVarString   `json:"availabilityZone"`
	Region           providerconfig.ConfigVarString   `json:"region"`
	Tags             map[string]string                `json:"tags"`
}

type Config struct {
	IdentityEndpoint string
	Username         string
	Password         string
	DomainName       string
	TenantName       string
	TokenID          string

	// Machine details
	Image            string
	Flavor           string
	SecurityGroups   []string
	Network          string
	Subnet           string
	FloatingIPPool   string
	AvailabilityZone string
	Region           string

	Tags map[string]string
}

const (
	machineUIDMetaKey = "machine-uid"
	securityGroupName = "kubernetes-v1"

	instanceReadyCheckPeriod  = 2 * time.Second
	instanceReadyCheckTimeout = 2 * time.Minute
)

// Protects floating ip assignment
var floatingIPAssignLock = &sync.Mutex{}

func (p *provider) getConfig(s runtime.RawExtension) (*Config, *providerconfig.Config, *RawConfig, error) {
	pconfig := providerconfig.Config{}
	err := json.Unmarshal(s.Raw, &pconfig)
	if err != nil {
		return nil, nil, nil, err
	}
	var rawConfig RawConfig
	err = json.Unmarshal(pconfig.CloudProviderSpec.Raw, &rawConfig)
	if err != nil {
		return nil, nil, nil, err
	}
	c := Config{}
	c.IdentityEndpoint, err = p.configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.IdentityEndpoint, "OS_AUTH_URL")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get the value of \"identityEndpoint\" field, error = %v", err)
	}
	c.Username, err = p.configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Username, "OS_USER_NAME")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get the value of \"username\" field, error = %v", err)
	}
	c.Password, err = p.configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Password, "OS_PASSWORD")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get the value of \"password\" field, error = %v", err)
	}
	// We ignore errors here because the OS domain is only required when using Identity API V3
	c.DomainName, _ = p.configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.DomainName, "OS_DOMAIN_NAME")
	c.TenantName, err = p.configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.TenantName, "OS_TENANT_NAME")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get the value of \"tenantName\" field, error = %v", err)
	}
	c.TokenID, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.TokenID)
	if err != nil {
		return nil, nil, nil, err
	}
	c.Image, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Image)
	if err != nil {
		return nil, nil, nil, err
	}
	c.Flavor, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Flavor)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, securityGroup := range rawConfig.SecurityGroups {
		securityGroupValue, err := p.configVarResolver.GetConfigVarStringValue(securityGroup)
		if err != nil {
			return nil, nil, nil, err
		}
		c.SecurityGroups = append(c.SecurityGroups, securityGroupValue)
	}
	c.Network, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Network)
	if err != nil {
		return nil, nil, nil, err
	}
	c.Subnet, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Subnet)
	if err != nil {
		return nil, nil, nil, err
	}
	c.FloatingIPPool, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.FloatingIPPool)
	if err != nil {
		return nil, nil, nil, err
	}
	c.AvailabilityZone, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.AvailabilityZone)
	if err != nil {
		return nil, nil, nil, err
	}
	c.Region, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Region)
	if err != nil {
		return nil, nil, nil, err
	}
	c.Tags = rawConfig.Tags
	if c.Tags == nil {
		c.Tags = map[string]string{}
	}

	return &c, &pconfig, &rawConfig, err
}

func setProviderConfig(rawConfig RawConfig, s runtime.RawExtension) (runtime.RawExtension, error) {
	pconfig := providerconfig.Config{}
	err := json.Unmarshal(s.Raw, &pconfig)
	if err != nil {
		return s, err
	}
	rawCloudProviderSpec, err := json.Marshal(rawConfig)
	if err != nil {
		return s, err
	}
	pconfig.CloudProviderSpec = runtime.RawExtension{Raw: rawCloudProviderSpec}
	rawPconfig, err := json.Marshal(pconfig)
	if err != nil {
		return s, err
	}

	return runtime.RawExtension{Raw: rawPconfig}, nil
}

func getClient(c *Config) (*gophercloud.ProviderClient, error) {
	opts := gophercloud.AuthOptions{
		IdentityEndpoint: c.IdentityEndpoint,
		Username:         c.Username,
		Password:         c.Password,
		DomainName:       c.DomainName,
		TenantName:       c.TenantName,
		TokenID:          c.TokenID,
	}

	return goopenstack.AuthenticatedClient(opts)
}

func (p *provider) AddDefaults(spec v1alpha1.MachineSpec) (v1alpha1.MachineSpec, bool, error) {
	var changed bool

	c, _, rawConfig, err := p.getConfig(spec.ProviderConfig)
	if err != nil {
		return spec, changed, cloudprovidererrors.TerminalError{
			Reason:  v1alpha1.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}

	client, err := getClient(c)
	if err != nil {
		return spec, changed, osErrorToTerminalError(err, "failed to get a openstack client")
	}

	if c.Region == "" {
		glog.V(4).Infof("Trying to default region for machine '%s'...", spec.Name)
		regions, err := getRegions(client)
		if err != nil {
			return spec, changed, osErrorToTerminalError(err, "failed to get regions")
		}
		if len(regions) == 1 {
			glog.V(4).Infof("Defaulted region for machine '%s' to '%s'", spec.Name, regions[0].ID)
			changed = true
			rawConfig.Region.Value = regions[0].ID
		} else {
			return spec, changed, fmt.Errorf("could not default region because got '%v' results", len(regions))
		}
	}

	if c.AvailabilityZone == "" {
		glog.V(4).Infof("Trying to default availability zone for machine '%s'...", spec.Name)
		availabilityZones, err := getAvailabilityZones(client, c.Region)
		if err != nil {
			return spec, changed, osErrorToTerminalError(err, "failed to get availability zones")
		}
		if len(availabilityZones) == 1 {
			glog.V(4).Infof("Defaulted availability zone for machine '%s' to '%s'", spec.Name, availabilityZones[0].ZoneName)
			changed = true
			rawConfig.AvailabilityZone.Value = availabilityZones[0].ZoneName
		}
	}

	if c.Network == "" {
		glog.V(4).Infof("Trying to default network for machine '%s'...", spec.Name)
		net, err := getDefaultNetwork(client, c.Region)
		if err != nil {
			return spec, changed, osErrorToTerminalError(err, "failed to default network")
		}
		if net != nil {
			glog.V(4).Infof("Defaulted network for machine '%s' to '%s'", spec.Name, net.Name)
			// Use the id as the name may not be unique
			rawConfig.Network.Value = net.ID
			changed = true
		}
	}

	if c.Subnet == "" {
		networkID := c.Network
		if rawConfig.Network.Value != "" {
			networkID = rawConfig.Network.Value
		}

		net, err := getNetwork(client, c.Region, networkID)
		if err != nil {
			return spec, changed, osErrorToTerminalError(err, fmt.Sprintf("failed to get network for subnet defaulting '%s", networkID))
		}
		subnet, err := getDefaultSubnet(client, net, c.Region)
		if err != nil {
			return spec, changed, osErrorToTerminalError(err, "error defaulting subnet")
		}
		if subnet != nil {
			glog.V(4).Infof("Defaulted subnet for machine '%s' to '%s'", spec.Name, *subnet)
			rawConfig.Subnet.Value = *subnet
			changed = true
		}
	}

	spec.ProviderConfig, err = setProviderConfig(*rawConfig, spec.ProviderConfig)
	if err != nil {
		return spec, changed, osErrorToTerminalError(err, "error marshaling providerconfig")
	}
	return spec, changed, nil
}

func (p *provider) Validate(spec v1alpha1.MachineSpec) error {
	c, _, _, err := p.getConfig(spec.ProviderConfig)
	if err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	client, err := getClient(c)
	if err != nil {
		return fmt.Errorf("failed to get a openstack client: %v", err)
	}

	// Required fields
	if _, err := getRegion(client, c.Region); err != nil {
		return fmt.Errorf("failed to get region %q: %v", c.Region, err)
	}

	if _, err := getImageByName(client, c.Region, c.Image); err != nil {
		return fmt.Errorf("failed to get image %q: %v", c.Image, err)
	}

	if _, err := getFlavor(client, c.Region, c.Flavor); err != nil {
		return fmt.Errorf("failed to get flavor %q: %v", c.Flavor, err)
	}

	if _, err := getNetwork(client, c.Region, c.Network); err != nil {
		return fmt.Errorf("failed to get network %q: %v", c.Network, err)
	}

	if _, err := getSubnet(client, c.Region, c.Subnet); err != nil {
		return fmt.Errorf("failed to get subnet %q: %v", c.Subnet, err)
	}

	if c.FloatingIPPool != "" {
		if _, err := getNetwork(client, c.Region, c.FloatingIPPool); err != nil {
			return fmt.Errorf("failed to get floating ip pool %q: %v", c.FloatingIPPool, err)
		}
	}

	if _, err := getAvailabilityZone(client, c.Region, c.AvailabilityZone); err != nil {
		return fmt.Errorf("failed to get availability zone %q: %v", c.AvailabilityZone, err)
	}

	// Optional fields
	if len(c.SecurityGroups) != 0 {
		for _, s := range c.SecurityGroups {
			if _, err := getSecurityGroup(client, c.Region, s); err != nil {
				return fmt.Errorf("failed to get security group %q: %v", s, err)
			}
		}
	}

	// validate reserved tags
	if _, ok := c.Tags[machineUIDMetaKey]; ok {
		return fmt.Errorf("the tag with the given name =%s is reserved, choose a different one", machineUIDMetaKey)
	}

	return nil
}

func (p *provider) Create(machine *v1alpha1.Machine, _ cloud.MachineUpdater, userdata string) (instance.Instance, error) {
	c, _, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, cloudprovidererrors.TerminalError{
			Reason:  v1alpha1.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}

	client, err := getClient(c)
	if err != nil {
		return nil, osErrorToTerminalError(err, "failed to get a openstack client")
	}

	flavor, err := getFlavor(client, c.Region, c.Flavor)
	if err != nil {
		return nil, osErrorToTerminalError(err, fmt.Sprintf("failed to get flavor %s", c.Flavor))
	}

	image, err := getImageByName(client, c.Region, c.Image)
	if err != nil {
		return nil, osErrorToTerminalError(err, fmt.Sprintf("failed to get image %s", c.Image))
	}

	network, err := getNetwork(client, c.Region, c.Network)
	if err != nil {
		return nil, osErrorToTerminalError(err, fmt.Sprintf("failed to get network %s", c.Network))
	}

	var ip *floatingips.FloatingIP
	if c.FloatingIPPool != "" {
		floatingIPAssignLock.Lock()
		defer floatingIPAssignLock.Unlock()
		floatingIPPool, err := getNetwork(client, c.Region, c.FloatingIPPool)
		if err != nil {
			return nil, osErrorToTerminalError(err, fmt.Sprintf("failed to get floating ip pool %q", c.FloatingIPPool))
		}

		freeFloatingIps, err := getFreeFloatingIPs(client, c.Region, floatingIPPool)
		if err != nil {
			return nil, osErrorToTerminalError(err, "failed to get free floating ips")
		}

		if len(freeFloatingIps) < 1 {
			ip, err = createFloatingIP(client, c.Region, floatingIPPool)
			if err != nil {
				return nil, osErrorToTerminalError(err, "failed to allocate a floating ip")
			}
		} else {
			ip = &freeFloatingIps[0]
		}
	}

	securityGroups := c.SecurityGroups
	if len(securityGroups) == 0 {
		glog.V(2).Infof("creating security group %s for worker nodes", securityGroupName)
		err = ensureKubernetesSecurityGroupExist(client, c.Region, securityGroupName)
		if err != nil {
			return nil, err
		}
		securityGroups = append(securityGroups, securityGroupName)
	}

	// we check against reserved tags in Validation method
	allTags := c.Tags
	allTags[machineUIDMetaKey] = string(machine.UID)

	serverOpts := osservers.CreateOpts{
		Name:             machine.Spec.Name,
		FlavorRef:        flavor.ID,
		ImageRef:         image.ID,
		UserData:         []byte(userdata),
		SecurityGroups:   securityGroups,
		AvailabilityZone: c.AvailabilityZone,
		Networks:         []osservers.Network{{UUID: network.ID}},
		Metadata:         allTags,
	}
	computeClient, err := goopenstack.NewComputeV2(client, gophercloud.EndpointOpts{Availability: gophercloud.AvailabilityPublic, Region: c.Region})
	if err != nil {
		return nil, osErrorToTerminalError(err, "failed to get compute client")
	}

	var server serverWithExt
	err = osservers.Create(computeClient, keypairs.CreateOptsExt{
		CreateOptsBuilder: serverOpts,
		KeyName:           "",
	}).ExtractInto(&server)
	if err != nil {
		return nil, osErrorToTerminalError(err, "failed to create server")
	}

	if ip != nil {
		// if we want to assign a floating ip to the instance, we have to wait until it is running
		// otherwise the instance has no port in the desired network
		instanceIsReady := func() (bool, error) {
			currentServer, err := osservers.Get(computeClient, server.ID).Extract()
			if err != nil {
				tErr := osErrorToTerminalError(err, fmt.Sprintf("failed to get current instance %s", server.ID))
				if isTerminalErr, _, _ := cloudprovidererrors.IsTerminalError(tErr); isTerminalErr {
					return true, tErr
				}
				// Only log the error but don't exit. in case of a network failure we want to retry
				glog.V(2).Infof("failed to get current instance %s: %v", server.ID, err)
				return false, nil
			}
			if currentServer.Status == "ACTIVE" {
				return true, nil
			}
			return false, nil
		}

		if err := wait.Poll(instanceReadyCheckPeriod, instanceReadyCheckTimeout, instanceIsReady); err != nil {
			return nil, fmt.Errorf("failed to wait for instance to be running. unable to proceed. %v", err)
		}

		if err := assignFloatingIP(client, c.Region, ip.ID, server.ID, network.ID); err != nil {
			return nil, fmt.Errorf("failed to assign a floating ip: %v", err)
		}
	}

	return &osInstance{server: &server}, nil
}

func (p *provider) Delete(machine *v1alpha1.Machine, _ cloud.MachineUpdater) error {
	instance, err := p.Get(machine)
	if err != nil {
		if err == cloudprovidererrors.ErrInstanceNotFound {
			return nil
		}
		return err
	}

	c, _, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return cloudprovidererrors.TerminalError{
			Reason:  v1alpha1.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}

	client, err := getClient(c)
	if err != nil {
		return osErrorToTerminalError(err, "failed to get a openstack client")
	}

	computeClient, err := goopenstack.NewComputeV2(client, gophercloud.EndpointOpts{Availability: gophercloud.AvailabilityPublic, Region: c.Region})
	if err != nil {
		return osErrorToTerminalError(err, "failed to get compute client")
	}

	err = osservers.Delete(computeClient, instance.ID()).ExtractErr()
	return osErrorToTerminalError(err, "failed to delete instance")
}

func (p *provider) Get(machine *v1alpha1.Machine) (instance.Instance, error) {
	c, _, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, cloudprovidererrors.TerminalError{
			Reason:  v1alpha1.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}

	client, err := getClient(c)
	if err != nil {
		return nil, osErrorToTerminalError(err, "failed to get a openstack client")
	}

	computeClient, err := goopenstack.NewComputeV2(client, gophercloud.EndpointOpts{Availability: gophercloud.AvailabilityPublic, Region: c.Region})
	if err != nil {
		return nil, osErrorToTerminalError(err, "failed to get compute client")
	}

	var allServers []serverWithExt
	pager := osservers.List(computeClient, osservers.ListOpts{Name: machine.Spec.Name})
	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		var servers []serverWithExt
		err = osservers.ExtractServersInto(page, &servers)
		if err != nil {
			return false, osErrorToTerminalError(err, "failed to extract instance info")
		}
		allServers = append(allServers, servers...)
		return true, nil
	})
	if err != nil {
		return nil, osErrorToTerminalError(err, "failed to list instances")
	}

	for i, s := range allServers {
		if s.Metadata[machineUIDMetaKey] == string(machine.UID) {
			return &osInstance{server: &allServers[i]}, nil
		}
	}

	return nil, cloudprovidererrors.ErrInstanceNotFound
}

func (p *provider) GetCloudConfig(spec v1alpha1.MachineSpec) (config string, name string, err error) {
	c, _, _, err := p.getConfig(spec.ProviderConfig)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse config: %v", err)
	}

	config = fmt.Sprintf(`
[Global]
auth-url = "%s"
username = "%s"
password = "%s"
domain-name="%s"
tenant-name = "%s"
region = "%s"
`, c.IdentityEndpoint, c.Username, c.Password, c.DomainName, c.TenantName, c.Region)
	return config, "openstack", nil
}

func (p *provider) MachineMetricsLabels(machine *v1alpha1.Machine) (map[string]string, error) {
	labels := make(map[string]string)

	c, _, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err == nil {
		labels["size"] = c.Flavor
		labels["image"] = c.Image
		labels["region"] = c.Region
	}

	return labels, err
}

type serverWithExt struct {
	osservers.Server
	osextendedstatus.ServerExtendedStatusExt
}

type osInstance struct {
	server *serverWithExt
}

func (d *osInstance) Name() string {
	return d.server.Name
}

func (d *osInstance) ID() string {
	return d.server.ID
}

func (d *osInstance) Addresses() []string {
	var addresses []string
	for _, networkAddresses := range d.server.Addresses {
		for _, element := range networkAddresses.([]interface{}) {
			address := element.(map[string]interface{})
			addresses = append(addresses, address["addr"].(string))
		}
	}

	return addresses
}

func (d *osInstance) Status() instance.Status {
	switch d.server.Status {
	case "IN_PROGRESS":
		return instance.StatusCreating
	case "ACTIVE":
		return instance.StatusRunning
	default:
		return instance.StatusUnknown
	}
}

// osErrorToTerminalError judges if the given error
// can be qualified as a "terminal" error, for more info see v1alpha1.MachineStatus
//
// if the given error doesn't qualify the error passed as an argument will be returned
func osErrorToTerminalError(err error, msg string) error {
	prepareAndReturnError := func() error {
		return fmt.Errorf("%s, due to %s", msg, err)
	}
	if err != nil {
		default401 := gophercloud.ErrDefault401{}

		switch err.Error() {
		case "Authentication failed", default401.Error():
			// authorization primitives come from MachineSpec
			// thus we are setting InvalidConfigurationMachineError
			return cloudprovidererrors.TerminalError{
				Reason:  v1alpha1.InvalidConfigurationMachineError,
				Message: "A request has been rejected due to invalid credentials which were taken from the MachineSpec",
			}
		default:
			return prepareAndReturnError()
		}
	}
	return err
}
