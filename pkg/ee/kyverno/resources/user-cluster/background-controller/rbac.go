//go:build ee

package resources

import (
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	clusterRoleBackgroundControllerName               = "kyverno:background-controller"
	clusterRoleBackgroundControllerCoreName           = "kyverno:background-controller:core"
	clusterRoleBackgroundControllerViewName           = "kyverno:background-controller:view"
	clusterRoleBackgroundControllerServiceAccountName = "kyverno-background-controller"
)

// ClusterRoleReconciler returns the function to create and update the Kyverno background controller aggregated ClusterRole in the user cluster.
func ClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return clusterRoleBackgroundControllerName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component": "background-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.13.2",
			}

			cr.AggregationRule = &rbacv1.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{
						MatchLabels: map[string]string{
							"rbac.kyverno.io/aggregate-to-background-controller": "true",
						},
					},
					{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component": "background-controller",
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

// CoreClusterRoleReconciler returns the function to create and update the Kyverno background controller core ClusterRole in the user cluster.
func CoreClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return clusterRoleBackgroundControllerCoreName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component": "background-controller",
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
					APIGroups: []string{"kyverno.io"},
					Resources: []string{
						"policies",
						"policies/status",
						"clusterpolicies",
						"clusterpolicies/status",
						"policyexceptions",
						"updaterequests",
						"updaterequests/status",
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
					APIGroups: []string{""},
					Resources: []string{
						"namespaces",
						"configmaps",
					},
					Verbs: []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"", "events.k8s.io"},
					Resources: []string{"events"},
					Verbs: []string{
						"create",
						"get",
						"list",
						"patch",
						"update",
						"watch",
					},
				},
				{
					APIGroups: []string{"reports.kyverno.io"},
					Resources: []string{
						"ephemeralreports",
						"clusterephemeralreports",
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
					APIGroups: []string{"networking.k8s.io"},
					Resources: []string{
						"ingresses",
						"ingressclasses",
						"networkpolicies",
					},
					Verbs: []string{
						"create",
						"update",
						"patch",
						"delete",
					},
				},
				{
					APIGroups: []string{"rbac.authorization.k8s.io"},
					Resources: []string{
						"rolebindings",
						"roles",
					},
					Verbs: []string{
						"create",
						"update",
						"patch",
						"delete",
					},
				},
				{
					APIGroups: []string{""},
					Resources: []string{
						"configmaps",
						"resourcequotas",
						"limitranges",
					},
					Verbs: []string{
						"create",
						"update",
						"patch",
						"delete",
					},
				},
			}
			return cr, nil
		}
	}
}

// ClusterRoleBindingReconciler returns the function to create and update the Kyverno background controller ClusterRoleBinding in the user cluster.
func ClusterRoleBindingReconciler(seedNamespace string) reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return clusterRoleBackgroundControllerName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = map[string]string{
				"app.kubernetes.io/component": "background-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.13.2",
			}

			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     clusterRoleBackgroundControllerName,
			}
			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      clusterRoleBackgroundControllerServiceAccountName,
					Namespace: seedNamespace,
				},
			}
			return crb, nil
		}
	}
}

// ViewClusterRoleBindingReconciler returns the function to create and update the Kyverno background controller view ClusterRoleBinding in the user cluster.
func ViewClusterRoleBindingReconciler(seedNamespace string) reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return clusterRoleBackgroundControllerViewName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = map[string]string{
				"app.kubernetes.io/component": "background-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.13.2",
			}

			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     "view",
			}
			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      clusterRoleBackgroundControllerServiceAccountName,
					Namespace: seedNamespace,
				},
			}
			return crb, nil
		}
	}
}
