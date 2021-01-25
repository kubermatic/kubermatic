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

package rbac

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog"
)

const (
	EtcdLauncherServiceAccountName = "etcd-launcher"
)

// syncProjectResource generates RBAC Role and Binding for a resource that belongs to a project.
// in order to support multiple cluster this code doesn't retrieve the project from the kube-api server
// instead it assumes that all required information is stored in OwnerReferences or in Labels (for cluster resources)
//
// note:
// the project resources live only on master cluster and cluster resources are on master and seed clusters
// we cannot use OwnerReferences for cluster resources because they are on clusters that don't have corresponding
// project resource and will be automatically gc'ed
func (c *resourcesController) syncProjectResource(ctx context.Context, key client.ObjectKey) error {
	obj := c.objectType.DeepCopyObject()
	if err := c.client.Get(ctx, key, obj); err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	metaObject, err := meta.Accessor(obj)
	if err != nil {
		return fmt.Errorf("failed to get object meta: %+v", err)
	}

	gvk := obj.GetObjectKind().GroupVersionKind()
	rmapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	projectName := ""
	for _, owner := range metaObject.GetOwnerReferences() {
		if owner.APIVersion == kubermaticv1.SchemeGroupVersion.String() && owner.Kind == kubermaticv1.ProjectKindName &&
			len(owner.Name) > 0 && len(owner.UID) > 0 {
			projectName = owner.Name
			break
		}
	}
	if len(projectName) == 0 {
		projectName = metaObject.GetLabels()[kubermaticv1.ProjectIDLabelKey]
	}

	if len(projectName) == 0 {
		return fmt.Errorf("unable to find owning project for the object name = %s, gvr = %s", metaObject.GetName(), rmapping)
	}

	if len(metaObject.GetNamespace()) == 0 {
		if err := ensureClusterRBACRoleForNamedResource(ctx, c.client, projectName, rmapping.Resource.Resource, gvk.Kind, metaObject); err != nil {
			return fmt.Errorf("failed to sync RBAC ClusterRole for %s resource for %s cluster provider, due to = %v", rmapping, c.providerName, err)
		}
		if err := ensureClusterRBACRoleBindingForNamedResource(ctx, c.client, projectName, rmapping.Resource.Resource, gvk.Kind, metaObject); err != nil {
			return fmt.Errorf("failed to sync RBAC ClusterRoleBinding for %s resource for %s cluster provider, due to = %v", rmapping, c.providerName, err)
		}
		if gvk.Kind == kubermaticv1.ClusterKindName {
			if err := c.ensureRBACRoleForClusterAddons(ctx, projectName, metaObject); err != nil {
				return fmt.Errorf("failed to sync RBAC Role for %s resource for %s cluster provider in namespace %s, due to = %v", rmapping, c.providerName, metaObject.GetNamespace(), err)
			}
			if err := c.ensureRBACRoleBindingForClusterAddons(ctx, projectName, metaObject); err != nil {
				return fmt.Errorf("failed to sync RBAC RoleBinding for %s resource for %s cluster provider in namespace %s, due to = %v", rmapping, c.providerName, metaObject.GetNamespace(), err)
			}
			if err := ensureClusterRBACRoleBindingForEtcdLauncher(ctx, c.client, projectName, metaObject); err != nil {
				return fmt.Errorf("failed to sync RBAC ClusterRoleBinding for %s resource for %s cluster provider: %v", rmapping, c.providerName, err)
			}
		}

		return nil
	}

	err = c.ensureRBACRoleForNamedResource(ctx,
		projectName,
		rmapping.Resource.Resource,
		gvk,
		metaObject.GetNamespace(),
		metaObject)
	if err != nil {
		return fmt.Errorf("failed to sync RBAC Role for %s resource for %s cluster provider in namespace %s: %v", rmapping, c.providerName, metaObject.GetNamespace(), err)
	}

	err = c.ensureRBACRoleBindingForNamedResource(ctx,
		projectName,
		rmapping.Resource.Resource,
		gvk,
		metaObject.GetNamespace(),
		metaObject)
	if err != nil {
		return fmt.Errorf("failed to sync RBAC RoleBinding for %s resource for %s cluster provider in namespace %s: %v", rmapping, c.providerName, metaObject.GetNamespace(), err)
	}

	return nil
}

