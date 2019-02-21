package rbacusercluster

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	clusterPolicyAPIGroup = "cluster.k8s.io"

	machinedeployments = "machinedeployments"
	machines           = "machines"

	resourceNameIndex = 2
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

// generateRBACClusterRole creates role for specific group
func generateRBACClusterRole(resourceName string) (*rbacv1.ClusterRole, error) {

	groupName, err := getGroupName(resourceName)
	if err != nil {
		return nil, err
	}
	verbs, err := generateVerbsForGroup(groupName)
	if err != nil {
		return nil, err
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: resourceName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{clusterPolicyAPIGroup},
				Resources: []string{machinedeployments, machines},
				Verbs:     verbs,
			},
		},
	}
	return clusterRole, nil
}

// generateRBACClusterRoleBinding creates role binding for specific group
func generateRBACClusterRoleBinding(resourceName string) (*rbacv1.ClusterRoleBinding, error) {
	groupName, err := getGroupName(resourceName)
	if err != nil {
		return nil, err
	}
	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: resourceName,
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
			Name:     resourceName,
		},
	}
	return binding, nil
}

func getGroupName(resourceName string) (string, error) {
	match, err := regexp.MatchString("system:kubermatic:[owners|editors|vievers]", resourceName)
	if err != nil {
		return "", err
	}
	if match {
		parts := strings.Split(resourceName, ":")
		return parts[resourceNameIndex], nil
	}
	return "", errors.New("can't get group name from resource name")
}
