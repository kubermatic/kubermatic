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

package vmwareclouddirector

import (
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// ServiceAccountsReconcilers returns the CSI serviceaccounts.
func ServiceAccountsReconcilers(c *kubermaticv1.Cluster) []reconciling.NamedServiceAccountReconcilerFactory {
	creators := []reconciling.NamedServiceAccountReconcilerFactory{
		ControllerServiceAccountReconciler(c),
	}
	return creators
}

// ControllerServiceAccountReconciler returns the CSI serviceaccount.
func ControllerServiceAccountReconciler(c *kubermaticv1.Cluster) reconciling.NamedServiceAccountReconcilerFactory {
	return func() (name string, create reconciling.ServiceAccountReconciler) {
		return resources.VMwareCloudDirectorCSIServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			sa.Labels = resources.BaseAppLabels(resources.VMwareCloudDirectorCSIServiceAccountName, nil)
			sa.Name = resources.VMwareCloudDirectorCSIServiceAccountName
			sa.Namespace = c.Status.NamespaceName
			return sa, nil
		}
	}
}
