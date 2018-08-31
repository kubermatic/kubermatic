package vsphere

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/kubermatic/machine-controller/pkg/cloudprovider/cloud"
	cloudprovidererrors "github.com/kubermatic/machine-controller/pkg/cloudprovider/errors"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/instance"
	"github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
)

type provider struct {
	configVarResolver *providerconfig.ConfigVarResolver
}

type netDeviceAndBackingInfo struct {
	device      *types.BaseVirtualDevice
	backingInfo *types.VirtualEthernetCardNetworkBackingInfo
}

// New returns a VSphere provider
func New(configVarResolver *providerconfig.ConfigVarResolver) cloud.Provider {
	return &provider{configVarResolver: configVarResolver}
}

type RawConfig struct {
	TemplateVMName  providerconfig.ConfigVarString `json:"templateVMName"`
	TemplateNetName providerconfig.ConfigVarString `json:"templateNetName"`
	VMNetName       providerconfig.ConfigVarString `json:"vmNetName"`
	Username        providerconfig.ConfigVarString `json:"username"`
	Password        providerconfig.ConfigVarString `json:"password"`
	VSphereURL      providerconfig.ConfigVarString `json:"vsphereURL"`
	Datacenter      providerconfig.ConfigVarString `json:"datacenter"`
	Cluster         providerconfig.ConfigVarString `json:"cluster"`
	Folder          providerconfig.ConfigVarString `json:"folder"`
	Datastore       providerconfig.ConfigVarString `json:"datastore"`
	CPUs            int32                          `json:"cpus"`
	MemoryMB        int64                          `json:"memoryMB"`
	AllowInsecure   providerconfig.ConfigVarBool   `json:"allowInsecure"`
}

type Config struct {
	TemplateVMName  string
	TemplateNetName string
	VMNetName       string
	Username        string
	Password        string
	VSphereURL      string
	Datacenter      string
	Cluster         string
	Folder          string
	Datastore       string
	AllowInsecure   bool
	CPUs            int32
	MemoryMB        int64
}

type Server struct {
	name      string
	id        string
	status    instance.Status
	addresses []string
}

func (vsphereServer Server) Name() string {
	return vsphereServer.name
}

func (vsphereServer Server) ID() string {
	return vsphereServer.id
}

func (vsphereServer Server) Addresses() []string {
	return vsphereServer.addresses
}

func (vsphereServer Server) Status() instance.Status {
	return vsphereServer.status
}

