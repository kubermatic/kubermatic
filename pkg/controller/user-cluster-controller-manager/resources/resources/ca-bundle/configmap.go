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

package cabundle

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// ConfigMapReconciler returns a ConfigMap containing the CA bundle for the usercluster.
func ConfigMapReconciler(caBundle resources.CABundle) reconciling.NamedConfigMapReconcilerFactory { //nolint:interfacer
	return func() (string, reconciling.ConfigMapReconciler) {
		return resources.CABundleConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}

			cm.Data[resources.CABundleConfigMapKey] = caBundle.String()
			return cm, nil
		}
	}
}
