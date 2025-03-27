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

package etcd

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// ServiceAccountReconciler returns a func to create/update the ServiceAccount used by etcd launcher.
func ServiceAccountReconciler() (string, reconciling.ServiceAccountReconciler) {
	return rbac.EtcdLauncherServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
		return sa, nil
	}
}

func KubeSystemServiceAccountReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedServiceAccountReconcilerFactory {
	return func() (string, reconciling.ServiceAccountReconciler) {
		return fmt.Sprintf("%s-%s", rbac.EtcdLauncherServiceAccountName, cluster.Name), func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			return sa, nil
		}
	}
}
