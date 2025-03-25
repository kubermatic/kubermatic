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

package webhook

import (
	"fmt"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	serviceAccountName = "usercluster-webhook"
	roleName           = "kubermatic:usercluster-webhook"
	roleBindingName    = "kubermatic:usercluster-webhook"
)

func ServiceAccountReconciler() (string, reconciling.ServiceAccountReconciler) {
	return serviceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
		return sa, nil
	}
}

func ClusterRole() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return roleName, func(r *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{appskubermaticv1.GroupName},
					Resources: []string{"applicationdefinitions", "applicationdefinitions/status"},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
				{
					APIGroups: []string{kubermaticv1.GroupName},
					Resources: []string{"resourcequotas", "resourcequotas/status"},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
			}
			return r, nil
		}
	}
}

func ClusterRoleBinding(namespace *corev1.Namespace) reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return GenerateRoleName(namespace), func(rb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			rb.OwnerReferences = []metav1.OwnerReference{genOwnerReference(namespace)}
			rb.RoleRef = rbacv1.RoleRef{
				Name:     roleName,
				Kind:     "ClusterRole",
				APIGroup: rbacv1.GroupName,
			}
			rb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      serviceAccountName,
					Namespace: namespace.Name,
				},
			}
			return rb, nil
		}
	}
}

func genOwnerReference(namespace *corev1.Namespace) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: namespace.APIVersion,
		Kind:       namespace.Kind,
		Name:       namespace.Name,
		UID:        namespace.UID,
	}
}

func GenerateRoleName(targetNamespace *corev1.Namespace) string {
	return fmt.Sprintf("%s-%s", roleBindingName, targetNamespace.Name)
}
