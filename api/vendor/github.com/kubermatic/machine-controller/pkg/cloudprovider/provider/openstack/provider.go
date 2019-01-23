package openstack

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"

	"github.com/gophercloud/gophercloud"
	goopenstack "github.com/gophercloud/gophercloud/openstack"
	osextendedstatus "github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/extendedstatus"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	osservers "github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	osfloatingips "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	osnetworks "github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/pagination"

	"github.com/kubermatic/machine-controller/pkg/cloudprovider/cloud"
	cloudprovidererrors "github.com/kubermatic/machine-controller/pkg/cloudprovider/errors"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/instance"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"

	"sigs.k8s.io/cluster-api/pkg/apis/cluster/common"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	floatingIPReleaseFinalizer = "kubermatic.io/release-openstack-floating-ip"
	floatingIPIDAnnotationKey  = "kubermatic.io/release-openstack-floating-ip"
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
	Region           providerconfig.ConfigVarString `json:"region"`

	// Machine details
	Image            providerconfig.ConfigVarString   `json:"image"`
	Flavor           providerconfig.ConfigVarString   `json:"flavor"`
	SecurityGroups   []providerconfig.ConfigVarString `json:"securityGroups"`
	Network          providerconfig.ConfigVarString   `json:"network"`
	Subnet           providerconfig.ConfigVarString   `json:"subnet"`
	FloatingIPPool   providerconfig.ConfigVarString   `json:"floatingIpPool"`
	AvailabilityZone providerconfig.ConfigVarString   `json:"availabilityZone"`
	// This tag is related to server metadata, not compute server's tag
	Tags map[string]string `json:"tags"`
}

