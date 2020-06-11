package resources

import (
	"github.com/kubermatic/kubermatic/pkg/resources"
	"github.com/kubermatic/kubermatic/pkg/resources/reconciling"

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
