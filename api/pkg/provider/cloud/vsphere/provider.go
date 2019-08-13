package vsphere

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/golang/glog"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/types"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	kruntime "k8s.io/apimachinery/pkg/util/runtime"
)

const (
	folderCleanupFinalizer = "kubermatic.io/cleanup-vsphere-folder"
)

// Provider represents the vsphere provider.
type Provider struct {
	dc             *kubermaticv1.DatacenterSpecVSphere
	clusterUpdater provider.ClusterUpdater
}

// Network represents a vsphere network backing.
type Network struct {
	Name string
}

// Folder represents a vsphere folder.
type Folder struct {
	Path string
}

// NewCloudProvider creates a new vSphere provider.
func NewCloudProvider(dc *kubermaticv1.Datacenter) (*Provider, error) {
	if dc.Spec.VSphere == nil {
		return nil, errors.New("datacenter is not a vSphere datacenter")
	}
	return &Provider{
		dc: dc.Spec.VSphere,
	}, nil
}

func (v *Provider) getClient(cloud kubermaticv1.CloudSpec) (*govmomi.Client, error) {
	u, err := url.Parse(fmt.Sprintf("%s/sdk", v.dc.Endpoint))
	if err != nil {
		return nil, err
	}

	c, err := govmomi.NewClient(context.Background(), u, v.dc.AllowInsecure)
	if err != nil {
		return nil, err
	}

	user := url.UserPassword(
		cloud.VSphere.InfraManagementUser.Username,
		cloud.VSphere.InfraManagementUser.Password)
	err = c.Login(context.Background(), user)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (v *Provider) getVsphereRootPath(cloud kubermaticv1.CloudSpec) (string, error) {
	if v.dc.RootPath == "" {
		return "", fmt.Errorf("missing property 'root_path' for datacenter %s", cloud.DatacenterName)
	}

	return v.dc.RootPath, nil
}

// createVMFolderForCluster adds a vm folder beneath the rootpath set in the datacenter.yamls with the name of the cluster.
func (v *Provider) createVMFolderForCluster(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	// Don't add the finalizer if the folder is passed in
	if cluster.Spec.Cloud.VSphere.Folder != "" {
		return cluster, nil
	}

	dcRootPath, err := v.getVsphereRootPath(cluster.Spec.Cloud)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := v.getClient(cluster.Spec.Cloud)
	if err != nil {
		return nil, err
	}
	defer logout(client)

	finder := find.NewFinder(client.Client, true)
	rootFolder, err := finder.Folder(ctx, dcRootPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't find rootpath, see: %v", err)
	}
	_, err = finder.Folder(ctx, cluster.Name)
	if err != nil {
		if _, ok := err.(*find.NotFoundError); !ok {
			return nil, fmt.Errorf("Failed to get cluster folder: %v", err)
		}
		if _, err = rootFolder.CreateFolder(ctx, cluster.Name); err != nil {
			return nil, fmt.Errorf("failed to create cluster folder %s/%s: %v", rootFolder, cluster.Name, err)
		}
	}

	cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
		if !kuberneteshelper.HasFinalizer(cluster, folderCleanupFinalizer) {
			cluster.Finalizers = append(cluster.Finalizers, folderCleanupFinalizer)
		}
		cluster.Spec.Cloud.VSphere.Folder = fmt.Sprintf("%s/%s", dcRootPath, cluster.Name)
	})
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

// GetNetworks returns a slice of VSphereNetworks of the datacenter from the passed cloudspec.
func (v *Provider) GetNetworks(cloud kubermaticv1.CloudSpec) ([]Network, error) {

	// For the GetNetworks request we use dc.Spec.VSphere.InfraManagementUser
	// if set because that is the user which will ultimatively configure
	// the networks - But it means users in the UI can see vsphere
	// networks without entering credentials
	if v.dc.InfraManagementUser != nil {
		cloud.VSphere.InfraManagementUser.Username = v.dc.InfraManagementUser.Username
		cloud.VSphere.InfraManagementUser.Password = v.dc.InfraManagementUser.Password
	}

	client, err := v.getClient(cloud)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialize vsphere client: %v", err)
	}
	defer logout(client)

	finder := find.NewFinder(client.Client, true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	vsphereDC, err := finder.Datacenter(ctx, v.dc.Datacenter)
	if err != nil {
		return nil, fmt.Errorf("failed to get vsphere datacenter: %v", err)
	}
	finder.SetDatacenter(vsphereDC)

	// finder is relative to datacenter, so * is fine for us.
	netRefs, err := finder.NetworkList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("couldn't retrieve network list: %v", err)
	}

	var networks []Network
	for _, netRef := range netRefs {
		backing, err := netRef.EthernetCardBackingInfo(ctx)
		if err != nil {
			// So, there are some netRefs (for example VmwareDistributedVirtualSwitch) which don't implement a BackingInfo.
			// And since the error isn't typed, we can't check for it.
			// And since the vsphere documentation is glorious, we dont know all netRef-types affected by this.
			// so instead of checking type for ignoring that special error, we have to ignore all of it.
			// return nil, fmt.Errorf("couldn't get network backing: %v (%s)", err, netRef.Reference().Type)
			continue
		}
		netBacking, ok := backing.(*types.VirtualEthernetCardNetworkBackingInfo)
		if !ok {
			// ignore virtual networks
			continue
		}

		network := Network{Name: netBacking.DeviceName}
		networks = append(networks, network)
	}

	return networks, nil
}