type Config struct {
	IdentityEndpoint string
	Username         string
	Password         string
	DomainName       string
	TenantName       string
	TokenID          string
	Region           string

	// Machine details
	Image            string
	Flavor           string
	SecurityGroups   []string
	Network          string
	Subnet           string
	FloatingIPPool   string
	AvailabilityZone string

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

func (p *provider) getConfig(s v1alpha1.ProviderSpec) (*Config, *providerconfig.Config, *RawConfig, error) {
	if s.Value == nil {
		return nil, nil, nil, fmt.Errorf("machine.spec.providerconfig.value is nil")
	}
	pconfig := providerconfig.Config{}
	err := json.Unmarshal(s.Value.Raw, &pconfig)
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
	// Ignore Region not found as Region might not be found and we can default it later
	c.Region, err = p.configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Region, "OS_REGION_NAME")
	if err != nil {
		glog.V(6).Infof("Region from configuration or environment variable not found")
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
	c.Tags = rawConfig.Tags
	if c.Tags == nil {
		c.Tags = map[string]string{}
	}

	return &c, &pconfig, &rawConfig, err
}

func setProviderSpec(rawConfig RawConfig, s v1alpha1.ProviderSpec) (*runtime.RawExtension, error) {
	if s.Value == nil {
		return nil, fmt.Errorf("machine.spec.providerconfig.value is nil")
	}
	pconfig := providerconfig.Config{}
	err := json.Unmarshal(s.Value.Raw, &pconfig)
	if err != nil {
		return nil, err
	}
	rawCloudProviderSpec, err := json.Marshal(rawConfig)
	if err != nil {
		return nil, err
	}
	pconfig.CloudProviderSpec = runtime.RawExtension{Raw: rawCloudProviderSpec}
	rawPconfig, err := json.Marshal(pconfig)
	if err != nil {
		return nil, err
	}

	return &runtime.RawExtension{Raw: rawPconfig}, nil
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

func (p *provider) AddDefaults(spec v1alpha1.MachineSpec) (v1alpha1.MachineSpec, error) {
	c, _, rawConfig, err := p.getConfig(spec.ProviderSpec)
	if err != nil {
		return spec, cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}

	client, err := getClient(c)
	if err != nil {
		return spec, osErrorToTerminalError(err, "failed to get a openstack client")
	}

	if c.Region == "" {
		glog.V(4).Infof("Trying to default region for machine '%s'...", spec.Name)
		regions, err := getRegions(client)
		if err != nil {
			return spec, osErrorToTerminalError(err, "failed to get regions")
		}
		if len(regions) == 1 {
			glog.V(4).Infof("Defaulted region for machine '%s' to '%s'", spec.Name, regions[0].ID)
			rawConfig.Region.Value = regions[0].ID
		} else {
			return spec, fmt.Errorf("could not default region because got '%v' results", len(regions))
		}
	}

	if c.AvailabilityZone == "" {
		glog.V(4).Infof("Trying to default availability zone for machine '%s'...", spec.Name)
		availabilityZones, err := getAvailabilityZones(client, c.Region)
		if err != nil {
			return spec, osErrorToTerminalError(err, "failed to get availability zones")
		}
		if len(availabilityZones) == 1 {
			glog.V(4).Infof("Defaulted availability zone for machine '%s' to '%s'", spec.Name, availabilityZones[0].ZoneName)
			rawConfig.AvailabilityZone.Value = availabilityZones[0].ZoneName
		}
	}

	if c.Network == "" {
		glog.V(4).Infof("Trying to default network for machine '%s'...", spec.Name)
		net, err := getDefaultNetwork(client, c.Region)
		if err != nil {
			return spec, osErrorToTerminalError(err, "failed to default network")
		}
		if net != nil {
			glog.V(4).Infof("Defaulted network for machine '%s' to '%s'", spec.Name, net.Name)
			// Use the id as the name may not be unique
			rawConfig.Network.Value = net.ID
		}
	}

	if c.Subnet == "" {
		networkID := c.Network
		if rawConfig.Network.Value != "" {
			networkID = rawConfig.Network.Value
		}

		net, err := getNetwork(client, c.Region, networkID)
		if err != nil {
			return spec, osErrorToTerminalError(err, fmt.Sprintf("failed to get network for subnet defaulting '%s", networkID))
		}
		subnet, err := getDefaultSubnet(client, net, c.Region)
		if err != nil {
			return spec, osErrorToTerminalError(err, "error defaulting subnet")
		}
		if subnet != nil {
			glog.V(4).Infof("Defaulted subnet for machine '%s' to '%s'", spec.Name, *subnet)
			rawConfig.Subnet.Value = *subnet
		}
	}

	spec.ProviderSpec.Value, err = setProviderSpec(*rawConfig, spec.ProviderSpec)
	if err != nil {
		return spec, osErrorToTerminalError(err, "error marshaling providerconfig")
	}
	return spec, nil
}

func (p *provider) Validate(spec v1alpha1.MachineSpec) error {
	c, _, _, err := p.getConfig(spec.ProviderSpec)
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

func (p *provider) Create(machine *v1alpha1.Machine, machineCreateDeleteData *cloud.MachineCreateDeleteData, userdata string) (instance.Instance, error) {
	c, _, _, err := p.getConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
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

	if err := waitUntilInstanceIsActive(computeClient, server.ID); err != nil {
		defer deleteInstanceDueToFatalLogged(computeClient, server.ID)
		return nil, fmt.Errorf("instance %s became not active: %v", server.ID, err)
	}

	// Find a free FloatingIP or allocate a new one
	if c.FloatingIPPool != "" {
		if err := assignFloatingIPToInstance(machineCreateDeleteData.Updater, machine, client, server.ID, c.FloatingIPPool, c.Region, network); err != nil {
			defer deleteInstanceDueToFatalLogged(computeClient, server.ID)
			return nil, fmt.Errorf("failed to assign a floating ip to instance %s: %v", server.ID, err)
		}
	}

	return &osInstance{server: &server}, nil
}

func waitUntilInstanceIsActive(computeClient *gophercloud.ServiceClient, serverID string) error {
	started := time.Now()
	glog.V(2).Infof("Waiting for the instance %s to become active...", serverID)

	instanceIsReady := func() (bool, error) {
		currentServer, err := osservers.Get(computeClient, serverID).Extract()
		if err != nil {
			tErr := osErrorToTerminalError(err, fmt.Sprintf("failed to get current instance %s", serverID))
			if isTerminalErr, _, _ := cloudprovidererrors.IsTerminalError(tErr); isTerminalErr {
				return true, tErr
			}
			// Only log the error but don't exit. in case of a network failure we want to retry
			glog.V(2).Infof("failed to get current instance %s: %v", serverID, err)
			return false, nil
		}
		if currentServer.Status == "ACTIVE" {
			return true, nil
		}
		return false, nil
	}

	if err := wait.Poll(instanceReadyCheckPeriod, instanceReadyCheckTimeout, instanceIsReady); err != nil {
		if err == wait.ErrWaitTimeout {
			// In case we have a timeout, include the timeout details
			return fmt.Errorf("instance became not active after %f seconds", instanceReadyCheckTimeout.Seconds())
		}
		// Some terminal error happened
		return fmt.Errorf("failed to wait for instance to become active: %v", err)
	}

	glog.V(2).Infof("Instance %s became active after %f seconds", serverID, time.Since(started).Seconds())
	return nil
}

func deleteInstanceDueToFatalLogged(computeClient *gophercloud.ServiceClient, serverID string) {
	glog.V(0).Infof("Deleting instance %s due to fatal error during machine creation...", serverID)
	if err := osservers.Delete(computeClient, serverID).ExtractErr(); err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to delete the instance %s. Please take care of manually deleting the instance: %v", serverID, err))
		return
	}
	glog.V(0).Infof("Instance %s got deleted", serverID)
}

func (p *provider) Cleanup(machine *v1alpha1.Machine, machineCreateDeleteData *cloud.MachineCreateDeleteData) (bool, error) {
	var hasFloatingIPReleaseFinalizer bool
	if finalizers := sets.NewString(machine.Finalizers...); finalizers.Has(floatingIPReleaseFinalizer) {
		hasFloatingIPReleaseFinalizer = true
	}

	instance, err := p.Get(machine)
	if err != nil {
		if err == cloudprovidererrors.ErrInstanceNotFound {
			if hasFloatingIPReleaseFinalizer {
				if err := p.cleanupFloatingIP(machine, machineCreateDeleteData.Updater); err != nil {
					return false, fmt.Errorf("failed to clean up floating ip: %v", err)
				}
			}
			return true, nil
		}
		return false, err
	}

	c, _, _, err := p.getConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return false, cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}

	client, err := getClient(c)
	if err != nil {
		return false, osErrorToTerminalError(err, "failed to get a openstack client")
	}

	computeClient, err := goopenstack.NewComputeV2(client, gophercloud.EndpointOpts{Availability: gophercloud.AvailabilityPublic, Region: c.Region})
	if err != nil {
		return false, osErrorToTerminalError(err, "failed to get compute client")
	}

	if err := osservers.Delete(computeClient, instance.ID()).ExtractErr(); err != nil && err.Error() != "Resource not found" {
		return false, osErrorToTerminalError(err, "failed to delete instance")
	}

	if hasFloatingIPReleaseFinalizer {
		return false, p.cleanupFloatingIP(machine, machineCreateDeleteData.Updater)
	}

	return false, nil
}

func (p *provider) Get(machine *v1alpha1.Machine) (instance.Instance, error) {
	c, _, _, err := p.getConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
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

func (p *provider) MigrateUID(machine *v1alpha1.Machine, new types.UID) error {
	c, _, _, err := p.getConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
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
		return osErrorToTerminalError(err, "failed to list instances")
	}

	for _, s := range allServers {
		if s.Metadata[machineUIDMetaKey] == string(machine.UID) {
			metadataOpts := osservers.MetadataOpts(s.Metadata)
			metadataOpts[machineUIDMetaKey] = string(new)
			response := osservers.UpdateMetadata(computeClient, s.ID, metadataOpts)
			if response.Err != nil {
				return fmt.Errorf("failed to update instance metadata with new UID: %v", err)
			}
		}
	}

	return nil
}

func (p *provider) GetCloudConfig(spec v1alpha1.MachineSpec) (config string, name string, err error) {
	c, _, _, err := p.getConfig(spec.ProviderSpec)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse config: %v", err)
	}

	cc := &CloudConfig{
		Global: GlobalOpts{
			AuthURL:    c.IdentityEndpoint,
			Username:   c.Username,
			Password:   c.Password,
			DomainName: c.DomainName,
			TenantName: c.TenantName,
			Region:     c.Region,
		},
		LoadBalancer: LoadBalancerOpts{
			ManageSecurityGroups: true,
		},
		BlockStorage: BlockStorageOpts{
			BSVersion:       "v2",
			TrustDevicePath: false,
			IgnoreVolumeAZ:  true,
		},
		Version: spec.Versions.Kubelet,
	}

	s, err := CloudConfigToString(cc)
	if err != nil {
		return "", "", fmt.Errorf("failed to convert the cloud-config to string: %v", err)
	}
	return s, "openstack", nil
}

