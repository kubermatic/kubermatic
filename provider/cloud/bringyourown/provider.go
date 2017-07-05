package bringyourown

import (
	"encoding/base64"
	"errors"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/extensions"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"
)

const (
	privateIntfAnnotationKey = "private-interface"
	clientKeyAnnotationKey   = "client-key"
	clientCertAnnotationKey  = "client-cert"
)

var _ provider.CloudProvider = (*bringyourown)(nil)

type bringyourown struct{}

// NewCloudProvider creates a new bringyourown provider.
func NewCloudProvider() provider.CloudProvider {
	return &bringyourown{}
}

func (b *bringyourown) MarshalCloudSpec(cloud *api.CloudSpec) (map[string]string, error) {
	as := map[string]string{
		privateIntfAnnotationKey: cloud.BringYourOwn.PrivateIntf,
	}
	if cloud.BringYourOwn.ClientKeyCert.Key != nil {
		as[clientKeyAnnotationKey] = base64.StdEncoding.EncodeToString(cloud.BringYourOwn.ClientKeyCert.Key)
	}
	if cloud.BringYourOwn.ClientKeyCert.Cert != nil {
		as[clientCertAnnotationKey] = base64.StdEncoding.EncodeToString(cloud.BringYourOwn.ClientKeyCert.Cert)
	}

	return as, nil
}

func (b *bringyourown) UnmarshalCloudSpec(as map[string]string) (*api.CloudSpec, error) {
	c := api.CloudSpec{
		BringYourOwn: &api.BringYourOwnCloudSpec{
			PrivateIntf: as[privateIntfAnnotationKey],
		},
	}
	if as[clientKeyAnnotationKey] != "" {
		key, err := base64.StdEncoding.DecodeString(as[clientKeyAnnotationKey])
		if err != nil {
			return &c, err
		}
		c.BringYourOwn.ClientKeyCert.Key = key
	}
	if as[clientCertAnnotationKey] != "" {
		cert, err := base64.StdEncoding.DecodeString(as[clientCertAnnotationKey])
		if err != nil {
			return &c, err
		}
		c.BringYourOwn.ClientKeyCert.Cert = cert
	}

	return &c, nil
}

func (b *bringyourown) CreateNodes(
	ctx context.Context,
	cluster *api.Cluster, spec *api.NodeSpec, instances int, keys []extensions.UserSSHKey,
) ([]*api.Node, error) {
	return nil, errors.New("not implemented")
}

func (b *bringyourown) InitializeCloudSpec(c *api.Cluster) error {
	return nil
}

func (b *bringyourown) Nodes(ctx context.Context, cluster *api.Cluster) ([]*api.Node, error) {
	return []*api.Node{}, nil
}

func (b *bringyourown) DeleteNodes(ctx context.Context, c *api.Cluster, UIDs []string) error {
	return errors.New("delete: unsupported operation")
}

func (b *bringyourown) CleanUp(c *api.Cluster) error {
	return nil
}
