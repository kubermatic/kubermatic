package azure

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"

	"github.com/kubermatic/machine-controller/pkg/cloudprovider/cloud"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/common/ssh"
	cloudprovidererrors "github.com/kubermatic/machine-controller/pkg/cloudprovider/errors"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/instance"
	kuberneteshelper "github.com/kubermatic/machine-controller/pkg/kubernetes"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	"k8s.io/apimachinery/pkg/types"

	common "sigs.k8s.io/cluster-api/pkg/apis/cluster/common"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	machineUIDTag = "Machine-UID"
	adminUserName = "kubermatic"

	finalizerPublicIP = "kubermatic.io/cleanup-azure-public-ip"
	finalizerNIC      = "kubermatic.io/cleanup-azure-nic"
	finalizerDisks    = "kubermatic.io/cleanup-azure-disks"
	finalizerVM       = "kubermatic.io/cleanup-azure-vm"
)

type provider struct {
	configVarResolver *providerconfig.ConfigVarResolver
}

// RawConfig is a direct representation of an Azure machine object's configuration
type RawConfig struct {
	SubscriptionID providerconfig.ConfigVarString `json:"subscriptionID"`
	TenantID       providerconfig.ConfigVarString `json:"tenantID"`
	ClientID       providerconfig.ConfigVarString `json:"clientID"`
	ClientSecret   providerconfig.ConfigVarString `json:"clientSecret"`

	Location        providerconfig.ConfigVarString `json:"location"`
	ResourceGroup   providerconfig.ConfigVarString `json:"resourceGroup"`
	VMSize          providerconfig.ConfigVarString `json:"vmSize"`
	VNetName        providerconfig.ConfigVarString `json:"vnetName"`
	SubnetName      providerconfig.ConfigVarString `json:"subnetName"`
	RouteTableName  providerconfig.ConfigVarString `json:"routeTableName"`
	AvailabilitySet providerconfig.ConfigVarString `json:"availabilitySet"`

	AssignPublicIP providerconfig.ConfigVarBool `json:"assignPublicIP"`
	Tags           map[string]string            `json:"tags"`
}

type config struct {
	SubscriptionID string
	TenantID       string
	ClientID       string
	ClientSecret   string

	Location        string
	ResourceGroup   string
	VMSize          string
	VNetName        string
	SubnetName      string
	RouteTableName  string
	AvailabilitySet string

	AssignPublicIP bool
	Tags           map[string]string
}

type azureVM struct {
	vm          *compute.VirtualMachine
	ipAddresses []string
	status      instance.Status
}

func (vm *azureVM) Addresses() []string {
	return vm.ipAddresses
}

func (vm *azureVM) ID() string {
	return *vm.vm.ID
}

func (vm *azureVM) Name() string {
	return *vm.vm.Name
}

func (vm *azureVM) Status() instance.Status {
	return vm.status
}

var imageReferences = map[providerconfig.OperatingSystem]compute.ImageReference{
	providerconfig.OperatingSystemCoreos: compute.ImageReference{
		Publisher: to.StringPtr("CoreOS"),
		Offer:     to.StringPtr("CoreOS"),
		Sku:       to.StringPtr("Stable"),
		Version:   to.StringPtr("latest"),
	},
	providerconfig.OperatingSystemCentOS: compute.ImageReference{
		Publisher: to.StringPtr("OpenLogic"),
		Offer:     to.StringPtr("CentOS"),
		Sku:       to.StringPtr("7-CI"), // https://docs.microsoft.com/en-us/azure/virtual-machines/linux/using-cloud-init
		Version:   to.StringPtr("latest"),
	},
	providerconfig.OperatingSystemUbuntu: compute.ImageReference{
		Publisher: to.StringPtr("Canonical"),
		Offer:     to.StringPtr("UbuntuServer"),
		// FIXME We'd like to use Ubuntu 18.04 eventually, but the docker's release
		// deb repo for `bionic` is empty, and we use `$RELEASE` in userdata.
		// Either Docker needs to fix their repo, or we need to hardcode `xenial`.
		Sku:     to.StringPtr("18.04-LTS"),
		Version: to.StringPtr("latest"),
	},
}