func (p *provider) AddDefaults(spec v1alpha1.MachineSpec) (v1alpha1.MachineSpec, bool, error) {
	changed := false

	cfg, _, rawCfg, err := p.getConfig(spec.ProviderConfig)
	if err != nil {
		return spec, changed, cloudprovidererrors.TerminalError{
			Reason:  v1alpha1.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}

	// default templatenetname to network of template if none specific was given and only one adapter exists.
	if cfg.TemplateNetName == "" && cfg.VMNetName != "" {
		ctx := context.TODO()
		client, err := getClient(cfg.Username, cfg.Password, cfg.VSphereURL, cfg.AllowInsecure)
		if err != nil {
			return spec, changed, fmt.Errorf("failed to get vsphere client: '%v'", err)
		}
		defer func() {
			if lerr := client.Logout(ctx); lerr != nil {
				utilruntime.HandleError(fmt.Errorf("vsphere client failed to logout: %s", lerr))
			}
		}()

		finder, err := getDatacenterFinder(cfg.Datacenter, client)
		if err != nil {
			return spec, changed, err
		}

		templateVM, err := finder.VirtualMachine(ctx, cfg.TemplateVMName)
		if err != nil {
			return spec, changed, err
		}

		availableNetworkDevices, err := getNetworkDevicesAndBackingsFromVM(ctx, templateVM, "")
		if err != nil {
			return spec, changed, err
		}

		if len(availableNetworkDevices) == 0 {
			glog.V(6).Infof("found no network adapter to default to in template vm %s", cfg.TemplateVMName)
		} else if len(availableNetworkDevices) > 1 {
			glog.V(6).Infof("found multiple network adapters in template vm %s but no explicit template net name is specified in the cluster", cfg.TemplateVMName)
		} else {
			eth := availableNetworkDevices[0].backingInfo
			rawCfg.TemplateNetName.Value = eth.DeviceName
			changed = true
		}
	}

	spec.ProviderConfig, err = setProviderConfig(*rawCfg, spec.ProviderConfig)
	if err != nil {
		return spec, changed, fmt.Errorf("error marshaling providerconfig: %s", err)
	}

	return spec, changed, nil
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

func getClient(username, password, address string, allowInsecure bool) (*govmomi.Client, error) {
	clientURL, err := url.Parse(fmt.Sprintf("%s/sdk", address))
	if err != nil {
		return nil, err
	}
	clientURL.User = url.UserPassword(username, password)

	return govmomi.NewClient(context.TODO(), clientURL, allowInsecure)
}

func (p *provider) getConfig(s runtime.RawExtension) (*Config, *providerconfig.Config, *RawConfig, error) {
	pconfig := providerconfig.Config{}
	err := json.Unmarshal(s.Raw, &pconfig)
	if err != nil {
		return nil, nil, nil, err
	}

	rawConfig := RawConfig{}
	err = json.Unmarshal(pconfig.CloudProviderSpec.Raw, &rawConfig)
	if err != nil {
		return nil, nil, nil, err
	}

	c := Config{}
	c.TemplateVMName, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.TemplateVMName)
	if err != nil {
		return nil, nil, nil, err
	}

	c.TemplateNetName, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.TemplateNetName)
	if err != nil {
		return nil, nil, nil, err
	}

	c.VMNetName, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.VMNetName)
	if err != nil {
		return nil, nil, nil, err
	}

	c.Username, err = p.configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Username, "VSPHERE_USERNAME")
	if err != nil {
		return nil, nil, nil, err
	}

	c.Password, err = p.configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Password, "VSPHERE_PASSWORD")
	if err != nil {
		return nil, nil, nil, err
	}

	c.VSphereURL, err = p.configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.VSphereURL, "VSPHERE_ADDRESS")
	if err != nil {
		return nil, nil, nil, err
	}

	c.Datacenter, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Datacenter)
	if err != nil {
		return nil, nil, nil, err
	}

	c.Cluster, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Cluster)
	if err != nil {
		return nil, nil, nil, err
	}

	c.Folder, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Folder)
	if err != nil {
		return nil, nil, nil, err
	}

	c.Datastore, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Datastore)
	if err != nil {
		return nil, nil, nil, err
	}

	c.AllowInsecure, err = p.configVarResolver.GetConfigVarBoolValueOrEnv(rawConfig.AllowInsecure, "VSPHERE_ALLOW_INSECURE")
	if err != nil {
		return nil, nil, nil, err
	}

	c.CPUs = rawConfig.CPUs
	c.MemoryMB = rawConfig.MemoryMB

	return &c, &pconfig, &rawConfig, nil
}

func (p *provider) Validate(spec v1alpha1.MachineSpec) error {
	config, _, _, err := p.getConfig(spec.ProviderConfig)
	if err != nil {
		return err
	}

	if config.VMNetName != "" && config.TemplateNetName == "" {
		return errors.New("specified target network (VMNetName) in cluster, but no source network (TemplateNetName) in machine")
	}

	client, err := getClient(config.Username, config.Password, config.VSphereURL, config.AllowInsecure)
	if err != nil {
		return fmt.Errorf("failed to get vsphere client: '%v'", err)
	}
	defer func() {
		if lerr := client.Logout(context.TODO()); lerr != nil {
			utilruntime.HandleError(fmt.Errorf("vsphere client failed to logout: %s", lerr))
		}
	}()

	finder, err := getDatacenterFinder(config.Datacenter, client)
	if err != nil {
		return err
	}

	_, err = finder.Datastore(context.TODO(), config.Datastore)
	if err != nil {
		return err
	}

	_, err = finder.ClusterComputeResource(context.TODO(), config.Cluster)
	return err
}

func machineInvalidConfigurationTerminalError(err error) error {
	return cloudprovidererrors.TerminalError{
		Reason:  v1alpha1.InvalidConfigurationMachineError,
		Message: err.Error(),
	}
}

