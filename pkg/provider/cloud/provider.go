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
	"errors"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/alibaba"
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
		return digitalocean.NewCloudProvider(secretKeyGetter), nil
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
		return openstack.NewCloudProvider(datacenter, secretKeyGetter)
	}
	if datacenter.Spec.Packet != nil {
		return packet.NewCloudProvider(secretKeyGetter), nil
	}
	if datacenter.Spec.Hetzner != nil {
		return hetzner.NewCloudProvider(secretKeyGetter), nil
	}
	if datacenter.Spec.VSphere != nil {
		return vsphere.NewCloudProvider(datacenter, secretKeyGetter)
	}
	if datacenter.Spec.GCP != nil {
		return gcp.NewCloudProvider(secretKeyGetter), nil
	}
	if datacenter.Spec.Fake != nil {
		return fake.NewCloudProvider(), nil
	}
	if datacenter.Spec.Kubevirt != nil {
		return kubevirt.NewCloudProvider(secretKeyGetter), nil
	}
	if datacenter.Spec.Alibaba != nil {
		return alibaba.NewCloudProvider(datacenter, secretKeyGetter)
	}
	return nil, errors.New("no cloudprovider found")
}