func (p *provider) MachineMetricsLabels(machine *v1alpha1.Machine) (map[string]string, error) {
	labels := make(map[string]string)

	c, _, _, err := p.getConfig(machine.Spec.ProviderSpec)
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
	if errUnauthorized, ok := err.(gophercloud.ErrDefault401); ok {
		return cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("A request has been rejected due to invalid credentials which were taken from the MachineSpec: %v", errUnauthorized),
		}
	}

	if errForbidden, ok := err.(gophercloud.ErrDefault403); ok {
		terr := cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("%s. The request against the OpenStack API is forbidden: %s", msg, errForbidden.Error()),
		}

		// The response from OpenStack might contain a more detailed message
		info := &forbiddenResponse{}
		if err := json.Unmarshal(errForbidden.Body, info); err != nil {
			// We just log here as we just do this to make the response more pretty
			glog.V(0).Infof("failed to unmarshal response body from 403 response from OpenStack API: %v\n%s", err, errForbidden.Body)
			return terr
		}

		// If we have more details, interpret them
		if info.Forbidden.Message != "" {
			terr.Message = fmt.Sprintf("%s. The request against the OpenStack API is forbidden: %s", msg, info.Forbidden.Message)
			if strings.Contains(info.Forbidden.Message, "Quota exceeded") {
				terr.Reason = common.InsufficientResourcesMachineError
			}
		}

		return terr
	}

	return fmt.Errorf("%s, due to %s", msg, err)
}