// GetFolders returns a slice of VSphereFolders of the datacenter from the passed cloudspec.
func (v *Provider) GetFolders(cloud kubermaticv1.CloudSpec) ([]Folder, error) {
	if v.dc.InfraManagementUser != nil {
		cloud.VSphere.InfraManagementUser.Username = v.dc.InfraManagementUser.Username
		cloud.VSphere.InfraManagementUser.Password = v.dc.InfraManagementUser.Password
	}

	client, err := v.getClient(cloud)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialize vsphere client: %v", err)
	}
	defer logout(client)

	finder := find.NewFinder(client.Client, true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	vsphereDC, err := finder.Datacenter(ctx, v.dc.Datacenter)
	if err != nil {
		return nil, fmt.Errorf("failed to get vsphere datacenter: %v", err)
	}
	finder.SetDatacenter(vsphereDC)

	// finder is relative to datacenter, so * is fine for us.
	folderRefs, err := finder.FolderList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("couldn't retrieve folder list: %v", err)
	}

	var folders []Folder
	for _, netRef := range folderRefs {
		folder := Folder{Path: netRef.Common.InventoryPath}
		folders = append(folders, folder)
	}

	return folders, nil
}

// DefaultCloudSpec adds defaults to the cloud spec
func (v *Provider) DefaultCloudSpec(cloud *kubermaticv1.CloudSpec) error {
	if v.dc.InfraManagementUser != nil {
		cloud.VSphere.InfraManagementUser = kubermaticv1.VSphereCredentials{
			Username: v.dc.InfraManagementUser.Username,
			Password: v.dc.InfraManagementUser.Password,
		}
	} else {
		cloud.VSphere.InfraManagementUser = kubermaticv1.VSphereCredentials{
			Username: cloud.VSphere.Username,
			Password: cloud.VSphere.Password,
		}
	}

	return nil
}

// ValidateCloudSpec validates whether a vsphere client can be constructued for the passed cloudspec.
func (v *Provider) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	client, err := v.getClient(spec)
	if err != nil {
		return err
	}
	logout(client)
	return nil
}

// InitializeCloudProvider initializes the vsphere cloud provider by setting up vm folders for the cluster.
func (v *Provider) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, secretKeySelector provider.SecretKeySelectorValueFunc) (*kubermaticv1.Cluster, error) {
	v.clusterUpdater = update
	return v.createVMFolderForCluster(cluster, update)
}

// CleanUpCloudProvider we always check if the folder is there and remove it if yes because we know its absolute path
// This covers cases where the finalizer was not added
// We also remove the finalizer if either the folder is not present or we successfully deleted it
func (v *Provider) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, _ provider.SecretKeySelectorValueFunc) (*kubermaticv1.Cluster, error) {
	v.clusterUpdater = update

	vsphereRootPath, err := v.getVsphereRootPath(cluster.Spec.Cloud)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := v.getClient(cluster.Spec.Cloud)
	if err != nil {
		return nil, err
	}
	defer logout(client)

	finder := find.NewFinder(client.Client, true)
	folder, err := finder.Folder(ctx, fmt.Sprintf("%s/%s", vsphereRootPath, cluster.Name))
	if err != nil {
		if _, ok := err.(*find.NotFoundError); !ok {
			return nil, fmt.Errorf("failed to get folder: %v", err)
		}
		// Folder is not there anymore, maybe someone deleted it manually
		cluster, err = v.clusterUpdater(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, folderCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}

		return cluster, nil
	}
	task, err := folder.Destroy(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to delete folder: %v", err)
	}
	if err := task.Wait(ctx); err != nil {
		return nil, fmt.Errorf("failed to wait for deletion of folder: %v", err)
	}

	cluster, err = v.clusterUpdater(cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kuberneteshelper.RemoveFinalizer(cluster, folderCleanupFinalizer)
	})
	if err != nil {
		return nil, err
	}

	glog.V(2).Infof("Successfully deleted vsphere folder %s/%s for cluster %s", vsphereRootPath, cluster.Name, cluster.Name)
	return cluster, nil
}

func logout(client *govmomi.Client) {
	if err := client.Logout(context.Background()); err != nil {
		kruntime.HandleError(fmt.Errorf("Failed to logout from vsphere: %v", err))
	}
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted
func (v *Provider) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}
