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

package kubermatic

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	serviceAccountName = "kubermatic-seed"
)

func ClusterRoleBindingName(cfg *kubermaticv1.KubermaticConfiguration) string {
	return fmt.Sprintf("%s:%s-seed:cluster-admin", cfg.Namespace, cfg.Name)
}

func ServiceAccountReconciler(cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed) reconciling.NamedServiceAccountReconcilerFactory {
	return func() (string, reconciling.ServiceAccountReconciler) {
		return serviceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			return sa, nil
		}
	}
}

func ClusterRoleBindingReconciler(cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed) reconciling.NamedClusterRoleBindingReconcilerFactory {
	name := ClusterRoleBindingName(cfg)

	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return name, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
			}

			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      serviceAccountName,
					Namespace: cfg.Namespace,
				},
			}

			return crb, nil
		}
	}
}

func CABundleConfigMapReconciler(caBundle *corev1.ConfigMap) reconciling.NamedConfigMapReconcilerFactory {
	return func() (string, reconciling.ConfigMapReconciler) {
		return resources.CABundleConfigMapName, func(c *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			c.Data = caBundle.Data

			return c, nil
		}
	}
}
