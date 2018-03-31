package hetzner

import (
	"context"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

type hetzner struct {
	dcs map[string]provider.DatacenterMeta
}

// NewCloudProvider creates a new hetzner provider.
func NewCloudProvider(dcs map[string]provider.DatacenterMeta) provider.CloudProvider {
	return &hetzner{
		dcs: dcs,
	}
}

// ValidateCloudSpec
func (h *hetzner) ValidateCloudSpec(cloud *kubermaticv1.CloudSpec) error {
	client := hcloud.NewClient(hcloud.WithToken(cloud.Hetzner.Token))
	_, _, err := client.ServerType.List(context.Background(), hcloud.ServerTypeListOpts{})
	return err
}

// InitializeCloudProvider
func (h *hetzner) InitializeCloudProvider(cloud *kubermaticv1.CloudSpec, name string) (*kubermaticv1.CloudSpec, error) {
	return nil, nil
}

// CleanUpCloudProvider
func (h *hetzner) CleanUpCloudProvider(*kubermaticv1.CloudSpec) error {
	return nil
}

// CreateNodeClass
func (h *hetzner) CreateNodeClass(c *kubermaticv1.Cluster, nSpec *apiv1.NodeSpec, keys []*kubermaticv1.UserSSHKey, version *apiv1.MasterVersion) (*v1alpha1.NodeClass, error) {
	return nil, nil
}

// NodeClassName
func (h *hetzner) NodeClassName(nSpec *apiv1.NodeSpec) string {
	return ""
}

// ValidateNodeSpec
func (h *hetzner) ValidateNodeSpec(cloudSpec *kubermaticv1.CloudSpec, nodeSpec *apiv1.NodeSpec) error {
	return nil
}
