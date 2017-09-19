package bringyourown

import (
	"encoding/base64"

	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/extensions"
	"github.com/kubermatic/kubermatic/api/provider"
)

const (
	privateIntfAnnotationKey = "private-interface"
	clientKeyAnnotationKey   = "client-key"
	clientCertAnnotationKey  = "client-cert"
)

type bringyourown struct{}

// NewCloudProvider creates a new bringyourown provider.
func NewCloudProvider() provider.CloudProvider {
	return &bringyourown{}
}

func (b *bringyourown) Initialize(cloud *api.CloudSpec, name string) (*api.CloudSpec, error) {
	return cloud, nil
}

func (b *bringyourown) CleanUp(*api.CloudSpec) error {
	return nil
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

func (b *bringyourown) CreateNodeClass(c *api.Cluster, nSpec *api.NodeSpec, keys []extensions.UserSSHKey, version *api.MasterVersion) (*v1alpha1.NodeClass, error) {
	return nil, nil
}

func (b *bringyourown) GetNodeClassName(nSpec *api.NodeSpec) string {
	return ""
}
