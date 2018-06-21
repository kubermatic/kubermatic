package vsphere

import (
	"context"
	"fmt"
	"net/url"

	"github.com/golang/glog"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"path"

	kruntime "k8s.io/apimachinery/pkg/util/runtime"
)

const (
	folderCleanupFinalizer = "kubermatic.io/cleanup-vsphere-folder"
)

type vsphere struct{}

// NewCloudProvider creates a new vSphere provider.
func NewCloudProvider() provider.CloudProvider {
	return &vsphere{}
}

func (v *vsphere) getClient(cloud *kubermaticv1.CloudSpec) (*govmomi.Client, error) {
	u, err := url.Parse(fmt.Sprintf("%s/sdk", cloud.VSphere.Endpoint))
	if err != nil {
		return nil, err
	}

	c, err := govmomi.NewClient(context.Background(), u, cloud.VSphere.AllowInsecure)
	if err != nil {
		return nil, err
	}

	user := url.UserPassword(cloud.VSphere.Username, cloud.VSphere.Password)
	err = c.Login(context.Background(), user)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// createVMFolderForCluster adds a vm folder beneath the rootpath set in the datacenter.yamls with the name of the cluster.
func (v *vsphere) createVMFolderForCluster(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := v.getClient(cluster.Spec.Cloud)
	if err != nil {
		return nil, err
	}
	defer logout(client)

	finder := find.NewFinder(client.Client, true)
	rootPath := path.Dir(cluster.Spec.Cloud.VSphere.VMFolder)
	rootFolder, err := finder.Folder(ctx, rootPath)
	clusterFolder := path.Join(rootPath, cluster.Name)
	if err != nil {
		return nil, fmt.Errorf("couldn't find rootpath '%s', see: %v", rootPath, err)
	}
	_, err = finder.Folder(ctx, cluster.Name)
	if err != nil {
		if _, ok := err.(*find.NotFoundError); !ok {
			return nil, fmt.Errorf("failed to get cluster folder '%s': %v", clusterFolder, err)
		}
		if _, err = rootFolder.CreateFolder(ctx, cluster.Name); err != nil {
			return nil, fmt.Errorf("failed to create cluster folder '%s': %v", clusterFolder, err)
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

// ValidateCloudSpec
func (v *vsphere) ValidateCloudSpec(spec *kubermaticv1.CloudSpec) error {
	client, err := v.getClient(spec)
	if err != nil {
		return err
	}
	logout(client)
	return nil
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := v.getClient(cluster.Spec.Cloud)
	if err != nil {
		return nil, err
	}
	defer logout(client)

	finder := find.NewFinder(client.Client, true)
	folder, err := finder.Folder(ctx, cluster.Spec.Cloud.VSphere.VMFolder)
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

	glog.V(4).Infof("Successfully deleted folder '%s' for cluster %s on vSphere", cluster.Spec.Cloud.VSphere.VMFolder, cluster.Name)
	return cluster, nil
}

func logout(client *govmomi.Client) {
	if err := client.Logout(context.Background()); err != nil {
		kruntime.HandleError(fmt.Errorf("failed to logout from vsphere: %v", err))
	}
}