func ensureClusterRBACRoleForNamedResource(ctx context.Context, cli client.Client, projectName string, objectResource string, objectKind string, object metav1.Object) error {
	for _, groupPrefix := range AllGroupsPrefixes {
		skip, generatedRole, err := shouldSkipClusterRBACRoleBindingForNamedResource(projectName, objectResource, objectKind, groupPrefix, object)
		if err != nil {
			return err
		}
		if skip {
			klog.V(4).Infof("skipping ClusterRole generation for named resource for group \"%s\" and resource \"%s\"", groupPrefix, objectResource)
			continue
		}

		var sharedExistingRole rbacv1.ClusterRole
		key := client.ObjectKey{Name: generatedRole.Name}
		if err := cli.Get(ctx, key, &sharedExistingRole); err != nil {
			if kerrors.IsNotFound(err) {
				if err := cli.Create(ctx, generatedRole); err != nil {
					return err
				}
				continue
			}
			return err
		}

		if equality.Semantic.DeepEqual(sharedExistingRole.Rules, generatedRole.Rules) {
			continue
		}
		existingRole := sharedExistingRole.DeepCopy()
		existingRole.Rules = generatedRole.Rules

		if err := cli.Update(ctx, existingRole); err != nil {
			return err
		}
	}
	return nil
}

func ensureClusterRBACRoleBindingForNamedResource(ctx context.Context, cli client.Client, projectName string, objectResource string, objectKind string, object metav1.Object) error {
	for _, groupPrefix := range AllGroupsPrefixes {
		skip, _, err := shouldSkipClusterRBACRoleBindingForNamedResource(projectName, objectResource, objectKind, groupPrefix, object)
		if err != nil {
			return err
		}
		if skip {
			klog.V(4).Infof("skipping operation on ClusterRoleBinding because corresponding ClusterRole was not(will not be) created for group %q and %q resource for project %q", groupPrefix, objectResource, projectName)
			continue
		}

		generatedRoleBinding := generateClusterRBACRoleBindingNamedResource(
			objectKind,
			object.GetName(),
			GenerateActualGroupNameFor(projectName, groupPrefix),
			metav1.OwnerReference{
				APIVersion: kubermaticv1.SchemeGroupVersion.String(),
				Kind:       objectKind,
				UID:        object.GetUID(),
				Name:       object.GetName(),
			},
		)

		var sharedExistingRoleBinding rbacv1.ClusterRoleBinding
		key := client.ObjectKey{Name: generatedRoleBinding.Name}
		if err := cli.Get(ctx, key, &sharedExistingRoleBinding); err != nil {
			if kerrors.IsNotFound(err) {
				if err := cli.Create(ctx, generatedRoleBinding); err != nil {
					return err
				}
				continue
			}
			return err
		}

		if equality.Semantic.DeepEqual(sharedExistingRoleBinding.Subjects, generatedRoleBinding.Subjects) {
			continue
		}
		existingRoleBinding := sharedExistingRoleBinding.DeepCopy()
		existingRoleBinding.Subjects = generatedRoleBinding.Subjects
		if err := cli.Update(ctx, existingRoleBinding); err != nil {
			return err
		}
	}
	return nil
}

// shouldSkipClusterRBACRoleBindingForNamedResource will tell you if you should skip the generation of ClusterResource or not,
// because for some kinds we actually don't create ClusterRole
//
// note that this method returns generated role if is not meant to be skipped
func shouldSkipClusterRBACRoleBindingForNamedResource(projectName string, objectResource string, objectKind string, groupPrefix string, object metav1.Object) (bool, *rbacv1.ClusterRole, error) {
	generatedRole, err := generateClusterRBACRoleNamedResource(
		objectKind,
		GenerateActualGroupNameFor(projectName, groupPrefix),
		objectResource,
		kubermaticv1.SchemeGroupVersion.Group,
		object.GetName(),
		metav1.OwnerReference{
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			Kind:       objectKind,
			UID:        object.GetUID(),
			Name:       object.GetName(),
		},
	)

	if err != nil {
		return false, generatedRole, err
	}
	if generatedRole == nil {
		return true, nil, nil
	}
	return false, generatedRole, nil
}

func (c *resourcesController) ensureRBACRoleForNamedResource(ctx context.Context, projectName string, resourceName string, gvk schema.GroupVersionKind, namespace string, object metav1.Object) error {
	for _, groupPrefix := range AllGroupsPrefixes {
		skip, generatedRole, err := shouldSkipRBACRoleBindingForNamedResource(projectName, resourceName, gvk, groupPrefix, namespace, object)
		if err != nil {
			return err
		}
		if skip {
			klog.V(4).Infof("skipping Role generation for named resource for group %q and resource %q in namespace %q", groupPrefix, resourceName, namespace)
			continue
		}
		var sharedExistingRole rbacv1.Role
		key := client.ObjectKey{Name: generatedRole.Name}
		if err := c.client.Get(ctx, key, &sharedExistingRole); err != nil {
			if kerrors.IsNotFound(err) {
				if err := c.client.Create(ctx, generatedRole); err != nil {
					return nil
				}
				continue
			}
			return nil
		}

		// make sure that existing rbac role has appropriate rules/policies
		if equality.Semantic.DeepEqual(sharedExistingRole.Rules, generatedRole.Rules) {
			continue
		}
		existingRole := sharedExistingRole.DeepCopy()
		existingRole.Rules = generatedRole.Rules
		if err := c.client.Update(ctx, existingRole); err != nil {
			return err
		}
	}
	return nil
}

