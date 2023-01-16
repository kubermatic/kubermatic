package image

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/reconciler/pkg/reconciling"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const RBACName = "image-cloner"

func imagesClusterRoleCreator(name string) reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return name, func(r *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"cdi.kubevirt.io"},
					Resources: []string{"datavolumes/source"},
					Verbs:     []string{"*"},
				},
			}
			return r, nil
		}
	}
}

func imagesRoleBindingCreator(name, namespace string) reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
		return name, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "default",
					Namespace: namespace,
				},
			}

			rb.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     RBACName,
			}

			return rb, nil
		}
	}
}

func ReconcileImagesRoleBinding(ctx context.Context, sourceNamespace, destinationNamespace string, cluster *kubermaticv1.Cluster,
	update provider.ClusterUpdater, client ctrlruntimeclient.Client, finalizers []string) (*kubermaticv1.Cluster, error) {
	cluster, err := update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		kuberneteshelper.AddFinalizer(updatedCluster, finalizers...)
	})
	if err != nil {
		return cluster, err
	}
	clusterRoleCreator := []reconciling.NamedClusterRoleReconcilerFactory{
		imagesClusterRoleCreator(RBACName),
	}

	if err = reconciling.ReconcileClusterRoles(ctx, clusterRoleCreator, "", client); err != nil {
		return cluster, err
	}

	roleBindingCreators := []reconciling.NamedRoleBindingReconcilerFactory{
		imagesRoleBindingCreator(fmt.Sprintf("%s-%s", RBACName, destinationNamespace), destinationNamespace),
	}
	if err = reconciling.ReconcileRoleBindings(ctx, roleBindingCreators, sourceNamespace, client); err != nil {
		return cluster, err
	}

	return cluster, nil
}

func DeleteImagesRoleBinding(ctx context.Context, name string, client ctrlruntimeclient.Client) error {
	roleBinding := &rbacv1.RoleBinding{}
	// TODO: doesn't this have to be
	if err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: ImagesNamespace}, roleBinding); err != nil {
		return ctrlruntimeclient.IgnoreNotFound(err)
	}
	return client.Delete(ctx, roleBinding)
}
