//go:build ee

package resources

import (
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	clusterRoleCleanupControllerName               = "kyverno:cleanup-controller"
	clusterRoleCleanupControllerCoreName           = "kyverno:cleanup-controller:core"
	clusterRoleCleanupControllerServiceAccountName = "kyverno-cleanup-controller"
)

// ClusterRoleReconciler returns the function to create and update the Kyverno cleanup controller aggregated ClusterRole in the user cluster.
func ClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return clusterRoleCleanupControllerName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component": "cleanup-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.13.2",
			}

			cr.AggregationRule = &rbacv1.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{
						MatchLabels: map[string]string{
							"rbac.kyverno.io/aggregate-to-cleanup-controller": "true",
						},
					},
					{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component": "cleanup-controller",
							"app.kubernetes.io/instance":  "kyverno",
							"app.kubernetes.io/part-of":   "kyverno",
						},
					},
				},
			}

			return cr, nil
		}
	}
}

// CoreClusterRoleReconciler returns the function to create and update the Kyverno cleanup controller core ClusterRole in the user cluster.
func CoreClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return clusterRoleCleanupControllerCoreName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component": "cleanup-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.13.2",
			}

			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"apiextensions.k8s.io"},
					Resources: []string{"customresourcedefinitions"},
					Verbs:     []string{"get"},
				},
				{
					APIGroups: []string{"admissionregistration.k8s.io"},
					Resources: []string{"validatingwebhookconfigurations"},
					Verbs: []string{
						"create",
						"delete",
						"get",
						"list",
						"update",
						"watch",
					},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"namespaces"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"kyverno.io"},
					Resources: []string{
						"clustercleanuppolicies",
						"cleanuppolicies",
					},
					Verbs: []string{"list", "watch"},
				},
				{
					APIGroups: []string{"kyverno.io"},
					Resources: []string{
						"globalcontextentries",
						"globalcontextentries/status",
					},
					Verbs: []string{
						"create",
						"delete",
						"get",
						"list",
						"patch",
						"update",
						"watch",
						"deletecollection",
					},
				},
				{
					APIGroups: []string{"kyverno.io"},
					Resources: []string{
						"clustercleanuppolicies/status",
						"cleanuppolicies/status",
					},
					Verbs: []string{"update"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"", "events.k8s.io"},
					Resources: []string{"events"},
					Verbs:     []string{"create", "patch", "update"},
				},
				{
					APIGroups: []string{"authorization.k8s.io"},
					Resources: []string{"subjectaccessreviews"},
					Verbs:     []string{"create"},
				},
			}
			return cr, nil
		}
	}
}

// ClusterRoleBindingReconciler returns the function to create and update the Kyverno cleanup controller ClusterRoleBinding in the user cluster.
func ClusterRoleBindingReconciler(seedNamespace string) reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return clusterRoleCleanupControllerName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = map[string]string{
				"app.kubernetes.io/component": "cleanup-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.13.2",
			}

			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     clusterRoleCleanupControllerName,
			}
			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      clusterRoleCleanupControllerServiceAccountName,
					Namespace: seedNamespace,
				},
			}
			return crb, nil
		}
	}
}