func (c *resourcesController) ensureRBACRoleBindingForNamedResource(ctx context.Context, projectName string, resourceName string, gvk schema.GroupVersionKind, namespace string, object metav1.Object) error {
	for _, groupPrefix := range AllGroupsPrefixes {

		skip, _, err := shouldSkipRBACRoleBindingForNamedResource(projectName, resourceName, gvk, groupPrefix, namespace, object)
		if err != nil {
			return err
		}
		if skip {
			klog.V(4).Infof("skipping operation on RoleBinding because corresponding Role was not(will not be) created for group %q and %q resource for project %q in namespace %q", groupPrefix, resourceName, projectName, namespace)
			continue
		}

		generatedRoleBinding := generateRBACRoleBindingNamedResource(
			gvk.Kind,
			object.GetName(),
			GenerateActualGroupNameFor(projectName, groupPrefix),
			namespace,
			metav1.OwnerReference{
				APIVersion: gvk.GroupVersion().String(),
				Kind:       gvk.Kind,
				UID:        object.GetUID(),
				Name:       object.GetName(),
			},
		)
		var sharedExistingRoleBinding rbacv1.RoleBinding
		key := client.ObjectKey{Name: generatedRoleBinding.Name}
		if err := c.client.Get(ctx, key, &sharedExistingRoleBinding); err != nil {
			if kerrors.IsNotFound(err) {
				if err := c.client.Create(ctx, generatedRoleBinding); err != nil {
					return nil
				}
				continue
			}
			return err
		}

		if equality.Semantic.DeepEqual(sharedExistingRoleBinding.Subjects, generatedRoleBinding.Subjects) {
			continue
		}
		existingRoleBinding := sharedExistingRoleBinding.DeepCopy()
		existingRoleBinding.Subjects = generatedRoleBinding.Subjects
		if err := c.client.Update(ctx, existingRoleBinding); err != nil {
			return err
		}
	}
	return nil
}

// shouldSkipRBACRoleBindingForNamedResource will tell you if you should skip the generation of ClusterResource or not,
// because for some kinds we actually don't create Role
//
// note that this method returns generated role if is not meant to be skipped
func shouldSkipRBACRoleBindingForNamedResource(projectName string, resourceName string, gvk schema.GroupVersionKind, groupPrefix string, namespace string, object metav1.Object) (bool, *rbacv1.Role, error) {
	generatedRole, err := generateRBACRoleNamedResource(
		gvk.Kind,
		GenerateActualGroupNameFor(projectName, groupPrefix),
		resourceName,
		gvk.Group,
		object.GetName(),
		namespace,
		metav1.OwnerReference{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
			UID:        object.GetUID(),
			Name:       object.GetName(),
		},
	)

	if err != nil {
		return false, generatedRole, err
	}
	if generatedRole == nil {
		return true, nil, nil
	}
	return false, generatedRole, nil
}

func (c *resourcesController) ensureRBACRoleForClusterAddons(ctx context.Context, projectName string, object metav1.Object) error {
	cluster, ok := object.(*kubermaticv1.Cluster)
	if !ok {
		return fmt.Errorf("ensureRBACRoleForClusterAddons called with non-cluster: %+v", object)
	}

	var roleList rbacv1.RoleList
	opts := &client.ListOptions{Namespace: cluster.Status.NamespaceName}
	if err := c.client.List(ctx, &roleList, opts); err != nil {
		return err
	}

	for _, groupPrefix := range AllGroupsPrefixes {
		skip, generatedRole, err := shouldSkipRBACRoleForClusterNamespaceResource(
			projectName,
			cluster,
			kubermaticv1.AddonResourceName,
			kubermaticv1.GroupName,
			kubermaticv1.AddonKindName,
			groupPrefix)
		if err != nil {
			return err
		}
		if skip {
			klog.V(4).Infof("skipping Role generation for cluster addons for group %q and cluster namespace %q", groupPrefix, cluster.Status.NamespaceName)
			continue
		}

		var sharedExistingRole rbacv1.Role
		key := client.ObjectKey{Name: generatedRole.Name, Namespace: cluster.Status.NamespaceName}
		if err := c.client.Get(ctx, key, &sharedExistingRole); err != nil {
			if kerrors.IsNotFound(err) {
				if err := c.client.Create(ctx, generatedRole); err != nil {
					return err
				}
				continue
			}
			return err
		}

		// make sure that existing rbac role has appropriate rules/policies

		if equality.Semantic.DeepEqual(sharedExistingRole.Rules, generatedRole.Rules) {
			continue
		}
		existingRole := sharedExistingRole.DeepCopy()
		existingRole.Rules = generatedRole.Rules
		if err := c.client.Update(ctx, existingRole); err != nil {
			return err
		}
	}
	return nil
}

