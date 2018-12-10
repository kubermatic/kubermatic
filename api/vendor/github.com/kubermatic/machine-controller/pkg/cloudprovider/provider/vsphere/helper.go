package vsphere

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"text/template"

	"github.com/golang/glog"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

const (
	snapshotName     = "machine-controller"
	snapshotDesc     = "Snapshot created by machine-controller"
	localTempDir     = "/tmp"
	metaDataTemplate = `instance-id: {{ .InstanceID}}
	local-hostname: {{ .Hostname }}`
)

var errSnapshotNotFound = errors.New("no snapshot with given name found")

func createClonedVM(ctx context.Context, vmName string, config *Config, dc *object.Datacenter, f *find.Finder, containerLinuxUserdata string) (*object.VirtualMachine, error) {
	templateVM, err := f.VirtualMachine(ctx, config.TemplateVMName)
	if err != nil {
		return nil, fmt.Errorf("failed to get template vm: %v", err)
	}

	glog.V(3).Infof("Template VM ref is %+v", templateVM)

	// Find the target folder, if its included in the provider config.
	var targetVMFolder *object.Folder
	if config.Folder != "" {
		// If non-absolute folder name is used, e.g. 'duplicate-folder' it can match
		// multiple folders and thus fail. It will also gladly match a folder from
		// a different datacenter. It is therefore preferable to use absolute folder
		// paths, e.g. '/Datacenter/vm/nested/folder'.
		// The target folder must already exist.
		targetVMFolder, err = f.Folder(ctx, config.Folder)
		if err != nil {
			return nil, fmt.Errorf("failed to get target folder: %v", err)
		}
	} else {
		// Do not query datacenter folders unless required
		datacenterFolders, err := dc.Folders(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get datacenter folders: %v", err)
		}
		targetVMFolder = datacenterFolders.VmFolder
	}

	// Create snapshot of the template VM if not already snapshotted.
	snapshot, err := findSnapshot(ctx, templateVM, snapshotName)
	if err != nil {
		if err != errSnapshotNotFound {
			return nil, fmt.Errorf("failed to find snapshot: %v", err)
		}
		snapshot, err = createSnapshot(ctx, templateVM, snapshotName, snapshotDesc)
		if err != nil {
			return nil, fmt.Errorf("failed to create snapshot: %v", err)
		}
	}

	snapshotRef := snapshot.Reference()

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
	desiredConfig := types.VirtualMachineConfigSpec{
		Flags: &types.VirtualMachineFlagInfo{
			DiskUuidEnabled: &diskUUIDEnabled,
		},
		NumCPUs:    config.CPUs,
		MemoryMB:   config.MemoryMB,
		VAppConfig: vAppAconfig,
	}

	// Create a cloned VM from the template VM's snapshot
	clonedVMTask, err := templateVM.Clone(ctx, targetVMFolder, vmName, types.VirtualMachineCloneSpec{Snapshot: &snapshotRef})
	if err != nil {
		return nil, fmt.Errorf("failed to clone template vm: %v", err)
	}

	if err := clonedVMTask.Wait(ctx); err != nil {
		return nil, fmt.Errorf("error when waiting for result of clone task: %v", err)
	}

	virtualMachine, err := f.VirtualMachine(ctx, vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual machine object after cloning: %v", err)
	}

	reconfigureTask, err := virtualMachine.Reconfigure(ctx, desiredConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to reconfigure vm: %v", err)
	}

	if err := reconfigureTask.Wait(ctx); err != nil {
		return nil, fmt.Errorf("error waiting for reconfigure task to finish: %v", err)
	}

	// Update network if requested
	if config.VMNetName != "" {
		if err := updateNetworkForVM(ctx, virtualMachine, config.TemplateNetName, config.VMNetName); err != nil {
			return nil, fmt.Errorf("couldn't set network for vm: %v", err)
		}
	}

	// Ubuntu wont boot with attached floppy device, because it tries to write to it
	// which fails, because the floppy device does not contain a floppy disk
	// Upstream issue: https://bugs.launchpad.net/cloud-images/+bug/1573095
	if err := removeFloppyDevice(ctx, virtualMachine); err != nil {
		return nil, fmt.Errorf("failed to remove floppy device: %v", err)
	}

	return virtualMachine, nil
}

