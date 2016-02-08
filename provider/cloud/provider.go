package cloud

import (
	"github.com/kubermatic/api/provider"
)

// Providers returns a map from cloud provider id to the actual provider.
func Providers() map[string]provider.CloudProvider {
	return map[string]provider.CloudProvider{
		provider.FakeCloudProvider:         NewFakeCloudProvider(),
		provider.DigitaloceanCloudProvider: nil,
	}
}
