package rbac

import (
	"context"
	"fmt"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	rbaclister "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CleanupFinalizerName = "kubermatic.io/controller-manager-rbac-cleanup"
)

func (c *projectController) sync(key client.ObjectKey) error {
	var originalProject kubermaticv1.Project
	if err := c.client.Get(c.ctx, key, &originalProject); err != nil {
		return err
	}

	project := originalProject.DeepCopy()

	if c.shouldDeleteProject(project) {
		if err := c.ensureProjectCleanup(project); err != nil {
			return fmt.Errorf("failed to cleanup project: %v", err)
		}
		return nil
	}

	if err := c.ensureCleanupFinalizerExists(project); err != nil {
		return fmt.Errorf("failed to ensure that the cleanup finalizer exists on the project: %v", err)
	}
	if err := c.ensureProjectOwner(project); err != nil {
		return fmt.Errorf("failed to ensure that the project owner exists in the owners group: %v", err)
	}
	if err := ensureClusterRBACRoleForNamedResource(project.Name, kubermaticv1.ProjectResourceName, kubermaticv1.ProjectKindName, project.GetObjectMeta(), c.masterClusterProvider.kubeClient, c.masterClusterProvider.kubeInformerProvider.KubeInformerFactoryFor(metav1.NamespaceAll).Rbac().V1().ClusterRoles().Lister()); err != nil {
		return fmt.Errorf("failed to ensure that the RBAC Role for the project exists: %v", err)
	}
	if err := ensureClusterRBACRoleBindingForNamedResource(project.Name, kubermaticv1.ProjectResourceName, kubermaticv1.ProjectKindName, project.GetObjectMeta(), c.masterClusterProvider.kubeClient, c.masterClusterProvider.kubeInformerProvider.KubeInformerFactoryFor(metav1.NamespaceAll).Rbac().V1().ClusterRoleBindings().Lister()); err != nil {
		return fmt.Errorf("failed to ensure that the RBAC RoleBinding for the project exists: %v", err)
	}
	if err := c.ensureClusterRBACRoleForResources(); err != nil {
		return fmt.Errorf("failed to ensure that the RBAC ClusterRoles for the project's resources exists: %v", err)
	}
	if err := c.ensureClusterRBACRoleBindingForResources(project.Name); err != nil {
		return fmt.Errorf("failed to ensure that the RBAC ClusterRoleBindings for the project's resources exists: %v", err)
	}
	if err := c.ensureRBACRoleForResources(); err != nil {
		return fmt.Errorf("failed to ensure that the RBAC Roles for the project's resources exists: %v", err)
	}
	if err := c.ensureRBACRoleBindingForResources(project.Name); err != nil {
		return fmt.Errorf("failed to ensure that the RBAC RolesBindings for the project's resources exists: %v", err)
	}
	if err := c.ensureProjectIsInActivePhase(project); err != nil {
		return fmt.Errorf("failed to ensure that the project is set to active: %v", err)
	}

	return nil
}

func (c *projectController) ensureCleanupFinalizerExists(project *kubermaticv1.Project) error {
	if !kuberneteshelper.HasFinalizer(project, CleanupFinalizerName) {
		kuberneteshelper.AddFinalizer(project, CleanupFinalizerName)
		return c.client.Update(c.ctx, project)
	}
	return nil
}

func (c *projectController) ensureProjectIsInActivePhase(project *kubermaticv1.Project) error {
	if project.Status.Phase == kubermaticv1.ProjectInactive {
		project.Status.Phase = kubermaticv1.ProjectActive
		return c.client.Update(c.ctx, project)
	}
	return nil
}

