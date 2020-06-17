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

package openshift

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	roleName = "system:openshift:sa-leader-election-configmaps"
	// TokenOwnerServiceAccountName is the name of the ServiceAccount used to back the
	// admin kubeconfig our API hands out
	TokenOwnerServiceAccountName        = "cluster-admin"
	tokenOwnerServiceAccountBindingName = "cluster-admin-serviceaccount"
)

// KubeSystemRoleCreator returns the func to create/update the Role for the machine controller to allow reading secrets
func KubeSchedulerRoleCreatorGetter() (string, reconciling.RoleCreator) {
	return roleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
		r.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs: []string{
					"get",
					"create",
					"update",
				},
			},
		}
		return r, nil
	}
}

func KubeSchedulerRoleBindingCreatorGetter() (string, reconciling.RoleBindingCreator) {
	return resources.MachineControllerRoleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
		rb.RoleRef = rbacv1.RoleRef{
			Name:     roleName,
			Kind:     "Role",
			APIGroup: rbacv1.GroupName,
		}
		rb.Subjects = []rbacv1.Subject{
			{
				Kind:     rbacv1.UserKind,
				Name:     resources.SchedulerCertUsername,
				APIGroup: rbacv1.GroupName,
			},
		}
		return rb, nil
	}
}

// TokenOwnerServiceAccount is the ServiceAccount that owns the secret which we put onto the
// kubeconfig that is in the seed
func TokenOwnerServiceAccount() (string, reconciling.ServiceAccountCreator) {
	return TokenOwnerServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
		return sa, nil
	}
}

// TokenOwnerServiceAccountClusterRoleBinding is the clusterrolebinding that gives the TokenOwnerServiceAccount
// admin powers
func TokenOwnerServiceAccountClusterRoleBinding() (string, reconciling.ClusterRoleBindingCreator) {
	return tokenOwnerServiceAccountBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
		crb.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      TokenOwnerServiceAccountName,
				Namespace: metav1.NamespaceSystem,
			},
		}
		crb.RoleRef = rbacv1.RoleRef{
			Name:     "cluster-admin",
			Kind:     "ClusterRole",
			APIGroup: rbacv1.GroupName,
		}
		return crb, nil
	}
}
