/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cloud

import (
	"crypto/x509"
	"errors"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/alibaba"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/anexia"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/azure"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/baremetal"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/bringyourown"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/digitalocean"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/edge"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/fake"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gcp"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/hetzner"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/nutanix"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/openstack"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/packet"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/vmwareclouddirector"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/vsphere"
)

func Provider(
	datacenter *kubermaticv1.Datacenter,
	secretKeyGetter provider.SecretKeySelectorValueFunc,
	caBundle *x509.CertPool,
) (provider.CloudProvider, error) {
	if datacenter.Spec.Digitalocean != nil {
		return digitalocean.NewCloudProvider(secretKeyGetter), nil
	}
	if datacenter.Spec.BringYourOwn != nil {
		return bringyourown.NewCloudProvider(), nil
	}
	if datacenter.Spec.Edge != nil {
		return edge.NewCloudProvider(), nil
	}
	if datacenter.Spec.AWS != nil {
		return aws.NewCloudProvider(datacenter, secretKeyGetter)
	}
	if datacenter.Spec.Azure != nil {
		return azure.New(datacenter, secretKeyGetter)
	}
	if datacenter.Spec.Openstack != nil {
		return openstack.NewCloudProvider(datacenter, secretKeyGetter, caBundle)
	}
	if datacenter.Spec.Packet != nil {
		return packet.NewCloudProvider(secretKeyGetter), nil
	}
	if datacenter.Spec.Hetzner != nil {
		return hetzner.NewCloudProvider(secretKeyGetter), nil
	}
	if datacenter.Spec.VMwareCloudDirector != nil {
		return vmwareclouddirector.NewCloudProvider(datacenter, secretKeyGetter)
	}
	if datacenter.Spec.VSphere != nil {
		return vsphere.NewCloudProvider(datacenter, secretKeyGetter, caBundle)
	}
	if datacenter.Spec.GCP != nil {
		return gcp.NewCloudProvider(secretKeyGetter), nil
	}
	if datacenter.Spec.Fake != nil {
		return fake.NewCloudProvider(), nil
	}
	if datacenter.Spec.Kubevirt != nil {
		return kubevirt.NewCloudProvider(datacenter, secretKeyGetter)
	}
	if datacenter.Spec.Alibaba != nil {
		return alibaba.NewCloudProvider(datacenter, secretKeyGetter)
	}
	if datacenter.Spec.Anexia != nil {
		return anexia.NewCloudProvider(datacenter, secretKeyGetter)
	}
	if datacenter.Spec.Nutanix != nil {
		return nutanix.NewCloudProvider(datacenter, secretKeyGetter)
	}
	if datacenter.Spec.Baremetal != nil {
		return baremetal.NewCloudProvider(), nil
	}
	return nil, errors.New("no cloudprovider found")
}
