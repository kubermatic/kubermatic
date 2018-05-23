package rbac

import (
	"fmt"

	"github.com/golang/glog"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"strings"

	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	cleanupFinalizerName = "kubermatic.io/controller-manager-rbac-cleanup"
)

func (c *Controller) sync(key string) error {
	sharedProject, err := c.projectLister.Get(key)
	if err != nil {
		if kerrors.IsNotFound(err) {
			glog.V(2).Infof("project '%s' in work queue no longer exists", key)
			return nil
		}
		return err
	}
	if c.shouldSkipProject(sharedProject) {
		glog.V(8).Infof("skipping project %s due to different worker (%s) assigned to it", key, c.workerName)
		return nil
	}
	if c.shouldDeleteProject(sharedProject) {
		return c.ensureProjectCleanup(sharedProject)
	}

	project := sharedProject.DeepCopy()
	err = c.ensureProjectInitialized(project)
	if err != nil {
		return err
	}
	err = c.ensureProjectOwner(project)
	if err != nil {
		return err
	}
	err = c.ensureProjectRBACRole(project)
	if err != nil {
		return err
	}
	err = c.ensureProjectRBACRoleBinding(project)
	if err != nil {
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
		project, err = c.kubermaticClient.KubermaticV1().Projects().Update(project)
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
		project, err = c.kubermaticClient.KubermaticV1().Projects().Update(project)
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
		if ref.Kind == "User" {
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
		if pg.Name == project.Name && pg.Group == generateOwnersGroupName(project.Name) {
			return nil
		}
	}
	owner.Spec.Projects = append(owner.Spec.Projects, kubermaticv1.ProjectGroup{Name: project.Name, Group: generateOwnersGroupName(project.Name)})
	_, err := c.kubermaticClient.KubermaticV1().Users().Update(owner)
	return err
}

// ensureProjectRBACRole makes sure that desired RBAC roles are created
func (c *Controller) ensureProjectRBACRole(project *kubermaticv1.Project) error {
	generatedRole, err := generateRBACRole(
		"projects",
		"Project",
		generateOwnersGroupName(project.Name),
		project.GroupVersionKind().Group, project.Name,
		metav1.OwnerReference{
			APIVersion: project.APIVersion,
			Kind:       project.Kind,
			UID:        project.GetUID(),
			Name:       project.Name,
		},
	)
	if err != nil {
		return err
	}
	sharedExistingRole, err := c.rbacClusterRoleLister.Get(generatedRole.Name)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
	}

	// make sure that existing rbac role has appropriate rules/policies
	if sharedExistingRole != nil {
		if equality.Semantic.DeepEqual(sharedExistingRole.Rules, generatedRole.Rules) {
			return nil
		}
		existingRole := sharedExistingRole.DeepCopy()
		existingRole.Rules = generatedRole.Rules
		_, err = c.kubeClient.RbacV1().ClusterRoles().Update(existingRole)
		return err
	}

	_, err = c.kubeClient.RbacV1().ClusterRoles().Create(generatedRole)
	return err
}

// ensureProjectRBACRoleBinding makes sure that project's groups are bind to appropriate roles
func (c *Controller) ensureProjectRBACRoleBinding(project *kubermaticv1.Project) error {
	generatedRoleBinding := generateRBACRoleBinding(
		"Project",
		generateOwnersGroupName(project.Name),
		metav1.OwnerReference{
			APIVersion: project.APIVersion,
			Kind:       project.Kind,
			UID:        project.GetUID(),
			Name:       project.Name,
		},
	)
	sharedExistingRoleBinding, err := c.rbacClusterRoleBindingLister.Get(generatedRoleBinding.Name)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
	}
	if sharedExistingRoleBinding != nil {
		if equality.Semantic.DeepEqual(sharedExistingRoleBinding.Subjects, generatedRoleBinding.Subjects) {
			return nil
		}
		existingRoleBinding := sharedExistingRoleBinding.DeepCopy()
		existingRoleBinding.Subjects = generatedRoleBinding.Subjects
		_, err = c.kubeClient.RbacV1().ClusterRoleBindings().Update(existingRoleBinding)
		return err
	}
	_, err = c.kubeClient.RbacV1().ClusterRoleBindings().Create(generatedRoleBinding)
	return err
}

// ensureProjectCleanup ensures proper clean up of dependent resources upon deletion
//
// In particular:
// - removes project/group reference from users object
// - removes cleanupFinalizer
func (c *Controller) ensureProjectCleanup(project *kubermaticv1.Project) error {
	sharedUsers, err := c.userLister.List(labels.Everything())
	if err != nil {
		return err
	}
	for _, sharedUser := range sharedUsers {
		updatedProjectGroup := []kubermaticv1.ProjectGroup{}
		for _, pg := range sharedUser.Spec.Projects {
			if strings.HasSuffix(pg.Group, project.Name) {
				continue
			}
			updatedProjectGroup = append(updatedProjectGroup, pg)
		}
		if len(updatedProjectGroup) != len(sharedUser.Spec.Projects) {
			user := sharedUser.DeepCopy()
			user.Spec.Projects = updatedProjectGroup
			_, err = c.kubermaticClient.KubermaticV1().Users().Update(user)
			if err != nil {
				return err
			}
		}
	}

	finalizers := sets.NewString(project.Finalizers...)
	finalizers.Delete(cleanupFinalizerName)
	project.Finalizers = finalizers.List()
	_, err = c.kubermaticClient.KubermaticV1().Projects().Update(project)
	return nil
}

func (c *Controller) shouldSkipProject(project *kubermaticv1.Project) bool {
	return project.Labels[kubermaticv1.WorkerNameLabelKey] != c.workerName
}

func (c *Controller) shouldDeleteProject(project *kubermaticv1.Project) bool {
	return project.DeletionTimestamp != nil && sets.NewString(project.Finalizers...).Has(cleanupFinalizerName)
}
