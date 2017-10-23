package fake

import (
	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	"github.com/kubermatic/kubermatic/api"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

type fakeCloudProvider struct {
	nodes map[string]*api.Node
}

// NewCloudProvider creates a new fake cloud provider
func NewCloudProvider() provider.CloudProvider {
	return &fakeCloudProvider{
		nodes: map[string]*api.Node{},
	}
}

func (p *fakeCloudProvider) Validate(*kubermaticv1.CloudSpec) error {
	return nil
}

func (p *fakeCloudProvider) Initialize(cloud *kubermaticv1.CloudSpec, name string) (*kubermaticv1.CloudSpec, error) {
	return cloud, nil
}

func (p *fakeCloudProvider) CleanUp(*kubermaticv1.CloudSpec) error {
	return nil
}

func (p *fakeCloudProvider) CreateNodeClass(c *kubermaticv1.Cluster, nSpec *api.NodeSpec, keys []*kubermaticv1.UserSSHKey, version *api.MasterVersion) (*v1alpha1.NodeClass, error) {
	return nil, nil
}

func (p *fakeCloudProvider) GetNodeClassName(nSpec *api.NodeSpec) string {
	return ""
}
