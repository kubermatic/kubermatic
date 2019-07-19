package cloud

import (
	"errors"

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

func Provider(datacenter *kubermaticv1.Datacenter) provider.CloudProvider {
	if datacenter.Spec.Digitalocean != nil {
		return digitalocean.NewCloudProvider()
	}
	if datacenter.Spec.BringYourOwn != nil {
		return bringyourown.NewCloudProvider()
	}
	if datacenter.Spec.AWS != nil {
		return aws.NewCloudProvider(datacenter)
	}
	if datacenter.Spec.Azure != nil {
		return azure.New(datacenter)
	}
	if datacenter.Spec.Openstack != nil {
		return openstack.NewCloudProvider(datacenter)
	}
	if datacenter.Spec.Packet != nil {
		return packet.NewCloudProvider()
	}
	if datacenter.Spec.Hetzner != nil {
		return hetzner.NewCloudProvider()
	}
	if datacenter.Spec.VSphere != nil {
		return vsphere.NewCloudProvider(datacenter)
	}
	if datacenter.Spec.GCP != nil {
		return gcp.NewCloudProvider()
	}
	return nil
}

func FakeProvider(datacenter *kubermaticv1.Datacenter) provider.CloudProvider {
	return fake.NewCloudProvider()
}

func OpenstackProvider(datacenter *kubermaticv1.Datacenter) (*openstack.Provider, error) {
	if datacenter.Spec.Openstack == nil {
		return nil, errors.New("datacenter is not an Openstack datacenter")
	}
	return openstack.NewCloudProvider(datacenter), nil
}

func VSphereProvider(datacenter *kubermaticv1.Datacenter) (*vsphere.Provider, error) {
	if datacenter.Spec.VSphere == nil {
		return nil, errors.New("datacenter is not a vSphere datacenter")
	}
	return vsphere.NewCloudProvider(datacenter), nil
}
