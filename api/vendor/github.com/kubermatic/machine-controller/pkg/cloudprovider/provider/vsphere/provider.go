package vsphere

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

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
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	common "sigs.k8s.io/cluster-api/pkg/apis/cluster/common"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
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
			Reason:  common.InvalidConfigurationMachineError,
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
			return spec, changed, fmt.Errorf("failed to get datacenter finder: %v", err)
		}

		templateVM, err := finder.VirtualMachine(ctx, cfg.TemplateVMName)
		if err != nil {
			return spec, changed, fmt.Errorf("failed to get virtual machine: %v", err)
		}

		availableNetworkDevices, err := getNetworkDevicesAndBackingsFromVM(ctx, templateVM, "")
		if err != nil {
			return spec, changed, fmt.Errorf("failed to get network devices for vm: %v", err)
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

	spec.ProviderConfig.Value, err = setProviderConfig(*rawCfg, spec.ProviderConfig)
	if err != nil {
		return spec, changed, fmt.Errorf("error marshaling providerconfig: %s", err)
	}

	return spec, changed, nil
}

func setProviderConfig(rawConfig RawConfig, s v1alpha1.ProviderConfig) (*runtime.RawExtension, error) {
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

func getClient(username, password, address string, allowInsecure bool) (*govmomi.Client, error) {
	clientURL, err := url.Parse(fmt.Sprintf("%s/sdk", address))
	if err != nil {
		return nil, err
	}
	clientURL.User = url.UserPassword(username, password)

	return govmomi.NewClient(context.TODO(), clientURL, allowInsecure)
}

func (p *provider) getConfig(s v1alpha1.ProviderConfig) (*Config, *providerconfig.Config, *RawConfig, error) {
	if s.Value == nil {
		return nil, nil, nil, fmt.Errorf("machine.spec.providerconfig.value is nil")
	}
	pconfig := providerconfig.Config{}
	err := json.Unmarshal(s.Value.Raw, &pconfig)
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config, _, _, err := p.getConfig(spec.ProviderConfig)
	if err != nil {
		return fmt.Errorf("failed to get config: %v", err)
	}

	if config.VMNetName != "" && config.TemplateNetName == "" {
		return errors.New("specified target network (VMNetName) in cluster, but no source network (TemplateNetName) in machine")
	}

	client, err := getClient(config.Username, config.Password, config.VSphereURL, config.AllowInsecure)
	if err != nil {
		return fmt.Errorf("failed to get vsphere client: '%v'", err)
	}
	defer func() {
		if err := client.Logout(context.Background()); err != nil {
			utilruntime.HandleError(fmt.Errorf("vsphere client failed to logout: %s", err))
		}
	}()

	finder, err := getDatacenterFinder(config.Datacenter, client)
	if err != nil {
		return fmt.Errorf("failed to get datacenter %s: %v", config.Datacenter, err)
	}

	if _, err := finder.Datastore(ctx, config.Datastore); err != nil {
		return fmt.Errorf("failed to get datastore %s: %v", config.Datastore, err)
	}

	if _, err := finder.ClusterComputeResource(ctx, config.Cluster); err != nil {
		return fmt.Errorf("failed to get cluster: %s: %v", config.Cluster, err)
	}

	return nil
}

func machineInvalidConfigurationTerminalError(err error) error {
	return cloudprovidererrors.TerminalError{
		Reason:  common.InvalidConfigurationMachineError,
		Message: err.Error(),
	}
}

func (p *provider) Create(machine *v1alpha1.Machine, _ cloud.MachineUpdater, userdata string) (instance.Instance, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, pc, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	client, err := getClient(config.Username, config.Password, config.VSphereURL, config.AllowInsecure)
	if err != nil {
		return nil, fmt.Errorf("failed to get vsphere client: '%v'", err)
	}
	defer func() {
		if err := client.Logout(context.Background()); err != nil {
			utilruntime.HandleError(fmt.Errorf("vsphere client failed to logout: %s", err))
		}
	}()

	var containerLinuxUserdata string
	if pc.OperatingSystem == providerconfig.OperatingSystemCoreos {
		containerLinuxUserdata = userdata
	}

	finder := find.NewFinder(client.Client, true)
	dc, err := finder.Datacenter(ctx, config.Datacenter)
	if err != nil {
		return nil, fmt.Errorf("failed to get datacenter: %v", err)
	}
	finder.SetDatacenter(dc)

	virtualMachine, err := createClonedVM(ctx,
		machine.Spec.Name,
		config,
		dc,
		finder,
		containerLinuxUserdata)
	if err != nil {
		return nil, machineInvalidConfigurationTerminalError(fmt.Errorf("failed to create cloned vm: '%v'", err))
	}

	if pc.OperatingSystem != providerconfig.OperatingSystemCoreos {
		localUserdataIsoFilePath, err := generateLocalUserdataISO(userdata, machine.Spec.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to generate local userdadata iso: %v", err)
		}

		defer func() {
			err := os.Remove(localUserdataIsoFilePath)
			if err != nil {
				utilruntime.HandleError(fmt.Errorf("failed to clean up local userdata iso file at %s: %v", localUserdataIsoFilePath, err))
			}
		}()

		if err := uploadAndAttachISO(ctx, finder, virtualMachine, localUserdataIsoFilePath, config.Datastore); err != nil {
			return nil, machineInvalidConfigurationTerminalError(fmt.Errorf("failed to upload and attach userdata iso: %v", err))
		}
	}

	powerOnTask, err := virtualMachine.PowerOn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to power on machine: %v", err)
	}

	if err := powerOnTask.Wait(ctx); err != nil {
		return nil, fmt.Errorf("error when waiting for vm powerOn task: %v", err)
	}

	return Server{name: virtualMachine.Name(), status: instance.StatusRunning, id: virtualMachine.Reference().Value}, nil
}

