package vsphere

import (
	"context"
	"fmt"
	"net/url"

	"github.com/vmware/govmomi"

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
	return cluster, nil
}

// CleanUpCloudProvider
func (v *vsphere) CleanUpCloudProvider(cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}
