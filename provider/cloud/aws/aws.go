package aws

import (
	"context"
	"strconv"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
)

const (
	accessKeyIDAnnotationKey     = "acccess-key-id"
	secretAccessKeyAnnotationKey = "secret-access-key"
)

type aws struct {
	datacenters map[string]provider.DatacenterMeta
}

// NewCloudProvider returns a new aws provider.
func NewCloudProvider(datacenters map[string]provider.DatacenterMeta) *provider.CloudProvider {
	return &aws{
		datacenters: datacenters,
	}
}

func (a *aws) PrepareCloudSpec(*api.Cluster) error {
	panic("not implemented")
}

func (a *aws) CreateAnnotations(cs *api.CloudSpec) (map[string]string, error) {
	return map[string]string{
		accessKeyIDAnnotationKey:     strconv.FormatInt(cs.AWS.AccessToken, 10),
		secretAccessKeyAnnotationKey: cs.AWS.SecretAccessKey,
	}, nil
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
