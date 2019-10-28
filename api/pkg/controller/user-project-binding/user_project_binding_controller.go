package userprojectbinding

import (
	"context"
	"fmt"
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const controllerName = "kubermatic_sync_projectbinding_controller"

func Add(mgr manager.Manager) error {
	r := &reconcileSyncProjectBinding{Client: mgr.GetClient(), ctx: context.TODO()}
	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to UserProjectBinding
	err = c.Watch(&source.Kind{Type: &kubermaticv1.UserProjectBinding{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	return nil
}

// reconcileSyncProjectBinding reconciles UserProjectBinding objects
type reconcileSyncProjectBinding struct {
	ctx context.Context
	client.Client
}

func (r *reconcileSyncProjectBinding) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	projectBinding := &kubermaticv1.UserProjectBinding{}
	if err := r.Get(r.ctx, request.NamespacedName, projectBinding); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if projectBinding.DeletionTimestamp != nil {
		return reconcile.Result{}, r.ensureNotProjectOwnerForBinding(projectBinding)
	}
	if rbac.ExtractGroupPrefix(projectBinding.Spec.Group) == rbac.OwnerGroupNamePrefix {
		return reconcile.Result{}, r.ensureProjectOwnerForBinding(projectBinding)
	}
	return reconcile.Result{}, r.ensureNotProjectOwnerForBinding(projectBinding)

}

// ensureProjectOwnerForBinding makes sure that the owner reference is set on the project resource for the given binding
func (r *reconcileSyncProjectBinding) ensureProjectOwnerForBinding(projectBinding *kubermaticv1.UserProjectBinding) error {
	project, err := r.getProjectForBinding(projectBinding)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return r.removeFinalizerFromBinding(projectBinding)
		}
		return err
	}

	for _, ref := range project.OwnerReferences {
		if ref.Kind == kubermaticv1.UserKindName {
			existingOwner := &kubermaticv1.User{}
			err := r.Get(r.ctx, client.ObjectKey{Namespace: metav1.NamespaceAll, Name: ref.Name}, existingOwner)
			if err != nil {
				return err
			}
			if strings.EqualFold(existingOwner.Spec.Email, projectBinding.Spec.UserEmail) {
				return nil
			}
		}
	}

	userObject, err := r.userForBinding(projectBinding)
	if err != nil {
		return err
	}
	project.OwnerReferences = append(project.OwnerReferences, metav1.OwnerReference{
		APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		Kind:       kubermaticv1.UserKindName,
		UID:        userObject.GetUID(),
		Name:       userObject.Name,
	})

	return r.Update(r.ctx, project)
}

// ensureNotProjectOwnerForBinding checks if the owner reference entry is removed from the project for the given binding
func (r *reconcileSyncProjectBinding) ensureNotProjectOwnerForBinding(projectBinding *kubermaticv1.UserProjectBinding) error {
	project, err := r.getProjectForBinding(projectBinding)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return r.removeFinalizerFromBinding(projectBinding)
		}
		return err
	}

	newOwnerRef := []metav1.OwnerReference{}
	for _, ref := range project.OwnerReferences {
		if ref.Kind == kubermaticv1.UserKindName {
			existingOwner := &kubermaticv1.User{}
			if err := r.Get(r.ctx, client.ObjectKey{Namespace: metav1.NamespaceAll, Name: ref.Name}, existingOwner); err != nil {
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
	err = r.Update(r.ctx, project)
	if err != nil {
		return err
	}
	return r.removeFinalizerFromBinding(projectBinding)
}

func (r *reconcileSyncProjectBinding) removeFinalizerFromBinding(projectBinding *kubermaticv1.UserProjectBinding) error {
	if kuberneteshelper.HasFinalizer(projectBinding, rbac.CleanupFinalizerName) {
		kuberneteshelper.RemoveFinalizer(projectBinding, rbac.CleanupFinalizerName)
		return r.Update(r.ctx, projectBinding)

	}
	return nil
}

func (r *reconcileSyncProjectBinding) userForBinding(projectBinding *kubermaticv1.UserProjectBinding) (*kubermaticv1.User, error) {

	users := &kubermaticv1.UserList{}
	if err := r.List(r.ctx, users); err != nil {
		return nil, err
	}

	for _, user := range users.Items {
		if strings.EqualFold(user.Spec.Email, projectBinding.Spec.UserEmail) {
			return &user, nil
		}
	}

	return nil, fmt.Errorf("a user resource for the given project binding %s not found", projectBinding.Name)
}

func (r *reconcileSyncProjectBinding) getProjectForBinding(projectBinding *kubermaticv1.UserProjectBinding) (*kubermaticv1.Project, error) {
	projectFromCache := &kubermaticv1.Project{}
	if err := r.Get(r.ctx, client.ObjectKey{Namespace: metav1.NamespaceAll, Name: projectBinding.Spec.ProjectID}, projectFromCache); err != nil {
		return nil, err
	}
	return projectFromCache, nil
}