// ensureProjectOwner makes sure that the owner of the project is assign to "owners" group
func (c *projectController) ensureProjectOwner(project *kubermaticv1.Project) error {
	var sharedOwner *kubermaticv1.User
	for _, ref := range project.OwnerReferences {
		if ref.Kind == kubermaticv1.UserKindName {
			var err error
			if sharedOwner, err = c.userLister.Get(ref.Name); err != nil {
				return err
			}
		}
	}
	if sharedOwner == nil {
		return fmt.Errorf("the given project %s doesn't have associated owner/user", project.Name)
	}
	owner := sharedOwner.DeepCopy()

	bindings, err := c.userProjectBindingLister.List(labels.Everything())
	if err != nil {
		return err
	}
	for _, binding := range bindings {
		if binding.Spec.ProjectID == project.Name && strings.EqualFold(binding.Spec.UserEmail, owner.Spec.Email) &&
			binding.Spec.Group == GenerateActualGroupNameFor(project.Name, OwnerGroupNamePrefix) {
			return nil
		}
	}
	ownerBinding := &kubermaticv1.UserProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.ProjectKindName,
					UID:        project.GetUID(),
					Name:       project.Name,
				},
			},
			Name:       rand.String(10),
			Finalizers: []string{CleanupFinalizerName},
		},
		Spec: kubermaticv1.UserProjectBindingSpec{
			UserEmail: owner.Spec.Email,
			ProjectID: project.Name,
			Group:     GenerateActualGroupNameFor(project.Name, OwnerGroupNamePrefix),
		},
	}

	_, err = c.masterClusterProvider.kubermaticClient.KubermaticV1().UserProjectBindings().Create(ownerBinding)
	return err
}

