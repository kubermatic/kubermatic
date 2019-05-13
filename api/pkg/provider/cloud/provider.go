package cloud

import (
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
func Providers(dcs map[string]provider.DatacenterMeta) provider.CloudRegistry {
	return map[string]provider.CloudProvider{
		provider.DigitaloceanCloudProvider: digitalocean.NewCloudProvider(dcs),
		provider.BringYourOwnCloudProvider: bringyourown.NewCloudProvider(),
		provider.AWSCloudProvider:          aws.NewCloudProvider(dcs),
		provider.AzureCloudProvider:        azure.New(dcs),
		provider.OpenstackCloudProvider:    openstack.NewCloudProvider(dcs),
		provider.PacketCloudProvider:       packet.NewCloudProvider(dcs),
		provider.HetznerCloudProvider:      hetzner.NewCloudProvider(dcs),
		provider.VSphereCloudProvider:      vsphere.NewCloudProvider(dcs),
		provider.FakeCloudProvider:         fake.NewCloudProvider(),
		provider.GCPCloudProvider:          gcp.NewCloudProvider(dcs),
	}
}
