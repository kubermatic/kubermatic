package resources

import (
	"context"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourcesForDeletion returns a list of resources that should be deleted when the Kyverno background controller is removed.
func ResourcesForDeletion(cluster *kubermaticv1.Cluster) []ctrlruntimeclient.Object {
	return []ctrlruntimeclient.Object{
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kyverno-background-controller",
				Namespace: cluster.Status.NamespaceName,
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kyverno-background-controller",
				Namespace: cluster.Status.NamespaceName,
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kyverno-background-controller-metrics",
				Namespace: cluster.Status.NamespaceName,
			},
		},
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kyverno:background-controller",
				Namespace: cluster.Status.NamespaceName,
			},
		},
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kyverno:background-controller",
				Namespace: cluster.Status.NamespaceName,
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kyverno-background-controller",
				Namespace: cluster.Status.NamespaceName,
			},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kyverno:background-controller",
			},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kyverno:background-controller:core",
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kyverno:background-controller",
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kyverno:background-controller:view",
			},
		},
	}
}

// CleanUpResources deletes all resources created for the Kyverno background controller.
func CleanUpResources(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	resources := ResourcesForDeletion(cluster)
	for _, resource := range resources {
		if err := client.Delete(ctx, resource); err != nil {
			return err
		}
	}
	return nil
}
