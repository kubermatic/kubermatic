package rbac

import (
	"fmt"
	"strings"

	"github.com/golang/glog"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
)

func (c *Controller) syncProjectBindings(key string) error {
	projectBindingFromCache, err := c.userProjectBindingLister.Get(key)
	if err != nil {
		if kerrors.IsNotFound(err) {
			glog.V(2).Infof("user project binding '%s' in queue no longer exists", key)
			return nil
		}
		return err
	}
	projectBinding := projectBindingFromCache.DeepCopy()
	if projectBinding.DeletionTimestamp != nil {
		return removeFinalizerFromBinding(projectBinding, c.masterClusterProvider.kubermaticClient)
	}
	if ExtractGroupPrefix(projectBinding.Spec.Group) == OwnerGroupNamePrefix {
		return c.ensureProjectOwnerForBinding(projectBinding)
	}
	return c.ensureNotProjectOwnerForBinding(projectBinding)
}

// ensureProjectOwnerForBinding makes sure that the owner reference is set on the project resource for the given binding
func (c *Controller) ensureProjectOwnerForBinding(projectBinding *kubermaticv1.UserProjectBinding) error {
	project, err := getProjectForBinding(c.projectLister, projectBinding)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return removeFinalizerFromBinding(projectBinding, c.masterClusterProvider.kubermaticClient)
		}
		return err
	}

	for _, ref := range project.OwnerReferences {
		if ref.Kind == kubermaticv1.UserKindName {
			existingOwner, err := c.userLister.Get(ref.Name)
			if err != nil {
				return err
			}
			if strings.EqualFold(existingOwner.Spec.Email, projectBinding.Spec.UserEmail) {
				return nil
			}
		}
	}

	userObject, err := userForBinding(c.userLister, projectBinding)
	if err != nil {
		return err
	}
	project.OwnerReferences = append(project.OwnerReferences, metav1.OwnerReference{
		APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		Kind:       kubermaticv1.UserKindName,
		UID:        userObject.GetUID(),
		Name:       userObject.Name,
	})

	_, err = c.masterClusterProvider.kubermaticClient.KubermaticV1().Projects().Update(project)
	return err
}

// ensureNotProjectOwnerForBinding checks if the owner reference entry is removed from the project for the given binding
func (c *Controller) ensureNotProjectOwnerForBinding(projectBinding *kubermaticv1.UserProjectBinding) error {
	project, err := getProjectForBinding(c.projectLister, projectBinding)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return removeFinalizerFromBinding(projectBinding, c.masterClusterProvider.kubermaticClient)
		}
		return err
	}

	newOwnerRef := []metav1.OwnerReference{}
	for _, ref := range project.OwnerReferences {
		if ref.Kind == kubermaticv1.UserKindName {
			existingOwner, err := c.userLister.Get(ref.Name)
			if err != nil {
				return err
			}
			if !strings.EqualFold(existingOwner.Spec.Email, projectBinding.Spec.UserEmail) {
				newOwnerRef = append(newOwnerRef, ref)
			}
		}
	}

	if len(newOwnerRef) == len(project.OwnerReferences) {
		return nil
	}

	project.OwnerReferences = newOwnerRef
	_, err = c.masterClusterProvider.kubermaticClient.KubermaticV1().Projects().Update(project)
	if err != nil {
		return err
	}
	return removeFinalizerFromBinding(projectBinding, c.masterClusterProvider.kubermaticClient)
}

func removeFinalizerFromBinding(projectBinding *kubermaticv1.UserProjectBinding, client kubermaticclientset.Interface) error {
	if sets.NewString(projectBinding.Finalizers...).Has(CleanupFinalizerName) {
		finalizers := sets.NewString(projectBinding.Finalizers...)
		finalizers.Delete(CleanupFinalizerName)
		projectBinding.Finalizers = finalizers.List()
		_, err := client.KubermaticV1().UserProjectBindings().Update(projectBinding)
		return err

	}
	return nil
}

func userForBinding(userLister kubermaticv1lister.UserLister, projectBinding *kubermaticv1.UserProjectBinding) (*kubermaticv1.User, error) {
	users, err := userLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		if strings.EqualFold(user.Spec.Email, projectBinding.Spec.UserEmail) {
			return user, nil
		}
	}

	return nil, fmt.Errorf("a user resource for the given project binding %s not found", projectBinding.Name)
}

func getProjectForBinding(projectLister kubermaticv1lister.ProjectLister, projectBinding *kubermaticv1.UserProjectBinding) (*kubermaticv1.Project, error) {
	for _, ref := range projectBinding.OwnerReferences {
		if ref.Kind == kubermaticv1.ProjectKindName {
			var err error
			var projectFromCache *kubermaticv1.Project
			if projectFromCache, err = projectLister.Get(ref.Name); err != nil {
				return nil, err
			}
			return projectFromCache.DeepCopy(), nil
		}
	}

	return nil, fmt.Errorf("the given project binding %s doesn't have associated project", projectBinding.Name)
}
