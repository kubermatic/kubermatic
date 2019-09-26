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
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/kubevirt"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/openstack"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/packet"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/vsphere"
)

func Provider(datacenter *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc) (provider.CloudProvider, error) {
	if datacenter.Spec.Digitalocean != nil {
		return digitalocean.NewCloudProvider(), nil
	}
	if datacenter.Spec.BringYourOwn != nil {
		return bringyourown.NewCloudProvider(), nil
	}
	if datacenter.Spec.AWS != nil {
		return aws.NewCloudProvider(datacenter, secretKeyGetter)
	}
	if datacenter.Spec.Azure != nil {
		return azure.New(datacenter, secretKeyGetter)
	}
	if datacenter.Spec.Openstack != nil {
		return openstack.NewCloudProvider(datacenter)
	}
	if datacenter.Spec.Packet != nil {
		return packet.NewCloudProvider(), nil
	}
	if datacenter.Spec.Hetzner != nil {
		return hetzner.NewCloudProvider(), nil
	}
	if datacenter.Spec.VSphere != nil {
		return vsphere.NewCloudProvider(datacenter)
	}
	if datacenter.Spec.GCP != nil {
		return gcp.NewCloudProvider(secretKeyGetter), nil
	}
	if datacenter.Spec.Fake != nil {
		return fake.NewCloudProvider(), nil
	}
	if datacenter.Spec.Kubevirt != nil {
		return kubevirt.NewCloudProvider(), nil
	}
	return nil, errors.New("no cloudprovider found")
}
