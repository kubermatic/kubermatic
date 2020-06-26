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

package resources

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	machineControllerRoleName        = "machine-controller"
	machineControllerRoleBindingName = "machine-controller"
	openshiftInfraNamespaceName      = "openshift-infra"
)

func MachineControllerRole() (types.NamespacedName, reconciling.RoleCreator) {
	return types.NamespacedName{Namespace: openshiftInfraNamespaceName, Name: machineControllerRoleName},
		func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups:     []string{""},
					Resources:     []string{"serviceaccounts"},
					ResourceNames: []string{"node-bootstrapper"},
					Verbs:         []string{"get"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"secrets"},
					Verbs:     []string{"get"},
				},
			}
			return r, nil
		}
}

func MachineControllerRoleBinding() (types.NamespacedName, reconciling.RoleBindingCreator) {
	return types.NamespacedName{Namespace: openshiftInfraNamespaceName, Name: machineControllerRoleBindingName},
		func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.RoleRef = rbacv1.RoleRef{
				Kind:     "Role",
				APIGroup: rbacv1.GroupName,
				Name:     machineControllerRoleName,
			}
			rb.Subjects = []rbacv1.Subject{
				{
					Kind:     "User",
					Name:     resources.MachineControllerCertUsername,
					APIGroup: rbacv1.GroupName,
				},
			}
			return rb, nil
		}
}
