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
}

func ForCluster(cluster *kubermaticv1.Cluster, dc *kubermaticv1.Datacenter) CloudConfig {
	infraNamespace := cluster.Status.NamespaceName

	if dc.Spec.Kubevirt != nil && dc.Spec.Kubevirt.NamespacedMode != nil && dc.Spec.Kubevirt.NamespacedMode.Enabled {
		infraNamespace = dc.Spec.Kubevirt.NamespacedMode.Namespace
	}

	return CloudConfig{
		Kubeconfig: "/etc/kubernetes/cloud/infra-kubeconfig",
		Namespace:  infraNamespace,
	}
}

func (c *CloudConfig) String() (string, error) {
	out, err := yaml.Marshal(c)
	if err != nil {
		return "", err
	}

	return string(out), nil
}