func getOSImageReference(os providerconfig.OperatingSystem) (*compute.ImageReference, error) {
	ref, supported := imageReferences[os]
	if !supported {
		return nil, fmt.Errorf("operating system %q not supported", os)
	}

	return &ref, nil
}

// New returns a digitalocean provider
func New(configVarResolver *providerconfig.ConfigVarResolver) cloud.Provider {
	return &provider{configVarResolver: configVarResolver}
}

func (p *provider) getConfig(s v1alpha1.ProviderConfig) (*config, *providerconfig.Config, error) {
	if s.Value == nil {
		return nil, nil, fmt.Errorf("machine.spec.providerconfig.value is nil")
	}
	pconfig := providerconfig.Config{}
	err := json.Unmarshal(s.Value.Raw, &pconfig)
	if err != nil {
		return nil, nil, err
	}
	rawCfg := RawConfig{}
	err = json.Unmarshal(pconfig.CloudProviderSpec.Raw, &rawCfg)
	if err != nil {
		return nil, nil, err
	}

	c := config{}
	c.SubscriptionID, err = p.configVarResolver.GetConfigVarStringValue(rawCfg.SubscriptionID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the value of \"subscriptionID\" field, error = %v", err)
	}

	c.TenantID, err = p.configVarResolver.GetConfigVarStringValue(rawCfg.TenantID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the value of \"tenantID\" field, error = %v", err)
	}

	c.ClientID, err = p.configVarResolver.GetConfigVarStringValue(rawCfg.ClientID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the value of \"clientID\" field, error = %v", err)
	}

	c.ClientSecret, err = p.configVarResolver.GetConfigVarStringValue(rawCfg.ClientSecret)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the value of \"clientSecret\" field, error = %v", err)
	}

	c.ResourceGroup, err = p.configVarResolver.GetConfigVarStringValue(rawCfg.ResourceGroup)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the value of \"resourceGroup\" field, error = %v", err)
	}

	c.Location, err = p.configVarResolver.GetConfigVarStringValue(rawCfg.Location)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the value of \"location\" field, error = %v", err)
	}

	c.VMSize, err = p.configVarResolver.GetConfigVarStringValue(rawCfg.VMSize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the value of \"vmSize\" field, error = %v", err)
	}

	c.VNetName, err = p.configVarResolver.GetConfigVarStringValue(rawCfg.VNetName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the value of \"vnetName\" field, error = %v", err)
	}

	c.SubnetName, err = p.configVarResolver.GetConfigVarStringValue(rawCfg.SubnetName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the value of \"subnetName\" field, error = %v", err)
	}

	c.RouteTableName, err = p.configVarResolver.GetConfigVarStringValue(rawCfg.RouteTableName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the value of \"routeTableName\" field, error = %v", err)
	}

	c.AssignPublicIP, err = p.configVarResolver.GetConfigVarBoolValue(rawCfg.AssignPublicIP)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the value of \"assignPublicIP\" field, error = %v", err)
	}

	c.AvailabilitySet, err = p.configVarResolver.GetConfigVarStringValue(rawCfg.AvailabilitySet)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the value of \"availabilitySet\" field, error = %v", err)
	}

	c.Tags = rawCfg.Tags

	return &c, &pconfig, nil
}

func getVMIPAddresses(ctx context.Context, c *config, vm *compute.VirtualMachine) ([]string, error) {
	var ipAddresses []string

	if vm.VirtualMachineProperties == nil {
		return nil, fmt.Errorf("machine is missing properties")
	}

	if vm.VirtualMachineProperties.NetworkProfile == nil {
		return nil, fmt.Errorf("machine has no network profile")
	}

	if vm.NetworkProfile.NetworkInterfaces == nil {
		return nil, fmt.Errorf("machine has no network interfaces data")
	}

	for n, iface := range *vm.NetworkProfile.NetworkInterfaces {
		if iface.ID == nil || len(*iface.ID) == 0 {
			return nil, fmt.Errorf("interface %d has no ID", n)
		}

		splitIfaceID := strings.Split(*iface.ID, "/")
		ifaceName := splitIfaceID[len(splitIfaceID)-1]
		addrs, err := getNICIPAddresses(ctx, c, ifaceName)
		if vm.NetworkProfile.NetworkInterfaces == nil {
			return nil, fmt.Errorf("failed to get addresses for interface %q: %v", ifaceName, err)
		}
		ipAddresses = append(ipAddresses, addrs...)
	}

	return ipAddresses, nil
}

