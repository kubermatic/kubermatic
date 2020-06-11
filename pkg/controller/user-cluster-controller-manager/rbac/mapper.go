package rbacusercluster

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/rbac"

	apps "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v1"
	batch "k8s.io/api/batch/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	clusterPolicyAPIGroup = "cluster.k8s.io"

	machinedeployments = "machinedeployments"
	machinesets        = "machinesets"
	machines           = "machines"

	resourceNameIndex = 2
)

// generateVerbsForGroup generates a set of verbs for a group
func generateVerbsForGroup(groupName string) ([]string, error) {
	// verbs for owners
	if groupName == rbac.OwnerGroupNamePrefix || groupName == rbac.EditorGroupNamePrefix {
		return []string{"*"}, nil
	}

	if groupName == rbac.ViewerGroupNamePrefix {
		return []string{"list", "get", "watch"}, nil
	}

	// unknown group passed
	return []string{}, fmt.Errorf("unable to generate verbs, unknown group name passed in = %s", groupName)
}

// GenerateRBACClusterRole creates role for specific group
func GenerateRBACClusterRole(resourceName string) (*rbacv1.ClusterRole, error) {

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
				Resources: []string{machinedeployments, machinesets, machines},
				Verbs:     verbs,
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps",
					"endpoints",
					"persistentvolumeclaims",
					"pods",
					"replicationcontrollers",
					"replicationcontrollers/scale",
					"serviceaccounts",
					"services",
					"nodes",
					"namespaces",
				},
				Verbs: verbs,
			},
			{
				APIGroups: []string{""},
				Resources: []string{"bindings",
					"events",
					"limitranges",
					"namespaces/status",
					"pods/log",
					"pods/status",
					"replicationcontrollers/status",
					"resourcequotas",
					"resourcequotas/status",
				},
				Verbs: verbs,
			},
			{
				APIGroups: []string{apps.GroupName},
				Resources: []string{"controllerrevisions",
					"daemonsets",
					"deployments",
					"deployments/scale",
					"replicasets",
					"replicasets/scale",
					"statefulsets",
					"statefulsets/scale",
				},
				Verbs: verbs,
			},
			{
				APIGroups: []string{autoscaling.GroupName},
				Resources: []string{"horizontalpodautoscalers"},
				Verbs:     verbs,
			},
			{
				APIGroups: []string{batch.GroupName},
				Resources: []string{"cronjobs", "jobs"},
				Verbs:     verbs,
			},
			{
				APIGroups: []string{extensions.GroupName},
				Resources: []string{"daemonsets",
					"deployments",
					"deployments/scale",
					"ingresses",
					"networkpolicies",
					"replicasets",
					"replicasets/scale",
					"replicationcontrollers/scale",
				},
				Verbs: verbs,
			},
			{
				APIGroups: []string{"networking.k8s.io"},
				Resources: []string{"ingresses", "networkpolicies"},
				Verbs:     verbs,
			},
		},
	}
	if groupName == rbac.OwnerGroupNamePrefix || groupName == rbac.EditorGroupNamePrefix {
		clusterRole.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     verbs,
			}}
	}
	return clusterRole, nil
}

// GenerateRBACClusterRoleBinding creates role binding for specific group
func GenerateRBACClusterRoleBinding(resourceName string) (*rbacv1.ClusterRoleBinding, error) {
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
	groupNamePattern := fmt.Sprintf("system:%s:[%s|%s|%s]", rbac.RBACResourcesNamePrefix, rbac.OwnerGroupNamePrefix, rbac.EditorGroupNamePrefix, rbac.ViewerGroupNamePrefix)
	match, err := regexp.MatchString(groupNamePattern, resourceName)
	if err != nil {
		return "", err
	}
	if match {
		parts := strings.Split(resourceName, ":")
		return parts[resourceNameIndex], nil
	}
	return "", errors.New("can't get group name from resource name")
}