func (c *resourcesController) ensureRBACRoleBindingForClusterAddons(ctx context.Context, projectName string, object metav1.Object) error {
	cluster, ok := object.(*kubermaticv1.Cluster)
	if !ok {
		return fmt.Errorf("ensureRBACRoleBindingForClusterAddons called with non-cluster: %+v", object)
	}

	for _, groupPrefix := range AllGroupsPrefixes {
		skip, _, err := shouldSkipRBACRoleForClusterNamespaceResource(
			projectName,
			cluster,
			kubermaticv1.AddonResourceName,
			kubermaticv1.GroupName,
			kubermaticv1.AddonKindName,
			groupPrefix)
		if err != nil {
			return err
		}
		if skip {
			klog.V(4).Infof("skipping RoleBinding generation for cluster addons for group %q and cluster namespace %q", groupPrefix, cluster.Status.NamespaceName)
			continue
		}

		generatedRoleBinding := generateRBACRoleBindingForClusterNamespaceResource(
			cluster,
			GenerateActualGroupNameFor(projectName, groupPrefix),
			kubermaticv1.AddonKindName,
		)

		var sharedExistingRoleBinding rbacv1.RoleBinding
		key := client.ObjectKey{Name: generatedRoleBinding.Name, Namespace: cluster.Status.NamespaceName}
		if err := c.client.Get(ctx, key, &sharedExistingRoleBinding); err != nil {
			if kerrors.IsNotFound(err) {
				if err := c.client.Create(ctx, generatedRoleBinding); err != nil {
					return err
				}
				continue
			}
			return err
		}

		// sharedExistingRoleBinding found
		if equality.Semantic.DeepEqual(sharedExistingRoleBinding.Subjects, generatedRoleBinding.Subjects) {
			continue
		}
		existingRoleBinding := sharedExistingRoleBinding.DeepCopy()
		existingRoleBinding.Subjects = generatedRoleBinding.Subjects
		if err := c.client.Update(ctx, existingRoleBinding); err != nil {
			return err
		}
	}
	return nil
}

// shouldSkipRBACRoleForClusterNamespaceResource will tell you if you should skip the generation of Role/Rolebinding or not,
// because for some groupPrefixes we actually don't create Role
//
// note that this method returns generated role if is not meant to be skipped
func shouldSkipRBACRoleForClusterNamespaceResource(projectName string, cluster *kubermaticv1.Cluster, policyResource, policyAPIGroups, kind, groupPrefix string) (bool, *rbacv1.Role, error) {
	generatedRole, err := generateRBACRoleForClusterNamespaceResource(
		cluster,
		GenerateActualGroupNameFor(projectName, groupPrefix),
		policyResource,
		policyAPIGroups,
		kind,
	)

	if err != nil {
		return false, generatedRole, err
	}
	if generatedRole == nil {
		return true, nil, nil
	}
	return false, generatedRole, nil
}

// ensureClusterRBACRoleBindingForEtcdLauncher ensures the ClusterRoleBinding required to allow the etcd launcher to get Clusters on the Seed
func ensureClusterRBACRoleBindingForEtcdLauncher(ctx context.Context, cli client.Client, projectName string, object metav1.Object) error {

	cluster, ok := object.(*kubermaticv1.Cluster)
	if !ok {
		return fmt.Errorf("ensureClusterRBACRoleBindingForEtcdLauncher called with non-cluster: %+v", object)
	}
	generatedRoleBinding := generateClusterRBACRoleBindingForResourceWithServiceAccount(
		cluster.Name,
		kubermaticv1.ClusterKindName,
		GenerateActualGroupNameFor(projectName, ViewerGroupNamePrefix),
		EtcdLauncherServiceAccountName,
		cluster.Status.NamespaceName,
		metav1.OwnerReference{
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			Kind:       kubermaticv1.ClusterKindName,
			UID:        object.GetUID(),
			Name:       object.GetName(),
		},
	)

	var sharedExistingRoleBinding rbacv1.ClusterRoleBinding
	key := client.ObjectKey{Name: generatedRoleBinding.Name}
	if err := cli.Get(ctx, key, &sharedExistingRoleBinding); err != nil {
		if kerrors.IsNotFound(err) {
			if err := cli.Create(ctx, generatedRoleBinding); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	if equality.Semantic.DeepEqual(sharedExistingRoleBinding.Subjects, generatedRoleBinding.Subjects) {
		return nil
	}

	existingRoleBinding := sharedExistingRoleBinding.DeepCopy()
	existingRoleBinding.Subjects = generatedRoleBinding.Subjects
	if err := cli.Update(ctx, existingRoleBinding); err != nil {
		return err
	}

	return nil
}
