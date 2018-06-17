package vsphere

import (
	"context"
	"fmt"
	"net/url"
	"runtime"

	"github.com/golang/glog"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	kruntime "k8s.io/apimachinery/pkg/util/runtime"
)

const (
	folderCleanupFinalizer = "kubermatic.io/cleanup-vsphere-folder"
)

type vsphere struct {
	dcs map[string]provider.DatacenterMeta
}

// NewCloudProvider creates a new vSphere provider.
func NewCloudProvider(dcs map[string]provider.DatacenterMeta) provider.CloudProvider {
	return &vsphere{
		dcs: dcs,
	}
}

func (v *vsphere) getClient(cloud *kubermaticv1.CloudSpec) (*govmomi.Client, error) {
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

	user := url.UserPassword(cloud.VSphere.Username, cloud.VSphere.Password)
	err = c.Login(context.Background(), user)
	if err != nil {
		return nil, err
	}

	runtime.SetFinalizer(c, logout)
	return c, nil
}

func (v *vsphere) getVsphereRootPath(cluster *kubermaticv1.Cluster) (string, error) {
	cloud := cluster.Spec.Cloud
	dc, found := v.dcs[cloud.DatacenterName]
	if !found || dc.Spec.VSphere == nil {
		return "", fmt.Errorf("invalid datacenter %q", cloud.DatacenterName)
	}

	if dc.Spec.VSphere.RootPath == "" {
		return "", fmt.Errorf("missing rootpath for datacenter %s", cloud.DatacenterName)
	}

	return dc.Spec.VSphere.RootPath, nil
}

// createVMFolderForCluster adds a vm folder beneath the rootpath set in the datacenter.yamls with the name of the cluster.
func (v *vsphere) createVMFolderForCluster(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	dcRootPath, err := v.getVsphereRootPath(cluster)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := v.getClient(cluster.Spec.Cloud)
	if err != nil {
		return nil, err
	}

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

	if !kubernetes.HasFinalizer(cluster, folderCleanupFinalizer) {
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = append(cluster.Finalizers, folderCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

// ValidateCloudSpec
func (v *vsphere) ValidateCloudSpec(spec *kubermaticv1.CloudSpec) error {
	_, err := v.getClient(spec)
	return err
}

// InitializeCloudProvider
func (v *vsphere) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return v.createVMFolderForCluster(cluster, update)
}

// CleanUpCloudProvider
// We always check if the folder is there and remove it if yes because we know its absolute path
// This covers cases where the finalizer was not added
// We also remove the finalizer if either the folder is not present or we successfully deleted it
func (v *vsphere) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	if !kubernetes.HasFinalizer(cluster, folderCleanupFinalizer) {
		return cluster, nil
	}

	vsphereRootPath, err := v.getVsphereRootPath(cluster)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := v.getClient(cluster.Spec.Cloud)
	if err != nil {
		return nil, err
	}

	finder := find.NewFinder(client.Client, true)
	folder, err := finder.Folder(ctx, fmt.Sprintf("%s/%s", vsphereRootPath, cluster.Name))
	if err != nil {
		if _, ok := err.(*find.NotFoundError); !ok {
			return nil, fmt.Errorf("failed to get folder: %v", err)
		}
		// Folder is not there anymore, maybe someone deleted it manually
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = kubernetes.RemoveFinalizer(cluster.Finalizers, folderCleanupFinalizer)
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
		cluster.Finalizers = kubernetes.RemoveFinalizer(cluster.Finalizers, folderCleanupFinalizer)
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
