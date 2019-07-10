package cloud

import (
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/aws"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/azure"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/bringyourown"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/digitalocean"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/fake"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/gcp"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/hetzner"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/openstack"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/packet"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/vsphere"
)

// Providers returns a map from cloud provider id to the actual provider.
func Providers(dc *kubermaticv1.SeedDatacenter) provider.CloudRegistry {
	return map[string]provider.CloudProvider{
		provider.DigitaloceanCloudProvider: digitalocean.NewCloudProvider(),
		provider.BringYourOwnCloudProvider: bringyourown.NewCloudProvider(),
		provider.AWSCloudProvider:          aws.NewCloudProvider(dc),
		provider.AzureCloudProvider:        azure.New(dc),
		provider.OpenstackCloudProvider:    openstack.NewCloudProvider(dc),
		provider.PacketCloudProvider:       packet.NewCloudProvider(dc),
		provider.HetznerCloudProvider:      hetzner.NewCloudProvider(),
		provider.VSphereCloudProvider:      vsphere.NewCloudProvider(dc),
		provider.FakeCloudProvider:         fake.NewCloudProvider(),
		provider.GCPCloudProvider:          gcp.NewCloudProvider(dc),
	}
}
