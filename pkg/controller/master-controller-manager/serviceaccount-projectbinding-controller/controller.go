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

package serviceaccountprojectbindingcontroller

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	serviceaccount "k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "kkp-serviceaccount-projectbinding-controller"
)

// reconcileServiceAccountProjectBinding reconciles User objects.
type reconcileServiceAccountProjectBinding struct {
	ctrlruntimeclient.Client

	recorder record.EventRecorder
	log      *zap.SugaredLogger
}

func Add(mgr manager.Manager, log *zap.SugaredLogger) error {
	r := &reconcileServiceAccountProjectBinding{
		Client: mgr.GetClient(),

		recorder: mgr.GetEventRecorderFor(controllerName),
		log:      log,
	}
	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to User
	isServiceAccount := predicate.NewPredicateFuncs(func(object ctrlruntimeclient.Object) bool {
		return kubermaticv1helper.IsProjectServiceAccount(object.GetName())
	})

	if err = c.Watch(&source.Kind{Type: &kubermaticv1.User{}}, &handler.EnqueueRequestForObject{}, isServiceAccount); err != nil {
		return err
	}

	// Notice when projects appear, then enqueue all service account users in that project
	enqueueRelatedUsers := handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		userList := &kubermaticv1.UserList{}
		if err := mgr.GetClient().List(context.Background(), userList); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list Users: %w", err))
			return []reconcile.Request{}
		}

		requests := []reconcile.Request{}
		for _, user := range userList.Items {
			if kubermaticv1helper.IsProjectServiceAccount(user.Spec.Email) && user.Spec.Project == a.GetName() {
				requests = append(requests, reconcile.Request{
					NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(&user),
				})
			}
		}

		return requests
	})

	// Only react to new projects
	onlyNewProjects := predicate.Funcs{
		CreateFunc: func(ce event.CreateEvent) bool {
			return true
		},
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Project{}}, enqueueRelatedUsers, onlyNewProjects); err != nil {
		return err
	}

	return nil
}

func (r *reconcileServiceAccountProjectBinding) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	sa := &kubermaticv1.User{}
	if err := r.Get(ctx, request.NamespacedName, sa); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get user: %w", err)
	}

	log := r.log.With("serviceaccount", sa.Name)
	log.Debug("Reconciling")

	err := r.reconcile(ctx, log, sa)
	if err != nil {
		r.recorder.Eventf(sa, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		log.Errorw("failed to reconcile", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *reconcileServiceAccountProjectBinding) reconcile(ctx context.Context, log *zap.SugaredLogger, user *kubermaticv1.User) error {
	if user.DeletionTimestamp != nil {
		return nil
	}

	// get project name from Owner Reference or from label
	projectName := user.Spec.Project
	if len(projectName) == 0 {
		// TODO: Is this fallback still needed?
		projectName = user.GetLabels()[kubermaticv1.ProjectIDLabelKey]
	}

	if len(projectName) == 0 {
		return errors.New("no project name specified")
	}

	// ensure the service account User is owned by the project and gets deleted appropriately
	if err := r.ensureOwnerReference(ctx, log, user); err != nil {
		return fmt.Errorf("failed to ensure owner reference to the project: %w", err)
	}

	labelSelector, err := labels.Parse(fmt.Sprintf("%s=%s", kubermaticv1.ProjectIDLabelKey, projectName))
	if err != nil {
		return fmt.Errorf("project name %q is not a valid label value", projectName)
	}

	bindings := &kubermaticv1.UserProjectBindingList{}
	if err := r.List(ctx, bindings, &ctrlruntimeclient.ListOptions{LabelSelector: labelSelector}); err != nil {
		return fmt.Errorf("failed to list UserProjectBindings for project: %w", err)
	}

	var binding *kubermaticv1.UserProjectBinding
	for _, b := range bindings.Items {
		// the first equality check is kind of redundant, but just to be on the safe side;
		// webhooks should ensure that the label matches the spec
		if b.Spec.ProjectID == projectName && strings.EqualFold(b.Spec.UserEmail, user.Spec.Email) {
			binding = b.DeepCopy()
			break
		}
	}

	if binding == nil {
		binding, err = r.createBinding(ctx, user, projectName)
		if err != nil {
			return fmt.Errorf("failed to create UserProjectBinding: %w", err)
		}
	}

	oldBinding := binding.DeepCopy()
	kuberneteshelper.EnsureUniqueOwnerReference(binding, metav1.OwnerReference{
		APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		Kind:       kubermaticv1.UserKindName,
		UID:        user.GetUID(),
		Name:       user.Name,
	})
	// NB: The owner ref pointing from the UserProjectBinding to the Project is
	// maintained by the user-project-binding controller, because that owner ref
	// is valid for all UserProjectBindings, not just those created for service
	// account Users.

	group, hasGroupLabel := user.Labels[serviceaccount.ServiceAccountLabelGroup]
	if hasGroupLabel {
		binding.Spec.Group = group
	}

	if err := r.Patch(ctx, binding, ctrlruntimeclient.MergeFrom(oldBinding)); err != nil {
		return fmt.Errorf("failed to update UserProjectBinding: %w", err)
	}

	// remove labelGroup from the User object
	if hasGroupLabel {
		delete(user.Labels, serviceaccount.ServiceAccountLabelGroup)
		if err := r.Update(ctx, user); err != nil {
			return fmt.Errorf("failed to remove label from User: %w", err)
		}
	}

	return nil
}

func (r *reconcileServiceAccountProjectBinding) ensureOwnerReference(ctx context.Context, log *zap.SugaredLogger, user *kubermaticv1.User) error {
	project := &kubermaticv1.Project{}
	err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Name: user.Spec.Project}, project)
	if err != nil {
		if kerrors.IsNotFound(err) {
			log.Debugw("Project does not exist", "project", user.Spec.Project)
			return nil
		}

		return fmt.Errorf("failed to get project: %w", err)
	}

	oldUser := user.DeepCopy()
	kuberneteshelper.EnsureUniqueOwnerReference(user, metav1.OwnerReference{
		APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		Kind:       kubermaticv1.ProjectKindName,
		UID:        project.UID,
		Name:       project.Name,
	})
	kuberneteshelper.SortOwnerReferences(user.OwnerReferences)

	if err := r.Patch(ctx, user, ctrlruntimeclient.MergeFrom(oldUser)); err != nil {
		return fmt.Errorf("failed to patch user: %w", err)
	}

	return nil
}

func (r *reconcileServiceAccountProjectBinding) createBinding(ctx context.Context, sa *kubermaticv1.User, projectName string) (*kubermaticv1.UserProjectBinding, error) {
	group, ok := sa.Labels[serviceaccount.ServiceAccountLabelGroup]
	if !ok {
		return nil, fmt.Errorf("label %s not found", serviceaccount.ServiceAccountLabelGroup)
	}

	binding := &kubermaticv1.UserProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   rand.String(10),
			Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: projectName},
		},
		Spec: kubermaticv1.UserProjectBindingSpec{
			ProjectID: projectName,
			UserEmail: sa.Spec.Email,
			Group:     group,
		},
	}

	if err := r.Create(ctx, binding); err != nil {
		return nil, err
	}

	return binding, nil
}
