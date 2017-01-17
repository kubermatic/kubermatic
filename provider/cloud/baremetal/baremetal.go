package baremetal

import (
	"github.com/kubermatic/api"
	"golang.org/x/net/context"
)

type baremetal struct {

}

func (*baremetal) InitializeCloudSpec(*api.Cluster) error {
	panic("implement me")
}

func (*baremetal) MarshalCloudSpec(*api.CloudSpec) (annotations map[string]string, err error) {
	panic("implement me")
}

func (*baremetal) UnmarshalCloudSpec(annotations map[string]string) (*api.CloudSpec, error) {
	panic("implement me")
}

func (*baremetal) CreateNodes(context.Context, *api.Cluster, *api.NodeSpec, int) ([]*api.Node, error) {
	panic("implement me")
}

func (*baremetal) Nodes(context.Context, *api.Cluster) ([]*api.Node, error) {
	panic("implement me")
}

func (*baremetal) DeleteNodes(ctx context.Context, c *api.Cluster, UIDs []string) error {
	panic("implement me")
}

func (*baremetal) CleanUp(*api.Cluster) error {
	panic("implement me")
}




