/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package userprojectbinding

import (
	"context"
	"fmt"
	"strings"

	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const controllerName = "kubermatic_sync_projectbinding_controller"

func Add(mgr manager.Manager) error {
	r := &reconcileSyncProjectBinding{Client: mgr.GetClient()}
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
	ctrlruntimeclient.Client
}

func (r *reconcileSyncProjectBinding) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	projectBinding := &kubermaticv1.UserProjectBinding{}
	if err := r.Get(ctx, request.NamespacedName, projectBinding); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if projectBinding.DeletionTimestamp != nil {
		return reconcile.Result{}, r.ensureNotProjectOwnerForBinding(ctx, projectBinding)
	}
	if rbac.ExtractGroupPrefix(projectBinding.Spec.Group) == rbac.OwnerGroupNamePrefix {
		return reconcile.Result{}, r.ensureProjectOwnerForBinding(ctx, projectBinding)
	}
	return reconcile.Result{}, r.ensureNotProjectOwnerForBinding(ctx, projectBinding)

}

// ensureProjectOwnerForBinding makes sure that the owner reference is set on the project resource for the given binding
func (r *reconcileSyncProjectBinding) ensureProjectOwnerForBinding(ctx context.Context, projectBinding *kubermaticv1.UserProjectBinding) error {
	project, err := r.getProjectForBinding(ctx, projectBinding)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return r.removeFinalizerFromBinding(ctx, projectBinding)
		}
		return err
	}

	for _, ref := range project.OwnerReferences {
		if ref.Kind == kubermaticv1.UserKindName {
			existingOwner := &kubermaticv1.User{}
			err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: metav1.NamespaceAll, Name: ref.Name}, existingOwner)
			if err != nil {
				return err
			}
			if strings.EqualFold(existingOwner.Spec.Email, projectBinding.Spec.UserEmail) {
				return nil
			}
		}
	}

	userObject, err := r.userForBinding(ctx, projectBinding)
	if err != nil {
		return err
	}
	project.OwnerReferences = append(project.OwnerReferences, metav1.OwnerReference{
		APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		Kind:       kubermaticv1.UserKindName,
		UID:        userObject.GetUID(),
		Name:       userObject.Name,
	})

	return r.Update(ctx, project)
}

// ensureNotProjectOwnerForBinding checks if the owner reference entry is removed from the project for the given binding
func (r *reconcileSyncProjectBinding) ensureNotProjectOwnerForBinding(ctx context.Context, projectBinding *kubermaticv1.UserProjectBinding) error {
	project, err := r.getProjectForBinding(ctx, projectBinding)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return r.removeFinalizerFromBinding(ctx, projectBinding)
		}
		return err
	}

	newOwnerRef := []metav1.OwnerReference{}
	for _, ref := range project.OwnerReferences {
		if ref.Kind == kubermaticv1.UserKindName {
			existingOwner := &kubermaticv1.User{}
			if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: metav1.NamespaceAll, Name: ref.Name}, existingOwner); err != nil {
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
	err = r.Update(ctx, project)
	if err != nil {
		return err
	}
	return r.removeFinalizerFromBinding(ctx, projectBinding)
}

func (r *reconcileSyncProjectBinding) removeFinalizerFromBinding(ctx context.Context, projectBinding *kubermaticv1.UserProjectBinding) error {
	if kuberneteshelper.HasFinalizer(projectBinding, rbac.CleanupFinalizerName) {
		kuberneteshelper.RemoveFinalizer(projectBinding, rbac.CleanupFinalizerName)
		return r.Update(ctx, projectBinding)

	}
	return nil
}

func (r *reconcileSyncProjectBinding) userForBinding(ctx context.Context, projectBinding *kubermaticv1.UserProjectBinding) (*kubermaticv1.User, error) {
	users := &kubermaticv1.UserList{}
	if err := r.List(ctx, users); err != nil {
		return nil, err
	}

	for _, user := range users.Items {
		if strings.EqualFold(user.Spec.Email, projectBinding.Spec.UserEmail) {
			return &user, nil
		}
	}

	return nil, fmt.Errorf("a user resource for the given project binding %s not found", projectBinding.Name)
}

func (r *reconcileSyncProjectBinding) getProjectForBinding(ctx context.Context, projectBinding *kubermaticv1.UserProjectBinding) (*kubermaticv1.Project, error) {
	projectFromCache := &kubermaticv1.Project{}
	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: metav1.NamespaceAll, Name: projectBinding.Spec.ProjectID}, projectFromCache); err != nil {
		return nil, err
	}
	return projectFromCache, nil
}
