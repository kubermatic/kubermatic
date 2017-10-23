package baremetal

import (
	"net/http"

	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	"github.com/kubermatic/kubermatic/api"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

type baremetal struct {
	datacenters map[string]provider.DatacenterMeta
	client      *http.Client
}

// NewCloudProvider returns a new bare-metal provider.
func NewCloudProvider(datacenters map[string]provider.DatacenterMeta) provider.CloudProvider {
	return &baremetal{
		datacenters: datacenters,
		client:      http.DefaultClient,
	}
}

func (b *baremetal) Validate(*kubermaticv1.CloudSpec) error {
	return nil
}

func (b *baremetal) Initialize(cloud *kubermaticv1.CloudSpec, name string) (*kubermaticv1.CloudSpec, error) {
	return cloud, nil
}

func (b *baremetal) CleanUp(*kubermaticv1.CloudSpec) error {
	return nil
}

func (b *baremetal) CreateNodeClass(c *kubermaticv1.Cluster, nSpec *api.NodeSpec, keys []*kubermaticv1.UserSSHKey, version *api.MasterVersion) (*v1alpha1.NodeClass, error) {
	return nil, nil
}

func (b *baremetal) GetNodeClassName(nSpec *api.NodeSpec) string {
	return ""
}
