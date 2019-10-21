package systembasicuser

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRoleBinding is needed to address CVE-2019-11253 for clusters that were
// created with a Kubernetes version < 1.14, to remove permissions from
// unauthenticated users to post data to the API and cause a DOS.
// For details, see https://github.com/kubernetes/kubernetes/issues/83253
func ClusterRoleBinding() (string, reconciling.ClusterRoleBindingCreator) {
	return "system:basic-user", func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
		crb.Subjects = []rbacv1.Subject{{
			APIGroup: rbacv1.GroupName,
			Name:     "system:authenticated",
			Kind:     rbacv1.GroupKind,
		}}

		return crb, nil
	}
}
