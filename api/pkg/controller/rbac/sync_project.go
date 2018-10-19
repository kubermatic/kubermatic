package rbac

import (
	"fmt"

	"github.com/golang/glog"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	rbaclister "k8s.io/client-go/listers/rbac/v1"
)

const (
	cleanupFinalizerName = "kubermatic.io/controller-manager-rbac-cleanup"
)

func (c *Controller) sync(key string) error {
	listerProject, err := c.projectLister.Get(key)
	if err != nil {
		if kerrors.IsNotFound(err) {
			glog.V(2).Infof("project '%s' in queue no longer exists", key)
			return nil
		}
		return err
	}
	project := listerProject.DeepCopy()

	if c.shouldDeleteProject(project) {
		if err := c.ensureProjectCleanup(project); err != nil {
			return fmt.Errorf("failed to cleanup project: %v", err)
		}
		return nil
	}

	if err = c.ensureCleanupFinalizerExists(project); err != nil {
		return fmt.Errorf("failed to ensure that the cleanup finalizer exists on the project: %v", err)
	}
	if err = c.ensureProjectOwner(project); err != nil {
		return fmt.Errorf("failed to ensure that the project owner exists in the owners group: %v", err)
	}
	if err = c.ensureClusterRBACRoleForNamedResource(project.Name, kubermaticv1.ProjectResourceName, kubermaticv1.ProjectKindName, project.GetObjectMeta(), c.masterClusterProvider.kubeClient, c.masterClusterProvider.rbacClusterRoleLister); err != nil {
		return fmt.Errorf("failed to ensure that the RBAC Role for the project exists: %v", err)
	}
	if err = c.ensureClusterRBACRoleBindingForNamedResource(project.Name, kubermaticv1.ProjectResourceName, kubermaticv1.ProjectKindName, project.GetObjectMeta(), c.masterClusterProvider.kubeClient, c.masterClusterProvider.rbacClusterRoleBindingLister); err != nil {
		return fmt.Errorf("failed to ensure that the RBAC RoleBinding for the project exists: %v", err)
	}
	if err = c.ensureClusterRBACRoleForResources(); err != nil {
		return fmt.Errorf("failed to ensure that the RBAC ClusterRole for the project exists: %v", err)
	}
	if err = c.ensureClusterRBACRoleBindingForResources(project.Name); err != nil {
		return fmt.Errorf("failed to ensure that the RBAC ClusterRoleBinding for the project exists: %v", err)
	}
	if err := c.ensureProjectIsInActivePhase(project); err != nil {
		return fmt.Errorf("failed to ensure that the project is set to active: %v", err)
	}

	return nil
}

func (c *Controller) ensureCleanupFinalizerExists(project *kubermaticv1.Project) error {
	var err error
	if !sets.NewString(project.Finalizers...).Has(cleanupFinalizerName) {
		finalizers := sets.NewString(project.Finalizers...)
		finalizers.Insert(cleanupFinalizerName)
		project.Finalizers = finalizers.List()
		project, err = c.masterClusterProvider.kubermaticClient.KubermaticV1().Projects().Update(project)
		if err != nil {
			return err
		}
	}
	return err
}

func (c *Controller) ensureProjectIsInActivePhase(project *kubermaticv1.Project) error {
	if project.Status.Phase == kubermaticv1.ProjectInactive {
		var err error
		project.Status.Phase = kubermaticv1.ProjectActive
		project, err = c.masterClusterProvider.kubermaticClient.KubermaticV1().Projects().Update(project)
		if err != nil {
			return err
		}
	}
	return nil
}

