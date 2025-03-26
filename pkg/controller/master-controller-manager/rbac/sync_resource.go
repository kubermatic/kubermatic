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

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EtcdLauncherServiceAccountName = "etcd-launcher"
)

// syncClusterScopedProjectResource generates RBAC Role and Binding for a cluster-scoped resource that belongs to a project.
// in order to support multiple cluster this code doesn't retrieve the project from the kube-api server
// instead it assumes that all required information is stored in OwnerReferences or in Labels (for cluster resources)
//
// note:
// the project resources live only on master cluster and cluster resources are on master and seed clusters
// we cannot use OwnerReferences for cluster resources because they are on clusters that don't have corresponding
// project resource and will be automatically gc'ed.
func (c *resourcesController) syncClusterScopedProjectResource(ctx context.Context, obj ctrlruntimeclient.Object) error {
	metaObject, err := meta.Accessor(obj)
	if err != nil {
		return fmt.Errorf("failed to get object meta: %w", err)
	}

	// just handle cluster-scoped resources
	if len(metaObject.GetNamespace()) != 0 {
		return nil
	}

	gvk := obj.GetObjectKind().GroupVersionKind()
	resourceName, err := getPluralResourceName(c.restMapper, obj)
	if err != nil {
		return err
	}

	projectName, err := getProjectName(obj)
	if err != nil {
		return err
	}

	log := c.log.With("project", projectName)

	if err := ensureClusterRBACRoleForNamedResource(ctx, log, c.client, projectName, resourceName, gvk.Kind, metaObject); err != nil {
		return fmt.Errorf("failed to sync RBAC ClusterRole: %w", err)
	}

	if err := ensureClusterRBACRoleBindingForNamedResource(ctx, log, c.client, projectName, resourceName, gvk.Kind, metaObject); err != nil {
		return fmt.Errorf("failed to sync RBAC ClusterRoleBinding: %w", err)
	}

	return nil
}

// syncNamespaceScopedProjectResource generates RBAC Role and Binding for a namespace-scoped resource that belongs to a project.
// in order to support multiple cluster this code doesn't retrieve the project from the kube-api server
// instead it assumes that all required information is stored in OwnerReferences or in Labels (for cluster resources)
//
// note:
// the project resources live only on master cluster and cluster resources are on master and seed clusters
// we cannot use OwnerReferences for cluster resources because they are on clusters that don't have corresponding
// project resource and will be automatically gc'ed.
func (c *resourcesController) syncNamespaceScopedProjectResource(ctx context.Context, obj ctrlruntimeclient.Object) error {
	metaObject, err := meta.Accessor(obj)
	if err != nil {
		return fmt.Errorf("failed to get object meta: %w", err)
	}

	// just handle namespaced resources
	if len(metaObject.GetNamespace()) == 0 {
		return nil
	}

	gvk := obj.GetObjectKind().GroupVersionKind()
	resourceName, err := getPluralResourceName(c.restMapper, obj)
	if err != nil {
		return err
	}

	projectName, err := getProjectName(obj)
	if err != nil {
		return err
	}

	log := c.log.With("project", projectName)

	if err := c.ensureRBACRoleForNamedResource(ctx, log, projectName, resourceName, gvk, metaObject.GetNamespace(), metaObject); err != nil {
		return fmt.Errorf("failed to sync RBAC Role: %w", err)
	}

	if err := c.ensureRBACRoleBindingForNamedResource(ctx, log, projectName, resourceName, gvk, metaObject.GetNamespace(), metaObject); err != nil {
		return fmt.Errorf("failed to sync RBAC RoleBinding: %w", err)
	}

	return nil
}

