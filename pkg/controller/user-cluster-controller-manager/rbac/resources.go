/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rbacusercluster

import (
	"fmt"
	"regexp"

	constrainttemplatesv1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1"
	configv1alpha1 "github.com/open-policy-agent/gatekeeper/v3/apis/config/v1alpha1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	clusterPolicyAPIGroup = "cluster.k8s.io"
)

// generateVerbsForGroup generates a set of verbs for a group.
func generateVerbsForGroup(groupName string) ([]string, error) {
	// verbs for owners
	if groupName == rbac.OwnerGroupNamePrefix || groupName == rbac.EditorGroupNamePrefix {
		return []string{"*"}, nil
	}

	if groupName == rbac.ViewerGroupNamePrefix {
		return []string{"list", "get", "watch"}, nil
	}

	// unknown group passed
	return []string{}, fmt.Errorf("unable to generate verbs, unknown group name %q given", groupName)
}

func newClusterRoleReconciler(resourceName string) (reconciling.NamedClusterRoleReconcilerFactory, error) {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return resourceName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			return CreateClusterRole(resourceName, cr)
		}
	}, nil
}

func CreateClusterRole(resourceName string, cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	groupName, err := getGroupName(resourceName)
	if err != nil {
		return nil, err
	}

	verbs, err := generateVerbsForGroup(groupName)
	if err != nil {
		return nil, err
	}

	cr.Name = resourceName // this is useful for our conformance tests, the reconciling framework would otherwise set it later
	cr.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{clusterPolicyAPIGroup},
			Resources: []string{"machinedeployments", "machinesets", "machines"},
			Verbs:     verbs,
		},
		{
			APIGroups: []string{appskubermaticv1.GroupName},
			Resources: []string{appskubermaticv1.ApplicationInstallationResourceName},
			Verbs:     verbs,
		},
		{
			APIGroups: []string{""},
			Resources: []string{
				"configmaps",
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
			Resources: []string{
				"bindings",
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
			APIGroups: []string{appsv1.GroupName},
			Resources: []string{
				"controllerrevisions",
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
			APIGroups: []string{autoscalingv1.GroupName},
			Resources: []string{"horizontalpodautoscalers"},
			Verbs:     verbs,
		},
		{
			APIGroups: []string{batchv1.GroupName},
			Resources: []string{"cronjobs", "jobs"},
			Verbs:     verbs,
		},
		{
			APIGroups: []string{networkingv1.GroupName},
			Resources: []string{"ingresses", "networkpolicies"},
			Verbs:     verbs,
		},
		{
			APIGroups: []string{constrainttemplatesv1.SchemeGroupVersion.Group},
			Resources: []string{"constrainttemplates"},
			Verbs:     verbs,
		},
		{
			APIGroups: []string{"constraints.gatekeeper.sh"},
			Resources: []string{"*"},
			Verbs:     verbs,
		},
		{
			APIGroups: []string{configv1alpha1.GroupVersion.Group},
			Resources: []string{"configs"},
			Verbs:     verbs,
		},
		{
			APIGroups: []string{velerov1.SchemeGroupVersion.Group},
			Resources: []string{"backups", "restores", "schedules"},
			Verbs:     verbs,
		},
	}

	if groupName == rbac.OwnerGroupNamePrefix || groupName == rbac.EditorGroupNamePrefix {
		cr.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     verbs,
			},
		}
	}

	return cr, nil
}

func newClusterRoleBindingReconciler(resourceName string) (reconciling.NamedClusterRoleBindingReconcilerFactory, error) {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return resourceName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			return CreateClusterRoleBinding(resourceName, crb)
		}
	}, nil
}

func CreateClusterRoleBinding(resourceName string, crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	groupName, err := getGroupName(resourceName)
	if err != nil {
		return nil, err
	}

	crb.Name = resourceName // this is useful for our conformance tests, the reconciling framework would otherwise set it later
	crb.Subjects = []rbacv1.Subject{
		{
			APIGroup: rbacv1.GroupName,
			Kind:     "Group",
			Name:     groupName,
		},
	}

	crb.RoleRef = rbacv1.RoleRef{
		APIGroup: rbacv1.GroupName,
		Kind:     "ClusterRole",
		Name:     resourceName,
	}

	return crb, nil
}

var groupNameRegex = regexp.MustCompile(fmt.Sprintf("system:%s:(%s|%s|%s)", rbac.RBACResourcesNamePrefix, rbac.OwnerGroupNamePrefix, rbac.EditorGroupNamePrefix, rbac.ViewerGroupNamePrefix))

func getGroupName(resourceName string) (string, error) {
	match := groupNameRegex.FindStringSubmatch(resourceName)
	if match != nil {
		return match[1], nil
	}
	return "", fmt.Errorf("can't get group name from resource name %q", resourceName)
}
