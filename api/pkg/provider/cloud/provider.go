package cloud

import (
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/aws"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/azure"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/bringyourown"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/digitalocean"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/fake"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/hetzner"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/openstack"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/vsphere"
)

// Providers returns a map from cloud provider id to the actual provider.
func Providers() provider.CloudRegistry {
	return map[string]provider.CloudProvider{
		provider.DigitaloceanCloudProvider: digitalocean.NewCloudProvider(),
		provider.BringYourOwnCloudProvider: bringyourown.NewCloudProvider(),
		provider.AWSCloudProvider:          aws.NewCloudProvider(),
		provider.AzureCloudProvider:        azure.New(),
		provider.OpenstackCloudProvider:    openstack.NewCloudProvider(),
		provider.HetznerCloudProvider:      hetzner.NewCloudProvider(),
		provider.VSphereCloudProvider:      vsphere.NewCloudProvider(),
		provider.FakeCloudProvider:         fake.NewCloudProvider(),
	}
}
