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

package rbacusercluster

import (
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

// ClusterRoleMatches compares cluster role Rules
func ClusterRoleMatches(existingRole, requestedRole *rbacv1.ClusterRole) bool {
	return equality.Semantic.DeepEqual(existingRole.Rules, requestedRole.Rules)
}

// ClusterRoleBindingMatches checks if cluster role bindings have the same Subjects and RoleRefs
func ClusterRoleBindingMatches(existingClusterRoleBinding, requestedClusterRoleBinding *rbacv1.ClusterRoleBinding) bool {
	if !equality.Semantic.DeepEqual(existingClusterRoleBinding.Subjects, requestedClusterRoleBinding.Subjects) {
		return false
	}
	if !equality.Semantic.DeepEqual(existingClusterRoleBinding.RoleRef, requestedClusterRoleBinding.RoleRef) {
		return false
	}

	return true
}
