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

package machinecontroller

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRoleBinding returns a ClusterRoleBinding for the machine-controller.
func ClusterRoleBindingCreator() reconciling.NamedClusterRoleBindingCreatorGetter {
	return createClusterRoleBindingCreator("controller",
		resources.MachineControllerClusterRoleName, rbacv1.Subject{
			Kind:     "User",
			Name:     resources.MachineControllerCertUsername,
			APIGroup: rbacv1.GroupName,
		})
}

// NodeBootstrapperClusterRoleBinding returns a ClusterRoleBinding for the machine-controller.
func NodeBootstrapperClusterRoleBindingCreator() reconciling.NamedClusterRoleBindingCreatorGetter {
	return createClusterRoleBindingCreator("kubelet-bootstrap",
		"system:node-bootstrapper", rbacv1.Subject{
			Kind:     "Group",
			Name:     "system:bootstrappers:machine-controller:default-node-token",
			APIGroup: rbacv1.GroupName,
		})
}

// NodeSignerClusterRoleBindingCreator returns a ClusterRoleBinding for the machine-controller.
func NodeSignerClusterRoleBindingCreator() reconciling.NamedClusterRoleBindingCreatorGetter {
	return createClusterRoleBindingCreator("node-signer",
		"system:certificates.k8s.io:certificatesigningrequests:nodeclient", rbacv1.Subject{
			Kind:     "Group",
			Name:     "system:bootstrappers:machine-controller:default-node-token",
			APIGroup: rbacv1.GroupName,
		})
}

func createClusterRoleBindingCreator(crbSuffix, cRoleRef string, subj rbacv1.Subject) reconciling.NamedClusterRoleBindingCreatorGetter {
	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return fmt.Sprintf("%s:%s", resources.MachineControllerClusterRoleBindingName, crbSuffix), func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = resources.BaseAppLabels(Name, nil)

			crb.RoleRef = rbacv1.RoleRef{
				Name:     cRoleRef,
				Kind:     "ClusterRole",
				APIGroup: rbacv1.GroupName,
			}
			crb.Subjects = []rbacv1.Subject{subj}
			return crb, nil
		}
	}
}
