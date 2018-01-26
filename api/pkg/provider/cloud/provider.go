package cloud

import (
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/aws"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/bringyourown"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/digitalocean"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/fake"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/openstack"
)

// Providers returns a map from cloud provider id to the actual provider.
func Providers(dcs map[string]provider.DatacenterMeta) provider.CloudRegistry {
	return map[string]provider.CloudProvider{
		provider.FakeCloudProvider:         fake.NewCloudProvider(),
		provider.DigitaloceanCloudProvider: digitalocean.NewCloudProvider(dcs),
		provider.BringYourOwnCloudProvider: bringyourown.NewCloudProvider(),
		provider.AWSCloudProvider:          aws.NewCloudProvider(dcs),
		provider.OpenstackCloudProvider:    openstack.NewCloudProvider(dcs),
	}
}
