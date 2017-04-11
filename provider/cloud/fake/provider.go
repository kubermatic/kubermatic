package fake

import (
	"errors"
	"fmt"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/util/rand"
)

const (
	tokenAnnotationKey = "token"
)

var _ provider.CloudProvider = (*fakeCloudProvider)(nil)

type fakeCloudProvider struct {
	nodes map[string]*api.Node
}

// NewCloudProvider creates a new fake cloud provider
func NewCloudProvider() provider.CloudProvider {
	return &fakeCloudProvider{
		nodes: map[string]*api.Node{},
	}
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

func (p *fakeCloudProvider) CreateNodes(
	ctx context.Context,
	cluster *api.Cluster,
	spec *api.NodeSpec,
	instances int,
) ([]*api.Node, error) {
	var ns []*api.Node

	for i := 0; i < instances; i++ {
		n := &api.Node{
			Metadata: api.Metadata{
				UID:  rand.String(4),
				Name: rand.String(8),
			},
			Spec: *spec,
		}

		p.nodes[n.Metadata.UID] = n
		ns = append(ns, n)
	}

	return ns, nil
}

func (p *fakeCloudProvider) InitializeCloudSpec(c *api.Cluster) error {
	return nil
}

func (p *fakeCloudProvider) Nodes(ctx context.Context, cluster *api.Cluster) ([]*api.Node, error) {
	var ns []*api.Node

	for _, n := range p.nodes {
		ns = append(ns, n)
	}

	return ns, nil
}

func (p *fakeCloudProvider) DeleteNodes(ctx context.Context, c *api.Cluster, UIDs []string) error {
	// @apinnecke: Removed error throwing to make code testable
	//return errors.New("delete: unsupported operation")

	for _, u := range UIDs {
		_, found := p.nodes[u]
		if !found {
			return fmt.Errorf("node %q not found", u)
		}

		delete(p.nodes, u)
	}

	return nil
}

func (p *fakeCloudProvider) CleanUp(c *api.Cluster) error {
	return nil
}
