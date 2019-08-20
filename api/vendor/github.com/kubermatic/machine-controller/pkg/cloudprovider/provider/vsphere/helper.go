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
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"text/template"

	"github.com/golang/glog"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

const (
	localTempDir     = "/tmp"
	metaDataTemplate = `instance-id: {{ .InstanceID}}
	local-hostname: {{ .Hostname }}`
)

func createClonedVM(ctx context.Context, vmName string, config *Config, session *Session, containerLinuxUserdata string) (*object.VirtualMachine, error) {
	templateVM, err := session.Finder.VirtualMachine(ctx, config.TemplateVMName)
	if err != nil {
		return nil, fmt.Errorf("failed to get template vm: %v", err)
	}

	glog.V(3).Infof("Template VM ref is %+v", templateVM)

	vmDevices, err := templateVM.Device(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list devices of template VM: %v", err)
	}

	// Find the target folder, if its included in the provider config.
	var targetVMFolder *object.Folder
	if config.Folder != "" {
		// If non-absolute folder name is used, e.g. 'duplicate-folder' it can match
		// multiple folders and thus fail. It will also gladly match a folder from
		// a different datacenter. It is therefore preferable to use absolute folder
		// paths, e.g. '/Datacenter/vm/nested/folder'.
		// The target folder must already exist.
		targetVMFolder, err = session.Finder.Folder(ctx, config.Folder)
		if err != nil {
			return nil, fmt.Errorf("failed to get target folder: %v", err)
		}
	} else {
		// Do not query datacenter folders unless required
		datacenterFolders, err := session.Datacenter.Folders(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get datacenter folders: %v", err)
		}
		targetVMFolder = datacenterFolders.VmFolder
	}

	var vAppAconfig *types.VmConfigSpec
	if containerLinuxUserdata != "" {
		userdataBase64 := base64.StdEncoding.EncodeToString([]byte(containerLinuxUserdata))

		// The properties describing userdata will already exist in the CoreOS VM template.
		// In order to overwrite them, we need to specify their numeric Key values,
		// which we'll extract from that template.
		var mvm mo.VirtualMachine
		if err := templateVM.Properties(ctx, templateVM.Reference(), []string{"config", "config.vAppConfig", "config.vAppConfig.property"}, &mvm); err != nil {
			return nil, fmt.Errorf("failed to extract vapp properties for coreos: %v", err)
		}

		var propertySpecs []types.VAppPropertySpec
		if mvm.Config.VAppConfig.GetVmConfigInfo() == nil {
			return nil, fmt.Errorf("no vm config found in template '%s'. Make sure you import the correct OVA with the appropriate coreos settings", config.TemplateVMName)
		}

		for _, item := range mvm.Config.VAppConfig.GetVmConfigInfo().Property {
			switch item.Id {
			case "guestinfo.coreos.config.data":
				propertySpecs = append(propertySpecs, types.VAppPropertySpec{
					ArrayUpdateSpec: types.ArrayUpdateSpec{
						Operation: types.ArrayUpdateOperationEdit,
					},
					Info: &types.VAppPropertyInfo{
						Key:   item.Key,
						Id:    item.Id,
						Value: userdataBase64,
					},
				})
			case "guestinfo.coreos.config.data.encoding":
				propertySpecs = append(propertySpecs, types.VAppPropertySpec{
					ArrayUpdateSpec: types.ArrayUpdateSpec{
						Operation: types.ArrayUpdateOperationEdit,
					},
					Info: &types.VAppPropertyInfo{
						Key:   item.Key,
						Id:    item.Id,
						Value: "base64",
					},
				})
			}
		}

		vAppAconfig = &types.VmConfigSpec{Property: propertySpecs}
	}

	diskUUIDEnabled := true

	deviceSpecs := []types.BaseVirtualDeviceConfigSpec{}
	if config.DiskSizeGB != nil {
		disks, err := getDisksFromVM(ctx, templateVM)
		if err != nil {
			return nil, fmt.Errorf("failed to get disks from VM: %v", err)
		}
		// If this is wrong, the resulting error is `Invalid operation for device '0`
		// so verify again this is legit
		if err := validateDiskResizing(disks, *config.DiskSizeGB); err != nil {
			return nil, err
		}

		glog.V(4).Infof("Increasing disk size to %d GB", *config.DiskSizeGB)
		disk := disks[0]
		disk.CapacityInBytes = *config.DiskSizeGB * int64(math.Pow(1024, 3))
		diskspec := &types.VirtualDeviceConfigSpec{Operation: types.VirtualDeviceConfigSpecOperationEdit, Device: disk}
		deviceSpecs = append(deviceSpecs, diskspec)
	}

	if config.VMNetName != "" {
		networkSpecs, err := GetNetworkSpecs(ctx, session, vmDevices, config.VMNetName)
		if err != nil {
			return nil, fmt.Errorf("failed to get network specifications: %v", err)
		}
		deviceSpecs = append(deviceSpecs, networkSpecs...)
	}

	// Create a cloned VM from the template VM's snapshot.
	// We split the cloning from the reconfiguring as those actions differ on the permission side.
	// It's nicer to tell which specific action failed due to lacking permissions.
	clonedVMTask, err := templateVM.Clone(ctx, targetVMFolder, vmName, types.VirtualMachineCloneSpec{})
	if err != nil {
		return nil, fmt.Errorf("failed to clone template vm: %v", err)
	}

	if err := clonedVMTask.Wait(ctx); err != nil {
		return nil, fmt.Errorf("error when waiting for result of clone task: %v", err)
	}

	virtualMachine, err := session.Finder.VirtualMachine(ctx, vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual machine object after cloning: %v", err)
	}

	vmConfig := types.VirtualMachineConfigSpec{
		DeviceChange: deviceSpecs,
		Flags: &types.VirtualMachineFlagInfo{
			DiskUuidEnabled: &diskUUIDEnabled,
		},
		NumCPUs:    config.CPUs,
		MemoryMB:   config.MemoryMB,
		VAppConfig: vAppAconfig,
	}
	reconfigureTask, err := virtualMachine.Reconfigure(ctx, vmConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to reconfigure the VM: %v", err)
	}
	if err := reconfigureTask.Wait(ctx); err != nil {
		return nil, fmt.Errorf("error when waiting for result of the reconfigure task: %v", err)
	}

	// Ubuntu wont boot with attached floppy device, because it tries to write to it
	// which fails, because the floppy device does not contain a floppy disk
	// Upstream issue: https://bugs.launchpad.net/cloud-images/+bug/1573095
	if err := removeFloppyDevice(ctx, virtualMachine); err != nil {
		return nil, fmt.Errorf("failed to remove floppy device: %v", err)
	}

	return virtualMachine, nil
}

