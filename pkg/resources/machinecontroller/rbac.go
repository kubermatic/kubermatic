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

package machinecontroller

import (
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	serviceAccountName        = "kubermatic-machine-controller"
	webhookServiceAccountName = "kubermatic-machine-controller-webhook"
	webhookRoleName           = "kubermatic:machine-controller-webhook"
	webhookRoleBindingName    = "kubermatic:machine-controller-webhook"
)

func ServiceAccountReconciler() (string, reconciling.ServiceAccountReconciler) {
	return serviceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
		return sa, nil
	}
}

func WebhookServiceAccountReconciler() (string, reconciling.ServiceAccountReconciler) {
	return webhookServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
		return sa, nil
	}
}
func WebhookRoleReconciler() (string, reconciling.RoleReconciler) {
	return webhookRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
		r.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{"operatingsystemmanager.k8c.io"},
				Resources: []string{"operatingsystemprofiles"},
				Verbs: []string{
					"get",
					"list",
				},
			},
		}
		return r, nil
	}
}

func WebhookRoleBindingReconciler() (string, reconciling.RoleBindingReconciler) {
	return webhookRoleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
		rb.RoleRef = rbacv1.RoleRef{
			Name:     webhookRoleName,
			Kind:     "Role",
			APIGroup: rbacv1.GroupName,
		}
		rb.Subjects = []rbacv1.Subject{
			{
				Kind: rbacv1.ServiceAccountKind,
				Name: webhookServiceAccountName,
			},
		}
		return rb, nil
	}
}
