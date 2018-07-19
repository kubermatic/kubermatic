package prometheus

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RoleBinding returns the RoleBinding for the prometheus
func RoleBinding(data *resources.TemplateData, existing *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	var rb *rbacv1.RoleBinding
	if existing != nil {
		rb = existing
	} else {
		rb = &rbacv1.RoleBinding{}
	}

	rb.Name = resources.PrometheusRoleBindingName
	rb.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	rb.Labels = resources.GetLabels(name)

	rb.RoleRef = rbacv1.RoleRef{
		Name:     resources.PrometheusRoleName,
		Kind:     "Role",
		APIGroup: rbacv1.GroupName,
	}
	rb.Subjects = []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      resources.PrometheusServiceAccountName,
			Namespace: data.Cluster.Status.NamespaceName,
		},
	}
	return rb, nil
}
