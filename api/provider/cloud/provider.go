package cloud

import (
	"github.com/kubermatic/kubermatic/api/provider"
	"github.com/kubermatic/kubermatic/api/provider/cloud/aws"
	"github.com/kubermatic/kubermatic/api/provider/cloud/baremetal"
	"github.com/kubermatic/kubermatic/api/provider/cloud/bringyourown"
	"github.com/kubermatic/kubermatic/api/provider/cloud/digitalocean"
	"github.com/kubermatic/kubermatic/api/provider/cloud/fake"
	"github.com/kubermatic/kubermatic/api/provider/cloud/openstack"
	"github.com/kubermatic/kubermatic/api/provider/cloud/otc"
)

// Providers returns a map from cloud provider id to the actual provider.
func Providers(dcs map[string]provider.DatacenterMeta) provider.CloudRegistry {
	return map[string]provider.CloudProvider{
		provider.FakeCloudProvider:         fake.NewCloudProvider(),
		provider.DigitaloceanCloudProvider: digitalocean.NewCloudProvider(dcs),
		provider.BringYourOwnCloudProvider: bringyourown.NewCloudProvider(),
		provider.AWSCloudProvider:          aws.NewCloudProvider(dcs),
		provider.BareMetalCloudProvider:    baremetal.NewCloudProvider(dcs),
		provider.OpenstackCloudProvider:    openstack.NewCloudProvider(dcs),
		provider.OTCCloudProvider:          otc.NewCloudProvider(dcs),
	}
}
