package bringyourown

import (
	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	"github.com/kubermatic/kubermatic/api"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

type bringyourown struct{}

// NewCloudProvider creates a new bringyourown provider.
func NewCloudProvider() provider.CloudProvider {
	return &bringyourown{}
}

func (b *bringyourown) Validate(*kubermaticv1.CloudSpec) error {
	return nil
}

func (b *bringyourown) Initialize(cloud *kubermaticv1.CloudSpec, name string) (*kubermaticv1.CloudSpec, error) {
	return nil, nil
}

func (b *bringyourown) CleanUp(*kubermaticv1.CloudSpec) error {
	return nil
}

func (b *bringyourown) CreateNodeClass(c *kubermaticv1.Cluster, nSpec *api.NodeSpec, keys []*kubermaticv1.UserSSHKey, version *api.MasterVersion) (*v1alpha1.NodeClass, error) {
	return nil, nil
}

func (b *bringyourown) GetNodeClassName(nSpec *api.NodeSpec) string {
	return ""
}