// syncClusterResource generates RBAC Role and Binding for resource that belongs to a Cluster.
// in order to support multiple cluster this code doesn't retrieve the project from the kube-api server
// instead it assumes that all required information is stored in OwnerReferences or in Labels (for cluster resources)
//
// note:
// the project resources live only on master cluster and cluster resources are on master and seed clusters
// we cannot use OwnerReferences for cluster resources because they are on clusters that don't have corresponding
// project resource and will be automatically gc'ed.
func (c *resourcesController) syncClusterResource(ctx context.Context, obj ctrlruntimeclient.Object) error {
	gvk := obj.GetObjectKind().GroupVersionKind()
	// handle only clusters
	if gvk.Kind != kubermaticv1.ClusterKindName {
		return nil
	}

	cluster := obj.(*kubermaticv1.Cluster)

	// skip reconciling for incomplete cluster objects
	if cluster.Status.NamespaceName == "" {
		return nil
	}

	// do not reconcile anyone once a cluster is in deletion
	if cluster.DeletionTimestamp != nil {
		return nil
	}

	rmapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	projectName, err := getProjectName(obj)
	if err != nil {
		return err
	}

	log := c.log.With("namespace", cluster.Status.NamespaceName, "project", projectName)

	if err := c.ensureRBACRoleForClusterAddons(ctx, log, projectName, cluster); err != nil {
		return fmt.Errorf("failed to sync RBAC Role: %w", err)
	}
	if err := c.ensureRBACRoleBindingForClusterAddons(ctx, log, projectName, cluster); err != nil {
		return fmt.Errorf("failed to sync RBAC RoleBinding: %w", err)
	}
	if err := c.ensureRBACForEtcdLauncher(ctx, c.client, cluster, projectName, rmapping); err != nil {
		return fmt.Errorf("failed to sync RBAC for etcd-launcher: %w", err)
	}
	if err := c.ensureRBACRoleForClusterConstraints(ctx, log, projectName, cluster); err != nil {
		return fmt.Errorf("failed to sync RBAC Role: %w", err)
	}
	if err := c.ensureRBACRoleBindingForClusterConstraints(ctx, log, projectName, cluster); err != nil {
		return fmt.Errorf("failed to sync RBAC RoleBinding: %w", err)
	}
	if err := c.ensureRBACRoleForEtcdBackupConfigs(ctx, log, projectName, cluster); err != nil {
		return fmt.Errorf("failed to sync RBAC Role: %w", err)
	}
	if err := c.ensureRBACRoleBindingForEtcdBackupConfigs(ctx, log, projectName, cluster); err != nil {
		return fmt.Errorf("failed to sync RBAC RoleBinding: %w", err)
	}
	if err := c.ensureRBACRoleForEtcdRestores(ctx, log, projectName, cluster); err != nil {
		return fmt.Errorf("failed to sync RBAC Role: %w", err)
	}
	if err := c.ensureRBACRoleBindingForEtcdRestores(ctx, log, projectName, cluster); err != nil {
		return fmt.Errorf("failed to sync RBAC RoleBinding: %w", err)
	}

	if userClusterMLAEnabled(cluster) {
		if err := c.ensureRBACRoleForClusterAlertmanagers(ctx, log, projectName, cluster); err != nil {
			return fmt.Errorf("failed to sync RBAC Role: %w", err)
		}
		if err := c.ensureRBACRoleBindingForClusterAlertmanagers(ctx, log, projectName, cluster); err != nil {
			return fmt.Errorf("failed to sync RBAC RoleBinding: %w", err)
		}
		if err := c.ensureRBACRoleForClusterAlertmanagerConfigSecrets(ctx, log, projectName, cluster); err != nil {
			return fmt.Errorf("failed to sync RBAC Role: %w", err)
		}
		if err := c.ensureRBACRoleBindingForClusterAlertmanagerConfigSecrets(ctx, log, projectName, cluster); err != nil {
			return fmt.Errorf("failed to sync RBAC RoleBinding: %w", err)
		}
		if err := c.ensureRBACRoleForClusterRuleGroups(ctx, log, projectName, cluster); err != nil {
			return fmt.Errorf("failed to sync RBAC Role: %w", err)
		}
		if err := c.ensureRBACRoleBindingForClusterRuleGroups(ctx, log, projectName, cluster); err != nil {
			return fmt.Errorf("failed to sync RBAC RoleBinding: %w", err)
		}
	}

	return nil
}

func userClusterMLAEnabled(cluster *kubermaticv1.Cluster) bool {
	return cluster.Spec.MLA != nil && (cluster.Spec.MLA.MonitoringEnabled || cluster.Spec.MLA.LoggingEnabled)
}

