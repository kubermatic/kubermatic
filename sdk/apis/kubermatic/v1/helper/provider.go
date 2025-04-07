/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package helper

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

// ExternalClusterCloudProviderName returns the provider name for the given ExternalClusterCloudSpec.
func ExternalClusterCloudProviderName(spec kubermaticv1.ExternalClusterCloudSpec) (string, error) {
	var clouds []kubermaticv1.ExternalClusterProvider
	if spec.AKS != nil {
		clouds = append(clouds, kubermaticv1.ExternalClusterAKSProvider)
	}
	if spec.EKS != nil {
		clouds = append(clouds, kubermaticv1.ExternalClusterEKSProvider)
	}
	if spec.GKE != nil {
		clouds = append(clouds, kubermaticv1.ExternalClusterGKEProvider)
	}
	if spec.KubeOne != nil {
		clouds = append(clouds, kubermaticv1.ExternalClusterKubeOneProvider)
	}
	if spec.BringYourOwn != nil {
		clouds = append(clouds, kubermaticv1.ExternalClusterBringYourOwnProvider)
	}
	if len(clouds) == 0 {
		return "", nil
	}
	if len(clouds) != 1 {
		return "", fmt.Errorf("only one cloud provider can be set in ExternalClusterCloudSpec, but found the following providers: %v", clouds)
	}
	return string(clouds[0]), nil
}

// ClusterCloudProviderName returns the provider name for the given CloudSpec.
func ClusterCloudProviderName(spec kubermaticv1.CloudSpec) (string, error) {
	var clouds []kubermaticv1.ProviderType
	if spec.AWS != nil {
		clouds = append(clouds, kubermaticv1.AWSCloudProvider)
	}
	if spec.Alibaba != nil {
		clouds = append(clouds, kubermaticv1.AlibabaCloudProvider)
	}
	if spec.Anexia != nil {
		clouds = append(clouds, kubermaticv1.AnexiaCloudProvider)
	}
	if spec.Azure != nil {
		clouds = append(clouds, kubermaticv1.AzureCloudProvider)
	}
	if spec.Baremetal != nil {
		clouds = append(clouds, kubermaticv1.BaremetalCloudProvider)
	}
	if spec.BringYourOwn != nil {
		clouds = append(clouds, kubermaticv1.BringYourOwnCloudProvider)
	}
	if spec.Edge != nil {
		clouds = append(clouds, kubermaticv1.EdgeCloudProvider)
	}
	if spec.Digitalocean != nil {
		clouds = append(clouds, kubermaticv1.DigitaloceanCloudProvider)
	}
	if spec.Fake != nil {
		clouds = append(clouds, kubermaticv1.FakeCloudProvider)
	}
	if spec.GCP != nil {
		clouds = append(clouds, kubermaticv1.GCPCloudProvider)
	}
	if spec.Hetzner != nil {
		clouds = append(clouds, kubermaticv1.HetznerCloudProvider)
	}
	if spec.Kubevirt != nil {
		clouds = append(clouds, kubermaticv1.KubevirtCloudProvider)
	}
	if spec.Openstack != nil {
		clouds = append(clouds, kubermaticv1.OpenstackCloudProvider)
	}
	if spec.Packet != nil {
		clouds = append(clouds, kubermaticv1.PacketCloudProvider)
	}
	if spec.VSphere != nil {
		clouds = append(clouds, kubermaticv1.VSphereCloudProvider)
	}
	if spec.Nutanix != nil {
		clouds = append(clouds, kubermaticv1.NutanixCloudProvider)
	}
	if spec.VMwareCloudDirector != nil {
		clouds = append(clouds, kubermaticv1.VMwareCloudDirectorCloudProvider)
	}
	if len(clouds) == 0 {
		return "", nil
	}
	if len(clouds) != 1 {
		return "", fmt.Errorf("only one cloud provider can be set in CloudSpec, but found the following providers: %v", clouds)
	}
	return string(clouds[0]), nil
}

// DatacenterCloudProviderName returns the provider name for the given Datacenter.
func DatacenterCloudProviderName(spec *kubermaticv1.DatacenterSpec) (string, error) {
	if spec == nil {
		return "", nil
	}
	var clouds []kubermaticv1.ProviderType
	if spec.BringYourOwn != nil {
		clouds = append(clouds, kubermaticv1.BringYourOwnCloudProvider)
	}
	if spec.Baremetal != nil {
		clouds = append(clouds, kubermaticv1.BaremetalCloudProvider)
	}
	if spec.Edge != nil {
		clouds = append(clouds, kubermaticv1.EdgeCloudProvider)
	}
	if spec.Digitalocean != nil {
		clouds = append(clouds, kubermaticv1.DigitaloceanCloudProvider)
	}
	if spec.AWS != nil {
		clouds = append(clouds, kubermaticv1.AWSCloudProvider)
	}
	if spec.Openstack != nil {
		clouds = append(clouds, kubermaticv1.OpenstackCloudProvider)
	}
	if spec.Packet != nil {
		clouds = append(clouds, kubermaticv1.PacketCloudProvider)
	}
	if spec.Hetzner != nil {
		clouds = append(clouds, kubermaticv1.HetznerCloudProvider)
	}
	if spec.VSphere != nil {
		clouds = append(clouds, kubermaticv1.VSphereCloudProvider)
	}
	if spec.Azure != nil {
		clouds = append(clouds, kubermaticv1.AzureCloudProvider)
	}
	if spec.GCP != nil {
		clouds = append(clouds, kubermaticv1.GCPCloudProvider)
	}
	if spec.Fake != nil {
		clouds = append(clouds, kubermaticv1.FakeCloudProvider)
	}
	if spec.Kubevirt != nil {
		clouds = append(clouds, kubermaticv1.KubevirtCloudProvider)
	}
	if spec.Alibaba != nil {
		clouds = append(clouds, kubermaticv1.AlibabaCloudProvider)
	}
	if spec.Anexia != nil {
		clouds = append(clouds, kubermaticv1.AnexiaCloudProvider)
	}
	if spec.Nutanix != nil {
		clouds = append(clouds, kubermaticv1.NutanixCloudProvider)
	}
	if spec.VMwareCloudDirector != nil {
		clouds = append(clouds, kubermaticv1.VMwareCloudDirectorCloudProvider)
	}
	if len(clouds) == 0 {
		return "", nil
	}
	if len(clouds) != 1 {
		return "", fmt.Errorf("only one cloud provider can be set in DatacenterSpec: %+v", spec)
	}
	return string(clouds[0]), nil
}