// ensureProjectOwner makes sure that the owner of the project is assign to "owners" group
func (c *Controller) ensureProjectOwner(project *kubermaticv1.Project) error {
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
		if binding.Spec.ProjectID == project.Name && binding.Spec.UserEmail == owner.Spec.Email && binding.Spec.Group == GenerateActualGroupNameFor(project.Name, OwnerGroupNamePrefix) {
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
			Name: rand.String(10),
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

func (c *Controller) ensureClusterRBACRoleForResources() error {
	for _, projectResource := range c.projectResources {
		for _, groupPrefix := range AllGroupsPrefixes {

			if projectResource.destination == destinationSeed {
				for _, seedClusterProvider := range c.seedClusterProviders {
					seedClusterRESTClient := seedClusterProvider.kubeClient
					err := ensureClusterRBACRoleForResource(seedClusterRESTClient, groupPrefix, projectResource.gvr.Resource, projectResource.kind, seedClusterProvider.rbacClusterRoleLister)
					if err != nil {
						return err
					}
				}
			} else {
				err := ensureClusterRBACRoleForResource(c.masterClusterProvider.kubeClient, groupPrefix, projectResource.gvr.Resource, projectResource.kind, c.masterClusterProvider.rbacClusterRoleLister)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c *Controller) ensureClusterRBACRoleBindingForResources(projectName string) error {
	for _, projectResource := range c.projectResources {
		for _, groupPrefix := range AllGroupsPrefixes {
			groupName := GenerateActualGroupNameFor(projectName, groupPrefix)

			if skip, err := shouldSkipRBACRoleBindingFor(groupName, projectResource.gvr.Resource, kubermaticv1.SchemeGroupVersion.Group, projectName, projectResource.kind); skip {
				continue
			} else if err != nil {
				return err
			}

			if projectResource.destination == destinationSeed {
				for _, seedClusterProvider := range c.seedClusterProviders {
					seedClusterRESTClient := seedClusterProvider.kubeClient
					err := ensureClusterRBACRoleBindingForResource(seedClusterRESTClient, groupName, projectResource.gvr.Resource, seedClusterProvider.rbacClusterRoleBindingLister)
					if err != nil {
						return err
					}
				}
			} else {
				err := ensureClusterRBACRoleBindingForResource(c.masterClusterProvider.kubeClient, groupName, projectResource.gvr.Resource, c.masterClusterProvider.rbacClusterRoleBindingLister)
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
		glog.V(5).Infof("skipping ClusterRole generation because the resource for group \"%s\" and resource \"%s\" will not be created", groupName, resource)
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

// ensureProjectCleanup ensures proper clean up of dependent resources upon deletion
//
// In particular:
// - removes no longer needed Subject from RBAC Binding for project's resources
// - removes cluster resources on master and seed because for them we use Labels not OwnerReferences
// - removes cleanupFinalizer
func (c *Controller) ensureProjectCleanup(project *kubermaticv1.Project) error {
	// cluster resources don't have OwnerReferences set thus we need to manually remove them
	for _, clusterProvider := range c.seedClusterProviders {
		if clusterProvider.clusterResourceLister == nil {
			return fmt.Errorf("there is no lister for cluster resources for cluster provider %s", clusterProvider.providerName)
		}
		clusterResources, err := clusterProvider.clusterResourceLister.List(labels.Everything())
		if err != nil {
			return err
		}
		for _, clusterResource := range clusterResources {
			if clusterProject := clusterResource.Labels[kubermaticv1.ProjectIDLabelKey]; clusterProject == project.Name {
				err := clusterProvider.kubermaticClient.KubermaticV1().Clusters().Delete(clusterResource.Name, &metav1.DeleteOptions{})
				if err != nil {
					return err
				}
			}
		}
	}

	// remove subjects from RBAC Bindings for project's resources
	for _, projectResource := range c.projectResources {
		for _, groupPrefix := range AllGroupsPrefixes {
			groupName := GenerateActualGroupNameFor(project.Name, groupPrefix)
			if skip, err := shouldSkipRBACRoleBindingFor(groupName, projectResource.gvr.Resource, kubermaticv1.SchemeGroupVersion.Group, project.Name, projectResource.kind); skip {
				continue
			} else if err != nil {
				return err
			}

			if projectResource.destination == destinationSeed {
				for _, seedClusterProvider := range c.seedClusterProviders {
					seedClusterRESTClient := seedClusterProvider.kubeClient
					err := cleanUpRBACRoleBindingFor(seedClusterRESTClient, groupName, projectResource.gvr.Resource)
					if err != nil {
						return err
					}
				}
			} else {
				err := cleanUpRBACRoleBindingFor(c.masterClusterProvider.kubeClient, groupName, projectResource.gvr.Resource)
				if err != nil {
					return err
				}
			}
		}
	}

	finalizers := sets.NewString(project.Finalizers...)
	finalizers.Delete(cleanupFinalizerName)
	project.Finalizers = finalizers.List()
	_, err := c.masterClusterProvider.kubermaticClient.KubermaticV1().Projects().Update(project)
	return err
}

func cleanUpRBACRoleBindingFor(kubeClient kubernetes.Interface, groupName, resource string) error {
	generatedClusterRoleBinding := generateClusterRBACRoleBindingForResource(resource, groupName)
	sharedExistingClusterRoleBinding, err := kubeClient.RbacV1().ClusterRoleBindings().Get(generatedClusterRoleBinding.Name, metav1.GetOptions{})
	if err != nil {
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
	_, err = kubeClient.RbacV1().ClusterRoleBindings().Update(existingClusterRoleBinding)
	return err
}

func (c *Controller) shouldDeleteProject(project *kubermaticv1.Project) bool {
	return project.DeletionTimestamp != nil && sets.NewString(project.Finalizers...).Has(cleanupFinalizerName)
}

// for some groups we actually don't create ClusterRole
// thus before doing something with ClusterRoleBinding check if the role was generated for the given resource and the group
//
// note: this method will add status to the log file
func shouldSkipRBACRoleBindingFor(groupName, policyResource, policyAPIGroups, projectName, kind string) (bool, error) {
	generatedClusterRole, err := generateClusterRBACRoleForResource(groupName, policyResource, policyAPIGroups, kind)
	if err != nil {
		return false, err
	}
	if generatedClusterRole == nil {
		glog.V(5).Infof("skipping operation on ClusterRoleBinding because corresponding ClusterRole was not(will not be) created for group \"%s\" and \"%s\" resource for project %s", groupName, policyResource, projectName)
		return true, nil
	}
	return false, nil
}