func (c *projectController) ensureClusterRBACRoleForResources() error {
	for _, projectResource := range c.projectResources {
		if len(projectResource.namespace) > 0 {
			continue
		}
		for _, groupPrefix := range AllGroupsPrefixes {

			if projectResource.destination == destinationSeed {
				for _, seedClusterProvider := range c.seedClusterProviders {
					seedClusterRESTClient := seedClusterProvider.kubeClient
					err := ensureClusterRBACRoleForResource(seedClusterRESTClient, groupPrefix, projectResource.gvr.Resource, projectResource.kind, seedClusterProvider.kubeInformerProvider.KubeInformerFactoryFor(metav1.NamespaceAll).Rbac().V1().ClusterRoles().Lister())
					if err != nil {
						return err
					}
				}
			} else {
				err := ensureClusterRBACRoleForResource(c.masterClusterProvider.kubeClient, groupPrefix, projectResource.gvr.Resource, projectResource.kind, c.masterClusterProvider.kubeInformerProvider.KubeInformerFactoryFor(metav1.NamespaceAll).Rbac().V1().ClusterRoles().Lister())
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c *projectController) ensureClusterRBACRoleBindingForResources(projectName string) error {
	for _, projectResource := range c.projectResources {
		if len(projectResource.namespace) > 0 {
			continue
		}
		for _, groupPrefix := range AllGroupsPrefixes {
			groupName := GenerateActualGroupNameFor(projectName, groupPrefix)

			if skip, err := shouldSkipClusterRBACRoleBindingFor(groupName, projectResource.gvr.Resource, kubermaticv1.SchemeGroupVersion.Group, projectName, projectResource.kind); skip {
				continue
			} else if err != nil {
				return err
			}

			if projectResource.destination == destinationSeed {
				for _, seedClusterProvider := range c.seedClusterProviders {
					seedClusterRESTClient := seedClusterProvider.kubeClient
					err := ensureClusterRBACRoleBindingForResource(seedClusterRESTClient, groupName, projectResource.gvr.Resource, seedClusterProvider.kubeInformerProvider.KubeInformerFactoryFor(metav1.NamespaceAll).Rbac().V1().ClusterRoleBindings().Lister())
					if err != nil {
						return err
					}
				}
			} else {
				err := ensureClusterRBACRoleBindingForResource(c.masterClusterProvider.kubeClient, groupName, projectResource.gvr.Resource, c.masterClusterProvider.kubeInformerProvider.KubeInformerFactoryFor(metav1.NamespaceAll).Rbac().V1().ClusterRoleBindings().Lister())
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func ensureClusterRBACRoleForResource(kubeClient kubernetes.Interface, groupName, resource, kind string, rbacLister rbaclister.ClusterRoleLister) error {
	generatedClusterRole, err := generateClusterRBACRoleForResource(groupName, resource, kubermaticv1.SchemeGroupVersion.Group, kind)
	if err != nil {
		return err
	}
	if generatedClusterRole == nil {
		klog.V(4).Infof("skipping ClusterRole generation because the resource for group %q and resource %q will not be created", groupName, resource)
		return nil
	}
	sharedExistingClusterRole, err := rbacLister.Get(generatedClusterRole.Name)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
		// the resource has not been found but for some reason sharedExistingClusterRoles is not nil
		sharedExistingClusterRole = nil
	}
	if sharedExistingClusterRole != nil {
		if equality.Semantic.DeepEqual(sharedExistingClusterRole.Rules, generatedClusterRole.Rules) {
			return nil
		}
		existingClusterRole := sharedExistingClusterRole.DeepCopy()
		existingClusterRole.Rules = generatedClusterRole.Rules
		_, err = kubeClient.RbacV1().ClusterRoles().Update(existingClusterRole)
		return err
	}

	_, err = kubeClient.RbacV1().ClusterRoles().Create(generatedClusterRole)
	return err
}

func ensureClusterRBACRoleBindingForResource(kubeClient kubernetes.Interface, groupName, resource string, rbacLister rbaclister.ClusterRoleBindingLister) error {
	generatedClusterRoleBinding := generateClusterRBACRoleBindingForResource(resource, groupName)

	sharedExistingClusterRoleBinding, err := rbacLister.Get(generatedClusterRoleBinding.Name)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
		// the resource has not been found but for some reason sharedExistingClusterRoleBinding is not nil
		sharedExistingClusterRoleBinding = nil
	}

	if sharedExistingClusterRoleBinding != nil {
		subjectsToAdd := []rbacv1.Subject{}

		for _, generatedRoleBindingSubject := range generatedClusterRoleBinding.Subjects {
			shouldAdd := true
			for _, existingRoleBindingSubject := range sharedExistingClusterRoleBinding.Subjects {
				if equality.Semantic.DeepEqual(existingRoleBindingSubject, generatedRoleBindingSubject) {
					shouldAdd = false
					break
				}
			}
			if shouldAdd {
				subjectsToAdd = append(subjectsToAdd, generatedRoleBindingSubject)
			}
		}

		if len(subjectsToAdd) == 0 {
			return nil
		}

		existingClusterRoleBinding := sharedExistingClusterRoleBinding.DeepCopy()
		existingClusterRoleBinding.Subjects = append(existingClusterRoleBinding.Subjects, subjectsToAdd...)
		_, err = kubeClient.RbacV1().ClusterRoleBindings().Update(existingClusterRoleBinding)
		return err
	}

	_, err = kubeClient.RbacV1().ClusterRoleBindings().Create(generatedClusterRoleBinding)
	return err
}

func (c *projectController) ensureRBACRoleForResources() error {
	for _, projectResource := range c.projectResources {
		if len(projectResource.namespace) == 0 {
			continue
		}
		for _, groupPrefix := range AllGroupsPrefixes {

			if projectResource.destination == destinationSeed {
				for _, seedClusterProvider := range c.seedClusterProviders {
					seedClusterRESTClient := seedClusterProvider.kubeClient
					err := ensureRBACRoleForResource(seedClusterRESTClient,
						groupPrefix,
						projectResource.gvr,
						projectResource.kind,
						projectResource.namespace,
						seedClusterProvider.kubeInformerProvider.KubeInformerFactoryFor(projectResource.namespace).Rbac().V1().Roles().Lister().Roles(projectResource.namespace))
					if err != nil {
						return err
					}
				}
			} else {
				err := ensureRBACRoleForResource(c.masterClusterProvider.kubeClient,
					groupPrefix,
					projectResource.gvr,
					projectResource.kind,
					projectResource.namespace,
					c.masterClusterProvider.kubeInformerProvider.KubeInformerFactoryFor(projectResource.namespace).Rbac().V1().Roles().Lister().Roles(projectResource.namespace))
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func ensureRBACRoleForResource(kubeClient kubernetes.Interface, groupName string, gvr schema.GroupVersionResource, kind string, namespace string, rbacLister rbaclister.RoleNamespaceLister) error {
	generatedRole, err := generateRBACRoleForResource(groupName, gvr.Resource, gvr.Group, kind, namespace)
	if err != nil {
		return err
	}
	if generatedRole == nil {
		klog.V(4).Infof("skipping Role generation because the resource for group %q and resource %q in namespace %q will not be created", groupName, gvr.Resource, namespace)
		return nil
	}
	sharedExistingRole, err := rbacLister.Get(generatedRole.Name)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
		// the resource has not been found but for some reason sharedExistingRoles is not nil
		sharedExistingRole = nil
	}
	if sharedExistingRole != nil {
		if equality.Semantic.DeepEqual(sharedExistingRole.Rules, generatedRole.Rules) {
			return nil
		}
		existingRole := sharedExistingRole.DeepCopy()
		existingRole.Rules = generatedRole.Rules
		_, err = kubeClient.RbacV1().Roles(namespace).Update(existingRole)
		return err
	}

	_, err = kubeClient.RbacV1().Roles(namespace).Create(generatedRole)
	return err
}

func (c *projectController) ensureRBACRoleBindingForResources(projectName string) error {
	for _, projectResource := range c.projectResources {
		if len(projectResource.namespace) == 0 {
			continue
		}
		for _, groupPrefix := range AllGroupsPrefixes {
			groupName := GenerateActualGroupNameFor(projectName, groupPrefix)

			if skip, err := shouldSkipRBACRoleBindingFor(groupName, projectResource.gvr.Resource, kubermaticv1.SchemeGroupVersion.Group, projectName, projectResource.kind, projectResource.namespace); skip {
				continue
			} else if err != nil {
				return err
			}

			if projectResource.destination == destinationSeed {
				for _, seedClusterProvider := range c.seedClusterProviders {
					seedClusterRESTClient := seedClusterProvider.kubeClient
					err := ensureRBACRoleBindingForResource(seedClusterRESTClient,
						groupName,
						projectResource.gvr.Resource,
						projectResource.namespace,
						seedClusterProvider.kubeInformerProvider.KubeInformerFactoryFor(projectResource.namespace).Rbac().V1().RoleBindings().Lister().RoleBindings(projectResource.namespace))
					if err != nil {
						return err
					}
				}
			} else {
				err := ensureRBACRoleBindingForResource(c.masterClusterProvider.kubeClient,
					groupName,
					projectResource.gvr.Resource,
					projectResource.namespace,
					c.masterClusterProvider.kubeInformerProvider.KubeInformerFactoryFor(projectResource.namespace).Rbac().V1().RoleBindings().Lister().RoleBindings(projectResource.namespace))
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func ensureRBACRoleBindingForResource(kubeClient kubernetes.Interface, groupName, resource, namespace string, rbacLister rbaclister.RoleBindingNamespaceLister) error {
	generatedRoleBinding := generateRBACRoleBindingForResource(resource, groupName, namespace)

	sharedExistingRoleBinding, err := rbacLister.Get(generatedRoleBinding.Name)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
		// the resource has not been found but for some reason sharedExistingRoleBinding is not nil
		sharedExistingRoleBinding = nil
	}

	if sharedExistingRoleBinding != nil {
		subjectsToAdd := []rbacv1.Subject{}

		for _, generatedRoleBindingSubject := range generatedRoleBinding.Subjects {
			shouldAdd := true
			for _, existingRoleBindingSubject := range sharedExistingRoleBinding.Subjects {
				if equality.Semantic.DeepEqual(existingRoleBindingSubject, generatedRoleBindingSubject) {
					shouldAdd = false
					break
				}
			}
			if shouldAdd {
				subjectsToAdd = append(subjectsToAdd, generatedRoleBindingSubject)
			}
		}

		if len(subjectsToAdd) == 0 {
			return nil
		}

		existingRoleBinding := sharedExistingRoleBinding.DeepCopy()
		existingRoleBinding.Subjects = append(existingRoleBinding.Subjects, subjectsToAdd...)
		_, err = kubeClient.RbacV1().RoleBindings(namespace).Update(existingRoleBinding)
		return err
	}

	_, err = kubeClient.RbacV1().RoleBindings(namespace).Create(generatedRoleBinding)
	return err
}

// ensureProjectCleanup ensures proper clean up of dependent resources upon deletion
//
// In particular:
// - removes no longer needed Subject from RBAC Binding for project's resources
// - removes cluster resources on master and seed because for them we use Labels not OwnerReferences
// - removes cleanupFinalizer
func (c *projectController) ensureProjectCleanup(project *kubermaticv1.Project) error {
	// cluster resources don't have OwnerReferences set thus we need to manually remove them
	for _, seedClient := range c.seedClientMap {
		var listObj kubermaticv1.ClusterList
		if err := seedClient.List(c.ctx, &listObj); err != nil {
			return err
		}

		for _, cluster := range listObj.Items {
			if clusterProject := cluster.Labels[kubermaticv1.ProjectIDLabelKey]; clusterProject == project.Name {
				if err := seedClient.Delete(c.ctx, &cluster); err != nil {
					return err
				}
			}
		}
	}

	// remove subjects from Cluster RBAC Bindings for project's resources
	for _, projectResource := range c.projectResources {
		if len(projectResource.namespace) > 0 {
			continue
		}
		for _, groupPrefix := range AllGroupsPrefixes {
			groupName := GenerateActualGroupNameFor(project.Name, groupPrefix)
			if skip, err := shouldSkipClusterRBACRoleBindingFor(groupName, projectResource.gvr.Resource, kubermaticv1.SchemeGroupVersion.Group, project.Name, projectResource.kind); skip {
				continue
			} else if err != nil {
				return err
			}

			if projectResource.destination == destinationSeed {
				for _, seedClient := range c.seedClientMap {
					err := cleanUpClusterRBACRoleBindingFor(c.ctx, seedClient, groupName, projectResource.gvr.Resource)
					if err != nil {
						return err
					}
				}
			} else {
				err := cleanUpClusterRBACRoleBindingFor(c.ctx, c.client, groupName, projectResource.gvr.Resource)
				if err != nil {
					return err
				}
			}
		}
	}

	// remove subjects from RBAC Bindings for project's resources
	for _, projectResource := range c.projectResources {
		if len(projectResource.namespace) == 0 {
			continue
		}
		for _, groupPrefix := range AllGroupsPrefixes {
			groupName := GenerateActualGroupNameFor(project.Name, groupPrefix)
			if skip, err := shouldSkipRBACRoleBindingFor(groupName, projectResource.gvr.Resource, kubermaticv1.SchemeGroupVersion.Group, project.Name, projectResource.kind, projectResource.namespace); skip {
				continue
			} else if err != nil {
				return err
			}

			if projectResource.destination == destinationSeed {
				for _, seedClient := range c.seedClientMap {
					err := cleanUpRBACRoleBindingFor(c.ctx, seedClient, groupName, projectResource.gvr.Resource, projectResource.namespace)
					if err != nil {
						return err
					}
				}
			} else {
				err := cleanUpRBACRoleBindingFor(c.ctx, c.client, groupName, projectResource.gvr.Resource, projectResource.namespace)
				if err != nil {
					return err
				}
			}
		}
	}

	kuberneteshelper.RemoveFinalizer(project, CleanupFinalizerName)
	return c.client.Update(c.ctx, project)
}

func cleanUpClusterRBACRoleBindingFor(ctx context.Context, c client.Client, groupName, resource string) error {
	generatedClusterRoleBinding := generateClusterRBACRoleBindingForResource(resource, groupName)
	var sharedExistingClusterRoleBinding rbacv1.ClusterRoleBinding
	key := types.NamespacedName{Name: generatedClusterRoleBinding.Name}
	if err := c.Get(ctx, key, &sharedExistingClusterRoleBinding); err != nil {
		return err
	}

	updatedListOfSubjectes := []rbacv1.Subject{}
	for _, existingRoleBindingSubject := range sharedExistingClusterRoleBinding.Subjects {
		shouldRemove := false
		for _, generatedRoleBindingSubject := range generatedClusterRoleBinding.Subjects {
			if equality.Semantic.DeepEqual(existingRoleBindingSubject, generatedRoleBindingSubject) {
				shouldRemove = true
				break
			}
		}
		if !shouldRemove {
			updatedListOfSubjectes = append(updatedListOfSubjectes, existingRoleBindingSubject)
		}
	}

	existingClusterRoleBinding := sharedExistingClusterRoleBinding.DeepCopy()
	existingClusterRoleBinding.Subjects = updatedListOfSubjectes

	return c.Update(ctx, existingClusterRoleBinding)
}

func cleanUpRBACRoleBindingFor(ctx context.Context, c client.Client, groupName, resource, namespace string) error {
	generatedRoleBinding := generateRBACRoleBindingForResource(resource, groupName, namespace)
	var sharedExistingRoleBinding rbacv1.RoleBinding
	key := types.NamespacedName{Name: generatedRoleBinding.Name, Namespace: namespace}
	if err := c.Get(ctx, key, &sharedExistingRoleBinding); err != nil {
		return err
	}

	updatedListOfSubjectes := []rbacv1.Subject{}
	for _, existingRoleBindingSubject := range sharedExistingRoleBinding.Subjects {
		shouldRemove := false
		for _, generatedRoleBindingSubject := range generatedRoleBinding.Subjects {
			if equality.Semantic.DeepEqual(existingRoleBindingSubject, generatedRoleBindingSubject) {
				shouldRemove = true
				break
			}
		}
		if !shouldRemove {
			updatedListOfSubjectes = append(updatedListOfSubjectes, existingRoleBindingSubject)
		}
	}

	existingRoleBinding := sharedExistingRoleBinding.DeepCopy()
	existingRoleBinding.Subjects = updatedListOfSubjectes
	return c.Update(ctx, existingRoleBinding)
}

func (c *projectController) shouldDeleteProject(project *kubermaticv1.Project) bool {
	return project.DeletionTimestamp != nil && sets.NewString(project.Finalizers...).Has(CleanupFinalizerName)
}

// for some groups we actually don't create ClusterRole
// thus before doing something with ClusterRoleBinding check if the role was generated for the given resource and the group
//
// note: this method will add status to the log file
func shouldSkipClusterRBACRoleBindingFor(groupName, policyResource, policyAPIGroups, projectName, kind string) (bool, error) {
	generatedClusterRole, err := generateClusterRBACRoleForResource(groupName, policyResource, policyAPIGroups, kind)
	if err != nil {
		return false, err
	}
	if generatedClusterRole == nil {
		klog.V(4).Infof("skipping operation on ClusterRoleBinding because corresponding ClusterRole was not(will not be) created for group %q and %q resource for project %q", groupName, policyResource, projectName)
		return true, nil
	}
	return false, nil
}

// for some groups we actually don't create Role
// thus before doing something with RoleBinding check if the role was generated for the given resource and the group
//
// note: this method will add status to the log file
func shouldSkipRBACRoleBindingFor(groupName, policyResource, policyAPIGroups, projectName, kind, namespace string) (bool, error) {
	generatedRole, err := generateRBACRoleForResource(groupName, policyResource, policyAPIGroups, kind, namespace)
	if err != nil {
		return false, err
	}
	if generatedRole == nil {
		klog.V(4).Infof("skipping operation on RoleBinding because corresponding Role was not(will not be) created for group %q and %q resource for project %q in namespace %q", groupName, policyResource, projectName, namespace)
		return true, nil
	}
	return false, nil
}
