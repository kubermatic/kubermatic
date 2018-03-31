package vsphere

import (
	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"context"
	"fmt"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/uuid"
	"github.com/vmware/govmomi"
	"net/url"
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

	u, err := url.Parse(dc.Spec.VSphere.Endpoint)
	if err != nil {
		return nil, err
	}

	c, err := govmomi.NewClient(context.Background(), u, true)
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
	_, err := v.getClient(spec)
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

// CreateNodeClass
func (v *vsphere) CreateNodeClass(c *kubermaticv1.Cluster, nSpec *apiv1.NodeSpec, keys []*kubermaticv1.UserSSHKey, version *apiv1.MasterVersion) (*v1alpha1.NodeClass, error) {
	return nil, nil
}

// NodeClassName
func (v *vsphere) NodeClassName(nSpec *apiv1.NodeSpec) string {
	return fmt.Sprintf("kubermatic-%s", uuid.ShortUID(5))
}
