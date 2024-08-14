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

	kubevirt "k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// ConfigMapsReconcilers returns the CSI controller ConfigMaps for KubeVirt.
func ConfigMapsReconcilers(data *resources.TemplateData) []reconciling.NamedConfigMapReconcilerFactory {
	creators := []reconciling.NamedConfigMapReconcilerFactory{
		ControllerConfigMapReconciler(data),
	}
	return creators
}

// ControllerConfigMapReconciler returns the CSI controller ConfigMap for KubeVirt.
func ControllerConfigMapReconciler(data *resources.TemplateData) reconciling.NamedConfigMapReconcilerFactory {
	return func() (name string, create reconciling.ConfigMapReconciler) {
		return resources.KubeVirtCSIConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			cm.Labels = resources.BaseAppLabels(resources.KubeVirtCSIConfigMapName, nil)
			kubevirtInfraNamespace := data.Cluster().Status.NamespaceName
			if data.DC().Spec.Kubevirt != nil && data.DC().Spec.Kubevirt.NamespacedMode {
				kubevirtInfraNamespace = kubevirt.DefaultNamespaceName
			}
			cm.Data[resources.KubeVirtCSINamespaceKey] = kubevirtInfraNamespace
			cm.Data[resources.KubeVirtCSIClusterLabelKey] = fmt.Sprintf("cluster-name=%s", data.Cluster().Name)
			return cm, nil
		}
	}
}
