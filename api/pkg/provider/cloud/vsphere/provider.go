package vsphere

import (
	"context"
	"fmt"
	"net/url"

	"github.com/golang/glog"
	"github.com/vmware/govmomi"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
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

// InitializeCloudProvider
func (v *vsphere) InitializeCloudProvider(spec *kubermaticv1.CloudSpec, name string) (*kubermaticv1.CloudSpec, error) {
	client, err := v.getClient(spec)
	defer func() {
		if err := client.Logout(context.TODO()); err != nil {
			glog.V(0).Infof("failed to logout from vsphere for %s: %v", name, err)
		}
	}()

	return nil, err
}

// ValidateCloudSpec
func (v *vsphere) ValidateCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

// CleanUpCloudProvider
func (v *vsphere) CleanUpCloudProvider(spec *kubermaticv1.CloudSpec) error {
	return nil
}

// ValidateNodeSpec
func (v *vsphere) ValidateNodeSpec(spec *kubermaticv1.CloudSpec, nSpec *apiv1.NodeSpec) error {
	return nil
}