func (p *provider) Delete(machine *v1alpha1.Machine, _ cloud.MachineUpdater) error {
	if _, err := p.Get(machine); err != nil {
		if err == cloudprovidererrors.ErrInstanceNotFound {
			return nil
		}
		return fmt.Errorf("failed to get instance: %v", err)
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
		if err := client.Logout(context.TODO()); err != nil {
			utilruntime.HandleError(fmt.Errorf("vsphere client failed to logout: %s", err))
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
	if err := destroyTask.Wait(context.TODO()); err != nil {
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
		return nil, fmt.Errorf("failed to get datacenter finder: %v", err)
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
		if err := pc.RetrieveOne(context.TODO(), virtualMachine.Reference(), []string{"guest"}, &moVirtualMachine); err != nil {
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

	u, err := url.Parse(passedURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse '%s' as url: %v", passedURL, err)
	}

	workingDir := c.Folder
	// Default to basedir
	if workingDir == "" {
		workingDir = fmt.Sprintf("/%s/vm", c.Datacenter)
	}

	cc := &CloudConfig{
		Global: GlobalOpts{
			User:         c.Username,
			Password:     c.Password,
			InsecureFlag: c.AllowInsecure,
			VCenterPort:  u.Port(),
		},
		Disk: DiskOpts{
			SCSIControllerType: "pvscsi",
		},
		Workspace: WorkspaceOpts{
			Datacenter:       c.Datacenter,
			VCenterIP:        u.Hostname(),
			DefaultDatastore: c.Datastore,
			Folder:           workingDir,
		},
		VirtualCenter: map[string]*VirtualCenterConfig{
			u.Hostname(): {
				VCenterPort: u.Port(),
				Datacenters: c.Datacenter,
				User:        c.Username,
				Password:    c.Password,
			},
		},
	}

	s, err := CloudConfigToString(cc)
	if err != nil {
		return "", "", fmt.Errorf("failed to convert the cloud-config to string: %v", err)
	}

	return s, "vsphere", nil
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
