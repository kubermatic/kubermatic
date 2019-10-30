package machinecontroller

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubeSystemRoleCreator returns the func to create/update the Role for the machine controller to allow reading secrets
func KubeSystemRoleCreator() reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return resources.MachineControllerRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Name = resources.MachineControllerRoleName
			r.Namespace = metav1.NamespaceSystem
			r.Labels = resources.BaseAppLabel(machinecontroller.Name, nil)

			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"secrets"},
					Verbs: []string{
						"create",
						"update",
						"list",
						"watch",
					},
				},
				{
					APIGroups:     []string{""},
					Resources:     []string{"endpoints"},
					ResourceNames: []string{"machine-controller"},
					Verbs:         []string{"*"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"endpoints"},
					Verbs:     []string{"create"},
				},
			}
			return r, nil
		}
	}
}

// EndpointReaderRoleCreator returns the func to create/update the Role for the machine controller to allow reading the kubernetes api endpoints
func EndpointReaderRoleCreator() reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return resources.MachineControllerRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Name = resources.MachineControllerRoleName
			r.Namespace = metav1.NamespaceDefault
			r.Labels = resources.BaseAppLabel(machinecontroller.Name, nil)

			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"endpoints"},
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

// ClusterInfoReaderRoleCreator returns the func to create/update the Role for the machine controller to allow
// the kubelet & kubeadm to read the cluster-info reading the cluster-info ConfigMap without authentication.
func ClusterInfoReaderRoleCreator() reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return resources.ClusterInfoReaderRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Name = resources.ClusterInfoReaderRoleName
			r.Namespace = metav1.NamespacePublic
			r.Labels = resources.BaseAppLabel(machinecontroller.Name, nil)

			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups:     []string{""},
					ResourceNames: []string{"cluster-info"},
					Resources:     []string{"configmaps"},
					Verbs:         []string{"get"},
				},
			}
			return r, nil
		}
	}
}

// KubePublicRoleCreator returns the func to create/update the Role for the machine controller to allow
// reading all configmaps in kube-public.
func KubePublicRoleCreator() reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return resources.MachineControllerRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Name = resources.MachineControllerRoleName
			r.Namespace = metav1.NamespacePublic
			r.Labels = resources.BaseAppLabel(machinecontroller.Name, nil)

			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps"},
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
