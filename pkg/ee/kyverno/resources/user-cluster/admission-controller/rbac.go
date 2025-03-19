//go:build ee

package resources

import (
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	clusterRoleAdmissionControllerName               = "kyverno:admission-controller"
	clusterRoleAdmissionControllerCoreName           = "kyverno:admission-controller:core"
	clusterRoleAdmissionControllerViewName           = "kyverno:admission-controller:view"
	clusterRoleAdmissionControllerServiceAccountName = "kyverno-admission-controller"
)

// ClusterRoleReconciler returns the function to create and update the Kyverno admission controller aggregated ClusterRole in the user cluster.
func ClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return clusterRoleAdmissionControllerName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component": "admission-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.13.2",
			}

			cr.AggregationRule = &rbacv1.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{
						MatchLabels: map[string]string{
							"rbac.kyverno.io/aggregate-to-admission-controller": "true",
						},
					},
					{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component": "admission-controller",
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

// CoreClusterRoleReconciler returns the function to create and update the Kyverno admission controller core ClusterRole in the user cluster.
func CoreClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return clusterRoleAdmissionControllerCoreName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component": "admission-controller",
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
					Resources: []string{
						"mutatingwebhookconfigurations",
						"validatingwebhookconfigurations",
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
					APIGroups: []string{"rbac.authorization.k8s.io"},
					Resources: []string{
						"roles",
						"clusterroles",
						"rolebindings",
						"clusterrolebindings",
					},
					Verbs: []string{"list", "watch"},
				},
				{
					APIGroups: []string{"kyverno.io"},
					Resources: []string{
						"policies",
						"policies/status",
						"clusterpolicies",
						"clusterpolicies/status",
						"updaterequests",
						"updaterequests/status",
						"globalcontextentries",
						"globalcontextentries/status",
						"policyexceptions",
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
					APIGroups: []string{"wgpolicyk8s.io"},
					Resources: []string{
						"policyreports",
						"policyreports/status",
						"clusterpolicyreports",
						"clusterpolicyreports/status",
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
					APIGroups: []string{"", "events.k8s.io"},
					Resources: []string{"events"},
					Verbs:     []string{"create", "update", "patch"},
				},
				{
					APIGroups: []string{"authorization.k8s.io"},
					Resources: []string{"subjectaccessreviews"},
					Verbs:     []string{"create"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{
						"configmaps",
						"namespaces",
					},
					Verbs: []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"coordination.k8s.io"},
					Resources: []string{"leases"},
					Verbs: []string{
						"create",
						"update",
						"patch",
						"get",
						"list",
						"watch",
					},
				},
			}
			return cr, nil
		}
	}
}

// ClusterRoleBindingReconciler returns the function to create and update the Kyverno admission controller ClusterRoleBinding in the user cluster.
func ClusterRoleBindingReconciler(seedNamespace string) reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return clusterRoleAdmissionControllerName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = map[string]string{
				"app.kubernetes.io/component": "admission-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.13.2",
			}

			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     clusterRoleAdmissionControllerName,
			}
			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      clusterRoleAdmissionControllerServiceAccountName,
					Namespace: seedNamespace,
				},
			}
			return crb, nil
		}
	}
}

// ViewClusterRoleBindingReconciler returns the function to create and update the Kyverno admission controller view ClusterRoleBinding in the user cluster.
func ViewClusterRoleBindingReconciler(seedNamespace string) reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return clusterRoleAdmissionControllerViewName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = map[string]string{
				"app.kubernetes.io/component": "admission-controller",
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
					Name:      clusterRoleAdmissionControllerServiceAccountName,
					Namespace: seedNamespace,
				},
			}
			return crb, nil
		}
	}
}
