package rbac

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (c *Controller) syncDependant(item *dependantQueueItem) error {
	projects, err := c.projectLister.List(labels.Everything())
	if err != nil {
		return err
	}

	owners := item.metaObject.GetOwnerReferences()
	var project *kubermaticv1.Project
	for _, p := range projects {
		for _, owner := range owners {
			if owner.APIVersion == kubermaticv1.SchemeGroupVersion.String() && owner.Kind == "Project" && owner.UID == p.UID {
				project = p
				break
			}
		}
		if project != nil {
			break
		}
	}
	if project == nil {
		return fmt.Errorf("unable to find owing project for the object name = %s, gvr = %s", item.metaObject.GetName(), item.gvr.String())
	}

	if err = c.ensureDependantRBACRole(project, item.gvr.Resource, item.kind, item.metaObject); err != nil {
		return err
	}
	return c.ensureDependantRBACRoleBindings(project, item.kind, item.metaObject)
}

func (c *Controller) ensureDependantRBACRole(project *kubermaticv1.Project, resources string, kind string, object metav1.Object) error {
	for _, groupName := range allGroups {
		generatedRole, err := generateRBACRole(
			kind,
			generateGroupNameFor(project.Name, groupName),
			resources,
			kubermaticv1.SchemeGroupVersion.Group,
			object.GetName(),
			metav1.OwnerReference{
				APIVersion: kubermaticv1.SchemeGroupVersion.String(),
				Kind:       kind,
				UID:        object.GetUID(),
				Name:       object.GetName(),
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
				continue
			}
			existingRole := sharedExistingRole.DeepCopy()
			existingRole.Rules = generatedRole.Rules
			if _, err = c.kubeClient.RbacV1().ClusterRoles().Update(existingRole); err != nil {
				return err
			}
			continue
		}

		if _, err = c.kubeClient.RbacV1().ClusterRoles().Create(generatedRole); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) ensureDependantRBACRoleBindings(project *kubermaticv1.Project, kind string, object metav1.Object) error {
	for _, groupName := range allGroups {
		generatedRoleBinding := generateRBACRoleBinding(
			kind,
			object.GetName(),
			generateGroupNameFor(project.Name, groupName),
			metav1.OwnerReference{
				APIVersion: kubermaticv1.SchemeGroupVersion.String(),
				Kind:       kind,
				UID:        object.GetUID(),
				Name:       object.GetName(),
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
				continue
			}
			existingRoleBinding := sharedExistingRoleBinding.DeepCopy()
			existingRoleBinding.Subjects = generatedRoleBinding.Subjects
			if _, err = c.kubeClient.RbacV1().ClusterRoleBindings().Update(existingRoleBinding); err != nil {
				return err
			}
			continue
		}
		if _, err = c.kubeClient.RbacV1().ClusterRoleBindings().Create(generatedRoleBinding); err != nil {
			return err
		}
	}
	return nil
}
