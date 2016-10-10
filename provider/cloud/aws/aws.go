package aws

import (
	"context"
	"strconv"

	"github.com/kubermatic/api"
)

const (
	accessKeyIDAnnotationKey     = "acccess-key-id"
	secretAccessKeyAnnotationKey = "secret-access-key"
)

type aws struct {
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
