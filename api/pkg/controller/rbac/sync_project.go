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
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	rbaclister "k8s.io/client-go/listers/rbac/v1"
)

const (
	cleanupFinalizerName = "kubermatic.io/controller-manager-rbac-cleanup"
)

func (c *Controller) sync(key string) error {
	sharedProject, err := c.projectLister.Get(key)
	if err != nil {
		if kerrors.IsNotFound(err) {
			glog.V(2).Infof("project '%s' in work projectQueue no longer exists", key)
			return nil
		}
		return err
	}
	if c.shouldSkipProject(sharedProject) {
		glog.V(8).Infof("skipping project %s due to different worker than (%s) is assigned to it", key, c.workerName)
		return nil
	}
	if c.shouldDeleteProject(sharedProject) {
		return c.ensureProjectCleanup(sharedProject)
	}

	project := sharedProject.DeepCopy()
	if err = c.ensureProjectInitialized(project); err != nil {
		return err
	}
	if err = c.ensureProjectOwner(project); err != nil {
		return err
	}
	if err = c.ensureClusterRBACRoleForNamedResource(project.Name, kubermaticv1.ProjectResourceName, kubermaticv1.ProjectKindName, project.GetObjectMeta(), c.kubeMasterClient, c.rbacClusterRoleMasterLister); err != nil {
		return err
	}
	if err = c.ensureClusterRBACRoleBindingForNamedResource(project.Name, kubermaticv1.ProjectKindName, project.GetObjectMeta(), c.kubeMasterClient, c.rbacClusterRoleBindingMasterLister); err != nil {
		return err
	}
	if err = c.ensureClusterRBACRoleForResources(c.rbacClusterRoleMasterLister); err != nil {
		return err
	}
	if err = c.ensureClusterRBACRoleBindingForResources(project.Name, c.rbacClusterRoleBindingMasterLister); err != nil {
		return err
	}
	err = c.ensureProjectIsInActivePhase(project)
	return err
}

func (c *Controller) ensureProjectInitialized(project *kubermaticv1.Project) error {
	var err error
	if !sets.NewString(project.Finalizers...).Has(cleanupFinalizerName) {
		finalizers := sets.NewString(project.Finalizers...)
		finalizers.Insert(cleanupFinalizerName)
		project.Finalizers = finalizers.List()
		project, err = c.kubermaticMasterClient.KubermaticV1().Projects().Update(project)
		if err != nil {
			return err
		}
	}
	return err
}

func (c *Controller) ensureProjectIsInActivePhase(project *kubermaticv1.Project) error {
	var err error
	if project.Status.Phase != kubermaticv1.ProjectActive {
		project.Status.Phase = kubermaticv1.ProjectActive
		project, err = c.kubermaticMasterClient.KubermaticV1().Projects().Update(project)
		if err != nil {
			return err
		}
	}
	return err
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

	for _, pg := range owner.Spec.Projects {
		if pg.Name == project.Name && pg.Group == GenerateActualGroupNameFor(project.Name, OwnerGroupNamePrefix) {
			return nil
		}
	}
	owner.Spec.Projects = append(owner.Spec.Projects, kubermaticv1.ProjectGroup{Name: project.Name, Group: GenerateActualGroupNameFor(project.Name, OwnerGroupNamePrefix)})
	_, err := c.kubermaticMasterClient.KubermaticV1().Users().Update(owner)
	return err
}

