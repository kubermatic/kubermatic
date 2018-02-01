package fake

import (
	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

type fakeCloudProvider struct {
	nodes map[string]*apiv1.Node
}

// NewCloudProvider creates a new fake cloud provider
func NewCloudProvider() provider.CloudProvider {
	return &fakeCloudProvider{
		nodes: map[string]*apiv1.Node{},
	}
}

func (p *fakeCloudProvider) ValidateCloudSpec(*kubermaticv1.CloudSpec) error {
	return nil
}

func (p *fakeCloudProvider) InitializeCloudProvider(cloud *kubermaticv1.CloudSpec, name string) (*kubermaticv1.CloudSpec, error) {
	return nil, nil
}

func (p *fakeCloudProvider) CleanUpCloudProvider(*kubermaticv1.CloudSpec) error {
	return nil
}

func (p *fakeCloudProvider) CreateNodeClass(c *kubermaticv1.Cluster, nSpec *apiv1.NodeSpec, keys []*kubermaticv1.UserSSHKey, version *apiv1.MasterVersion) (*v1alpha1.NodeClass, error) {
	return nil, nil
}

func (p *fakeCloudProvider) NodeClassName(nSpec *apiv1.NodeSpec) string {
	return ""
}

func (p *fakeCloudProvider) ValidateNodeSpec(cloudSpec *kubermaticv1.CloudSpec, nodeSpec *apiv1.NodeSpec) error {
	return nil
}