// forbiddenResponse is a potential response body from the OpenStack API when the request is forbidden (code: 403)
type forbiddenResponse struct {
	Forbidden struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"forbidden"`
}

func (p *provider) cleanupFloatingIP(machine *v1alpha1.Machine, updater cloud.MachineUpdater) error {
	floatingIPID, exists := machine.Annotations[floatingIPIDAnnotationKey]
	if !exists {
		return osErrorToTerminalError(fmt.Errorf("failed to release floating ip"),
			fmt.Sprintf("%s finalizer exists but %s annotation does not", floatingIPReleaseFinalizer, floatingIPIDAnnotationKey))
	}

	c, _, _, err := p.getConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}

	client, err := getClient(c)
	if err != nil {
		return osErrorToTerminalError(err, "failed to get a openstack client")
	}
	netClient, err := goopenstack.NewNetworkV2(client, gophercloud.EndpointOpts{Region: c.Region})
	if err != nil {
		return fmt.Errorf("failed to create the networkv2 client for region %s: %v", c.Region, err)
	}
	if err := osfloatingips.Delete(netClient, floatingIPID).ExtractErr(); err != nil && err.Error() != "Resource not found" {
		return fmt.Errorf("failed to delete floating ip %s: %v", floatingIPID, err)
	}
	if _, err := updater(machine, func(m *v1alpha1.Machine) {
		finalizers := sets.NewString(m.Finalizers...)
		finalizers.Delete(floatingIPReleaseFinalizer)
		m.Finalizers = finalizers.List()
	}); err != nil {
		return fmt.Errorf("failed to delete %s finalizer from Machine: %v", floatingIPReleaseFinalizer, err)
	}

	return nil
}

func assignFloatingIPToInstance(machineUpdater cloud.MachineUpdater, machine *v1alpha1.Machine, client *gophercloud.ProviderClient, instanceID, floatingIPPoolName, region string, network *osnetworks.Network) error {
	port, err := getInstancePort(client, region, instanceID, network.ID)
	if err != nil {
		return fmt.Errorf("failed to get instance port for network %s in region %s: %v", network.ID, region, err)
	}

	netClient, err := goopenstack.NewNetworkV2(client, gophercloud.EndpointOpts{Region: region})
	if err != nil {
		return fmt.Errorf("failed to create the networkv2 client for region %s: %v", region, err)
	}

	floatingIPPool, err := getNetwork(client, region, floatingIPPoolName)
	if err != nil {
		return osErrorToTerminalError(err, fmt.Sprintf("failed to get floating ip pool %q", floatingIPPoolName))
	}

	// We're only interested in the part which is vulnerable to concurrent access
	started := time.Now()
	glog.V(2).Infof("Assigning a floating IP to instance %s", instanceID)

	floatingIPAssignLock.Lock()
	defer floatingIPAssignLock.Unlock()

	freeFloatingIps, err := getFreeFloatingIPs(client, region, floatingIPPool)
	if err != nil {
		return osErrorToTerminalError(err, "failed to get free floating ips")
	}

	var ip *osfloatingips.FloatingIP
	if len(freeFloatingIps) < 1 {
		if ip, err = createFloatingIP(client, region, port.ID, floatingIPPool); err != nil {
			return osErrorToTerminalError(err, "failed to allocate a floating ip")
		}
		if _, err = machineUpdater(machine, func(m *v1alpha1.Machine) {
			m.Finalizers = append(m.Finalizers, floatingIPReleaseFinalizer)
			if m.Annotations == nil {
				m.Annotations = map[string]string{}
			}
			m.Annotations[floatingIPIDAnnotationKey] = ip.ID
		}); err != nil {
			return fmt.Errorf("failed to add floating ip release finalizer after allocating floating ip: %v", err)
		}
	} else {
		freeIP := freeFloatingIps[0]
		ip, err = osfloatingips.Update(netClient, freeIP.ID, osfloatingips.UpdateOpts{
			PortID: &port.ID,
		}).Extract()
		if err != nil {
			return fmt.Errorf("failed to update FloatingIP %s(%s): %v", freeIP.ID, freeIP.FloatingIP, err)
		}

		// We're now going to wait 3 seconds and check if the IP is still ours. If not, we're going to fail
		// On our reference system it took ~3 seconds for a full FloatingIP allocation (Including creating a new one). It took ~600ms just for assigning one.
		time.Sleep(floatingReassignIPCheckPeriod)
		currentIP, err := osfloatingips.Get(netClient, ip.ID).Extract()
		if err != nil {
			return fmt.Errorf("failed to load FloatingIP %s after assignment has been done: %v", ip.FloatingIP, err)
		}
		// Verify if the port is still the one we set it to
		if currentIP.PortID != port.ID {
			return fmt.Errorf("floatingIP %s got reassigned", currentIP.FloatingIP)
		}
	}
	secondsTook := time.Since(started).Seconds()

	glog.V(2).Infof("Successfully assigned the FloatingIP %s to instance %s. Took %f seconds(without the recheck wait period %f seconds). ", ip.FloatingIP, instanceID, secondsTook, floatingReassignIPCheckPeriod.Seconds())
	return nil
}