func getProjectName(obj ctrlruntimeclient.Object) (string, error) {
	projectName := ""
	for _, owner := range obj.GetOwnerReferences() {
		if owner.APIVersion == kubermaticv1.SchemeGroupVersion.String() && owner.Kind == kubermaticv1.ProjectKindName &&
			len(owner.Name) > 0 && len(owner.UID) > 0 {
			projectName = owner.Name
			break
		}
	}
	if len(projectName) == 0 {
		projectName = obj.GetLabels()[kubermaticv1.ProjectIDLabelKey]
	}

	if len(projectName) == 0 {
		return "", fmt.Errorf("unable to find owning project for %s %s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName())
	}
	return projectName, nil
}

func ensureClusterRBACRoleForNamedResource(ctx context.Context, log *zap.SugaredLogger, cli ctrlruntimeclient.Client, projectName string, objectResource string, objectKind string, object metav1.Object) error {
	for _, groupPrefix := range AllGroupsPrefixes {
		skip, generatedRole, err := shouldSkipClusterRBACRoleBindingForNamedResource(projectName, objectResource, objectKind, groupPrefix, object)
		if err != nil {
			return err
		}
		if skip {
			log.Debugw("skipping ClusterRole generation for named resource", "group", groupPrefix, "resource", objectResource)
			continue
		}

		var sharedExistingRole rbacv1.ClusterRole
		key := ctrlruntimeclient.ObjectKey{Name: generatedRole.Name}
		if err := cli.Get(ctx, key, &sharedExistingRole); err != nil {
			if apierrors.IsNotFound(err) {
				if err := cli.Create(ctx, generatedRole); err != nil {
					return err
				}
				continue
			}
			return err
		}

		if equality.Semantic.DeepEqual(sharedExistingRole.Rules, generatedRole.Rules) &&
			equality.Semantic.DeepEqual(sharedExistingRole.Labels, generatedRole.Labels) {
			continue
		}
		existingRole := sharedExistingRole.DeepCopy()
		existingRole.Rules = generatedRole.Rules
		existingRole.Labels = generatedRole.Labels

		if err := cli.Update(ctx, existingRole); err != nil {
			return err
		}
	}

	return nil
}

func ensureClusterRBACRoleBindingForNamedResource(ctx context.Context, log *zap.SugaredLogger, cli ctrlruntimeclient.Client, projectName string, objectResource string, objectKind string, object metav1.Object) error {
	for _, groupPrefix := range AllGroupsPrefixes {
		skip, _, err := shouldSkipClusterRBACRoleBindingForNamedResource(projectName, objectResource, objectKind, groupPrefix, object)
		if err != nil {
			return err
		}
		if skip {
			log.Debugw("skipping operation on ClusterRoleBinding because corresponding ClusterRole was not(will not be) created", "group", groupPrefix, "resource", objectResource)
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
		key := ctrlruntimeclient.ObjectKey{Name: generatedRoleBinding.Name}
		if err := cli.Get(ctx, key, &sharedExistingRoleBinding); err != nil {
			if apierrors.IsNotFound(err) {
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
// note that this method returns generated role if is not meant to be skipped.
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

func (c *resourcesController) ensureRBACRoleForNamedResource(ctx context.Context, log *zap.SugaredLogger, projectName string, resourceName string, gvk schema.GroupVersionKind, namespace string, object metav1.Object) error {
	for _, groupPrefix := range AllGroupsPrefixes {
		skip, generatedRole, err := shouldSkipRBACRoleBindingForNamedResource(projectName, resourceName, gvk, groupPrefix, namespace, object)
		if err != nil {
			return err
		}
		if skip {
			log.Debugw("skipping Role generation for named resource", "group", groupPrefix, "resource", resourceName, "namespace", namespace)
			continue
		}
		var sharedExistingRole rbacv1.Role
		key := ctrlruntimeclient.ObjectKey{Name: generatedRole.Name}
		if err := c.client.Get(ctx, key, &sharedExistingRole); err != nil {
			if apierrors.IsNotFound(err) {
				if err := c.client.Create(ctx, generatedRole); err != nil {
					return nil
				}
				continue
			}
			return nil
		}

		// make sure that existing rbac role has appropriate rules/policies
		if equality.Semantic.DeepEqual(sharedExistingRole.Rules, generatedRole.Rules) &&
			equality.Semantic.DeepEqual(sharedExistingRole.Labels, generatedRole.Labels) {
			continue
		}
		existingRole := sharedExistingRole.DeepCopy()
		existingRole.Rules = generatedRole.Rules
		existingRole.Labels = generatedRole.Labels
		if err := c.client.Update(ctx, existingRole); err != nil {
			return err
		}
	}

	return nil
}

func (c *resourcesController) ensureRBACRoleBindingForNamedResource(ctx context.Context, log *zap.SugaredLogger, projectName string, resourceName string, gvk schema.GroupVersionKind, namespace string, object metav1.Object) error {
	for _, groupPrefix := range AllGroupsPrefixes {
		skip, _, err := shouldSkipRBACRoleBindingForNamedResource(projectName, resourceName, gvk, groupPrefix, namespace, object)
		if err != nil {
			return err
		}
		if skip {
			log.Debugw("skipping operation on RoleBinding because corresponding Role was not (will not be) created", "group", groupPrefix, "resource", resourceName, "namespace", namespace)
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
		key := ctrlruntimeclient.ObjectKey{Name: generatedRoleBinding.Name}
		if err := c.client.Get(ctx, key, &sharedExistingRoleBinding); err != nil {
			if apierrors.IsNotFound(err) {
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
// note that this method returns generated role if is not meant to be skipped.
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

func (c *resourcesController) ensureRBACRoleForClusterAddons(ctx context.Context, log *zap.SugaredLogger, projectName string, cluster *kubermaticv1.Cluster) error {
	var roleList rbacv1.RoleList
	opts := &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}
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
			log.Debugw("skipping Role generation for cluster addons", "group", groupPrefix)
			continue
		}

		var sharedExistingRole rbacv1.Role
		key := ctrlruntimeclient.ObjectKey{Name: generatedRole.Name, Namespace: cluster.Status.NamespaceName}
		if err := c.client.Get(ctx, key, &sharedExistingRole); err != nil {
			if apierrors.IsNotFound(err) {
				if err := c.client.Create(ctx, generatedRole); err != nil {
					return err
				}
				continue
			}
			return err
		}

		// make sure that existing rbac role has appropriate rules/policies
		if equality.Semantic.DeepEqual(sharedExistingRole.Rules, generatedRole.Rules) &&
			equality.Semantic.DeepEqual(sharedExistingRole.Labels, generatedRole.Labels) {
			continue
		}
		existingRole := sharedExistingRole.DeepCopy()
		existingRole.Rules = generatedRole.Rules
		existingRole.Labels = generatedRole.Labels
		if err := c.client.Update(ctx, existingRole); err != nil {
			return err
		}
	}

	return nil
}

func (c *resourcesController) ensureRBACRoleBindingForClusterAddons(ctx context.Context, log *zap.SugaredLogger, projectName string, cluster *kubermaticv1.Cluster) error {
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
			log.Debugw("skipping RoleBinding generation for cluster addons", "group", groupPrefix)
			continue
		}

		generatedRoleBinding := generateRBACRoleBindingForClusterNamespaceResource(
			cluster,
			GenerateActualGroupNameFor(projectName, groupPrefix),
			kubermaticv1.AddonKindName,
		)

		var sharedExistingRoleBinding rbacv1.RoleBinding
		key := ctrlruntimeclient.ObjectKey{Name: generatedRoleBinding.Name, Namespace: cluster.Status.NamespaceName}
		if err := c.client.Get(ctx, key, &sharedExistingRoleBinding); err != nil {
			if apierrors.IsNotFound(err) {
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

func (c *resourcesController) ensureRBACRoleForEtcdLauncher(ctx context.Context, cluster *kubermaticv1.Cluster, resourceName string, groupName string, kindName string) error {
	var roleList rbacv1.RoleList
	opts := &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}
	if err := c.client.List(ctx, &roleList, opts); err != nil {
		return err
	}

	generatedRole, err := generateRBACRoleForClusterNamespaceResourceAndServiceAccount(
		cluster,
		[]string{"get", "list"},
		EtcdLauncherServiceAccountName,
		resourceName,
		groupName,
		kindName)
	if err != nil {
		return err
	}

	var sharedExistingRole rbacv1.Role
	key := ctrlruntimeclient.ObjectKey{Name: generatedRole.Name, Namespace: cluster.Status.NamespaceName}
	if err := c.client.Get(ctx, key, &sharedExistingRole); err != nil {
		if apierrors.IsNotFound(err) {
			if err := c.client.Create(ctx, generatedRole); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	// make sure that existing rbac role has appropriate rules/policies
	if equality.Semantic.DeepEqual(sharedExistingRole.Rules, generatedRole.Rules) &&
		equality.Semantic.DeepEqual(sharedExistingRole.Labels, generatedRole.Labels) {
		return nil
	}
	existingRole := sharedExistingRole.DeepCopy()
	existingRole.Rules = generatedRole.Rules
	existingRole.Labels = generatedRole.Labels
	if err := c.client.Update(ctx, existingRole); err != nil {
		return err
	}

	return nil
}

func (c *resourcesController) ensureRBACRoleBindingForEtcdLauncher(ctx context.Context, cluster *kubermaticv1.Cluster, kindName string) error {
	generatedRoleBinding := generateRBACRoleBindingForEtcdLauncherServiceAccount(
		cluster,
		EtcdLauncherServiceAccountName,
		kindName,
	)

	var sharedExistingRoleBinding rbacv1.RoleBinding
	key := ctrlruntimeclient.ObjectKey{Name: generatedRoleBinding.Name, Namespace: cluster.Status.NamespaceName}
	if err := c.client.Get(ctx, key, &sharedExistingRoleBinding); err != nil {
		if apierrors.IsNotFound(err) {
			if err := c.client.Create(ctx, generatedRoleBinding); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	// sharedExistingRoleBinding found
	if equality.Semantic.DeepEqual(sharedExistingRoleBinding.Subjects, generatedRoleBinding.Subjects) {
		return nil
	}
	existingRoleBinding := sharedExistingRoleBinding.DeepCopy()
	existingRoleBinding.Subjects = generatedRoleBinding.Subjects
	if err := c.client.Update(ctx, existingRoleBinding); err != nil {
		return err
	}

	return nil
}

// shouldSkipRBACRoleForClusterNamespaceResource will tell you if you should skip the generation of Role/Rolebinding or not,
// because for some groupPrefixes we actually don't create Role
//
// note that this method returns generated role if is not meant to be skipped.
func shouldSkipRBACRoleForClusterNamespaceResource(projectName string, cluster *kubermaticv1.Cluster, policyResource, policyAPIGroups, kind, groupPrefix string) (bool, *rbacv1.Role, error) {
	if cluster.Status.NamespaceName == "" {
		return true, nil, nil
	}

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

// shouldSkipRBACRoleForClusterNamespaceNamedResource will tell you if you should skip the generation of Role/Rolebinding of a named resource or not,
// because for some groupPrefixes we actually don't create Role
// note that this method returns generated role if is not meant to be skipped.
func shouldSkipRBACRoleForClusterNamespaceNamedResource(projectName string, cluster *kubermaticv1.Cluster, resourceName, policyAPIGroups, policyResource, kind, groupPrefix string) (bool, *rbacv1.Role, error) {
	if cluster.Status.NamespaceName == "" {
		return true, nil, nil
	}

	generatedRole, err := generateRBACRoleForClusterNamespaceNamedResource(
		cluster,
		GenerateActualGroupNameFor(projectName, groupPrefix),
		policyAPIGroups,
		policyResource,
		kind,
		resourceName,
	)
	if err != nil {
		return false, generatedRole, err
	}
	if generatedRole == nil {
		return true, nil, nil
	}
	return false, generatedRole, nil
}

// ensureClusterRBACRoleBindingForEtcdLauncher ensures the ClusterRoleBinding required to allow the etcd launcher to get Clusters on the Seed.
func (c *resourcesController) ensureClusterRBACRoleBindingForEtcdLauncher(ctx context.Context, resourceName, resourceKind, namespace, projectName string, cluster *kubermaticv1.Cluster) error {
	generatedRoleBinding := generateEtcdRBACRoleBindingForResourceWithServiceAccount(
		resourceName,
		resourceKind,
		GenerateActualGroupNameFor(projectName, ViewerGroupNamePrefix),
		cluster.Name,
		EtcdLauncherServiceAccountName,
		namespace,
		metav1.OwnerReference{
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			Kind:       kubermaticv1.ClusterKindName,
			UID:        cluster.GetUID(),
			Name:       cluster.GetName(),
		},
	)

	var sharedExistingRoleBinding rbacv1.ClusterRoleBinding
	key := ctrlruntimeclient.ObjectKey{Name: generatedRoleBinding.Name}
	if err := c.client.Get(ctx, key, &sharedExistingRoleBinding); err != nil {
		if apierrors.IsNotFound(err) {
			if err := c.client.Create(ctx, generatedRoleBinding); err != nil {
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
	if err := c.client.Update(ctx, existingRoleBinding); err != nil {
		return err
	}

	return nil
}

func (c *resourcesController) ensureRBACRoleForClusterAlertmanagers(ctx context.Context, log *zap.SugaredLogger, projectName string, cluster *kubermaticv1.Cluster) error {
	var roleList rbacv1.RoleList
	opts := &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}
	if err := c.client.List(ctx, &roleList, opts); err != nil {
		return err
	}

	for _, groupPrefix := range AllGroupsPrefixes {
		skip, generatedRole, err := shouldSkipRBACRoleForClusterNamespaceNamedResource(
			projectName,
			cluster,
			alertmanagerName,
			kubermaticv1.GroupName,
			kubermaticv1.AlertmanagerResourceName,
			kubermaticv1.AlertmanagerKindName,
			groupPrefix)
		if err != nil {
			return err
		}
		if skip {
			log.Debugw("skipping Role generation for cluster alertmanagers", "group", groupPrefix)
			continue
		}

		var sharedExistingRole rbacv1.Role
		key := ctrlruntimeclient.ObjectKey{Name: generatedRole.Name, Namespace: cluster.Status.NamespaceName}
		if err := c.client.Get(ctx, key, &sharedExistingRole); err != nil {
			if apierrors.IsNotFound(err) {
				if err := c.client.Create(ctx, generatedRole); err != nil {
					return err
				}
				continue
			}
			return err
		}

		// make sure that existing rbac role has appropriate rules/policies

		if equality.Semantic.DeepEqual(sharedExistingRole.Rules, generatedRole.Rules) &&
			equality.Semantic.DeepEqual(sharedExistingRole.Labels, generatedRole.Labels) {
			continue
		}
		existingRole := sharedExistingRole.DeepCopy()
		existingRole.Rules = generatedRole.Rules
		existingRole.Labels = generatedRole.Labels
		if err := c.client.Update(ctx, existingRole); err != nil {
			return err
		}
	}

	return nil
}

func (c *resourcesController) ensureRBACRoleBindingForClusterAlertmanagers(ctx context.Context, log *zap.SugaredLogger, projectName string, cluster *kubermaticv1.Cluster) error {
	for _, groupPrefix := range AllGroupsPrefixes {
		skip, _, err := shouldSkipRBACRoleForClusterNamespaceNamedResource(
			projectName,
			cluster,
			alertmanagerName,
			kubermaticv1.GroupName,
			kubermaticv1.AlertmanagerResourceName,
			kubermaticv1.AlertmanagerKindName,
			groupPrefix)
		if err != nil {
			return err
		}
		if skip {
			log.Debugw("skipping RoleBinding generation for cluster alertmanagers", "group", groupPrefix)
			continue
		}

		generatedRoleBinding := generateRBACRoleBindingForClusterNamespaceNamedResource(
			cluster,
			GenerateActualGroupNameFor(projectName, groupPrefix),
			kubermaticv1.AlertmanagerKindName,
			alertmanagerName,
		)

		var sharedExistingRoleBinding rbacv1.RoleBinding
		key := ctrlruntimeclient.ObjectKey{Name: generatedRoleBinding.Name, Namespace: cluster.Status.NamespaceName}
		if err := c.client.Get(ctx, key, &sharedExistingRoleBinding); err != nil {
			if apierrors.IsNotFound(err) {
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

func (c *resourcesController) ensureRBACRoleForClusterAlertmanagerConfigSecrets(ctx context.Context, log *zap.SugaredLogger, projectName string, cluster *kubermaticv1.Cluster) error {
	var roleList rbacv1.RoleList
	opts := &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}
	if err := c.client.List(ctx, &roleList, opts); err != nil {
		return err
	}

	for _, groupPrefix := range AllGroupsPrefixes {
		skip, generatedRole, err := shouldSkipRBACRoleForClusterNamespaceNamedResource(
			projectName,
			cluster,
			defaultAlertmanagerConfigSecretName,
			corev1.GroupName,
			"secrets",
			"Secret",
			groupPrefix)
		if err != nil {
			return err
		}
		if skip {
			log.Debugw("skipping Role generation for cluster alertmanager config secrets ", "group", groupPrefix)
			continue
		}

		var sharedExistingRole rbacv1.Role
		key := ctrlruntimeclient.ObjectKey{Name: generatedRole.Name, Namespace: cluster.Status.NamespaceName}
		if err := c.client.Get(ctx, key, &sharedExistingRole); err != nil {
			if apierrors.IsNotFound(err) {
				if err := c.client.Create(ctx, generatedRole); err != nil {
					return err
				}
				continue
			}
			return err
		}

		// make sure that existing rbac role has appropriate rules/policies
		if equality.Semantic.DeepEqual(sharedExistingRole.Rules, generatedRole.Rules) &&
			equality.Semantic.DeepEqual(sharedExistingRole.Labels, generatedRole.Labels) {
			continue
		}
		existingRole := sharedExistingRole.DeepCopy()
		existingRole.Rules = generatedRole.Rules
		existingRole.Labels = generatedRole.Labels
		if err := c.client.Update(ctx, existingRole); err != nil {
			return err
		}
	}

	return nil
}

func (c *resourcesController) ensureRBACRoleBindingForClusterAlertmanagerConfigSecrets(ctx context.Context, log *zap.SugaredLogger, projectName string, cluster *kubermaticv1.Cluster) error {
	for _, groupPrefix := range AllGroupsPrefixes {
		skip, _, err := shouldSkipRBACRoleForClusterNamespaceNamedResource(
			projectName,
			cluster,
			defaultAlertmanagerConfigSecretName,
			corev1.GroupName,
			"secrets",
			"Secret",
			groupPrefix)
		if err != nil {
			return err
		}
		if skip {
			log.Debugw("skipping RoleBinding generation for cluster alertmanager config secrets", "group", groupPrefix)
			continue
		}

		generatedRoleBinding := generateRBACRoleBindingForClusterNamespaceNamedResource(
			cluster,
			GenerateActualGroupNameFor(projectName, groupPrefix),
			"Secret",
			defaultAlertmanagerConfigSecretName,
		)

		var sharedExistingRoleBinding rbacv1.RoleBinding
		key := ctrlruntimeclient.ObjectKey{Name: generatedRoleBinding.Name, Namespace: cluster.Status.NamespaceName}
		if err := c.client.Get(ctx, key, &sharedExistingRoleBinding); err != nil {
			if apierrors.IsNotFound(err) {
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

func (c *resourcesController) ensureRBACRoleForClusterConstraints(ctx context.Context, log *zap.SugaredLogger, projectName string, cluster *kubermaticv1.Cluster) error {
	var roleList rbacv1.RoleList
	opts := &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}
	if err := c.client.List(ctx, &roleList, opts); err != nil {
		return err
	}

	for _, groupPrefix := range AllGroupsPrefixes {
		skip, generatedRole, err := shouldSkipRBACRoleForClusterNamespaceResource(
			projectName,
			cluster,
			kubermaticv1.ConstraintResourceName,
			kubermaticv1.GroupName,
			kubermaticv1.ConstraintKind,
			groupPrefix)
		if err != nil {
			return err
		}
		if skip {
			log.Debugw("skipping Role generation for cluster constraints", "group", groupPrefix)
			continue
		}

		var sharedExistingRole rbacv1.Role
		key := ctrlruntimeclient.ObjectKey{Name: generatedRole.Name, Namespace: cluster.Status.NamespaceName}
		if err := c.client.Get(ctx, key, &sharedExistingRole); err != nil {
			if apierrors.IsNotFound(err) {
				if err := c.client.Create(ctx, generatedRole); err != nil {
					return err
				}
				continue
			}
			return err
		}

		// make sure that existing rbac role has appropriate rules/policies
		if equality.Semantic.DeepEqual(sharedExistingRole.Rules, generatedRole.Rules) &&
			equality.Semantic.DeepEqual(sharedExistingRole.Labels, generatedRole.Labels) {
			continue
		}
		existingRole := sharedExistingRole.DeepCopy()
		existingRole.Rules = generatedRole.Rules
		existingRole.Labels = generatedRole.Labels
		if err := c.client.Update(ctx, existingRole); err != nil {
			return err
		}
	}

	return nil
}

func (c *resourcesController) ensureRBACRoleBindingForClusterConstraints(ctx context.Context, log *zap.SugaredLogger, projectName string, cluster *kubermaticv1.Cluster) error {
	for _, groupPrefix := range AllGroupsPrefixes {
		skip, _, err := shouldSkipRBACRoleForClusterNamespaceResource(
			projectName,
			cluster,
			kubermaticv1.ConstraintResourceName,
			kubermaticv1.GroupName,
			kubermaticv1.ConstraintKind,
			groupPrefix)
		if err != nil {
			return err
		}
		if skip {
			log.Debugw("skipping RoleBinding generation for cluster constraints", "group", groupPrefix)
			continue
		}

		generatedRoleBinding := generateRBACRoleBindingForClusterNamespaceResource(
			cluster,
			GenerateActualGroupNameFor(projectName, groupPrefix),
			kubermaticv1.ConstraintKind,
		)

		var sharedExistingRoleBinding rbacv1.RoleBinding
		key := ctrlruntimeclient.ObjectKey{Name: generatedRoleBinding.Name, Namespace: cluster.Status.NamespaceName}
		if err := c.client.Get(ctx, key, &sharedExistingRoleBinding); err != nil {
			if apierrors.IsNotFound(err) {
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

func (c *resourcesController) ensureRBACRoleForClusterRuleGroups(ctx context.Context, log *zap.SugaredLogger, projectName string, cluster *kubermaticv1.Cluster) error {
	var roleList rbacv1.RoleList
	opts := &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}
	if err := c.client.List(ctx, &roleList, opts); err != nil {
		return err
	}

	for _, groupPrefix := range AllGroupsPrefixes {
		skip, generatedRole, err := shouldSkipRBACRoleForClusterNamespaceResource(
			projectName,
			cluster,
			kubermaticv1.RuleGroupResourceName,
			kubermaticv1.GroupName,
			kubermaticv1.RuleGroupKindName,
			groupPrefix)
		if err != nil {
			return err
		}
		if skip {
			log.Debugw("skipping Role generation for cluster constraints", "group", groupPrefix)
			continue
		}

		var sharedExistingRole rbacv1.Role
		key := ctrlruntimeclient.ObjectKey{Name: generatedRole.Name, Namespace: cluster.Status.NamespaceName}
		if err := c.client.Get(ctx, key, &sharedExistingRole); err != nil {
			if apierrors.IsNotFound(err) {
				if err := c.client.Create(ctx, generatedRole); err != nil {
					return err
				}
				continue
			}
			return err
		}

		// make sure that existing rbac role has appropriate rules/policies
		if equality.Semantic.DeepEqual(sharedExistingRole.Rules, generatedRole.Rules) &&
			equality.Semantic.DeepEqual(sharedExistingRole.Labels, generatedRole.Labels) {
			continue
		}
		existingRole := sharedExistingRole.DeepCopy()
		existingRole.Rules = generatedRole.Rules
		existingRole.Labels = generatedRole.Labels
		if err := c.client.Update(ctx, existingRole); err != nil {
			return err
		}
	}

	return nil
}

func (c *resourcesController) ensureClusterRBACRoleForEtcdLauncher(ctx context.Context, projectName string, cluster *kubermaticv1.Cluster) error {
	generatedClusterRole, err := generateClusterRBACRoleNamedResource(
		"Configmap",
		GenerateActualGroupNameFor(projectName, ViewerGroupNamePrefix),
		"configmaps",
		"",
		fmt.Sprintf("cluster-%s-ca-bundle", cluster.Name),
		metav1.OwnerReference{
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			Kind:       kubermaticv1.ClusterKindName,
			UID:        cluster.GetUID(),
			Name:       cluster.GetName(),
		},
	)
	if err != nil {
		return err
	}
	var existingClusterRole rbacv1.ClusterRole
	key := ctrlruntimeclient.ObjectKey{Name: generatedClusterRole.Name}
	if err := c.client.Get(ctx, key, &existingClusterRole); err != nil {
		if apierrors.IsNotFound(err) {
			if err := c.client.Create(ctx, generatedClusterRole); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	if equality.Semantic.DeepEqual(existingClusterRole.Rules, generatedClusterRole.Rules) &&
		equality.Semantic.DeepEqual(existingClusterRole.Labels, generatedClusterRole.Labels) {
		return nil
	}
	updatedClusterRole := existingClusterRole.DeepCopy()
	updatedClusterRole.Rules = generatedClusterRole.Rules
	updatedClusterRole.Labels = generatedClusterRole.Labels
	if err := c.client.Update(ctx, updatedClusterRole); err != nil {
		return err
	}

	return nil
}

func (c *resourcesController) ensureRBACRoleBindingForClusterRuleGroups(ctx context.Context, log *zap.SugaredLogger, projectName string, cluster *kubermaticv1.Cluster) error {
	for _, groupPrefix := range AllGroupsPrefixes {
		skip, _, err := shouldSkipRBACRoleForClusterNamespaceResource(
			projectName,
			cluster,
			kubermaticv1.RuleGroupResourceName,
			kubermaticv1.GroupName,
			kubermaticv1.RuleGroupKindName,
			groupPrefix)
		if err != nil {
			return err
		}
		if skip {
			log.Debugw("skipping RoleBinding generation for cluster constraints", "group", groupPrefix)
			continue
		}

		generatedRoleBinding := generateRBACRoleBindingForClusterNamespaceResource(
			cluster,
			GenerateActualGroupNameFor(projectName, groupPrefix),
			kubermaticv1.RuleGroupKindName,
		)

		var sharedExistingRoleBinding rbacv1.RoleBinding
		key := ctrlruntimeclient.ObjectKey{Name: generatedRoleBinding.Name, Namespace: cluster.Status.NamespaceName}
		if err := c.client.Get(ctx, key, &sharedExistingRoleBinding); err != nil {
			if apierrors.IsNotFound(err) {
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

func (c *resourcesController) ensureRBACForEtcdLauncher(ctx context.Context, cli ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, projectName string, rmapping *meta.RESTMapping) error {
	if err := c.ensureClusterRBACRoleForEtcdLauncher(ctx, projectName, cluster); err != nil {
		return fmt.Errorf("failed to sync RBAC ClusterRole for %s resource for %s cluster provider: %w", formatMapping(rmapping), c.providerName, err)
	}
	if err := c.ensureClusterRBACRoleBindingForEtcdLauncher(ctx, cluster.Name, kubermaticv1.ClusterKindName, cluster.Status.NamespaceName, projectName, cluster); err != nil {
		return fmt.Errorf("failed to sync RBAC ClusterRoleBinding for %s resource for %s cluster provider: %w", formatMapping(rmapping), c.providerName, err)
	}
	if err := c.ensureClusterRBACRoleBindingForEtcdLauncher(ctx, fmt.Sprintf("cluster-%s-ca-bundle", cluster.Name), "Configmap", cluster.Status.NamespaceName, projectName, cluster); err != nil {
		return fmt.Errorf("failed to sync RBAC ClusterRoleBinding for %s resource for %s cluster provider: %w", formatMapping(rmapping), c.providerName, err)
	}
	if err := c.ensureRBACRoleForEtcdLauncher(ctx, cluster, kubermaticv1.EtcdRestoreResourceName, kubermaticv1.GroupName, kubermaticv1.EtcdRestoreKindName); err != nil {
		return fmt.Errorf("failed to sync etcd restore RBAC Role for %s resource for %s cluster provider: %w", formatMapping(rmapping), c.providerName, err)
	}
	if err := c.ensureRBACRoleBindingForEtcdLauncher(ctx, cluster, kubermaticv1.EtcdRestoreKindName); err != nil {
		return fmt.Errorf("failed to sync etcd restore RBAC ClusterRoleBinding for %s resource for %s cluster provider: %w", formatMapping(rmapping), c.providerName, err)
	}
	if err := c.ensureRBACRoleForEtcdLauncher(ctx, cluster, "secrets", "", "Secret"); err != nil {
		return fmt.Errorf("failed to sync etcd restore RBAC Role for %s resource for %s cluster provider: %w", formatMapping(rmapping), c.providerName, err)
	}
	if err := c.ensureRBACRoleBindingForEtcdLauncher(ctx, cluster, "Secret"); err != nil {
		return fmt.Errorf("failed to sync etcd restore RBAC RoleBinding for %s resource for %s cluster provider: %w", formatMapping(rmapping), c.providerName, err)
	}
	if err := c.ensureRBACRoleForEtcdLauncher(ctx, cluster, "pods", "", "Pod"); err != nil {
		return fmt.Errorf("failed to sync etcd restore RBAC Role for %s resource for %s cluster provider: %w", formatMapping(rmapping), c.providerName, err)
	}
	if err := c.ensureRBACRoleBindingForEtcdLauncher(ctx, cluster, "Pod"); err != nil {
		return fmt.Errorf("failed to sync etcd restore RBAC RoleBinding for %s resource for %s cluster provider: %w", formatMapping(rmapping), c.providerName, err)
	}
	if err := c.ensureRBACRoleForEtcdLauncher(ctx, cluster, "statefulsets", "apps", "StatefulSet"); err != nil {
		return fmt.Errorf("failed to sync etcd launcher RBAC Role for %s resource for %s cluster provider: %w", formatMapping(rmapping), c.providerName, err)
	}
	if err := c.ensureRBACRoleBindingForEtcdLauncher(ctx, cluster, "StatefulSet"); err != nil {
		return fmt.Errorf("failed to sync etcd launcher RBAC CluclustersterRoleBinding for %s resource for %s cluster provider: %w", formatMapping(rmapping), c.providerName, err)
	}

	return nil
}

func (c *resourcesController) ensureRBACRoleForEtcdBackupConfigs(ctx context.Context, log *zap.SugaredLogger, projectName string, cluster *kubermaticv1.Cluster) error {
	var roleList rbacv1.RoleList
	opts := &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}
	if err := c.client.List(ctx, &roleList, opts); err != nil {
		return err
	}

	for _, groupPrefix := range AllGroupsPrefixes {
		skip, generatedRole, err := shouldSkipRBACRoleForClusterNamespaceResource(
			projectName,
			cluster,
			kubermaticv1.EtcdBackupConfigResourceName,
			kubermaticv1.GroupName,
			kubermaticv1.EtcdBackupConfigKindName,
			groupPrefix)
		if err != nil {
			return err
		}
		if skip {
			log.Debugw("skipping Role generation for cluster etcdBackupConfigs", "group", groupPrefix)
			continue
		}

		var sharedExistingRole rbacv1.Role
		key := ctrlruntimeclient.ObjectKey{Name: generatedRole.Name, Namespace: cluster.Status.NamespaceName}
		if err := c.client.Get(ctx, key, &sharedExistingRole); err != nil {
			if apierrors.IsNotFound(err) {
				if err := c.client.Create(ctx, generatedRole); err != nil {
					return err
				}
				continue
			}
			return err
		}

		// make sure that existing rbac role has appropriate rules/policies
		if equality.Semantic.DeepEqual(sharedExistingRole.Rules, generatedRole.Rules) &&
			equality.Semantic.DeepEqual(sharedExistingRole.Labels, generatedRole.Labels) {
			continue
		}
		existingRole := sharedExistingRole.DeepCopy()
		existingRole.Rules = generatedRole.Rules
		existingRole.Labels = generatedRole.Labels
		if err := c.client.Update(ctx, existingRole); err != nil {
			return err
		}
	}

	return nil
}

func (c *resourcesController) ensureRBACRoleBindingForEtcdBackupConfigs(ctx context.Context, log *zap.SugaredLogger, projectName string, cluster *kubermaticv1.Cluster) error {
	for _, groupPrefix := range AllGroupsPrefixes {
		skip, _, err := shouldSkipRBACRoleForClusterNamespaceResource(
			projectName,
			cluster,
			kubermaticv1.EtcdBackupConfigResourceName,
			kubermaticv1.GroupName,
			kubermaticv1.EtcdBackupConfigKindName,
			groupPrefix)
		if err != nil {
			return err
		}
		if skip {
			log.Debugw("skipping RoleBinding generation for cluster etcdBackupConfigs", "group", groupPrefix)
			continue
		}

		generatedRoleBinding := generateRBACRoleBindingForClusterNamespaceResource(
			cluster,
			GenerateActualGroupNameFor(projectName, groupPrefix),
			kubermaticv1.EtcdBackupConfigKindName,
		)

		var sharedExistingRoleBinding rbacv1.RoleBinding
		key := ctrlruntimeclient.ObjectKey{Name: generatedRoleBinding.Name, Namespace: cluster.Status.NamespaceName}
		if err := c.client.Get(ctx, key, &sharedExistingRoleBinding); err != nil {
			if apierrors.IsNotFound(err) {
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

func (c *resourcesController) ensureRBACRoleForEtcdRestores(ctx context.Context, log *zap.SugaredLogger, projectName string, cluster *kubermaticv1.Cluster) error {
	var roleList rbacv1.RoleList
	opts := &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}
	if err := c.client.List(ctx, &roleList, opts); err != nil {
		return err
	}

	for _, groupPrefix := range AllGroupsPrefixes {
		skip, generatedRole, err := shouldSkipRBACRoleForClusterNamespaceResource(
			projectName,
			cluster,
			kubermaticv1.EtcdRestoreResourceName,
			kubermaticv1.GroupName,
			kubermaticv1.EtcdRestoreKindName,
			groupPrefix)
		if err != nil {
			return err
		}
		if skip {
			log.Debugw("skipping Role generation for cluster etcdRestores", "group", groupPrefix)
			continue
		}

		var sharedExistingRole rbacv1.Role
		key := ctrlruntimeclient.ObjectKey{Name: generatedRole.Name, Namespace: cluster.Status.NamespaceName}
		if err := c.client.Get(ctx, key, &sharedExistingRole); err != nil {
			if apierrors.IsNotFound(err) {
				if err := c.client.Create(ctx, generatedRole); err != nil {
					return err
				}
				continue
			}
			return err
		}

		// make sure that existing rbac role has appropriate rules/policies
		if equality.Semantic.DeepEqual(sharedExistingRole.Rules, generatedRole.Rules) &&
			equality.Semantic.DeepEqual(sharedExistingRole.Labels, generatedRole.Labels) {
			continue
		}
		existingRole := sharedExistingRole.DeepCopy()
		existingRole.Rules = generatedRole.Rules
		existingRole.Labels = generatedRole.Labels
		if err := c.client.Update(ctx, existingRole); err != nil {
			return err
		}
	}

	return nil
}

func (c *resourcesController) ensureRBACRoleBindingForEtcdRestores(ctx context.Context, log *zap.SugaredLogger, projectName string, cluster *kubermaticv1.Cluster) error {
	for _, groupPrefix := range AllGroupsPrefixes {
		skip, _, err := shouldSkipRBACRoleForClusterNamespaceResource(
			projectName,
			cluster,
			kubermaticv1.EtcdRestoreResourceName,
			kubermaticv1.GroupName,
			kubermaticv1.EtcdRestoreKindName,
			groupPrefix)
		if err != nil {
			return err
		}
		if skip {
			log.Debugw("skipping RoleBinding generation for cluster etcdRestores", "group", groupPrefix)
			continue
		}

		generatedRoleBinding := generateRBACRoleBindingForClusterNamespaceResource(
			cluster,
			GenerateActualGroupNameFor(projectName, groupPrefix),
			kubermaticv1.EtcdRestoreKindName,
		)

		var sharedExistingRoleBinding rbacv1.RoleBinding
		key := ctrlruntimeclient.ObjectKey{Name: generatedRoleBinding.Name, Namespace: cluster.Status.NamespaceName}
		if err := c.client.Get(ctx, key, &sharedExistingRoleBinding); err != nil {
			if apierrors.IsNotFound(err) {
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