func getNICIPAddresses(ctx context.Context, c *config, ifaceName string) ([]string, error) {
	ifClient, err := getInterfacesClient(c)
	if err != nil {
		return nil, fmt.Errorf("failed to create interfaces client: %v", err)
	}

	netIf, err := ifClient.Get(ctx, c.ResourceGroup, ifaceName, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get interface %q: %v", ifaceName, err.Error())
	}

	var ipAddresses []string

	if netIf.IPConfigurations != nil {
		for _, conf := range *netIf.IPConfigurations {
			var name string
			if conf.Name != nil {
				name = *conf.Name
			} else {
				glog.Warningf("IP configuration of NIC %q was returned with no name, trying to dissect the ID.", ifaceName)
				if conf.ID == nil || len(*conf.ID) == 0 {
					return nil, fmt.Errorf("IP configuration of NIC %q was returned with no ID", ifaceName)
				}
				splitConfID := strings.Split(*conf.ID, "/")
				name = splitConfID[len(splitConfID)-1]
			}

			addrStrings, err := getIPAddressStrings(ctx, c, name)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve IP string for IP %q: %v", name, err)
			}

			ipAddresses = append(ipAddresses, addrStrings...)
		}
	}

	return ipAddresses, nil
}

func getIPAddressStrings(ctx context.Context, c *config, addrName string) ([]string, error) {
	ipClient, err := getIPClient(c)
	if err != nil {
		return nil, fmt.Errorf("failed to create IP address client: %v", err)
	}

	ip, err := ipClient.Get(ctx, c.ResourceGroup, addrName, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get IP %q: %v", addrName, err)
	}

	if ip.IPConfiguration == nil {
		return nil, fmt.Errorf("IP %q has nil IPConfiguration", addrName)
	}

	var ipAddresses []string
	if ip.IPConfiguration.PublicIPAddress != nil && ip.IPConfiguration.PublicIPAddress.IPAddress != nil {
		ipAddresses = append(ipAddresses, *ip.IPConfiguration.PublicIPAddress.IPAddress)
	}
	if ip.IPConfiguration.PrivateIPAddress != nil {
		ipAddresses = append(ipAddresses, *ip.IPConfiguration.PrivateIPAddress)
	}

	return ipAddresses, nil
}

func (p *provider) AddDefaults(spec v1alpha1.MachineSpec) (v1alpha1.MachineSpec, bool, error) {
	return spec, false, nil
}