func (p *provider) Create(machine *v1alpha1.Machine, _ cloud.MachineUpdater, userdata string) (instance.Instance, error) {
	config, pc, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	client, err := getClient(config.Username, config.Password, config.VSphereURL, config.AllowInsecure)
	if err != nil {
		return nil, fmt.Errorf("failed to get vsphere client: '%v'", err)
	}
	defer func() {
		if lerr := client.Logout(context.TODO()); lerr != nil {
			utilruntime.HandleError(fmt.Errorf("vsphere client failed to logout: %s", lerr))
		}
	}()

	var containerLinuxUserdata string
	if pc.OperatingSystem == providerconfig.OperatingSystemCoreos {
		containerLinuxUserdata = userdata
	}

	if err = createLinkClonedVM(machine.Spec.Name,
		config.TemplateVMName,
		config.Datacenter,
		config.Cluster,
		config.Folder,
		config.CPUs,
		config.MemoryMB,
		client,
		containerLinuxUserdata); err != nil {
		return nil, machineInvalidConfigurationTerminalError(fmt.Errorf("failed to create linked vm: '%v'", err))
	}

	finder, err := getDatacenterFinder(config.Datacenter, client)
	if err != nil {
		return nil, err
	}
	virtualMachine, err := finder.VirtualMachine(context.TODO(), machine.Spec.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual machine object: %v", err)
	}

	// Map networks
	if config.VMNetName != "" {
		err = updateNetworkForVM(context.TODO(), virtualMachine, config.TemplateNetName, config.VMNetName)
		if err != nil {
			return nil, fmt.Errorf("couldn't set network for vm: %v", err)
		}
	}

	if pc.OperatingSystem != providerconfig.OperatingSystemCoreos {
		localUserdataIsoFilePath, err := generateLocalUserdataISO(userdata, machine.Spec.Name)
		if err != nil {
			return nil, err
		}

		defer func() {
			err := os.Remove(localUserdataIsoFilePath)
			if err != nil {
				utilruntime.HandleError(fmt.Errorf("failed to clean up local userdata iso file at %s: %v", localUserdataIsoFilePath, err))
			}
		}()

		err = uploadAndAttachISO(finder, virtualMachine, localUserdataIsoFilePath, config.Datastore, client)
		if err != nil {
			return nil, machineInvalidConfigurationTerminalError(fmt.Errorf("failed to upload and attach userdata iso: %v", err))
		}
	}

	// Ubuntu wont boot with attached floppy device, because it tries to write to it
	// which fails, because the floppy device does not contain a floppy disk
	// Upstream issue: https://bugs.launchpad.net/cloud-images/+bug/1573095
	err = removeFloppyDevice(virtualMachine)
	if err != nil {
		return nil, err
	}

	powerOnTask, err := virtualMachine.PowerOn(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to power on machine: %v", err)
	}

	powerOnTaskContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = powerOnTask.Wait(powerOnTaskContext)
	if err != nil {
		return nil, fmt.Errorf("timed out waiting to power on vm %s: %v", virtualMachine.Name(), err)
	}

	return Server{name: virtualMachine.Name(), status: instance.StatusRunning, id: virtualMachine.Reference().Value}, nil
}

func (p *provider) Delete(machine *v1alpha1.Machine, _ cloud.MachineUpdater) error {
	if _, err := p.Get(machine); err != nil {
		if err == cloudprovidererrors.ErrInstanceNotFound {
			return nil
		}
		return err
	}

	config, pc, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	client, err := getClient(config.Username, config.Password, config.VSphereURL, config.AllowInsecure)
	if err != nil {
		return fmt.Errorf("failed to get vsphere client: '%v'", err)
	}
	defer func() {
		if lerr := client.Logout(context.TODO()); lerr != nil {
			utilruntime.HandleError(fmt.Errorf("vsphere client failed to logout: %s", lerr))
		}
	}()
	finder := find.NewFinder(client.Client, true)

	// We can't use getDatacenterFinder because we need the dc object to
	// be able to initialize the Datastore Filemanager to delete the instaces
	// folder on the storage - This doesn't happen automatically because there
	// is still the cloud-init iso
	dc, err := finder.Datacenter(context.TODO(), config.Datacenter)
	if err != nil {
		return fmt.Errorf("failed to get vsphere datacenter: %v", err)
	}
	finder.SetDatacenter(dc)

	virtualMachine, err := finder.VirtualMachine(context.TODO(), machine.Spec.Name)
	if err != nil {
		return fmt.Errorf("failed to get virtual machine object: %v", err)
	}

	powerState, err := virtualMachine.PowerState(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to get virtual machine power state: %v", err)
	}

	// We cannot destroy a VM thats powered on, but we also
	// cannot power off a machine that is already off.
	if powerState != types.VirtualMachinePowerStatePoweredOff {
		powerOffTask, err := virtualMachine.PowerOff(context.TODO())
		if err != nil {
			return fmt.Errorf("failed to poweroff vm %s: %v", virtualMachine.Name(), err)
		}
		if err = powerOffTask.Wait(context.TODO()); err != nil {
			return fmt.Errorf("failed to poweroff vm %s: %v", virtualMachine.Name(), err)
		}
	}

	destroyTask, err := virtualMachine.Destroy(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to destroy vm %s: %v", virtualMachine.Name(), err)
	}
	if err = destroyTask.Wait(context.TODO()); err != nil {
		return fmt.Errorf("failed to destroy vm %s: %v", virtualMachine.Name(), err)
	}

	if pc.OperatingSystem != providerconfig.OperatingSystemCoreos {
		datastore, err := finder.Datastore(context.TODO(), config.Datastore)
		if err != nil {
			return fmt.Errorf("failed to get datastore %s: %v", config.Datastore, err)
		}
		filemanager := datastore.NewFileManager(dc, false)

		err = filemanager.Delete(context.TODO(), virtualMachine.Name())
		if err != nil {
			return fmt.Errorf("failed to delete storage of deleted instance %s: %v", virtualMachine.Name(), err)
		}
	}

	glog.V(2).Infof("Successfully destroyed vm %s", virtualMachine.Name())
	return nil
}

