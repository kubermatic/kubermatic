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
	"fmt"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/api/v3/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v3/pkg/provider"
	"k8c.io/kubermatic/v3/pkg/provider/cloud/alibaba"
	"k8c.io/kubermatic/v3/pkg/provider/cloud/anexia"
	"k8c.io/kubermatic/v3/pkg/provider/cloud/aws"
	"k8c.io/kubermatic/v3/pkg/provider/cloud/azure"
	"k8c.io/kubermatic/v3/pkg/provider/cloud/bringyourown"
	"k8c.io/kubermatic/v3/pkg/provider/cloud/digitalocean"
	"k8c.io/kubermatic/v3/pkg/provider/cloud/fake"
	"k8c.io/kubermatic/v3/pkg/provider/cloud/gcp"
	"k8c.io/kubermatic/v3/pkg/provider/cloud/hetzner"
	"k8c.io/kubermatic/v3/pkg/provider/cloud/kubevirt"
	"k8c.io/kubermatic/v3/pkg/provider/cloud/nutanix"
	"k8c.io/kubermatic/v3/pkg/provider/cloud/openstack"
	"k8c.io/kubermatic/v3/pkg/provider/cloud/packet"
	"k8c.io/kubermatic/v3/pkg/provider/cloud/vmwareclouddirector"
	"k8c.io/kubermatic/v3/pkg/provider/cloud/vsphere"
)

func Provider(datacenter *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc, caBundle *x509.CertPool) (provider.CloudProvider, error) {
	providerName, err := kubermaticv1helper.DatacenterCloudProviderName(&datacenter.Spec.Provider)
	if err != nil {
		return nil, err
	}

	switch providerName {
	case kubermaticv1.CloudProviderDigitalocean:
		return digitalocean.NewCloudProvider(secretKeyGetter), nil
	case kubermaticv1.CloudProviderBringYourOwn:
		return bringyourown.NewCloudProvider(), nil
	case kubermaticv1.CloudProviderAWS:
		return aws.NewCloudProvider(datacenter, secretKeyGetter)
	case kubermaticv1.CloudProviderAzure:
		return azure.New(datacenter, secretKeyGetter)
	case kubermaticv1.CloudProviderOpenStack:
		return openstack.NewCloudProvider(datacenter, secretKeyGetter, caBundle)
	case kubermaticv1.CloudProviderPacket:
		return packet.NewCloudProvider(secretKeyGetter), nil
	case kubermaticv1.CloudProviderHetzner:
		return hetzner.NewCloudProvider(secretKeyGetter), nil
	case kubermaticv1.CloudProviderVMwareCloudDirector:
		return vmwareclouddirector.NewCloudProvider(datacenter, secretKeyGetter)
	case kubermaticv1.CloudProviderVSphere:
		return vsphere.NewCloudProvider(datacenter, secretKeyGetter, caBundle)
	case kubermaticv1.CloudProviderGCP:
		return gcp.NewCloudProvider(secretKeyGetter), nil
	case kubermaticv1.CloudProviderFake:
		return fake.NewCloudProvider(), nil
	case kubermaticv1.CloudProviderKubeVirt:
		return kubevirt.NewCloudProvider(datacenter, secretKeyGetter)
	case kubermaticv1.CloudProviderAlibaba:
		return alibaba.NewCloudProvider(datacenter, secretKeyGetter)
	case kubermaticv1.CloudProviderAnexia:
		return anexia.NewCloudProvider(datacenter, secretKeyGetter)
	case kubermaticv1.CloudProviderNutanix:
		return nutanix.NewCloudProvider(datacenter, secretKeyGetter)
	}

	return nil, fmt.Errorf("unknown cloud provider %s", providerName)
}