func (p *provider) Create(machine *v1alpha1.Machine, update cloud.MachineUpdater, userdata string) (instance.Instance, error) {
	config, providerCfg, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("failed to parse MachineSpec, due to %v", err),
		}
	}

	vmClient, err := getVMClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM client: %v", err)
	}

	osRef, err := getOSImageReference(providerCfg.OperatingSystem)
	if err != nil {
		return nil, err
	}

	// We genete a random SSH key, since Azure won't let us create a VM without an SSH key or a password
	key, err := ssh.NewKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ssh key: %v", err)
	}

	ifaceName := machine.Spec.Name + "-netiface"
	publicIPName := ifaceName + "-pubip"
	var publicIP *network.PublicIPAddress
	if config.AssignPublicIP {
		if !kuberneteshelper.HasFinalizer(machine, finalizerPublicIP) {
			if machine, err = update(machine.Namespace, machine.Name, func(updatedMachine *v1alpha1.Machine) {
				updatedMachine.Finalizers = append(updatedMachine.Finalizers, finalizerPublicIP)
			}); err != nil {
				return nil, err
			}
		}
		publicIP, err = createPublicIPAddress(context.TODO(), publicIPName, machine.UID, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create public IP: %v", err)
		}
	}

	if !kuberneteshelper.HasFinalizer(machine, finalizerNIC) {
		if machine, err = update(machine.Namespace, machine.Name, func(updatedMachine *v1alpha1.Machine) {
			updatedMachine.Finalizers = append(updatedMachine.Finalizers, finalizerNIC)
		}); err != nil {
			return nil, err
		}
	}
	iface, err := createNetworkInterface(context.TODO(), ifaceName, machine.UID, config, publicIP)
	if err != nil {
		return nil, fmt.Errorf("failed to generate main network interface: %v", err)
	}

	tags := make(map[string]*string, len(config.Tags)+1)
	for k, v := range config.Tags {
		tags[k] = to.StringPtr(v)
	}
	tags[machineUIDTag] = to.StringPtr(string(machine.UID))

	vmSpec := compute.VirtualMachine{
		Location: &config.Location,
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			HardwareProfile: &compute.HardwareProfile{VMSize: compute.VirtualMachineSizeTypes(config.VMSize)},
			NetworkProfile: &compute.NetworkProfile{
				NetworkInterfaces: &[]compute.NetworkInterfaceReference{
					{
						ID:                                  iface.ID,
						NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{Primary: to.BoolPtr(true)},
					},
				},
			},
			OsProfile: &compute.OSProfile{
				AdminUsername: to.StringPtr(adminUserName),
				ComputerName:  &machine.Spec.Name,
				LinuxConfiguration: &compute.LinuxConfiguration{
					DisablePasswordAuthentication: to.BoolPtr(true),
					SSH: &compute.SSHConfiguration{
						PublicKeys: &[]compute.SSHPublicKey{
							{
								Path:    to.StringPtr(fmt.Sprintf("/home/%s/.ssh/authorized_keys", adminUserName)),
								KeyData: &key.PublicKey,
							},
						},
					},
				},
				CustomData: to.StringPtr(base64.StdEncoding.EncodeToString([]byte(userdata))),
			},
			StorageProfile: &compute.StorageProfile{ImageReference: osRef},
		},
		Tags: tags,
	}

	if config.AvailabilitySet != "" {
		// Azure expects the full path to the resource
		asURI := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/availabilitySets/%s", config.SubscriptionID, config.ResourceGroup, config.AvailabilitySet)
		vmSpec.VirtualMachineProperties.AvailabilitySet = &compute.SubResource{ID: to.StringPtr(asURI)}
	}

	glog.Infof("Creating machine %q", machine.Spec.Name)
	if !kuberneteshelper.HasFinalizer(machine, finalizerDisks) {
		if machine, err = update(machine.Namespace, machine.Name, func(updatedMachine *v1alpha1.Machine) {
			updatedMachine.Finalizers = append(updatedMachine.Finalizers, finalizerDisks)
		}); err != nil {
			return nil, err
		}
	}
	if !kuberneteshelper.HasFinalizer(machine, finalizerVM) {
		if machine, err = update(machine.Namespace, machine.Name, func(updatedMachine *v1alpha1.Machine) {
			updatedMachine.Finalizers = append(updatedMachine.Finalizers, finalizerVM)
		}); err != nil {
			return nil, err
		}
	}
	future, err := vmClient.CreateOrUpdate(context.TODO(), config.ResourceGroup, machine.Spec.Name, vmSpec)
	if err != nil {
		return nil, fmt.Errorf("trying to create a VM: %v", err)
	}

	err = future.WaitForCompletion(context.TODO(), vmClient.Client)
	if err != nil {
		return nil, fmt.Errorf("waiting for operation returned: %v", err.Error())
	}

	vm, err := future.Result(*vmClient)
	if err != nil {
		return nil, fmt.Errorf("decoding result: %v", err.Error())
	}

	// get the actual VM object filled in with additional data
	vm, err = vmClient.Get(context.TODO(), config.ResourceGroup, machine.Spec.Name, "")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updated data for VM %q: %v", machine.Spec.Name, err)
	}

	ipAddresses, err := getVMIPAddresses(context.TODO(), config, &vm)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve IP addresses for VM %q: %v", machine.Spec.Name, err.Error())
	}

	status, err := getVMStatus(context.TODO(), config, machine.Spec.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve status for VM %q: %v", machine.Spec.Name, err.Error())
	}

	return &azureVM{vm: &vm, ipAddresses: ipAddresses, status: status}, nil
}

