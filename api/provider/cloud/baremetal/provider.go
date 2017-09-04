package baremetal

import (
	"fmt"
	"net/http"

	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/provider"
)

const (
	clusterNameKey = "bm-cluster-name"
)

type baremetal struct {
	datacenters map[string]provider.DatacenterMeta
	client      *http.Client
}

// NewCloudProvider returns a new bare-metal provider.
func NewCloudProvider(datacenters map[string]provider.DatacenterMeta) provider.CloudProvider {
	return &baremetal{
		datacenters: datacenters,
		client:      http.DefaultClient,
	}
}

func (b *baremetal) Initialize(cloud *api.CloudSpec, name string) (*api.CloudSpec, error) {
	return cloud, nil
}

func (b *baremetal) CleanUp(*api.CloudSpec) error {
	return nil
}

func (*baremetal) MarshalCloudSpec(cs *api.CloudSpec) (annotations map[string]string, err error) {
	annotations = map[string]string{
		clusterNameKey: cs.BareMetal.Name,
	}
	return annotations, nil
}

func (*baremetal) UnmarshalCloudSpec(annotations map[string]string) (*api.CloudSpec, error) {
	cs := api.CloudSpec{BareMetal: &api.BareMetalCloudSpec{}}

	name, ok := annotations[clusterNameKey]
	if !ok {
		return nil, fmt.Errorf("couldn't find key %q in annotations while unmarshalling CloudSpec", clusterNameKey)
	}
	cs.BareMetal.Name = name

	return &cs, nil
}

func (b *baremetal) CreateNodeClass(c *api.Cluster, nSpec *api.NodeSpec, keys []v1.UserSSHKey, version *api.MasterVersion) (*v1alpha1.NodeClass, error) {
	return nil, nil
}

func (b *baremetal) GetNodeClassName(nSpec *api.NodeSpec) string {
	return ""
}
