package cloud

import (
	"github.com/kubermatic/api/provider"
	"github.com/kubermatic/api/provider/cloud/digitalocean"
	"github.com/kubermatic/api/provider/cloud/fake"
)

// Providers returns a map from cloud provider id to the actual provider.
func Providers() provider.CloudRegistry {
	return map[string]provider.CloudProvider{
		provider.FakeCloudProvider:         fake.NewCloudProvider(),
		provider.DigitaloceanCloudProvider: digitalocean.NewCloudProvider(),
	}
}
