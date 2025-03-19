package resources

import (
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

// RoleReconciler returns the function to create and update the Kyverno cleanup controller role.
func RoleReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedRoleReconcilerFactory {
	return func() (string, reconciling.RoleReconciler) {
		return "kyverno:cleanup-controller", func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Labels = map[string]string{
				"app.kubernetes.io/component": "cleanup-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.13.2",
			}

			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"secrets"},
					Verbs:     []string{"create"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"secrets"},
					Verbs:     []string{"delete", "get", "list", "update", "watch"},
					ResourceNames: []string{
						"kyverno-cleanup-controller.kyverno.svc.kyverno-tls-ca",
						"kyverno-cleanup-controller.kyverno.svc.kyverno-tls-pair",
					},
				},
				{
					APIGroups:     []string{""},
					Resources:     []string{"configmaps"},
					Verbs:         []string{"get", "list", "watch"},
					ResourceNames: []string{"kyverno", "kyverno-metrics"},
				},
				{
					APIGroups: []string{"coordination.k8s.io"},
					Resources: []string{"leases"},
					Verbs:     []string{"create"},
				},
				{
					APIGroups:     []string{"coordination.k8s.io"},
					Resources:     []string{"leases"},
					Verbs:         []string{"delete", "get", "patch", "update"},
					ResourceNames: []string{"kyverno-cleanup-controller"},
				},
				{
					APIGroups: []string{"apps"},
					Resources: []string{"deployments"},
					Verbs:     []string{"get", "list", "watch"},
				},
			}

			return r, nil
		}
	}
}

// RoleBindingReconciler returns the function to create and update the Kyverno cleanup controller role binding.
func RoleBindingReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
		return "kyverno:cleanup-controller", func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Labels = map[string]string{
				"app.kubernetes.io/component": "cleanup-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.13.2",
			}

			rb.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     "kyverno:cleanup-controller",
			}

			rb.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      name,
					Namespace: cluster.Status.NamespaceName,
				},
			}

			return rb, nil
		}
	}
}
