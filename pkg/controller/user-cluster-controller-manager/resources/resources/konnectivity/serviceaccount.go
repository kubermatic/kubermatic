/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package konnectivity

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// ServiceAccountCreator returns a func to create/update the ServiceAccount used by konnectivity agents.
func ServiceAccountCreator() reconciling.NamedServiceAccountCreatorGetter {
	return func() (string, reconciling.ServiceAccountCreator) {
		return resources.KonnectivityServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			// TODO(pratik): these labels, why?
			sa.Labels = map[string]string{
				"kubernetes.io/cluster-service":   "true",
				"addonmanager.kubernetes.io/mode": "Reconcile",
			}
			return sa, nil
		}
	}
}
