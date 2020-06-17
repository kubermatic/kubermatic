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

package systembasicuser

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRoleBinding is needed to address CVE-2019-11253 for clusters that were
// created with a Kubernetes version < 1.14, to remove permissions from
// unauthenticated users to post data to the API and cause a DOS.
// For details, see https://github.com/kubernetes/kubernetes/issues/83253
func ClusterRoleBinding() (string, reconciling.ClusterRoleBindingCreator) {
	return "system:basic-user", func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
		crb.Subjects = []rbacv1.Subject{{
			APIGroup: rbacv1.GroupName,
			Name:     "system:authenticated",
			Kind:     rbacv1.GroupKind,
		}}

		return crb, nil
	}
}
