package cloud

import (
	"github.com/kubermatic/api/provider"
	"github.com/kubermatic/api/provider/cloud/aws"
	"github.com/kubermatic/api/provider/cloud/baremetal"
	"github.com/kubermatic/api/provider/cloud/bringyourown"
	"github.com/kubermatic/api/provider/cloud/digitalocean"
	"github.com/kubermatic/api/provider/cloud/fake"
)

// Providers returns a map from cloud provider id to the actual provider.
func Providers(dcs map[string]provider.DatacenterMeta) provider.CloudRegistry {
	return map[string]provider.CloudProvider{
		provider.FakeCloudProvider:         fake.NewCloudProvider(),
		provider.DigitaloceanCloudProvider: digitalocean.NewCloudProvider(dcs),
		provider.BringYourOwnCloudProvider: bringyourown.NewCloudProvider(),
		provider.AWSCloudProvider:          aws.NewCloudProvider(dcs),
		provider.BareMetalCloudProvider:    baremetal.NewCloudProvider(dcs),
	}
}
