package rbac

import (
	"fmt"
	"github.com/golang/glog"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

func (c *Controller) sync(key string) error {
	projectFromCache, err := c.projectLister.Get(key)
	if err != nil {
		if kerrors.IsNotFound(err) {
			glog.V(2).Infof("project '%s' in work queue no longer exists", key)
			return nil
		}
		return err
	}
	project := projectFromCache.DeepCopy()

	// ensure owner
	err = c.ensureProjectOwner(project)
	return err
}

// ensureProjectOwner makes sure that the owner of the project is assign to "owners" group
func (c *Controller) ensureProjectOwner(project *kubermaticv1.Project) error {
	var owner *kubermaticv1.User
	for _, ref := range project.OwnerReferences {
		if ref.Kind == "User" {
			var err error
			if owner, err = c.userLister.Get(ref.Name); err != nil {
				return err
			}
		}
	}
	if owner == nil {
		return fmt.Errorf("the given project %s doesn't have associated owner/user", project.Name)
	}

	for _, pg := range owner.Spec.Projects {
		if pg.Name == project.Name {
			return nil
		}
	}
	owner.Spec.Projects = append(owner.Spec.Projects, kubermaticv1.ProjectGroup{Name: project.Name, Group: generateOwnersGroupName(project.Name)})
	_, err := c.kubermaticClient.KubermaticV1().Users().Update(owner)
	return err
}
