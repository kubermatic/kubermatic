package vsphere

import (
	"context"
	"fmt"
	"net/url"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
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

	return c, nil
}

// createVMFolderForCluster adds a vm folder beneath the rootpath set in the datacenter.yamls with the name of the cluster.
func (v *vsphere) createVMFolderForCluster(cluster *kubermaticv1.Cluster) error {
	cloud := cluster.Spec.Cloud
	dc, found := v.dcs[cloud.DatacenterName]
	if !found || dc.Spec.VSphere == nil {
		return fmt.Errorf("invalid datacenter %q", cloud.DatacenterName)
	}

	if dc.Spec.VSphere.RootPath == "" {
		return fmt.Errorf("missing rootpth for datacenter %s", cloud.DatacenterName)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := v.getClient(cloud)
	if err != nil {
		return err
	}
	defer client.Logout(ctx)

	finder := find.NewFinder(client.Client, true)
	rootFolder, err := finder.Folder(ctx, dc.Spec.VSphere.RootPath)
	if err != nil {
		return fmt.Errorf("couldn't find rootpath, see: %s", err)
	}

	_, err = rootFolder.CreateFolder(ctx, cluster.ObjectMeta.Name)
	if err != nil && soap.IsSoapFault(err) {
		soapFault := soap.ToSoapFault(err)
		if _, ok := soapFault.VimFault().(types.FileAlreadyExists); ok {
			return nil
		}
	} else if err != nil {
		return fmt.Errorf("couldn't create cluster vm folder, see: %s", err)
	}

	return nil
}

// ValidateCloudSpec
func (v *vsphere) ValidateCloudSpec(spec *kubermaticv1.CloudSpec) error {
	client, err := v.getClient(spec)
	if err != nil {
		return err
	}

	if err := client.Logout(context.TODO()); err != nil {
		return fmt.Errorf("failed to logout from vSphere: %v", err)
	}

	return nil
}

// InitializeCloudProvider
func (v *vsphere) InitializeCloudProvider(cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	err := v.createVMFolderForCluster(cluster)
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

// CleanUpCloudProvider
func (v *vsphere) CleanUpCloudProvider(cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	// TODO: Implement cleanUpVMFolderForCluster, atm there is no DeleteFolder function in govmomi...
	return cluster, nil
}
