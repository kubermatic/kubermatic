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

package multus

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// ConfigMapCreator returns a ConfigMap containing the config for Multus
func ConfigMapCreator() reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.MultusConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}

			cm.Name = "multus-cni-config"
			cm.Namespace = "kube-system"

			labels := map[string]string{"tier": "node"}
			cm.Labels = resources.BaseAppLabels(resources.MultusConfigMapName, labels)

			cm.Data["cni-conf.json"] = `
{
  "name": "multus-cni-network",
  "type": "multus",
  "capabilities": {
    "portMappings": true
  },
  "delegates": [
    {
      "cniVersion": "0.3.1",
      "name": "default-cni-network",
      "plugins": [
        {
          "type": "flannel",
          "name": "flannel.1",
            "delegate": {
              "isDefaultGateway": true,
              "hairpinMode": true
            }
          },
          {
            "type": "portmap",
            "capabilities": {
              "portMappings": true
            }
          }
      ]
    }
  ],
  "kubeconfig": "/etc/cni/net.d/multus.d/multus.kubeconfig"
}
`
			return cm, nil
		}
	}
}
