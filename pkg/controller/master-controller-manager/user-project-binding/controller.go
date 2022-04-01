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

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
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

const controllerName = "kkp-user-project-binding-controller"

// reconcileSyncProjectBinding reconciles UserProjectBinding objects.
type reconcileSyncProjectBinding struct {
	ctrlruntimeclient.Client

	log *zap.SugaredLogger
}

func Add(mgr manager.Manager, log *zap.SugaredLogger) error {
	r := &reconcileSyncProjectBinding{
		Client: mgr.GetClient(),

		log: log,
	}

	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to UserProjectBinding
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.UserProjectBinding{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	return nil
}

func (r *reconcileSyncProjectBinding) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	projectBinding := &kubermaticv1.UserProjectBinding{}
	if err := r.Get(ctx, request.NamespacedName, projectBinding); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	log := r.log.With("userprojectbinding", projectBinding.Name)
	log.Debug("Reconciling")

	err := r.reconcile(ctx, log, projectBinding)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *reconcileSyncProjectBinding) reconcile(ctx context.Context, log *zap.SugaredLogger, projectBinding *kubermaticv1.UserProjectBinding) error {
	project, err := r.getProjectForBinding(ctx, projectBinding)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return r.removeFinalizerFromBinding(ctx, projectBinding)
		}

		return fmt.Errorf("failed to get project: %w", err)
	}

	user, err := r.getUserForBinding(ctx, projectBinding)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return r.removeFinalizerFromBinding(ctx, projectBinding)
		}

		return fmt.Errorf("failed to get user from binding: %w", err)
	}

	if projectBinding.DeletionTimestamp != nil {
		return r.ensureNotProjectOwnerForBinding(ctx, user, project, projectBinding)
	}

	if err := r.ensureBindingIsOwnedByProject(ctx, project, projectBinding); err != nil {
		return err
	}

	if rbac.ExtractGroupPrefix(projectBinding.Spec.Group) == rbac.OwnerGroupNamePrefix {
		return r.ensureProjectOwnerForBinding(ctx, user, project, projectBinding)
	}

	return r.ensureNotProjectOwnerForBinding(ctx, user, project, projectBinding)
}

func (r *reconcileSyncProjectBinding) ensureBindingIsOwnedByProject(ctx context.Context, project *kubermaticv1.Project, projectBinding *kubermaticv1.UserProjectBinding) error {
	oldBinding := projectBinding.DeepCopy()

	kuberneteshelper.EnsureUniqueOwnerReference(projectBinding, metav1.OwnerReference{
		APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		Kind:       kubermaticv1.ProjectKindName,
		UID:        project.GetUID(),
		Name:       project.Name,
	})

	return r.Patch(ctx, projectBinding, ctrlruntimeclient.MergeFrom(oldBinding))
}

// ensureProjectOwnerForBinding makes sure that the owner reference is set
// on the project resource for the given binding.
func (r *reconcileSyncProjectBinding) ensureProjectOwnerForBinding(ctx context.Context, user *kubermaticv1.User, project *kubermaticv1.Project, projectBinding *kubermaticv1.UserProjectBinding) error {
	oldProject := project.DeepCopy()

	kuberneteshelper.EnsureOwnerReference(project, metav1.OwnerReference{
		APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		Kind:       kubermaticv1.UserKindName,
		UID:        user.GetUID(),
		Name:       user.Name,
	})

	if err := r.Patch(ctx, project, ctrlruntimeclient.MergeFrom(oldProject)); err != nil {
		return err
	}

	return r.addFinalizerToBinding(ctx, projectBinding)
}

// ensureNotProjectOwnerForBinding checks if the owner reference entry is removed
// from the project for the given binding.
func (r *reconcileSyncProjectBinding) ensureNotProjectOwnerForBinding(ctx context.Context, user *kubermaticv1.User, project *kubermaticv1.Project, projectBinding *kubermaticv1.UserProjectBinding) error {
	oldProject := project.DeepCopy()

	kuberneteshelper.RemoveOwnerReferences(project, metav1.OwnerReference{
		APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		Kind:       kubermaticv1.UserKindName,
		UID:        user.GetUID(),
		Name:       user.Name,
	})

	if err := r.Patch(ctx, project, ctrlruntimeclient.MergeFrom(oldProject)); err != nil {
		return err
	}

	return r.removeFinalizerFromBinding(ctx, projectBinding)
}

func (r *reconcileSyncProjectBinding) addFinalizerToBinding(ctx context.Context, projectBinding *kubermaticv1.UserProjectBinding) error {
	return kuberneteshelper.TryAddFinalizer(ctx, r, projectBinding, rbac.CleanupFinalizerName)
}

func (r *reconcileSyncProjectBinding) removeFinalizerFromBinding(ctx context.Context, projectBinding *kubermaticv1.UserProjectBinding) error {
	return kuberneteshelper.TryRemoveFinalizer(ctx, r, projectBinding, rbac.CleanupFinalizerName)
}

func (r *reconcileSyncProjectBinding) getUserForBinding(ctx context.Context, projectBinding *kubermaticv1.UserProjectBinding) (*kubermaticv1.User, error) {
	users := &kubermaticv1.UserList{}
	if err := r.List(ctx, users); err != nil {
		return nil, err
	}

	for _, user := range users.Items {
		if strings.EqualFold(user.Spec.Email, projectBinding.Spec.UserEmail) {
			return &user, nil
		}
	}

	return nil, fmt.Errorf("cannot find user with e-mail %q for project binding %s", projectBinding.Spec.UserEmail, projectBinding.Name)
}

func (r *reconcileSyncProjectBinding) getProjectForBinding(ctx context.Context, projectBinding *kubermaticv1.UserProjectBinding) (*kubermaticv1.Project, error) {
	project := &kubermaticv1.Project{}
	err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Name: projectBinding.Spec.ProjectID}, project)

	return project, err
}
