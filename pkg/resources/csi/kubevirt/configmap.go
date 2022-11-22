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

package kubevirt

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// ConfigMapsCreators returns the CSI controller ConfigMaps for KubeVirt.
func ConfigMapsCreators(data *resources.TemplateData) []reconciling.NamedConfigMapCreatorGetter {
	creators := []reconciling.NamedConfigMapCreatorGetter{
		ControllerConfigMapCreator(data),
	}
	return creators
}

// ControllerConfigMapCreator returns the CSI controller ConfigMap for KubeVirt.
func ControllerConfigMapCreator(data *resources.TemplateData) reconciling.NamedConfigMapCreatorGetter {
	return func() (name string, create reconciling.ConfigMapCreator) {
		return resources.KubeVirtCSIConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			cm.Labels = resources.BaseAppLabels(resources.KubeVirtCSIConfigMapName, nil)
			cm.Data[resources.KubeVirtCSINamespaceKey] = data.Cluster().Status.NamespaceName
			cm.Data[resources.KubeVirtCSIClusterLabelKey] = fmt.Sprintf("cluster-name=%s", data.Cluster().Name)
			return cm, nil
		}
	}
}
