package kubermatic

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	serviceAccountName                  = "kubermatic-seed"
	seedControllerManagerDeploymentName = "kubermatic-seed-controller-manager"
	openIDAuthFeatureFlag               = "OpenIDAuthPlugin"
	nameLabel                           = "app.kubernetes.io/name"
)

func clusterRoleBindingName(cfg *operatorv1alpha1.KubermaticConfiguration) string {
	return fmt.Sprintf("%s:%s-seed:cluster-admin", cfg.Namespace, cfg.Name)
}

func ServiceAccountCreator(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed) reconciling.NamedServiceAccountCreatorGetter {
	return func() (string, reconciling.ServiceAccountCreator) {
		return serviceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			return sa, nil
		}
	}
}

func ClusterRoleBindingCreator(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed) reconciling.NamedClusterRoleBindingCreatorGetter {
	name := clusterRoleBindingName(cfg)

	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return name, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
			}

			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      serviceAccountName,
					Namespace: cfg.Namespace,
				},
			}

			return crb, nil
		}
	}
}
