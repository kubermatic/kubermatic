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
	serviceaccount "k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
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

	log *zap.SugaredLogger
}

func Add(mgr manager.Manager, log *zap.SugaredLogger) error {
	r := &reconcileServiceAccountProjectBinding{
		Client: mgr.GetClient(),

		log: log,
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

	return nil
}

func (r *reconcileServiceAccountProjectBinding) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("serviceaccount", request.Name)

	err := r.ensureServiceAccountProjectBinding(ctx, log, request.Name)
	if err != nil {
		log.Errorw("failed to reconcile", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *reconcileServiceAccountProjectBinding) ensureServiceAccountProjectBinding(ctx context.Context, log *zap.SugaredLogger, saName string) error {
	log.Debug("Reconciling")

	sa := &kubermaticv1.User{}
	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Name: saName}, sa); err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// get project name from Owner Reference or from label
	projectName := sa.Spec.Project
	if len(projectName) == 0 {
		// TODO: Is this fallback still needed?
		projectName = sa.GetLabels()[kubermaticv1.ProjectIDLabelKey]
	}

	if len(projectName) == 0 {
		return errors.New("no project name specified")
	}

	labelSelector, err := labels.Parse(fmt.Sprintf("%s=%s", kubermaticv1.ProjectIDLabelKey, projectName))
	if err != nil {
		return fmt.Errorf("project name %q is not a valid label value", projectName)
	}

	bindings := &kubermaticv1.UserProjectBindingList{}
	if err := r.List(ctx, bindings, &ctrlruntimeclient.ListOptions{LabelSelector: labelSelector}); err != nil {
		return fmt.Errorf("failed to list UserProjectBindings for project: %w", err)
	}

	bindingExist := false
	var existingBinding kubermaticv1.UserProjectBinding
	for _, binding := range bindings.Items {
		if binding.Spec.ProjectID == projectName && strings.EqualFold(binding.Spec.UserEmail, sa.Spec.Email) {
			bindingExist = true
			existingBinding = binding
			break
		}
	}
	if bindingExist {
		group, ok := sa.Labels[serviceaccount.ServiceAccountLabelGroup]
		if ok {
			existingBinding.Spec.Group = group
			if err := r.Update(ctx, &existingBinding); err != nil {
				return err
			}
		}
	} else if err := r.createBinding(ctx, sa, projectName); err != nil {
		return err
	}

	// remove labelGroup from sa
	if _, ok := sa.Labels[serviceaccount.ServiceAccountLabelGroup]; ok {
		delete(sa.Labels, serviceaccount.ServiceAccountLabelGroup)
		if err := r.Update(ctx, sa); err != nil {
			return err
		}
	}

	return nil
}

func (r *reconcileServiceAccountProjectBinding) createBinding(ctx context.Context, sa *kubermaticv1.User, projectName string) error {
	group, ok := sa.Labels[serviceaccount.ServiceAccountLabelGroup]
	if !ok {
		return fmt.Errorf("label %s not found", serviceaccount.ServiceAccountLabelGroup)
	}

	binding := &kubermaticv1.UserProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.UserKindName,
					UID:        sa.GetUID(),
					Name:       sa.Name,
				},
			},
			Name:   rand.String(10),
			Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: projectName},
		},
		Spec: kubermaticv1.UserProjectBindingSpec{
			ProjectID: projectName,
			UserEmail: sa.Spec.Email,
			Group:     group,
		},
	}

	return r.Create(ctx, binding)
}