func uploadAndAttachISO(ctx context.Context, session *Session, vmRef *object.VirtualMachine, localIsoFilePath, datastoreName string) error {
	datastore, err := session.Finder.Datastore(ctx, datastoreName)
	if err != nil {
		return fmt.Errorf("failed to get datastore: %v", err)
	}
	p := soap.DefaultUpload
	remoteIsoFilePath := fmt.Sprintf("%s/%s", vmRef.Name(), "cloud-init.iso")
	glog.V(3).Infof("Uploading userdata ISO to datastore %+v, destination iso is %s\n", datastore, remoteIsoFilePath)
	if err := datastore.UploadFile(ctx, localIsoFilePath, remoteIsoFilePath, &p); err != nil {
		return fmt.Errorf("failed to upload iso: %v", err)
	}
	glog.V(3).Infof("Uploaded ISO file %s", localIsoFilePath)

	// Find the cd-rom device and insert the cloud init iso file into it.
	devices, err := vmRef.Device(ctx)
	if err != nil {
		return fmt.Errorf("failed to get devices: %v", err)
	}

	// passing empty cd-rom name so that the first one gets returned
	cdrom, err := devices.FindCdrom("")
	if err != nil {
		return fmt.Errorf("failed to find cdrom device: %v", err)
	}
	cdrom.Connectable.StartConnected = true
	iso := datastore.Path(remoteIsoFilePath)
	return vmRef.EditDevice(ctx, devices.InsertIso(cdrom, iso))
}

