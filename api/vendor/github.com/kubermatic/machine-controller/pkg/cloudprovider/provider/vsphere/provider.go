/*
Copyright 2019 The Machine Controller Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vsphere

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/common"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	cloudprovidererrors "github.com/kubermatic/machine-controller/pkg/cloudprovider/errors"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/instance"
	cloudprovidertypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/types"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
)

const (
	// We set this field on the virtual machine to the name of
	// the machine to indicate creation succeeded.
	// If this is not set correctly, .Get will delete the instance
	creationCompleteFieldName = "kubernetes-worker-complete"
	// machineUIDFieldName is the name of the field in which we
	// store the machines UID
	machineUIDFieldName = "kubernetes-machine-uid"
)

type provider struct {
	configVarResolver *providerconfig.ConfigVarResolver
}

// New returns a VSphere provider
func New(configVarResolver *providerconfig.ConfigVarResolver) cloudprovidertypes.Provider {
	return &provider{configVarResolver: configVarResolver}
}

type RawConfig struct {
	TemplateVMName providerconfig.ConfigVarString `json:"templateVMName"`
	VMNetName      providerconfig.ConfigVarString `json:"vmNetName,omitempty"`
	Username       providerconfig.ConfigVarString `json:"username,omitempty"`
	Password       providerconfig.ConfigVarString `json:"password,omitempty"`
	VSphereURL     providerconfig.ConfigVarString `json:"vsphereURL,omitempty"`
	Datacenter     providerconfig.ConfigVarString `json:"datacenter"`
	Cluster        providerconfig.ConfigVarString `json:"cluster"`
	Folder         providerconfig.ConfigVarString `json:"folder,omitempty"`
	Datastore      providerconfig.ConfigVarString `json:"datastore"`
	CPUs           int32                          `json:"cpus"`
	MemoryMB       int64                          `json:"memoryMB"`
	DiskSizeGB     *int64                         `json:"diskSizeGB,omitempty"`
	AllowInsecure  providerconfig.ConfigVarBool   `json:"allowInsecure"`
}

type Config struct {
	TemplateVMName string
	VMNetName      string
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
	DiskSizeGB     *int64
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

func (p *provider) AddDefaults(spec v1alpha1.MachineSpec) (v1alpha1.MachineSpec, error) {
	return spec, nil
}

func (p *provider) getConfig(s v1alpha1.ProviderSpec) (*Config, *providerconfig.Config, *RawConfig, error) {
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
	c.DiskSizeGB = rawConfig.DiskSizeGB

	return &c, &pconfig, &rawConfig, nil
}

func (p *provider) Validate(spec v1alpha1.MachineSpec) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config, _, _, err := p.getConfig(spec.ProviderSpec)
	if err != nil {
		return fmt.Errorf("failed to get config: %v", err)
	}

	session, err := NewSession(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create vCenter session: %v", err)
	}
	defer session.Logout()

	if _, err := session.Finder.Datastore(ctx, config.Datastore); err != nil {
		return fmt.Errorf("failed to get datastore %s: %v", config.Datastore, err)
	}

	if _, err := session.Finder.ClusterComputeResource(ctx, config.Cluster); err != nil {
		return fmt.Errorf("failed to get cluster: %s: %v", config.Cluster, err)
	}

	templateVM, err := session.Finder.VirtualMachine(ctx, config.TemplateVMName)
	if err != nil {
		return fmt.Errorf("failed to get template vm %q: %v", config.TemplateVMName, err)
	}

	disks, err := getDisksFromVM(ctx, templateVM)
	if err != nil {
		return fmt.Errorf("failed to get disks from VM: %v", err)
	}
	if diskLen := len(disks); diskLen != 1 {
		return fmt.Errorf("expected vm to have exactly one disk, had %d", diskLen)
	}

	if config.DiskSizeGB != nil {
		if err := validateDiskResizing(disks, *config.DiskSizeGB); err != nil {
			return err
		}
	}

	return nil
}

func machineInvalidConfigurationTerminalError(err error) error {
	return cloudprovidererrors.TerminalError{
		Reason:  common.InvalidConfigurationMachineError,
		Message: err.Error(),
	}
}

func (p *provider) Create(machine *v1alpha1.Machine, _ *cloudprovidertypes.ProviderData, userdata string) (instance.Instance, error) {
	ctx := context.Background()

	config, pc, _, err := p.getConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	session, err := NewSession(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vCenter session: %v", err)
	}
	defer session.Logout()

	var containerLinuxUserdata string
	if pc.OperatingSystem == providerconfig.OperatingSystemCoreos {
		containerLinuxUserdata = userdata
	}

	virtualMachine, err := createClonedVM(ctx,
		machine.Spec.Name,
		config,
		session,
		containerLinuxUserdata,
	)
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

		if err := uploadAndAttachISO(ctx, session, virtualMachine, localUserdataIsoFilePath, config.Datastore); err != nil {
			// Destroy VM to avoid a leftover.
			destroyTask, vmErr := virtualMachine.Destroy(ctx)
			if vmErr != nil {
				return nil, fmt.Errorf("failed to destroy vm %s after failing upload and attach userdata iso: %v / %v", virtualMachine.Name(), err, vmErr)
			}
			if vmErr := destroyTask.Wait(ctx); vmErr != nil {
				return nil, fmt.Errorf("failed to destroy vm %s after failing upload and attach userdata iso: %v / %v", virtualMachine.Name(), err, vmErr)
			}
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

	// Add a custom field to indicate to our Get that creation succeeded
	// If the field is not set, Get will delete the instance
	customFieldManager, err := object.GetCustomFieldsManager(session.Client.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to get customFieldManager: %v", err)
	}
	machineUIDFieldKey, err := createOrGetFieldIndex(ctx, machineUIDFieldName, customFieldManager)
	if err != nil {
		return nil, fmt.Errorf("failed to get field key for field %q: %v", machineUIDFieldName, err)
	}
	if err := customFieldManager.Set(ctx, virtualMachine.Reference(), machineUIDFieldKey, string(machine.UID)); err != nil {
		return nil, fmt.Errorf("failed to set field %q to value %q: %v", machineUIDFieldKey, string(machine.UID), err)
	}
	creationCompleteFieldKey, err := createOrGetFieldIndex(ctx, creationCompleteFieldName, customFieldManager)
	if err != nil {
		return nil, fmt.Errorf("failed to get field key for field %q: %v", creationCompleteFieldName, err)
	}
	if err := customFieldManager.Set(ctx, virtualMachine.Reference(), creationCompleteFieldKey, machine.Spec.Name); err != nil {
		return nil, fmt.Errorf("failed to set field %q to value %q: %v", creationCompleteFieldName, machine.Spec.Name, err)
	}

	return Server{name: virtualMachine.Name(), status: instance.StatusRunning, id: virtualMachine.Reference().Value}, nil
}

func createOrGetFieldIndex(ctx context.Context, fieldName string, customFieldManager *object.CustomFieldsManager) (int32, error) {
	key, err := customFieldManager.FindKey(ctx, fieldName)
	if err != nil {
		if !strings.Contains(err.Error(), "key name not found") {
			return 0, fmt.Errorf("error trying to get field with key %q: %v", fieldName, err)
		}
		field, err := customFieldManager.Add(ctx, fieldName, "VirtualMachine", nil, nil)
		if err != nil {
			return 0, fmt.Errorf("failed to add field %q: %v", fieldName, err)
		}
		key = field.Key
	}
	return key, nil
}

func (p *provider) Cleanup(machine *v1alpha1.Machine, data *cloudprovidertypes.ProviderData) (bool, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, pc, _, err := p.getConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return false, fmt.Errorf("failed to parse config: %v", err)
	}

	session, err := NewSession(ctx, config)
	if err != nil {
		return false, fmt.Errorf("failed to create vCenter session: %v", err)
	}
	defer session.Logout()

	virtualMachine, err := p.get(ctx, machine, session.Finder)
	if err != nil {
		if cloudprovidererrors.IsNotFound(err) {
			return true, nil
		}
		return false, fmt.Errorf("failed to get instance from vSphere: %v", err)
	}

	powerState, err := virtualMachine.PowerState(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get virtual machine power state: %v", err)
	}

	// We cannot destroy a VM thats powered on, but we also
	// cannot power off a machine that is already off.
	if powerState != types.VirtualMachinePowerStatePoweredOff {
		powerOffTask, err := virtualMachine.PowerOff(ctx)
		if err != nil {
			return false, fmt.Errorf("failed to poweroff vm %s: %v", virtualMachine.Name(), err)
		}
		if err = powerOffTask.Wait(ctx); err != nil {
			return false, fmt.Errorf("failed to poweroff vm %s: %v", virtualMachine.Name(), err)
		}
	}

	virtualMachineDeviceList, err := virtualMachine.Device(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get devices for virtual machine: %v", err)
	}

	pvs := &corev1.PersistentVolumeList{}
	if err := data.Client.List(data.Ctx, &ctrlruntimeclient.ListOptions{}, pvs); err != nil {
		return false, fmt.Errorf("failed to list PVs: %v", err)
	}

	for _, pv := range pvs.Items {
		if pv.Spec.VsphereVolume == nil {
			continue
		}
		for _, device := range virtualMachineDeviceList {
			if virtualMachineDeviceList.Type(device) == object.DeviceTypeDisk {
				fileName := device.GetVirtualDevice().Backing.(types.BaseVirtualDeviceFileBackingInfo).GetVirtualDeviceFileBackingInfo().FileName
				if pv.Spec.VsphereVolume.VolumePath == fileName {
					if err := virtualMachine.RemoveDevice(ctx, true, device); err != nil {
						return false, fmt.Errorf("error detaching pv-backing disk %s: %v", fileName, err)
					}
				}
			}
		}
	}

	destroyTask, err := virtualMachine.Destroy(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to destroy vm %s: %v", virtualMachine.Name(), err)
	}
	if err := destroyTask.Wait(ctx); err != nil {
		return false, fmt.Errorf("failed to destroy vm %s: %v", virtualMachine.Name(), err)
	}

	if pc.OperatingSystem != providerconfig.OperatingSystemCoreos {
		datastore, err := session.Finder.Datastore(ctx, config.Datastore)
		if err != nil {
			return false, fmt.Errorf("failed to get datastore %s: %v", config.Datastore, err)
		}
		filemanager := datastore.NewFileManager(session.Datacenter, false)

		if err := filemanager.Delete(ctx, virtualMachine.Name()); err != nil {
			if err.Error() == fmt.Sprintf("File [%s] %s was not found", datastore.Name(), virtualMachine.Name()) {
				return true, nil
			}
			return false, fmt.Errorf("failed to delete storage of deleted instance %s: %v", virtualMachine.Name(), err)
		}
	}

	glog.V(2).Infof("Successfully destroyed vm %s", virtualMachine.Name())
	return true, nil
}

func (p *provider) Get(machine *v1alpha1.Machine, data *cloudprovidertypes.ProviderData) (instance.Instance, error) {
	ctx := context.Background()

	config, _, _, err := p.getConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	session, err := NewSession(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vCenter session: %v", err)
	}
	defer session.Logout()

	virtualMachineList, err := session.Finder.VirtualMachineList(ctx, machine.Spec.Name)
	if err != nil {
		if err.Error() == fmt.Sprintf("vm '%s' not found", machine.Spec.Name) {
			return nil, cloudprovidererrors.ErrInstanceNotFound
		}
		return nil, fmt.Errorf("failed to list virtual machines: %v", err)
	}

	var virtualMachine *object.VirtualMachine
	for _, virtualMachineItem := range virtualMachineList {
		// Check if the creationCompleteFieldName is set to machine.UID
		// If that is not the case, the creation didn't complete successfully and
		// we must delete the instance so it gets recreated
		creationCompleteFieldValue, err := getValueForField(ctx,
			virtualMachineItem, creationCompleteFieldName)
		if err != nil {
			return nil, fmt.Errorf("failed to get value for field: %v", err)
		}
		machineCreationCompletedSuccessfully := creationCompleteFieldValue == machine.Spec.Name
		if !machineCreationCompletedSuccessfully {
			glog.V(4).Infof("Cleaning up instance %q whose creation didn't complete", machine.Spec.Name)
			if _, err := p.Cleanup(machine, data); err != nil {
				return nil, fmt.Errorf("failed to delete instance whose creation didn't complete: %v", err)
			}
			continue
		}

		machineUIDFieldValue, err := getValueForField(ctx, virtualMachineItem, machineUIDFieldName)
		if err != nil {
			return nil, fmt.Errorf("Failed to get value for field %q: %v", machineUIDFieldName, err)
		}
		if machineUIDFieldValue == string(machine.UID) {
			virtualMachine = virtualMachineItem
			break
		}
	}
	if virtualMachine == nil {
		return nil, cloudprovidererrors.ErrInstanceNotFound
	}

	powerState, err := virtualMachine.PowerState(ctx)
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

	// virtualMachine.IsToolsRunning panics when executed on a VM that is not powered on
	addresses := []string{}
	if powerState == types.VirtualMachinePowerStatePoweredOn {
		isGuestToolsRunning, err := virtualMachine.IsToolsRunning(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("failed to check if guest utils are running: %v", err)
		}
		if isGuestToolsRunning {
			var moVirtualMachine mo.VirtualMachine
			pc := property.DefaultCollector(session.Client.Client)
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
			glog.V(3).Infof("Can't fetch the IP addresses for machine %s, the VMware guest utils are not running yet. This might take a few minutes", machine.Spec.Name)
		}
	}

	return Server{name: virtualMachine.Name(), status: status, addresses: addresses, id: virtualMachine.Reference().Value}, nil
}

func (p *provider) MigrateUID(machine *v1alpha1.Machine, new ktypes.UID) error {
	ctx := context.Background()

	config, _, _, err := p.getConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	session, err := NewSession(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create vCenter session: %v", err)
	}
	defer session.Logout()

	virtualMachine, err := p.get(ctx, machine, session.Finder)
	if err != nil {
		if cloudprovidererrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get instance from vSphere: %v", err)
	}

	customFieldManager, err := object.GetCustomFieldsManager(session.Client.Client)
	if err != nil {
		return fmt.Errorf("failed to get customFieldManager: %v", err)
	}
	machineUIDFieldKey, err := createOrGetFieldIndex(ctx, machineUIDFieldName, customFieldManager)
	if err != nil {
		return fmt.Errorf("failed to get field key for field %q: %v", machineUIDFieldName, err)
	}
	if err := customFieldManager.Set(ctx, virtualMachine.Reference(), machineUIDFieldKey, string(new)); err != nil {
		return fmt.Errorf("failed to set field %q to value %q: %v", machineUIDFieldKey, string(new), err)
	}
	return nil
}

func (p *provider) GetCloudConfig(spec v1alpha1.MachineSpec) (config string, name string, err error) {
	c, _, _, err := p.getConfig(spec.ProviderSpec)
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

	c, _, _, err := p.getConfig(machine.Spec.ProviderSpec)
	if err == nil {
		labels["size"] = fmt.Sprintf("%d-cpus-%d-mb", c.CPUs, c.MemoryMB)
		labels["dc"] = c.Datacenter
		labels["cluster"] = c.Cluster
	}

	return labels, err
}

func (p *provider) SetMetricsForMachines(machines v1alpha1.MachineList) error {
	return nil
}

func (p *provider) get(ctx context.Context, machine *v1alpha1.Machine, datacenterFinder *find.Finder) (*object.VirtualMachine, error) {
	virtualMachineList, err := datacenterFinder.VirtualMachineList(ctx, machine.Spec.Name)
	if err != nil {
		if err.Error() == fmt.Sprintf("vm '%s' not found", machine.Spec.Name) {
			return nil, cloudprovidererrors.ErrInstanceNotFound
		}
		return nil, fmt.Errorf("failed to list virtual machines: %v", err)
	}

	for _, virtualMachine := range virtualMachineList {
		machineUIDFieldValue, err := getValueForField(ctx, virtualMachine, machineUIDFieldName)
		if err != nil {
			return nil, fmt.Errorf("Failed to get value for field %q: %v", machineUIDFieldName, err)
		}
		if machineUIDFieldValue == string(machine.UID) {
			return virtualMachine, nil
		}
	}

	return nil, cloudprovidererrors.ErrInstanceNotFound
}
