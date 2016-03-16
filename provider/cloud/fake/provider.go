package fake

import (
	"errors"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"
)

const (
	tokenAnnotationKey = "token"
)

var _ provider.CloudProvider = (*fakeCloudProvider)(nil)

type fakeCloudProvider struct{}

// NewCloudProvider creates a new fake cloud provider
func NewCloudProvider() provider.CloudProvider {
	return &fakeCloudProvider{}
}

func (p *fakeCloudProvider) CreateAnnotations(cloud *api.CloudSpec) (map[string]string, error) {
	as := map[string]string{}
	as[tokenAnnotationKey] = cloud.Fake.Token

	return as, nil
}

func (p *fakeCloudProvider) Cloud(as map[string]string) (*api.CloudSpec, error) {
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

func (p *fakeCloudProvider) CreateNodes(
	ctx context.Context,
	cluster *api.Cluster,
	spec *api.NodeSpec,
	instances int,
) ([]*api.Node, error) {
	return nil, errors.New("not implemented")
}

func (p *fakeCloudProvider) PrepareCloudSpec(c *api.Cluster) error {
	return nil
}

func (p *fakeCloudProvider) Nodes(ctx context.Context, cluster *api.Cluster) ([]*api.Node, error) {
	nodes := []*api.Node{
		&api.Node{
			Metadata: api.Metadata{
				Name: "server1",
			},
			Spec: api.NodeSpec{
				DC: "fake-fra1",
				Fake: &api.FakeNodeSpec{
					Type: "standard-1",
					OS:   "CoreOS alpha 1234",
				},
			},
		},
		&api.Node{
			Metadata: api.Metadata{
				Name: "server2",
			},
			Spec: api.NodeSpec{
				DC: "fake-fra1",
				Fake: &api.FakeNodeSpec{
					Type: "standard-1",
					OS:   "CoreOS alpha 1234",
				},
			},
		},
	}
	return nodes, nil
}
