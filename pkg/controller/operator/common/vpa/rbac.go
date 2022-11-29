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

package vpa

import (
	"k8c.io/reconciler/pkg/reconciling"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	MetricsReaderRoleName       = "system:vpa-metrics-reader"
	TargetReaderRoleName        = "system:vpa-target-reader"
	ActorRoleName               = "system:vpa-actor"
	CheckpointActorRoleName     = "system:vpa-checkpoint-actor"
	EvictionerRoleName          = "system:vpa-evictioner"
	AdmissionControllerRoleName = "system:vpa-admission-controller"
	StatusReaderRoleName        = "system:vpa-status-reader"
)

func ClusterRoleReconcilers() []reconciling.NamedClusterRoleReconcilerFactory {
	return []reconciling.NamedClusterRoleReconcilerFactory{
		clusterRoleReconciler(MetricsReaderRoleName, []rbacv1.PolicyRule{
			{
				APIGroups: []string{"metrics.k8s.io"},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list"},
			},
		}),

		clusterRoleReconciler(TargetReaderRoleName, []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*/scale"},
				Verbs:     []string{"get", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"replicationcontrollers"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"daemonsets", "deployments", "replicasets", "statefulsets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"batch"},
				Resources: []string{"jobs", "cronjobs"},
				Verbs:     []string{"get", "list", "watch"},
			},
		}),

		clusterRoleReconciler(ActorRoleName, []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "nodes", "limitranges"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"get", "list", "watch", "create"},
			},
			{
				APIGroups: []string{"poc.autoscaling.k8s.io"},
				Resources: []string{"verticalpodautoscalers"},
				Verbs:     []string{"get", "list", "watch", "patch"},
			},
			{
				APIGroups: []string{"autoscaling.k8s.io"},
				Resources: []string{"verticalpodautoscalers"},
				Verbs:     []string{"get", "list", "watch", "patch"},
			},
		}),

		clusterRoleReconciler(CheckpointActorRoleName, []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
				Verbs:     []string{"get", "list"},
			},
			{
				APIGroups: []string{"poc.autoscaling.k8s.io"},
				Resources: []string{"verticalpodautoscalercheckpoints"},
				Verbs:     []string{"get", "list", "watch", "create", "patch", "delete"},
			},
			{
				APIGroups: []string{"autoscaling.k8s.io"},
				Resources: []string{"verticalpodautoscalercheckpoints"},
				Verbs:     []string{"get", "list", "watch", "create", "patch", "delete"},
			},
		}),

		clusterRoleReconciler(EvictionerRoleName, []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods/eviction"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups: []string{"apps", "extensions"},
				Resources: []string{"replicasets"},
				Verbs:     []string{"get"},
			},
		}),

		clusterRoleReconciler(AdmissionControllerRoleName, []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "configmaps", "nodes", "limitranges"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"admissionregistration.k8s.io"},
				Resources: []string{"mutatingwebhookconfigurations"},
				Verbs:     []string{"get", "list", "create", "delete"},
			},
			{
				APIGroups: []string{"poc.autoscaling.k8s.io"},
				Resources: []string{"verticalpodautoscalers"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"autoscaling.k8s.io"},
				Resources: []string{"verticalpodautoscalers"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"coordination.k8s.io"},
				Resources: []string{"leases"},
				Verbs:     []string{"create", "update", "get", "list", "watch"},
			},
		}),

		clusterRoleReconciler(StatusReaderRoleName, []rbacv1.PolicyRule{
			{
				APIGroups: []string{"coordination.k8s.io"},
				Resources: []string{"leases"},
				Verbs:     []string{"get", "list", "watch"},
			},
		}),
	}
}

func clusterRoleReconciler(name string, rules []rbacv1.PolicyRule) reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return name, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Rules = rules
			return cr, nil
		}
	}
}

func ClusterRoleBindingReconcilers() []reconciling.NamedClusterRoleBindingReconcilerFactory {
	recommender := rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      RecommenderName,
		Namespace: metav1.NamespaceSystem,
	}

	updater := rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      UpdaterName,
		Namespace: metav1.NamespaceSystem,
	}

	admissionController := rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      AdmissionControllerName,
		Namespace: metav1.NamespaceSystem,
	}

	return []reconciling.NamedClusterRoleBindingReconcilerFactory{
		clusterRoleBindingReconciler(MetricsReaderRoleName, rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     MetricsReaderRoleName,
		}, []rbacv1.Subject{
			recommender,
		}),

		clusterRoleBindingReconciler(TargetReaderRoleName, rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     TargetReaderRoleName,
		}, []rbacv1.Subject{
			recommender,
			updater,
			admissionController,
		}),

		clusterRoleBindingReconciler(ActorRoleName, rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     ActorRoleName,
		}, []rbacv1.Subject{
			recommender,
			updater,
		}),

		clusterRoleBindingReconciler(CheckpointActorRoleName, rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     CheckpointActorRoleName,
		}, []rbacv1.Subject{
			recommender,
		}),

		clusterRoleBindingReconciler(EvictionerRoleName, rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     EvictionerRoleName,
		}, []rbacv1.Subject{
			updater,
		}),

		clusterRoleBindingReconciler(AdmissionControllerRoleName, rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     AdmissionControllerRoleName,
		}, []rbacv1.Subject{
			admissionController,
		}),

		clusterRoleBindingReconciler(StatusReaderRoleName, rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     StatusReaderRoleName,
		}, []rbacv1.Subject{
			updater,
		}),
	}
}

func clusterRoleBindingReconciler(name string, roleRef rbacv1.RoleRef, subjects []rbacv1.Subject) reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return name, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.RoleRef = roleRef
			crb.Subjects = subjects
			return crb, nil
		}
	}
}
