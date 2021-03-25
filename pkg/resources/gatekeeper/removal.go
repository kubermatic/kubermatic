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

package gatekeeper

import (
	"k8c.io/kubermatic/v2/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetResourcesToRemoveOnDelete(namespace string) []ctrlruntimeclient.Object {
	var toRemove []ctrlruntimeclient.Object

	// Deployment
	toRemove = append(toRemove, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.GatekeeperControllerDeploymentName,
			Namespace: namespace,
		},
	})
	toRemove = append(toRemove, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.GatekeeperAuditDeploymentName,
			Namespace: namespace,
		},
	})
	// Service
	toRemove = append(toRemove, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.GatekeeperWebhookServiceName,
			Namespace: namespace,
		},
	})
	// RBACs
	toRemove = append(toRemove, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.GatekeeperServiceAccountName,
			Namespace: namespace,
		},
	})
	toRemove = append(toRemove, &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.GatekeeperRoleName,
			Namespace: namespace,
		},
	})
	toRemove = append(toRemove, &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.GatekeeperRoleBindingName,
			Namespace: namespace,
		},
	})

	return toRemove
}
