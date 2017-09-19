package fake

import (
	"errors"

	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/extensions"
	"github.com/kubermatic/kubermatic/api/provider"
)

const (
	tokenAnnotationKey = "token"
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

func (p *fakeCloudProvider) Initialize(cloud *api.CloudSpec, name string) (*api.CloudSpec, error) {
	return cloud, nil
}

func (p *fakeCloudProvider) CleanUp(*api.CloudSpec) error {
	return nil
}

func (p *fakeCloudProvider) MarshalCloudSpec(cloud *api.CloudSpec) (map[string]string, error) {
	as := map[string]string{}
	as[tokenAnnotationKey] = cloud.Fake.Token

	return as, nil
}

func (p *fakeCloudProvider) UnmarshalCloudSpec(as map[string]string) (*api.CloudSpec, error) {
	c := api.CloudSpec{
		Fake: &api.FakeCloudSpec{},
	}

	var found bool
	c.Fake.Token, found = as[tokenAnnotationKey]
	if !found {
		return nil, errors.New("no token found in fake cloud provider")
	}

	return &c, nil
}

func (p *fakeCloudProvider) CreateNodeClass(c *api.Cluster, nSpec *api.NodeSpec, keys []extensions.UserSSHKey, version *api.MasterVersion) (*v1alpha1.NodeClass, error) {
	return nil, nil
}

func (p *fakeCloudProvider) GetNodeClassName(nSpec *api.NodeSpec) string {
	return ""
}
