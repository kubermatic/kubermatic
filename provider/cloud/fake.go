package cloud

import (
	"errors"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
)

const (
	tokenAnnotationKey  = "token"
	regionAnnotationKey = "region"
	dcAnnotationKey     = "dc"
)

var _ provider.CloudProvider = (*fakeCloudProvider)(nil)

type fakeCloudProvider struct {
}

// NewFakeCloudProvider creates a new fake cloud provider
func NewFakeCloudProvider() provider.CloudProvider {
	return &fakeCloudProvider{}
}

func (p *fakeCloudProvider) CreateAnnotations(cloud *api.CloudSpec) (map[string]string, error) {
	as := map[string]string{}
	as[tokenAnnotationKey] = cloud.Fake.Token
	as[regionAnnotationKey] = cloud.Fake.Region
	as[dcAnnotationKey] = cloud.Fake.Dc

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

	c.Fake.Region, found = as[regionAnnotationKey]
	if !found {
		return nil, errors.New("no region found in fake cloud provider")
	}

	c.Fake.Dc, found = as[dcAnnotationKey]
	if !found {
		return nil, errors.New("no datacenter found in fake cloud provider")
	}

	return &c, nil
}

func (p *fakeCloudProvider) CreateNode(cluster *api.Cluster, spec *api.NodeSpec) (*api.Node, error) {
	return nil, errors.New("not implemented")
}

func (p *fakeCloudProvider) Nodes(cluster *api.Cluster) ([]*api.Node, error) {
	nodes := []*api.Node{
		&api.Node{
			Metadata: api.Metadata{
				Name: "server1",
			},
			Spec: api.NodeSpec{
				Type: "standard-1",
				OS:   "CoreOS alpha 1234",
			},
		},
		&api.Node{
			Metadata: api.Metadata{
				Name: "server2",
			},
			Spec: api.NodeSpec{
				Type: "standard-1",
				OS:   "CoreOS alpha 1234",
			},
		},
	}
	return nodes, nil
}

func (p *fakeCloudProvider) Name() string {
	return provider.FakeCloudProvider
}