func (p *provider) Delete(machine *v1alpha1.Machine, update cloud.MachineUpdater) error {
	config, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return fmt.Errorf("failed to parse MachineSpec: %v", err)
	}

	_, err = p.Get(machine)
	// If a defunct VM got created, the `Get` call returns an error - But not because the request
	// failed but because the VM has an invalid config hence always delete except on err == cloudprovidererrors.ErrInstanceNotFound
	if err == nil || (err != nil && err != cloudprovidererrors.ErrInstanceNotFound) {
		glog.Infof("deleting VM %q", machine.Name)
		if err = deleteVMsByMachineUID(context.TODO(), config, machine.UID); err != nil {
			return fmt.Errorf("failed to delete instance for  machine %q: %v", machine.Name, err)
		}
	}

	if machine, err = update(machine.Namespace, machine.Name, func(updatedMachine *v1alpha1.Machine) {
		updatedMachine.Finalizers = kuberneteshelper.RemoveFinalizer(updatedMachine.Finalizers, finalizerVM)
	}); err != nil {
		return err
	}

	glog.Infof("deleting disks of VM %q", machine.Name)
	if err = deleteDisksByMachineUID(context.TODO(), config, machine.UID); err != nil {
		return fmt.Errorf("failed to remove disks of machine %q: %v", machine.Name, err)
	}
	if machine, err = update(machine.Namespace, machine.Name, func(updatedMachine *v1alpha1.Machine) {
		updatedMachine.Finalizers = kuberneteshelper.RemoveFinalizer(updatedMachine.Finalizers, finalizerDisks)
	}); err != nil {
		return err
	}

	glog.Infof("deleting network interfaces of VM %q", machine.Name)
	if err = deleteInterfacesByMachineUID(context.TODO(), config, machine.UID); err != nil {
		return fmt.Errorf("failed to remove network interfaces of machine %q: %v", machine.Name, err)
	}
	if machine, err = update(machine.Namespace, machine.Name, func(updatedMachine *v1alpha1.Machine) {
		updatedMachine.Finalizers = kuberneteshelper.RemoveFinalizer(updatedMachine.Finalizers, finalizerNIC)
	}); err != nil {
		return err
	}

	glog.Infof("deleting public IP addresses of VM %q", machine.Name)
	if err = deleteIPAddressesByMachineUID(context.TODO(), config, machine.UID); err != nil {
		return fmt.Errorf("failed to remove public IP addresses of machine %q: %v", machine.Name, err)
	}
	if machine, err = update(machine.Namespace, machine.Name, func(updatedMachine *v1alpha1.Machine) {
		updatedMachine.Finalizers = kuberneteshelper.RemoveFinalizer(updatedMachine.Finalizers, finalizerPublicIP)
	}); err != nil {
		return err
	}

	return nil
}

func getVMByUID(ctx context.Context, c *config, uid types.UID) (*compute.VirtualMachine, error) {
	vmClient, err := getVMClient(c)
	if err != nil {
		return nil, err
	}

	list, err := vmClient.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var allServers []compute.VirtualMachine

	for list.NotDone() {
		allServers = append(allServers, list.Values()...)
		if err = list.Next(); err != nil {
			return nil, fmt.Errorf("failed to iterate the result list: %s", err)
		}
	}

	for _, vm := range allServers {
		if vm.Tags != nil && vm.Tags[machineUIDTag] != nil && *vm.Tags[machineUIDTag] == string(uid) {
			return &vm, nil
		}
	}

	return nil, cloudprovidererrors.ErrInstanceNotFound
}

func getVMStatus(ctx context.Context, c *config, vmName string) (instance.Status, error) {
	vmClient, err := getVMClient(c)
	if err != nil {
		return instance.StatusUnknown, err
	}

	iv, err := vmClient.InstanceView(ctx, c.ResourceGroup, vmName)
	if err != nil {
		return instance.StatusUnknown, fmt.Errorf("failed to get instance view for machine %q: %v", vmName, err)
	}

	if iv.Statuses == nil {
		return instance.StatusUnknown, nil
	}

	// it seems that this field should contain two entries: a provisioning status and a power status
	if len(*iv.Statuses) < 2 {
		provisioningStatus := (*iv.Statuses)[0]
		if provisioningStatus.Code == nil {
			glog.Warningf("azure provisioning status has missing code")
			return instance.StatusUnknown, nil
		}

		switch *provisioningStatus.Code {
		case "":
			return instance.StatusUnknown, nil
		case "ProvisioningState/deleting":
			return instance.StatusDeleting, nil
		default:
			glog.Warningf("unknown Azure provisioning status %q", *provisioningStatus.Code)
			return instance.StatusUnknown, nil
		}
	}

	// the second field is supposed to be the power status
	// https://docs.microsoft.com/en-us/azure/virtual-machines/windows/tutorial-manage-vm#vm-power-states
	powerStatus := (*iv.Statuses)[1]
	if powerStatus.Code == nil {
		glog.Warningf("azure power status has missing code")
		return instance.StatusUnknown, nil
	}

	switch *powerStatus.Code {
	case "":
		return instance.StatusUnknown, nil
	case "PowerState/running":
		return instance.StatusRunning, nil
	case "PowerState/starting":
		return instance.StatusCreating, nil
	default:
		glog.Warningf("unknown Azure power status %q", *powerStatus.Code)
		return instance.StatusUnknown, nil
	}
}

