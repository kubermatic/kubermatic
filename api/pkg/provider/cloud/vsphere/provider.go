package vsphere

import (
	"context"
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
	dcs map[string]provider.DatacenterMeta
}

// Network represents a vsphere network backing.
type Network struct {
	Name string
}

// NewCloudProvider creates a new vSphere provider.
func NewCloudProvider(dcs map[string]provider.DatacenterMeta) provider.CloudProvider {
	return &Provider{
		dcs: dcs,
	}
}

func (v *Provider) getClient(cloud *kubermaticv1.CloudSpec) (*govmomi.Client, error) {
	dc, found := v.dcs[cloud.DatacenterName]
	if !found || dc.Spec.VSphere == nil {
		return nil, fmt.Errorf("invalid datacenter %q", cloud.DatacenterName)
	}

	u, err := url.Parse(fmt.Sprintf("%s/sdk", dc.Spec.VSphere.Endpoint))
	if err != nil {
		return nil, err
	}

	c, err := govmomi.NewClient(context.Background(), u, dc.Spec.VSphere.AllowInsecure)
	if err != nil {
		return nil, err
	}

	var username, password string
	if dc.Spec.VSphere.InstanceManagementUser != nil {
		username = dc.Spec.VSphere.InstanceManagementUser.Username
		password = dc.Spec.VSphere.InstanceManagementUser.Password
	} else {
		username = cloud.VSphere.Username
		password = cloud.VSphere.Password
	}
	user := url.UserPassword(username, password)
	err = c.Login(context.Background(), user)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (v *Provider) getVsphereRootPath(spec *kubermaticv1.CloudSpec) (string, error) {
	dc, found := v.dcs[spec.DatacenterName]
	if !found || dc.Spec.VSphere == nil {
		return "", fmt.Errorf("invalid datacenter %q", spec.DatacenterName)
	}

	if dc.Spec.VSphere.RootPath == "" {
		return "", fmt.Errorf("missing property 'root_path' for datacenter %s", spec.DatacenterName)
	}

	return dc.Spec.VSphere.RootPath, nil
}

// createVMFolderForCluster adds a vm folder beneath the rootpath set in the datacenter.yamls with the name of the cluster.
func (v *Provider) createVMFolderForCluster(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
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
	_, err = finder.Folder(ctx, cluster.ObjectMeta.Name)
	if err != nil {
		if _, ok := err.(*find.NotFoundError); !ok {
			return nil, fmt.Errorf("Failed to get cluster folder: %v", err)
		}
		if _, err = rootFolder.CreateFolder(ctx, cluster.Name); err != nil {
			return nil, fmt.Errorf("failed to create cluster folder %s/%s: %v", rootFolder, cluster.Name, err)
		}
	}

	if !kuberneteshelper.HasFinalizer(cluster, folderCleanupFinalizer) {
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = append(cluster.Finalizers, folderCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

// GetNetworks returns a slice of VSphereNetworks of the datacenter from the passed cloudspec.
func (v *Provider) GetNetworks(spec *kubermaticv1.CloudSpec) ([]Network, error) {
	client, err := v.getClient(spec)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialize vsphere client: %v", err)
	}
	defer logout(client)

	finder := find.NewFinder(client.Client, true)

	dc, found := v.dcs[spec.DatacenterName]
	if !found || dc.Spec.VSphere == nil {
		return nil, fmt.Errorf("invalid datacenter %q", spec.DatacenterName)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	vsphereDC, err := finder.Datacenter(ctx, dc.Spec.VSphere.Datacenter)
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

// ValidateCloudSpec validates whether a vsphere client can be constructued for the passed cloudspec.
func (v *Provider) ValidateCloudSpec(spec *kubermaticv1.CloudSpec) error {
	client, err := v.getClient(spec)
	if err != nil {
		return err
	}
	logout(client)
	return nil
}

// InitializeCloudProvider initializes the vsphere cloud provider by setting up vm folders for the cluster.
func (v *Provider) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return v.createVMFolderForCluster(cluster, update)
}

// CleanUpCloudProvider we always check if the folder is there and remove it if yes because we know its absolute path
// This covers cases where the finalizer was not added
// We also remove the finalizer if either the folder is not present or we successfully deleted it
func (v *Provider) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
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
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = kuberneteshelper.RemoveFinalizer(cluster.Finalizers, folderCleanupFinalizer)
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

	cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
		cluster.Finalizers = kuberneteshelper.RemoveFinalizer(cluster.Finalizers, folderCleanupFinalizer)
	})
	if err != nil {
		return nil, err
	}

	glog.V(4).Infof("Successfully deleted vsphere folder %s/%s for cluster %s", vsphereRootPath, cluster.Name, cluster.Name)
	return cluster, nil
}

func logout(client *govmomi.Client) {
	if err := client.Logout(context.Background()); err != nil {
		kruntime.HandleError(fmt.Errorf("Failed to logout from vsphere: %v", err))
	}
}
