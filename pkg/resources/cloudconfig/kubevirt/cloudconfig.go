/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package kubevirt

import (
	"gopkg.in/yaml.v3"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
)

type CloudConfig struct {
	// Kubeconfig used to connect to the cluster that runs KubeVirt
	Kubeconfig string `yaml:"kubeconfig"`
	// Namespace used in KubeVirt cloud-controller-manager as infra cluster namespace.
	Namespace string `yaml:"namespace"`
	// InstancesV2 used in KubeVirt cloud-controller-manager as metadata information about the infra cluster nodes
	InstancesV2 InstancesV2 `yaml:"instancesV2"`
	// LoadBalancer configures cloud-controller-manager LoadBalancer interface configurations.
	LoadBalancer LoadBalancer `yaml:"loadBalancer"`
}

type InstancesV2 struct {
	ZoneAndRegionEnabled bool `yaml:"zoneAndRegionEnabled"`
}

type LoadBalancer struct {
	// Enabled activates the load balancer interface of the CCM
	Enabled bool `yaml:"enabled"`
}

func ForCluster(cluster *kubermaticv1.Cluster, dc *kubermaticv1.Datacenter) CloudConfig {
	cloudConfig := CloudConfig{
		Kubeconfig: "/etc/kubernetes/cloud/infra-kubeconfig",
		Namespace:  cluster.Status.NamespaceName,
		InstancesV2: InstancesV2{
			ZoneAndRegionEnabled: true,
		},
		LoadBalancer: LoadBalancer{
			Enabled: true,
		},
	}

	if dc.Spec.Kubevirt != nil && dc.Spec.Kubevirt.NamespacedMode != nil && dc.Spec.Kubevirt.NamespacedMode.Enabled {
		cloudConfig.Namespace = dc.Spec.Kubevirt.NamespacedMode.Namespace
	}

	if dc.Spec.Kubevirt != nil &&
		dc.Spec.Kubevirt.CCMZoneAndRegionEnabled != nil &&
		!*dc.Spec.Kubevirt.CCMZoneAndRegionEnabled {
		cloudConfig.InstancesV2.ZoneAndRegionEnabled = false
	}

	if dc.Spec.Kubevirt != nil &&
		dc.Spec.Kubevirt.CCMLoadBalancerEnabled != nil &&
		!*dc.Spec.Kubevirt.CCMLoadBalancerEnabled {
		cloudConfig.LoadBalancer.Enabled = false
	}

	return cloudConfig
}

func (c *CloudConfig) String() (string, error) {
	out, err := yaml.Marshal(c)
	if err != nil {
		return "", err
	}

	return string(out), nil
}
