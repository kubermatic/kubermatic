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

package synccontroller

import (
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
)

func cbslReconcilerFactory(cbsl *kubermaticv1.ClusterBackupStorageLocation) reconciling.NamedClusterBackupStorageLocationReconcilerFactory {
	return func() (string, reconciling.ClusterBackupStorageLocationReconciler) {
		return cbsl.Name, func(existing *kubermaticv1.ClusterBackupStorageLocation) (*kubermaticv1.ClusterBackupStorageLocation, error) {
			if existing.ObjectMeta.Labels == nil {
				existing.ObjectMeta.Labels = map[string]string{}
			}
			for k, v := range cbsl.ObjectMeta.Labels {
				existing.ObjectMeta.Labels[k] = v
			}
			existing.Spec = cbsl.Spec
			return existing, nil
		}
	}
}
