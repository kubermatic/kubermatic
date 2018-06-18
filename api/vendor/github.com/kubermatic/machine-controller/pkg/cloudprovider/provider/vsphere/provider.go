package vsphere

import (
	"context"
	"encoding/json"
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

// New returns a VSphere provider
func New(configVarResolver *providerconfig.ConfigVarResolver) cloud.Provider {
	return &provider{configVarResolver: configVarResolver}
}

type RawConfig struct {
	TemplateVMName providerconfig.ConfigVarString `json:"templateVMName"`
	Username       providerconfig.ConfigVarString `json:"username"`
	Password       providerconfig.ConfigVarString `json:"password"`
	VSphereURL     providerconfig.ConfigVarString `json:"vsphereURL"`
	Datacenter     providerconfig.ConfigVarString `json:"datacenter"`
	Cluster        providerconfig.ConfigVarString `json:"cluster"`
	Folder         providerconfig.ConfigVarString `json:"folder"`
	Datastore      providerconfig.ConfigVarString `json:"datastore"`
	CPUs           int32                          `json:"cpus"`
	MemoryMB       int64                          `json:"memoryMB"`
	AllowInsecure  providerconfig.ConfigVarBool   `json:"allowInsecure"`
}

type Config struct {
	TemplateVMName string
	Username       string
	Password       string
	VSphereURL     string
	Datacenter     string
	Cluster        string
	Folder         string
	Datastore      string
	AllowInsecure  bool
	CPUs           int32
	MemoryMB       int64
}

type VSphereServer struct {
	name      string
	id        string
	status    instance.Status
	addresses []string
}

func (vsphereServer VSphereServer) Name() string {
	return vsphereServer.name
}

func (vsphereServer VSphereServer) ID() string {
	return vsphereServer.id
}

func (vsphereServer VSphereServer) Addresses() []string {
	return vsphereServer.addresses
}

func (vsphereServer VSphereServer) Status() instance.Status {
	return vsphereServer.status
}

func (p *provider) AddDefaults(spec v1alpha1.MachineSpec) (v1alpha1.MachineSpec, bool, error) {
	return spec, false, nil
}

func getClient(username, password, address string, allowInsecure bool) (*govmomi.Client, error) {
	clientUrl, err := url.Parse(fmt.Sprintf("%s/sdk", address))
	if err != nil {
		return nil, err
	}
	clientUrl.User = url.UserPassword(username, password)

	return govmomi.NewClient(context.TODO(), clientUrl, allowInsecure)
}

func (p *provider) getConfig(s runtime.RawExtension) (*Config, *providerconfig.Config, error) {
	pconfig := providerconfig.Config{}
	err := json.Unmarshal(s.Raw, &pconfig)
	if err != nil {
		return nil, nil, err
	}

	rawConfig := RawConfig{}
	err = json.Unmarshal(pconfig.CloudProviderSpec.Raw, &rawConfig)
	if err != nil {
		return nil, nil, err
	}

	c := Config{}
	c.TemplateVMName, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.TemplateVMName)
	if err != nil {
		return nil, nil, err
	}

	c.Username, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Username)
	if err != nil {
		return nil, nil, err
	}

	c.Password, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Password)
	if err != nil {
		return nil, nil, err
	}

	c.VSphereURL, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.VSphereURL)
	if err != nil {
		return nil, nil, err
	}

	c.Datacenter, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Datacenter)
	if err != nil {
		return nil, nil, err
	}

	c.Cluster, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Cluster)
	if err != nil {
		return nil, nil, err
	}

	c.Folder, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Folder)
	if err != nil {
		return nil, nil, err
	}

	c.Datastore, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Datastore)
	if err != nil {
		return nil, nil, err
	}

	c.AllowInsecure, err = p.configVarResolver.GetConfigVarBoolValue(rawConfig.AllowInsecure)
	if err != nil {
		return nil, nil, err
	}

	c.CPUs = rawConfig.CPUs
	c.MemoryMB = rawConfig.MemoryMB

	return &c, &pconfig, nil
}

func (p *provider) Validate(spec v1alpha1.MachineSpec) error {
	config, _, err := p.getConfig(spec.ProviderConfig)
	if err != nil {
		return err
	}

	client, err := getClient(config.Username, config.Password, config.VSphereURL, config.AllowInsecure)
	if err != nil {
		return fmt.Errorf("failed to get vsphere client: '%v'", err)
	}
	defer client.Logout(context.TODO())

	finder, err := getDatacenterFinder(config.Datacenter, client)
	if err != nil {
		return err
	}

	_, err = finder.Datastore(context.TODO(), config.Datastore)
	if err != nil {
		return err
	}

	_, err = finder.ClusterComputeResource(context.TODO(), config.Cluster)
	if err != nil {
		return err
	}
	return nil
}

func machineInvalidConfigurationTerminalError(err error) error {
	return cloudprovidererrors.TerminalError{
		Reason:  v1alpha1.InvalidConfigurationMachineError,
		Message: err.Error(),
	}
}

func (p *provider) Create(machine *v1alpha1.Machine, userdata string) (instance.Instance, error) {
	config, pc, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	client, err := getClient(config.Username, config.Password, config.VSphereURL, config.AllowInsecure)
	if err != nil {
		return nil, fmt.Errorf("failed to get vsphere client: '%v'", err)
	}
	defer client.Logout(context.TODO())

	var containerLinuxUserdata string
	if pc.OperatingSystem == providerconfig.OperatingSystemCoreos {
		containerLinuxUserdata = userdata
	}

	if err = createLinkClonedVm(machine.Spec.Name,
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

	if pc.OperatingSystem != providerconfig.OperatingSystemCoreos {
		localUserdataIsoFilePath, err := generateLocalUserdataIso(userdata, machine.Spec.Name)
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

	return VSphereServer{name: virtualMachine.Name(), status: instance.StatusRunning, id: virtualMachine.Reference().Value}, nil
}

func (p *provider) Delete(machine *v1alpha1.Machine, _ instance.Instance) error {
	config, pc, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	client, err := getClient(config.Username, config.Password, config.VSphereURL, config.AllowInsecure)
	if err != nil {
		return fmt.Errorf("failed to get vsphere client: '%v'", err)
	}
	defer client.Logout(context.TODO())
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

	// We can't destroy a VM thats powered on...
	powerOffTask, err := virtualMachine.PowerOff(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to poweroff vm %s: %v", virtualMachine.Name(), err)
	}
	powerOffTask.Wait(context.TODO())

	destroyTask, err := virtualMachine.Destroy(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to destroy vm %s: %v", virtualMachine.Name(), err)
	}
	destroyTask.Wait(context.TODO())

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
	config, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	client, err := getClient(config.Username, config.Password, config.VSphereURL, config.AllowInsecure)
	if err != nil {
		return nil, fmt.Errorf("failed to get vsphere client: '%v'", err)
	}
	defer client.Logout(context.TODO())

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

	return VSphereServer{name: virtualMachine.Name(), status: status, addresses: addresses, id: virtualMachine.Reference().Value}, nil
}

func (p *provider) GetCloudConfig(spec v1alpha1.MachineSpec) (config string, name string, err error) {
	//TODO: Implement this
	return "", "", nil
}
