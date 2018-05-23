package rbac

import (
	"fmt"
	"github.com/golang/glog"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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
	if sharedProject.Labels[kubermaticv1.WorkerNameLabelKey] != c.workerName {
		glog.V(8).Infof("skipping project %s due to different worker assigned to it", key)
		return nil
	}

	project := sharedProject.DeepCopy()
	err = c.ensureProjectOwner(project)
	if err != nil {
		return err
	}
	err = c.ensureProjectRBACRole(project)
	if err != nil {
		return err
	}
	err = c.ensureProjectRBACRoleBinding(project)
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
	generatedRole, err := generateRBACRole("projects", "Project", generateOwnersGroupName(project.Name), project.GroupVersionKind().Group, project.Name)
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
	generatedRoleBinding := generateRBACRoleBinding("Project", generateOwnersGroupName(project.Name))
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