func (p *provider) Get(machine *v1alpha1.Machine) (instance.Instance, error) {
	config, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MachineSpec: %v", err)
	}

	vm, err := getVMByUID(context.TODO(), config, machine.UID)
	if err != nil {
		if err == cloudprovidererrors.ErrInstanceNotFound {
			return nil, cloudprovidererrors.ErrInstanceNotFound
		}

		return nil, fmt.Errorf("failed to find machine %q by its UID: %v", machine.UID, err)
	}

	ipAddresses, err := getVMIPAddresses(context.TODO(), config, vm)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve IP addresses for VM %v: %v", vm.Name, err)
	}

	status, err := getVMStatus(context.TODO(), config, machine.Spec.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve status for VM %v: %v", vm.Name, err)
	}

	return &azureVM{vm: vm, ipAddresses: ipAddresses, status: status}, nil
}

func (p *provider) GetCloudConfig(spec v1alpha1.MachineSpec) (config string, name string, err error) {
	c, _, err := p.getConfig(spec.ProviderConfig)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse config: %v", err)
	}

	cc := &CloudConfig{
		Cloud:               "AZUREPUBLICCLOUD",
		TenantID:            c.TenantID,
		SubscriptionID:      c.SubscriptionID,
		AADClientID:         c.ClientID,
		AADClientSecret:     c.ClientSecret,
		ResourceGroup:       c.ResourceGroup,
		Location:            c.Location,
		VNetName:            c.VNetName,
		SubnetName:          c.SubnetName,
		RouteTableName:      c.RouteTableName,
		UseInstanceMetadata: true,
	}

	s, err := CloudConfigToString(cc)
	if err != nil {
		return "", "", fmt.Errorf("failed to convert cloud-config to string: %v", err)
	}

	return s, "azure", nil
}

func (p *provider) Validate(spec v1alpha1.MachineSpec) error {
	c, providerCfg, err := p.getConfig(spec.ProviderConfig)
	if err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	if c.SubscriptionID == "" {
		return errors.New("subscriptionID is missing")
	}

	if c.TenantID == "" {
		return errors.New("tenantID is missing")
	}

	if c.ClientID == "" {
		return errors.New("clientID is missing")
	}

	if c.ClientSecret == "" {
		return errors.New("clientSecret is missing")
	}

	if c.ResourceGroup == "" {
		return errors.New("resourceGroup is missing")
	}

	if c.VMSize == "" {
		return errors.New("vmSize is missing")
	}

	if c.VNetName == "" {
		return errors.New("vnetName is missing")
	}

	if c.SubnetName == "" {
		return errors.New("subnetName is missing")
	}

	vmClient, err := getVMClient(c)
	if err != nil {
		return fmt.Errorf("failed to (create) vm client: %v", err.Error())
	}

	_, err = vmClient.ListAll(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to list all: %v", err.Error())
	}

	if _, err := getVirtualNetwork(context.TODO(), c); err != nil {
		return fmt.Errorf("failed to get virtual network: %v", err)
	}

	if _, err := getSubnet(context.TODO(), c); err != nil {
		return fmt.Errorf("failed to get subnet: %v", err)
	}

	_, err = getOSImageReference(providerCfg.OperatingSystem)
	return nil
}

func (p *provider) MachineMetricsLabels(machine *v1alpha1.Machine) (map[string]string, error) {
	labels := make(map[string]string)

	c, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err == nil {
		labels["size"] = c.VMSize
		labels["location"] = c.Location
	}

	return labels, err
}
