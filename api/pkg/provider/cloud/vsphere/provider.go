package vsphere

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"

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

type Session struct {
	Client     *govmomi.Client
	Finder     *find.Finder
	Datacenter *object.Datacenter
}

// Logout closes the idling vCenter connections
func (s *Session) Logout() {
	if err := s.Client.Logout(context.Background()); err != nil {
		kruntime.HandleError(fmt.Errorf("vSphere client failed to logout: %s", err))
	}
}

func (v *Provider) newSession(ctx context.Context, cloud kubermaticv1.CloudSpec) (*Session, error) {
	u, err := url.Parse(fmt.Sprintf("%s/sdk", v.dc.Endpoint))
	if err != nil {
		return nil, err
	}

	client, err := govmomi.NewClient(context.Background(), u, v.dc.AllowInsecure)
	if err != nil {
		return nil, err
	}

	user := url.UserPassword(cloud.VSphere.InfraManagementUser.Username, cloud.VSphere.InfraManagementUser.Password)
	if v.dc.InfraManagementUser != nil {
		user = url.UserPassword(v.dc.InfraManagementUser.Username, v.dc.InfraManagementUser.Password)
	}

	if err = client.Login(ctx, user); err != nil {
		return nil, err
	}

	finder := find.NewFinder(client.Client, true)
	datacenter, err := finder.Datacenter(ctx, v.dc.Datacenter)
	if err != nil {
		return nil, fmt.Errorf("failed to get vSphere datacenter %q: %v", v.dc.Datacenter, err)
	}
	finder.SetDatacenter(datacenter)

	return &Session{
		Datacenter: datacenter,
		Finder:     finder,
		Client:     client,
	}, nil
}

// getVMRootPath is a helper func to get the root path for VM's
// We extracted it because we use it in several places
func (v *Provider) getVMRootPath(cloud kubermaticv1.CloudSpec) string {
	// Each datacenter root directory for VM's is: ${DATACENTER_NAME}/vm
	rootPath := path.Join("/", v.dc.Datacenter, "vm")
	// We offer a different root path though in case people would like to store all Kubermatic VM's below a certain directory
	if v.dc.RootPath != "" {
		rootPath = path.Clean(v.dc.RootPath)
	}
	return rootPath
}

// InitializeCloudProvider initializes the vsphere cloud provider by setting up vm folders for the cluster.
func (v *Provider) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, secretKeySelector provider.SecretKeySelectorValueFunc) (*kubermaticv1.Cluster, error) {
	v.clusterUpdater = update
	ctx := context.Background()

	session, err := v.newSession(ctx, cluster.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to create vCenter session: %v", err)
	}
	defer session.Logout()

	rootPath := v.getVMRootPath(cluster.Spec.Cloud)
	if cluster.Spec.Cloud.VSphere.Folder == "" {
		// If the user did not specify a folder, we create a own folder for this cluster to improve
		// the VM management in vCenter
		clusterFolder := path.Join(rootPath, cluster.Name)
		if err := createVMFolder(ctx, session, clusterFolder); err != nil {
			return nil, fmt.Errorf("failed to create the VM folder %q: %v", clusterFolder, err)
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.AddFinalizer(cluster, folderCleanupFinalizer)
			cluster.Spec.Cloud.VSphere.Folder = clusterFolder
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

// GetNetworks returns a slice of VSphereNetworks of the datacenter from the passed cloudspec.
func (v *Provider) GetNetworks(cloud kubermaticv1.CloudSpec) ([]NetworkInfo, error) {
	ctx := context.Background()
	// For the GetNetworks request we use dc.Spec.VSphere.InfraManagementUser
	// if set because that is the user which will ultimatively configure
	// the networks - But it means users in the UI can see vsphere
	// networks without entering credentials
	session, err := v.newSession(ctx, cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to create vCenter session: %v", err)
	}
	defer session.Logout()

	return getPossibleVMNetworks(ctx, session)
}

// GetVMFolders returns a slice of VSphereFolders of the datacenter from the passed cloudspec.
func (v *Provider) GetVMFolders(cloud kubermaticv1.CloudSpec) ([]Folder, error) {
	ctx := context.TODO()

	session, err := v.newSession(ctx, cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to create vCenter session: %v", err)
	}
	defer session.Logout()

	// We simply list all folders & filter out afterwards.
	// Filtering here is not possible as vCenter only lists the first level when giving a path.
	// vCenter only lists folders recursively if you just specify "*".
	folderRefs, err := session.Finder.FolderList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("couldn't retrieve folder list: %v", err)
	}

	rootPath := v.getVMRootPath(cloud)
	var folders []Folder
	for _, folderRef := range folderRefs {
		// We filter by rootPath. If someone configures it, we should respect it.
		if !strings.HasPrefix(folderRef.InventoryPath, rootPath+"/") && folderRef.InventoryPath != rootPath {
			continue
		}

		folder := Folder{Path: folderRef.Common.InventoryPath}
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
	session, err := v.newSession(context.TODO(), spec)
	if err != nil {
		return fmt.Errorf("failed to create vCenter session: %v", err)
	}
	defer session.Logout()
	return nil
}

// CleanUpCloudProvider we always check if the folder is there and remove it if yes because we know its absolute path
// This covers cases where the finalizer was not added
// We also remove the finalizer if either the folder is not present or we successfully deleted it
func (v *Provider) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, _ provider.SecretKeySelectorValueFunc) (*kubermaticv1.Cluster, error) {
	v.clusterUpdater = update
	ctx := context.TODO()

	session, err := v.newSession(ctx, cluster.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to create vCenter session: %v", err)
	}
	defer session.Logout()

	if kuberneteshelper.HasFinalizer(cluster, folderCleanupFinalizer) {
		if err := deleteVMFolder(ctx, session, cluster.Spec.Cloud.VSphere.Folder); err != nil {
			return nil, err
		}
		cluster, err = v.clusterUpdater(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, folderCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted
func (v *Provider) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}