func updateNetworkForVM(ctx context.Context, vm *object.VirtualMachine, currentNetName string, newNetName string) error {
	newNet, err := getNetworkFromVM(ctx, vm, newNetName)
	if err != nil {
		return fmt.Errorf("failed to get network from vm: %v", err)
	}

	availableData, err := getNetworkDevicesAndBackingsFromVM(ctx, vm, currentNetName)
	if err != nil {
		return fmt.Errorf("failed to get network devices for vm: %v", err)
	}
	if len(availableData) == 0 {
		return errors.New("found no matching network adapter")
	}

	netDev := availableData[0].device
	currentBacking := availableData[0].backingInfo

	glog.V(6).Infof("changing network `%s` to `%s` for vm `%s`", currentBacking.DeviceName, newNetName, vm.Name())
	currentBacking.DeviceName = newNetName
	currentBacking.Network = newNet

	return vm.EditDevice(ctx, *netDev)
}

func getNetworkDevicesAndBackingsFromVM(ctx context.Context, vm *object.VirtualMachine, netNameFilter string) ([]netDeviceAndBackingInfo, error) {
	devices, err := vm.Device(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't get devices for vm, see: %s", err)
	}

	var availableData []netDeviceAndBackingInfo

	for i, device := range devices {
		ethDevice, ok := device.(types.BaseVirtualEthernetCard)
		if !ok {
			continue
		}

		ethCard := ethDevice.GetVirtualEthernetCard()
		ethBacking := ethCard.Backing.(*types.VirtualEthernetCardNetworkBackingInfo)

		if netNameFilter == "" || ethBacking.DeviceName == netNameFilter {
			data := netDeviceAndBackingInfo{device: &devices[i], backingInfo: ethBacking}
			availableData = append(availableData, data)
		}
	}

	return availableData, nil
}

func getNetworkFromVM(ctx context.Context, vm *object.VirtualMachine, netName string) (*types.ManagedObjectReference, error) {
	cfg, err := vm.QueryConfigTarget(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't get query config for vm, see: %v", err)
	}

	for _, net := range cfg.Network {
		summary := net.Network.GetNetworkSummary()

		if summary.Accessible && summary.Name == netName {
			return summary.Network, nil
		}
	}

	return nil, fmt.Errorf("no accessible network with the name %s found", netName)
}

func createSnapshot(ctx context.Context, vm *object.VirtualMachine, snapshotName string, snapshotDesc string) (object.Reference, error) {
	task, err := vm.CreateSnapshot(ctx, snapshotName, snapshotDesc, false, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %v", err)
	}

	taskInfo, err := task.WaitForResult(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error waiting for task completion: %v", err)
	}
	glog.Infof("taskInfo.Result is %s", taskInfo.Result)
	return taskInfo.Result.(object.Reference), nil
}

func findSnapshot(ctx context.Context, vm *object.VirtualMachine, name string) (object.Reference, error) {
	var moVirtualMachine mo.VirtualMachine

	if err := vm.Properties(ctx, vm.Reference(), []string{"snapshot"}, &moVirtualMachine); err != nil {
		return nil, fmt.Errorf("failed to get vm properties: %v", err)
	}

	if moVirtualMachine.Snapshot == nil {
		return nil, errSnapshotNotFound
	}

	snapshotCandidates := []object.Reference{}
	for _, snapshotTree := range moVirtualMachine.Snapshot.RootSnapshotList {
		addMatchingSnapshotToList(&snapshotCandidates, snapshotTree, name)
	}

	switch len(snapshotCandidates) {
	case 0:
		return nil, errSnapshotNotFound
	case 1:
		return snapshotCandidates[0], nil
	default:
		glog.Warningf("VM %s seems to have more than one snapshots with name %s. Using a random snapshot.", vm, name)
		return snapshotCandidates[0], nil
	}
}

// VirtualMachineSnapshotTree is a tree (As the name suggests) so we need to use recursion to get all elements
func addMatchingSnapshotToList(list *[]object.Reference, tree types.VirtualMachineSnapshotTree, name string) {
	for _, childTree := range tree.ChildSnapshotList {
		addMatchingSnapshotToList(list, childTree, name)
	}
	if tree.Name == name || tree.Snapshot.Value == name {
		*list = append(*list, &tree.Snapshot)
	}
}

func uploadAndAttachISO(ctx context.Context, f *find.Finder, vmRef *object.VirtualMachine, localIsoFilePath, datastoreName string) error {

	datastore, err := f.Datastore(ctx, datastoreName)
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

func getDatacenterFinder(datacenter string, client *govmomi.Client) (*find.Finder, error) {
	finder := find.NewFinder(client.Client, true)
	dc, err := finder.Datacenter(context.TODO(), datacenter)
	if err != nil {
		return nil, fmt.Errorf("failed to get vsphere datacenter: %v", err)
	}
	finder.SetDatacenter(dc)
	return finder, nil
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
