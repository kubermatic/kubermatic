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

func createLinkClonedVM(vmName, vmImage, datacenter, clusterName, folder string, cpus int32, memoryMB int64, client *govmomi.Client, containerLinuxUserdata string) error {
	f := find.NewFinder(client.Client, true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dc, err := f.Datacenter(ctx, datacenter)
	if err != nil {
		return err
	}
	f.SetDatacenter(dc)

	templateVM, err := f.VirtualMachine(ctx, vmImage)
	if err != nil {
		return err
	}

	glog.V(3).Infof("Template VM ref is %+v", templateVM)
	datacenterFolders, err := dc.Folders(ctx)
	if err != nil {
		return fmt.Errorf("failed to get datacenter folders: %v", err)
	}

	// Find the target folder, if its include in the provider config.
	var targetVMFolderRefPtr *types.ManagedObjectReference
	if folder != "" {
		// If non-absolute folder name is used, e.g. 'duplicate-folder' it can match
		// multiple folders and thus fail. It will also gladly match a folder from
		// a different datacenter. It is therefore preferable to use absolute folder
		// paths, e.g. '/Datacenter/vm/nested/folder'.
		// The target folder must already exist.
		targetVMFolder, folderErr := f.Folder(ctx, folder)
		if folderErr != nil {
			return fmt.Errorf("failed to get target folder: %v", folderErr)
		}
		targetVMFolderRef := targetVMFolder.Reference()
		targetVMFolderRefPtr = &targetVMFolderRef
	}

	// Create snapshot of the template VM if not already snapshotted.
	snapshot, err := findSnapshot(ctx, templateVM, snapshotName)
	if err != nil {
		if err != errSnapshotNotFound {
			return fmt.Errorf("failed to find snapshot: %v", err)
		}
		snapshot, err = createSnapshot(ctx, templateVM, snapshotName, snapshotDesc)
		if err != nil {
			return fmt.Errorf("failed to create snapshot: %v", err)
		}
	}

	clsComputeRes, err := f.ClusterComputeResource(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster %s: %v", clusterName, err)
	}
	glog.V(3).Infof("Cluster is %+v", clsComputeRes)

	resPool, err := clsComputeRes.ResourcePool(ctx)
	if err != nil {
		return fmt.Errorf("failed to get ressource pool: %v", err)
	}
	glog.V(3).Infof("Cluster resource pool is %+v", resPool)

	if resPool == nil {
		return fmt.Errorf("no resource pool found for cluster %s", clusterName)
	}

	resPoolRef := resPool.Reference()
	snapshotRef := snapshot.Reference()

	var vAppAconfig *types.VmConfigSpec
	if containerLinuxUserdata != "" {
		userdataBase64 := base64.StdEncoding.EncodeToString([]byte(containerLinuxUserdata))

		// The properties describing userdata will already exist in the CoreOS VM template.
		// In order to overwrite them, we need to specify their numeric Key values,
		// which we'll extract from that template.
		var mvm mo.VirtualMachine
		err = templateVM.Properties(ctx, templateVM.Reference(), []string{"config", "config.vAppConfig", "config.vAppConfig.property"}, &mvm)
		if err != nil {
			return err
		}

		var propertySpecs []types.VAppPropertySpec
		if mvm.Config.VAppConfig.GetVmConfigInfo() == nil {
			return fmt.Errorf("no vm config found in template '%s'. Make sure you import the correct OVA with the appropriate coreos settings", vmImage)
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
	cloneSpec := &types.VirtualMachineCloneSpec{
		Config: &types.VirtualMachineConfigSpec{
			Flags: &types.VirtualMachineFlagInfo{
				DiskUuidEnabled: &diskUUIDEnabled,
			},
			NumCPUs:    cpus,
			MemoryMB:   memoryMB,
			VAppConfig: vAppAconfig,
		},
		Location: types.VirtualMachineRelocateSpec{
			Pool:         &resPoolRef,
			Folder:       targetVMFolderRefPtr,
			DiskMoveType: string(types.VirtualMachineRelocateDiskMoveOptionsCreateNewChildDiskBacking),
		},
		Snapshot: &snapshotRef,
	}

	// Create a link cloned VM from the template VM's snapshot
	clonedVMTask, err := templateVM.Clone(ctx, datacenterFolders.VmFolder, vmName, *cloneSpec)
	if err != nil {
		return err
	}

	_, err = clonedVMTask.WaitForResult(ctx, nil)
	return err
}

func updateNetworkForVM(ctx context.Context, vm *object.VirtualMachine, currentNetName string, newNetName string) error {
	newNet, err := getNetworkFromVM(ctx, vm, newNetName)
	if err != nil {
		return err
	}

	availableData, err := getNetworkDevicesAndBackingsFromVM(ctx, vm, currentNetName)
	if err != nil {
		return err
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

	var availableBackings []string
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

		availableBackings = append(availableBackings, ethBacking.DeviceName)
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
		return nil, err
	}

	taskInfo, err := task.WaitForResult(ctx, nil)
	if err != nil {
		return nil, err
	}
	glog.Infof("taskInfo.Result is %s", taskInfo.Result)
	return taskInfo.Result.(object.Reference), nil
}

func findSnapshot(ctx context.Context, vm *object.VirtualMachine, name string) (object.Reference, error) {
	var moVirtualMachine mo.VirtualMachine

	err := vm.Properties(ctx, vm.Reference(), []string{"snapshot"}, &moVirtualMachine)
	if err != nil {
		return nil, err
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

func uploadAndAttachISO(f *find.Finder, vmRef *object.VirtualMachine, localIsoFilePath, datastoreName string, client *govmomi.Client) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	datastore, err := f.Datastore(ctx, datastoreName)
	if err != nil {
		return err
	}
	p := soap.DefaultUpload
	remoteIsoFilePath := fmt.Sprintf("%s/%s", vmRef.Name(), "cloud-init.iso")
	glog.V(3).Infof("Uploading userdata ISO to datastore %+v, destination iso is %s\n", datastore, remoteIsoFilePath)
	err = datastore.UploadFile(ctx, localIsoFilePath, remoteIsoFilePath, &p)
	if err != nil {
		return err
	}
	glog.V(3).Infof("Uploaded ISO file %s", localIsoFilePath)

	// Find the cd-rom devide and insert the cloud init iso file into it.
	devices, err := vmRef.Device(ctx)
	if err != nil {
		return err
	}

	// passing empty cd-rom name so that the first one gets returned
	cdrom, err := devices.FindCdrom("")
	cdrom.Connectable.StartConnected = true
	if err != nil {
		return err
	}
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
		err := os.RemoveAll(userdataDir)
		if err != nil {
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
	err = metadataTmpl.Execute(metadata, templateContext)
	if err != nil {
		return "", fmt.Errorf("failed to render metadata: %v", err)
	}

	err = ioutil.WriteFile(userdataFilePath, []byte(userdata), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to locally write userdata file to %s: %v", userdataFilePath, err)
	}

	err = ioutil.WriteFile(metadataFilePath, metadata.Bytes(), 0644)
	if err != nil {
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
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error executing command `%s %s`: output: `%s`, error: `%v`", command, args, string(output), err)
	}

	return isoFilePath, nil
}

func removeFloppyDevice(virtualMachine *object.VirtualMachine) error {
	vmDevices, err := virtualMachine.Device(context.TODO())
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

	err = virtualMachine.RemoveDevice(context.TODO(), false, floppyDevice)
	if err != nil {
		return fmt.Errorf("failed to remove floppy device: %v", err)
	}

	return nil
}