func generateLocalUserdataISO(userdata, name string) (string, error) {
	// We must create a directory, because the iso-generation commands
	// take a directory as input
	userdataDir, err := ioutil.TempDir(localTempDir, name)
	if err != nil {
		return "", fmt.Errorf("failed to create local temp directory for userdata at %s: %v", userdataDir, err)
	}
	defer func() {
		if err := os.RemoveAll(userdataDir); err != nil {
			utilruntime.HandleError(fmt.Errorf("error cleaning up local userdata tempdir %s: %v", userdataDir, err))
		}
	}()

	userdataFilePath := fmt.Sprintf("%s/user-data", userdataDir)
	metadataFilePath := fmt.Sprintf("%s/meta-data", userdataDir)
	isoFilePath := fmt.Sprintf("%s/%s.iso", localTempDir, name)

	metadataTmpl, err := template.New("metadata").Parse(metaDataTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse metadata template: %v", err)
	}
	metadata := &bytes.Buffer{}
	templateContext := struct {
		InstanceID string
		Hostname   string
	}{
		InstanceID: name,
		Hostname:   name,
	}
	if err = metadataTmpl.Execute(metadata, templateContext); err != nil {
		return "", fmt.Errorf("failed to render metadata: %v", err)
	}

	if err := ioutil.WriteFile(userdataFilePath, []byte(userdata), 0644); err != nil {
		return "", fmt.Errorf("failed to locally write userdata file to %s: %v", userdataFilePath, err)
	}

	if err := ioutil.WriteFile(metadataFilePath, metadata.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("failed to locally write metadata file to %s: %v", userdataFilePath, err)
	}

	var command string
	var args []string

	if _, err := exec.LookPath("genisoimage"); err == nil {
		command = "genisoimage"
		args = []string{"-o", isoFilePath, "-volid", "cidata", "-joliet", "-rock", userdataDir}
	} else if _, err := exec.LookPath("mkisofs"); err == nil {
		command = "mkisofs"
		args = []string{"-o", isoFilePath, "-V", "cidata", "-J", "-R", userdataDir}
	} else {
		return "", errors.New("system is missing genisoimage or mkisofs, can't generate userdata iso without it")
	}

	cmd := exec.Command(command, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("error executing command `%s %s`: output: `%s`, error: `%v`", command, args, string(output), err)
	}

	return isoFilePath, nil
}

func removeFloppyDevice(ctx context.Context, virtualMachine *object.VirtualMachine) error {
	vmDevices, err := virtualMachine.Device(ctx)
	if err != nil {
		return fmt.Errorf("failed to get device list: %v", err)
	}

	// If there is more than one floppy device attached, you will simply get the first one. We
	// assume this wont happen.
	floppyDevice, err := vmDevices.FindFloppy("")
	if err != nil {
		if err.Error() == "no floppy device found" {
			return nil
		}
		return fmt.Errorf("failed to find floppy: %v", err)
	}

	if err := virtualMachine.RemoveDevice(ctx, false, floppyDevice); err != nil {
		return fmt.Errorf("failed to remove floppy device: %v", err)
	}

	return nil
}

func getValueForField(ctx context.Context, vm *object.VirtualMachine, fieldName string) (string, error) {
	var mvm mo.VirtualMachine
	if err := vm.Properties(ctx, vm.Reference(), nil, &mvm); err != nil {
		return "", fmt.Errorf("failed to get properties: %v", err)
	}

	var key int32
	for _, availableField := range mvm.AvailableField {
		if availableField.Name == fieldName {
			key = availableField.Key
			break
		}
	}

	for _, value := range mvm.Value {
		if value.GetCustomFieldValue().Key == key {
			stringVal, ok := value.(*types.CustomFieldStringValue)
			if ok {
				return stringVal.Value, nil
			}
			break
		}
	}

	return "", nil
}

func getDisksFromVM(ctx context.Context, vm *object.VirtualMachine) ([]*types.VirtualDisk, error) {
	var props mo.VirtualMachine
	if err := vm.Properties(ctx, vm.Reference(), nil, &props); err != nil {
		return nil, fmt.Errorf("error getting VM template reference: %v", err)
	}
	l := object.VirtualDeviceList(props.Config.Hardware.Device)
	disks := l.SelectByType((*types.VirtualDisk)(nil))

	var result []*types.VirtualDisk
	for _, disk := range disks {
		if assertedDisk := disk.(*types.VirtualDisk); assertedDisk != nil {
			result = append(result, assertedDisk)
		}
	}
	return result, nil
}

func validateDiskResizing(disks []*types.VirtualDisk, requestedSize int64) error {
	if diskLen := len(disks); diskLen != 1 {
		return fmt.Errorf("expected vm to have exactly one disk, got %d", diskLen)
	}
	requestedCapacityInBytes := requestedSize * int64(math.Pow(1024, 3))
	if requestedCapacityInBytes < disks[0].CapacityInBytes {
		attachedDiskSizeInGiB := disks[0].CapacityInBytes / int64(math.Pow(1024, 3))
		return fmt.Errorf("requested diskSizeGB %d is smaller than size of attached disk(%dGiB)", requestedSize, attachedDiskSizeInGiB)
	}
	return nil
}
