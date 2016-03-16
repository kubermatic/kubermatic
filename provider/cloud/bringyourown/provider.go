package bringyourown

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/kubermatic/api"
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

func (do *bringyourown) CreateAnnotations(cloud *api.CloudSpec) (map[string]string, error) {
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

func (do *bringyourown) Cloud(as map[string]string) (*api.CloudSpec, error) {
	c := api.CloudSpec{
		BringYourOwn: &api.BringYourOwnCloudSpec{
			PrivateIntf: as[privateIntfAnnotationKey],
		},
	}
	if as[clientKeyAnnotationKey] != "" {
		c.BringYourOwn.ClientKeyCert.Key, _ =
			base64.StdEncoding.DecodeString(as[clientKeyAnnotationKey])
	}
	if as[clientCertAnnotationKey] != "" {
		c.BringYourOwn.ClientKeyCert.Cert, _ =
			base64.StdEncoding.DecodeString(as[clientCertAnnotationKey])
	}

	return &c, nil
}

func (do *bringyourown) CreateNodes(
	ctx context.Context,
	cluster *api.Cluster, spec *api.NodeSpec, instances int,
) ([]*api.Node, error) {
	return nil, errors.New("not implemented")
}

func (do *bringyourown) PrepareCloudSpec(c *api.Cluster) error {
	if c.Status.RootCA.Key != nil && c.Status.RootCA.Cert != nil {
		clientCA, err := c.CreateKeyCert("seed-etcd-client-ca")
		if err != nil {
			return fmt.Errorf("failed to create a client ca for cluster %q", c.Metadata.Name)
		}
		c.Spec.Cloud.BringYourOwn.ClientKeyCert = *clientCA
	}

	return nil
}

func (do *bringyourown) Nodes(ctx context.Context, cluster *api.Cluster) ([]*api.Node, error) {
	return []*api.Node{}, nil
}
