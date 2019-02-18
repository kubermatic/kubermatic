package rbacusercluster

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	clusterPolicyAPIGroup = "cluster.k8s.io"

	machinedeployments = "machinedeployments"
	machines           = "machines"
	nodes              = "nodes"
)

// generateVerbsForGroup generates a set of verbs for a group
func generateVerbsForGroup(groupName string) ([]string, error) {
	// verbs for owners or editors
	if groupName == rbac.OwnerGroupNamePrefix || groupName == rbac.EditorGroupNamePrefix {
		return []string{"create", "list", "get", "update", "delete"}, nil
	}

	if groupName == rbac.ViewerGroupNamePrefix {
		return []string{"list", "get"}, nil
	}

	// unknown group passed
	return []string{}, fmt.Errorf("unable to generate verbs, unknown group name passed in = %s", groupName)
}

// GenerateRBACClusterRole creates role for specific group
func GenerateRBACClusterRole(groupName string) (*rbacv1.ClusterRole, error) {
	verbs, err := generateVerbsForGroup(groupName)
	if err != nil {
		return nil, err
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s:%s", rbac.RBACResourcesNamePrefix, groupName),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{clusterPolicyAPIGroup},
				Resources: []string{machinedeployments, machines},
				Verbs:     verbs,
			},
			{
				APIGroups: []string{""},
				Resources: []string{nodes},
				Verbs:     verbs,
			},
		},
	}
	return clusterRole, nil
}

// GenerateRBACClusterRoleBinding creates role binding for specific group
func GenerateRBACClusterRoleBinding(groupName string) *rbacv1.ClusterRoleBinding {
	name := fmt.Sprintf("%s:%s", rbac.RBACResourcesNamePrefix, groupName)
	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     "Group",
				Name:     groupName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     name,
		},
	}
	return binding
}
