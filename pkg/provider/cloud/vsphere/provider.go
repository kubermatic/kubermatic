/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"

	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/pkg/provider"
	"github.com/kubermatic/kubermatic/pkg/resources"
	kruntime "k8s.io/apimachinery/pkg/util/runtime"
)

const (
	folderCleanupFinalizer = "kubermatic.io/cleanup-vsphere-folder"
)

// Provider represents the vsphere provider.
type Provider struct {
	dc                *kubermaticv1.DatacenterSpecVSphere
	secretKeySelector provider.SecretKeySelectorValueFunc
}

// Folder represents a vsphere folder.
type Folder struct {
	Path string
}

// NewCloudProvider creates a new vSphere provider.
func NewCloudProvider(dc *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc) (*Provider, error) {
	if dc.Spec.VSphere == nil {
		return nil, errors.New("datacenter is not a vSphere datacenter")
	}
	return &Provider{
		dc:                dc.Spec.VSphere,
		secretKeySelector: secretKeyGetter,
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

func newSession(ctx context.Context, dc *kubermaticv1.DatacenterSpecVSphere, username, password string) (*Session, error) {
	u, err := url.Parse(fmt.Sprintf("%s/sdk", dc.Endpoint))
	if err != nil {
		return nil, err
	}

	client, err := govmomi.NewClient(context.Background(), u, dc.AllowInsecure)
	if err != nil {
		return nil, err
	}

	user := url.UserPassword(username, password)
	if dc.InfraManagementUser != nil {
		user = url.UserPassword(dc.InfraManagementUser.Username, dc.InfraManagementUser.Password)
	}

	if err = client.Login(ctx, user); err != nil {
		return nil, err
	}

	finder := find.NewFinder(client.Client, true)
	datacenter, err := finder.Datacenter(ctx, dc.Datacenter)
	if err != nil {
		return nil, fmt.Errorf("failed to get vSphere datacenter %q: %v", dc.Datacenter, err)
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
func getVMRootPath(dc *kubermaticv1.DatacenterSpecVSphere) string {
	// Each datacenter root directory for VM's is: ${DATACENTER_NAME}/vm
	rootPath := path.Join("/", dc.Datacenter, "vm")
	// We offer a different root path though in case people would like to store all Kubermatic VM's below a certain directory
	if dc.RootPath != "" {
		rootPath = path.Clean(dc.RootPath)
	}
	return rootPath
}

// InitializeCloudProvider initializes the vsphere cloud provider by setting up vm folders for the cluster.
func (v *Provider) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	ctx := context.Background()

	username, password, err := GetCredentialsForCluster(cluster.Spec.Cloud, v.secretKeySelector, v.dc)
	if err != nil {
		return nil, err
	}
	session, err := newSession(ctx, v.dc, username, password)
	if err != nil {
		return nil, fmt.Errorf("failed to create vCenter session: %v", err)
	}
	defer session.Logout()

	rootPath := getVMRootPath(v.dc)
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
func GetNetworks(dc *kubermaticv1.DatacenterSpecVSphere, username, password string) ([]NetworkInfo, error) {
	ctx := context.Background()
	// For the GetNetworks request we use dc.Spec.VSphere.InfraManagementUser
	// if set because that is the user which will ultimatively configure
	// the networks - But it means users in the UI can see vsphere
	// networks without entering credentials
	session, err := newSession(ctx, dc, username, password)
	if err != nil {
		return nil, fmt.Errorf("failed to create vCenter session: %v", err)
	}
	defer session.Logout()

	return getPossibleVMNetworks(ctx, session)
}

// GetVMFolders returns a slice of VSphereFolders of the datacenter from the passed cloudspec.
func GetVMFolders(dc *kubermaticv1.DatacenterSpecVSphere, username, password string) ([]Folder, error) {
	ctx := context.TODO()

	session, err := newSession(ctx, dc, username, password)
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

	rootPath := getVMRootPath(dc)
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
	return nil
}

// ValidateCloudSpec validates whether a vsphere client can be constructued for the passed cloudspec.
func (v *Provider) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	username, password, err := GetCredentialsForCluster(spec, v.secretKeySelector, v.dc)
	if err != nil {
		return err
	}

	if spec.VSphere.DatastoreCluster != "" && spec.VSphere.Datastore != "" {
		return errors.New("either datastore or datastore cluster can be selected")
	}

	session, err := newSession(context.TODO(), v.dc, username, password)
	if err != nil {
		return fmt.Errorf("failed to create vCenter session: %v", err)
	}
	defer session.Logout()
	return nil
}

// CleanUpCloudProvider we always check if the folder is there and remove it if yes because we know its absolute path
// This covers cases where the finalizer was not added
// We also remove the finalizer if either the folder is not present or we successfully deleted it
func (v *Provider) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	ctx := context.TODO()
	username, password, err := GetCredentialsForCluster(cluster.Spec.Cloud, v.secretKeySelector, v.dc)
	if err != nil {
		return nil, err
	}

	session, err := newSession(ctx, v.dc, username, password)
	if err != nil {
		return nil, fmt.Errorf("failed to create vCenter session: %v", err)
	}
	defer session.Logout()

	if kuberneteshelper.HasFinalizer(cluster, folderCleanupFinalizer) {
		if err := deleteVMFolder(ctx, session, cluster.Spec.Cloud.VSphere.Folder); err != nil {
			return nil, err
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
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

// Precedence if not infraManagementUser:
// * User from cluster
// * User from Secret
// Precedence if infraManagementUser:
// * User from clusters infraManagementUser
// * User from cluster
// * User form clusters secret infraManagementUser
// * User from clusters secret
func getUsernameAndPassword(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc, infraManagementUser bool) (username, password string, err error) {
	if infraManagementUser {
		username = cloud.VSphere.InfraManagementUser.Username
		password = cloud.VSphere.InfraManagementUser.Password
	}
	if username == "" {
		username = cloud.VSphere.Username
	}
	if password == "" {
		password = cloud.VSphere.Password
	}

	if username != "" && password != "" {
		return username, password, nil
	}

	if cloud.VSphere.CredentialsReference == nil {
		return "", "", errors.New("cluster contains no password an and empty credentialsReference")
	}

	if username == "" && infraManagementUser {
		username, err = secretKeySelector(cloud.VSphere.CredentialsReference, resources.VsphereInfraManagementUserUsername)
		if err != nil {
			return "", "", err
		}
	}
	if username == "" {
		username, err = secretKeySelector(cloud.VSphere.CredentialsReference, resources.VsphereUsername)
		if err != nil {
			return "", "", err
		}
	}

	if password == "" && infraManagementUser {
		password, err = secretKeySelector(cloud.VSphere.CredentialsReference, resources.VsphereInfraManagementUserPassword)
		if err != nil {
			return "", "", err
		}
	}

	if password == "" {
		password, err = secretKeySelector(cloud.VSphere.CredentialsReference, resources.VspherePassword)
		if err != nil {
			return "", "", err
		}
	}

	if username == "" {
		return "", "", errors.New("unable to get username")
	}

	if password == "" {
		return "", "", errors.New("unable to get password")
	}

	return username, password, nil
}

func GetCredentialsForCluster(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc, dc *kubermaticv1.DatacenterSpecVSphere) (string, string, error) {
	var username, password string
	var err error

	// InfraManagementUser from Datacenter
	if dc != nil && dc.InfraManagementUser != nil {
		if dc.InfraManagementUser.Username != "" && dc.InfraManagementUser.Password != "" {
			return dc.InfraManagementUser.Username, dc.InfraManagementUser.Password, nil
		}
	}

	// InfraManagementUser from Cluster
	username, password, err = getUsernameAndPassword(cloud, secretKeySelector, true)
	if err != nil {
		return "", "", err
	}

	return username, password, nil
}