func (p *provider) Get(machine *v1alpha1.Machine) (instance.Instance, error) {

	config, _, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	client, err := getClient(config.Username, config.Password, config.VSphereURL, config.AllowInsecure)
	if err != nil {
		return nil, fmt.Errorf("failed to get vsphere client: '%v'", err)
	}
	defer func() {
		if lerr := client.Logout(context.TODO()); lerr != nil {
			utilruntime.HandleError(fmt.Errorf("vsphere client failed to logout: %s", lerr))
		}
	}()

	finder, err := getDatacenterFinder(config.Datacenter, client)
	if err != nil {
		return nil, err
	}
	virtualMachine, err := finder.VirtualMachine(context.TODO(), machine.Spec.Name)
	if err != nil {
		if err.Error() == fmt.Sprintf("vm '%s' not found", machine.Spec.Name) {
			return nil, cloudprovidererrors.ErrInstanceNotFound
		}
		return nil, fmt.Errorf("failed to get server: %v", err)
	}

	powerState, err := virtualMachine.PowerState(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to get powerstate: %v", err)
	}

	var status instance.Status
	switch powerState {
	case types.VirtualMachinePowerStatePoweredOn:
		status = instance.StatusRunning
	default:
		status = instance.StatusUnknown
	}

	isGuestToolsRunning, err := virtualMachine.IsToolsRunning(context.TODO())
	addresses := []string{}
	if isGuestToolsRunning {
		var moVirtualMachine mo.VirtualMachine
		pc := property.DefaultCollector(client.Client)
		err = pc.RetrieveOne(context.TODO(), virtualMachine.Reference(), []string{"guest"}, &moVirtualMachine)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve guest info: %v", err)
		}

		for _, nic := range moVirtualMachine.Guest.Net {
			for _, address := range nic.IpAddress {
				// Exclude ipv6 link-local addresses and default Docker bridge
				if !strings.HasPrefix(address, "fe80:") && !strings.HasPrefix(address, "172.17.") {
					addresses = append(addresses, address)
				}
			}
		}
	} else {
		glog.Warningf("vmware guest utils for machine %s are not running, can't match it to a node!", machine.Spec.Name)
	}

	return Server{name: virtualMachine.Name(), status: status, addresses: addresses, id: virtualMachine.Reference().Value}, nil
}

func (p *provider) GetCloudConfig(spec v1alpha1.MachineSpec) (config string, name string, err error) {
	c, _, _, err := p.getConfig(spec.ProviderConfig)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse config: %v", err)
	}

	passedURL := c.VSphereURL
	// Required because url.Parse returns an empty string for the hostname if there was no schema
	if !strings.HasPrefix(passedURL, "https://") {
		passedURL = "https://" + passedURL
	}

	url, err := url.Parse(passedURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse '%s' as url: %v", passedURL, err)
	}
	port := "443"
	if url.Port() != "" {
		port = url.Port()
	}

	var insecureFlag string
	if c.AllowInsecure {
		insecureFlag = "1"
	} else {
		insecureFlag = "0"
	}

	workingDir := c.Folder
	// Default to basedir
	if workingDir == "" {
		workingDir = fmt.Sprintf("/%s/vm", c.Datacenter)
	}

	config = fmt.Sprintf(`
[Global]
server = "%s"
port = "%s"
user = "%s"
password = "%s"
insecure-flag = "%s" #set to 1 if the vCenter uses a self-signed cert
datastore = "%s"
working-dir = "%s"
datacenter = "%s"
`, url.Hostname(), port, c.Username, c.Password, insecureFlag, c.Datastore, workingDir, c.Datacenter)
	return config, "vsphere", nil
}

func (p *provider) MachineMetricsLabels(machine *v1alpha1.Machine) (map[string]string, error) {
	labels := make(map[string]string)

	c, _, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err == nil {
		labels["size"] = fmt.Sprintf("%d-cpus-%d-mb", c.CPUs, c.MemoryMB)
		labels["dc"] = c.Datacenter
		labels["cluster"] = c.Cluster
	}

	return labels, err
}
