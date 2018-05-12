package etcdoperator

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterRoleBinding returns the ClusterRoleBinding for the etcd-operator
func ClusterRoleBinding(data *resources.TemplateData, existing *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	var rb *rbacv1.ClusterRoleBinding
	if existing != nil {
		rb = existing
	} else {
		rb = &rbacv1.ClusterRoleBinding{}
	}

	rb.Name = data.Cluster.Status.NamespaceName + "-etcd-operator"
	rb.Namespace = data.Cluster.Status.NamespaceName
	rb.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

	rb.RoleRef = rbacv1.RoleRef{
		Name:     "etcd-operator",
		Kind:     "ClusterRole",
		APIGroup: "rbac.authorization.k8s.io",
	}
	rb.Subjects = []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      resources.EtcdOperatorServiceAccountName,
			Namespace: data.Cluster.Status.NamespaceName,
		},
	}
	return rb, nil
}
