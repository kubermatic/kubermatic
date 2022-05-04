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

package nodelocaldns

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// ServiceAccountCreator creates the service account for Node Local DNS cache.
func ServiceAccountCreator() reconciling.NamedServiceAccountCreatorGetter {
	return func() (string, reconciling.ServiceAccountCreator) {
		return resources.NodeLocalDNSServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			if sa.Labels == nil {
				sa.Labels = map[string]string{}
			}
			sa.Labels["kubernetes.io/cluster-service"] = "true"
			sa.Labels[addonManagerModeKey] = reconcileModeValue

			return sa, nil
		}
	}
}
