package aws

import (
	"context"

	"github.com/kubermatic/api"
)

type aws struct {
}

func (a *aws) PrepareCloudSpec(*api.Cluster) error {
	panic("not implemented")
}

func (a *aws) CreateAnnotations(*api.CloudSpec) (map[string]string, error) {
	panic("not implemented")
}

func (a *aws) Cloud(annotations map[string]string) (*api.CloudSpec, error) {
	panic("not implemented")
}

func (a *aws) CreateNodes(context.Context, *api.Cluster, *api.NodeSpec, int) ([]*api.Node, error) {
	panic("not implemented")
}

func (a *aws) Nodes(context.Context, *api.Cluster) ([]*api.Node, error) {
	panic("not implemented")
}

func (a *aws) DeleteNodes(ctx context.Context, c *api.Cluster, UIDs []string) error {
	panic("not implemented")
}
