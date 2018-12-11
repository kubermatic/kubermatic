package metricsserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RolebindingAuthReader returns the RoleBinding used by the metrics-server to get access to the token subject review API
func RolebindingAuthReader(_ resources.RoleBindingDataProvider, existing *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	var rb *rbacv1.RoleBinding
	if existing != nil {
		rb = existing
	} else {
		rb = &rbacv1.RoleBinding{}
	}

	rb.Name = resources.MetricsServerAuthReaderRoleName
	rb.Namespace = metav1.NamespaceSystem
	rb.Labels = resources.BaseAppLabel(name, nil)

	rb.RoleRef = rbacv1.RoleRef{
		Name:     "extension-apiserver-authentication-reader",
		Kind:     "Role",
		APIGroup: rbacv1.GroupName,
	}
	rb.Subjects = []rbacv1.Subject{
		{
			Kind:     "User",
			Name:     resources.MetricsServerCertUsername,
			APIGroup: rbacv1.GroupName,
		},
	}
	return rb, nil
}
