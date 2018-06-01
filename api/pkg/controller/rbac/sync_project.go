package rbac

import (
	"fmt"
	"strings"

	"github.com/golang/glog"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
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
			glog.V(2).Infof("project '%s' in work projectQueue no longer exists", key)
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
	if err = c.ensureProjectInitialized(project); err != nil {
		return err
	}
	if err = c.ensureProjectOwner(project); err != nil {
		return err
	}
	if err = c.ensureRBACRoleFor(project.Name, kubermaticv1.ProjectResourceName, kubermaticv1.ProjectKindName, project.GetObjectMeta()); err != nil {
		return err
	}
	if err = c.ensureRBACRoleBindingFor(project.Name, kubermaticv1.ProjectKindName, project.GetObjectMeta()); err != nil {
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
		if pg.Name == project.Name && pg.Group == generateGroupNameFor(project.Name, ownerGroupName) {
			return nil
		}
	}
	owner.Spec.Projects = append(owner.Spec.Projects, kubermaticv1.ProjectGroup{Name: project.Name, Group: generateGroupNameFor(project.Name, ownerGroupName)})
	_, err := c.kubermaticClient.KubermaticV1().Users().Update(owner)
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
			if _, err = c.kubermaticClient.KubermaticV1().Users().Update(user); err != nil {
				return err
			}
		}
	}

	finalizers := sets.NewString(project.Finalizers...)
	finalizers.Delete(cleanupFinalizerName)
	project.Finalizers = finalizers.List()
	_, err = c.kubermaticClient.KubermaticV1().Projects().Update(project)
	return err
}

func (c *Controller) shouldSkipProject(project *kubermaticv1.Project) bool {
	return project.Labels[kubermaticv1.WorkerNameLabelKey] != c.workerName
}

func (c *Controller) shouldDeleteProject(project *kubermaticv1.Project) bool {
	return project.DeletionTimestamp != nil && sets.NewString(project.Finalizers...).Has(cleanupFinalizerName)
}