func (c *Controller) ensureClusterRBACRoleForResources(rbacClusterRoleLister rbaclister.ClusterRoleLister) error {
	for _, projectResource := range c.projectResources {
		for _, groupPrefix := range AllGroupsPrefixes {
			err := ensureClusterRBACRoleForResource(c.kubeMasterClient, groupPrefix, projectResource.gvr.Resource, rbacClusterRoleLister)
			if err != nil {
				return err
			}

			if projectResource.destination == destinationSeed {
				for _, seedClusterProvider := range c.seedClusterProviders {
					seedClusterRESTClient := seedClusterProvider.kubeClient
					err := ensureClusterRBACRoleForResource(seedClusterRESTClient, groupPrefix, projectResource.gvr.Resource, seedClusterProvider.rbacClusterRoleLister)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (c *Controller) ensureClusterRBACRoleBindingForResources(projectName string, rbacClusterRoleBindingLister rbaclister.ClusterRoleBindingLister) error {
	for _, projectResource := range c.projectResources {
		for _, groupPrefix := range AllGroupsPrefixes {
			groupName := GenerateActualGroupNameFor(projectName, groupPrefix)

			// for some resources we actually don't create ClusterRole
			// thus before creating ClusterRoleBinding check if the role was generated for the given resource and the group
			generatedClusterRole, err := generateClusterRBACRoleForResource(groupName, projectResource.gvr.Resource, kubermaticv1.SchemeGroupVersion.Group)
			if err != nil {
				return err
			}
			if generatedClusterRole == nil {
				glog.V(5).Infof("skipping ClusterRoleBinding generation because corresponding ClusterRole will not be created for group \"%s\" and \"%s\" resource for project %s", groupPrefix, projectResource.gvr.Resource, projectName)
				continue
			}

			err = ensureClusterRBACRoleBindingForResource(c.kubeMasterClient, groupName, projectResource.gvr.Resource, rbacClusterRoleBindingLister)
			if err != nil {
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
			}
		}
	}
	return nil
}

func ensureClusterRBACRoleForResource(kubeClient kubernetes.Interface, groupName, resource string, rbacLister rbaclister.ClusterRoleLister) error {
	generatedClusterRole, err := generateClusterRBACRoleForResource(groupName, resource, kubermaticv1.SchemeGroupVersion.Group)
	if err != nil {
		return err
	}
	if generatedClusterRole == nil {
		glog.V(5).Infof("skipping ClusterRole generation because ClusterRole for group \"%s\" and resource \"%s\" will not be created", groupName, resource)
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
// - removes project/group reference from users object
// - removes no longer needed Subject from RBAC Binding for project's resources
// - removes cleanupFinalizer
func (c *Controller) ensureProjectCleanup(project *kubermaticv1.Project) error {
	sharedUsers, err := c.userLister.List(labels.Everything())
	if err != nil {
		return err
	}
	for _, sharedUser := range sharedUsers {
		updatedProjectGroup := []kubermaticv1.ProjectGroup{}
		for _, pg := range sharedUser.Spec.Projects {
			if pg.Name == project.Name {
				continue
			}
			updatedProjectGroup = append(updatedProjectGroup, pg)
		}
		if len(updatedProjectGroup) != len(sharedUser.Spec.Projects) {
			user := sharedUser.DeepCopy()
			user.Spec.Projects = updatedProjectGroup
			if _, err = c.kubermaticMasterClient.KubermaticV1().Users().Update(user); err != nil {
				return err
			}
		}
	}

	// remove subjects from RBAC Bindings for project's resources
	for _, projectResource := range c.projectResources {
		for _, groupPrefix := range AllGroupsPrefixes {
			groupName := GenerateActualGroupNameFor(project.Name, groupPrefix)
			err := cleanUpRBACRoleBindingFor(c.kubeMasterClient, groupName, projectResource.gvr.Resource)
			if err != nil {
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
			}
		}
	}

	finalizers := sets.NewString(project.Finalizers...)
	finalizers.Delete(cleanupFinalizerName)
	project.Finalizers = finalizers.List()
	_, err = c.kubermaticMasterClient.KubermaticV1().Projects().Update(project)
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

func (c *Controller) shouldSkipProject(project *kubermaticv1.Project) bool {
	return project.Labels[kubermaticv1.WorkerNameLabelKey] != c.workerName
}

func (c *Controller) shouldDeleteProject(project *kubermaticv1.Project) bool {
	return project.DeletionTimestamp != nil && sets.NewString(project.Finalizers...).Has(cleanupFinalizerName)
}
