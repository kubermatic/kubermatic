package bringyourown

import (
	"errors"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"
)

const ()

var _ provider.CloudProvider = (*bringyourown)(nil)

type bringyourown struct{}

// NewCloudProvider creates a new bringyourown provider.
func NewCloudProvider() provider.CloudProvider {
	return &bringyourown{}
}

func (do *bringyourown) CreateAnnotations(cloud *api.CloudSpec) (map[string]string, error) {
	return map[string]string{}, nil
}

func (do *bringyourown) Cloud(annotations map[string]string) (*api.CloudSpec, error) {
	c := api.CloudSpec{
		BringYourOwn: &api.BringYourOwnCloudSpec{},
	}

	return &c, nil
}

func (do *bringyourown) CreateNodes(
	ctx context.Context,
	cluster *api.Cluster, spec *api.NodeSpec, instances int,
) ([]*api.Node, error) {
	return nil, errors.New("not implemented")
}

func (do *bringyourown) Nodes(ctx context.Context, cluster *api.Cluster) ([]*api.Node, error) {
	return []*api.Node{}, nil
}
