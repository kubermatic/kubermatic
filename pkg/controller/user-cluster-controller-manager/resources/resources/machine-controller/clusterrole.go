package machinecontroller

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	Name = "machine-controller"
)

// ClusterRole returns a cluster role for the machine controller (user-cluster)
func ClusterRoleCreator() reconciling.NamedClusterRoleCreatorGetter {
	return func() (string, reconciling.ClusterRoleCreator) {
		return resources.MachineControllerClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = resources.BaseAppLabels(Name, nil)

			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"apiextensions.k8s.io"},
					Resources: []string{"customresourcedefinitions"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups:     []string{"apiextensions.k8s.io"},
					Resources:     []string{"customresourcedefinitions"},
					ResourceNames: []string{"machines.machine.k8s.io"},
					Verbs:         []string{"*"},
				},
				{
					APIGroups: []string{"machine.k8s.io"},
					Resources: []string{"machines"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"list", "get"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"persistentvolumes", "secrets", "configmaps"},
					Verbs:     []string{"list", "get", "watch"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"pods/eviction"},
					Verbs:     []string{"create"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"events"},
					Verbs:     []string{"create", "patch"},
				},
				{
					APIGroups: []string{"cluster.k8s.io"},
					Resources: []string{"machines", "machines/finalizers",
						"machinesets", "machinesets/status", "machinesets/finalizers",
						"machinedeployments", "machinedeployments/status", "machinedeployments/finalizers",
						"clusters", "clusters/status", "clusters/finalizers"},
					Verbs: []string{"*"},
				},
			}
			return cr, nil
		}
	}
}
