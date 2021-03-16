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

package serviceaccount

import (
	"context"
	"fmt"
	"strings"

	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	serviceaccount "k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName         = "kubermatic_serviceaccount_projectbinding_controller"
	MainServiceAccountName = "main-serviceaccount"
	ServiceAcountName      = "serviceaccount"
	OwnerAnnotationKey     = "owner"
)

func Add(mgr manager.Manager) error {
	r := &reconcileServiceAccountProjectBinding{Client: mgr.GetClient()}
	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to User
	err = c.Watch(&source.Kind{Type: &kubermaticv1.User{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	if err = c.Watch(&source.Kind{Type: &kubermaticv1.UserProjectBinding{}}, enqueueUserProjectBindings(mgr.GetClient())); err != nil {
		return fmt.Errorf("failed to establish watch for the UserProjectBinding %v", err)
	}
	return nil
}

// reconcileServiceAccountProjectBinding reconciles User objects
type reconcileServiceAccountProjectBinding struct {
	client.Client
}

func (r *reconcileServiceAccountProjectBinding) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	resourceName := request.Name
	// project service account
	if strings.HasPrefix(resourceName, ServiceAcountName) {
		if err := r.ensureServiceAccountProjectBinding(ctx, resourceName); err != nil {
			logger := log.Logger.With("controller", controllerName)
			logger.Errorw("failed to reconcile in controller", "error", err)
			return reconcile.Result{}, err
		}
	}
	// main service account
	if strings.HasPrefix(resourceName, MainServiceAccountName) {
		if err := r.ensureMainServiceAccountProjectBinding(ctx, resourceName); err != nil {
			logger := log.Logger.With("controller", controllerName)
			logger.Errorw("failed to reconcile in controller", "error", err)
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *reconcileServiceAccountProjectBinding) ensureMainServiceAccountProjectBinding(ctx context.Context, saName string) error {
	sa := &kubermaticv1.User{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: metav1.NamespaceAll, Name: saName}, sa); err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	group, ok := sa.GetLabels()[serviceaccount.ServiceAccountLabelGroup]
	if !ok {
		return fmt.Errorf("unable to find group for the service account = %s", sa.Name)
	}

	// get service account owner
	ownerEmail, ok := sa.GetAnnotations()[OwnerAnnotationKey]
	if !ok {
		return fmt.Errorf("unable to find owner for the service account = %s", sa.Name)
	}

	ownerList := &kubermaticv1.UserList{}
	if err := r.List(ctx, ownerList); err != nil {
		return err
	}

	var ownerUser *kubermaticv1.User
	for _, owner := range ownerList.Items {
		if strings.EqualFold(owner.Spec.Email, ownerEmail) {
			ownerUser = owner.DeepCopy()
		}
	}

	if ownerUser == nil {
		return fmt.Errorf("unable to find owner for the email = %s", ownerEmail)
	}

	if err := r.deleteMainServiceAccountBindings(ctx, sa); err != nil {
		return err
	}

	if err := r.createMainServiceAccountBindings(ctx, sa, group, ownerUser.Spec.Email); err != nil {
		return err
	}

	return nil
}

func (r *reconcileServiceAccountProjectBinding) deleteMainServiceAccountBindings(ctx context.Context, sa *kubermaticv1.User) error {
	allMemberMappings := &kubermaticv1.UserProjectBindingList{}
	if err := r.List(ctx, allMemberMappings); err != nil {
		return err
	}

	for _, memberMapping := range allMemberMappings.Items {
		if strings.EqualFold(memberMapping.Spec.UserEmail, sa.Spec.Email) {
			existingBinding := memberMapping.DeepCopy()
			if err := r.Delete(ctx, existingBinding); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *reconcileServiceAccountProjectBinding) createMainServiceAccountBindings(ctx context.Context, sa *kubermaticv1.User, group, ownerEmail string) error {
	allMemberMappings := &kubermaticv1.UserProjectBindingList{}
	if err := r.List(ctx, allMemberMappings); err != nil {
		return err
	}

	for _, memberMapping := range allMemberMappings.Items {
		projectID := memberMapping.Spec.ProjectID

		saGroup := fmt.Sprintf("%s-%s", group, projectID)

		// create a new binding for the owned projects
		if strings.EqualFold(memberMapping.Spec.UserEmail, ownerEmail) && strings.HasPrefix(memberMapping.Spec.Group, rbac.OwnerGroupNamePrefix) {
			sa.Labels[serviceaccount.ServiceAccountLabelGroup] = saGroup
			if err := r.createBinding(ctx, sa, projectID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *reconcileServiceAccountProjectBinding) ensureServiceAccountProjectBinding(ctx context.Context, saName string) error {
	sa := &kubermaticv1.User{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: metav1.NamespaceAll, Name: saName}, sa); err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// get project name from Owner Reference or from label
	projectName := ""
	for _, owner := range sa.GetOwnerReferences() {
		if owner.APIVersion == kubermaticv1.SchemeGroupVersion.String() && owner.Kind == kubermaticv1.ProjectKindName &&
			len(owner.Name) > 0 && len(owner.UID) > 0 {
			projectName = owner.Name
			break
		}
	}
	if len(projectName) == 0 {
		projectName = sa.GetLabels()[kubermaticv1.ProjectIDLabelKey]
	}

	if len(projectName) == 0 {
		return fmt.Errorf("unable to find owing project for the service account = %s", sa.Name)
	}

	labelSelector, err := labels.Parse(fmt.Sprintf("%s=%s", kubermaticv1.ProjectIDLabelKey, projectName))
	if err != nil {
		return err
	}

	bindings := &kubermaticv1.UserProjectBindingList{}
	if err := r.List(ctx, bindings, &client.ListOptions{LabelSelector: labelSelector}); err != nil {
		return err
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
		return fmt.Errorf("label %s not found for sa %s", serviceaccount.ServiceAccountLabelGroup, sa.Name)
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

// enqueueUserProjectBindings enqueues the human UserProjectBindings for changes
func enqueueUserProjectBindings(c client.Reader) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(a client.Object) []reconcile.Request {

		userProjectBinding := &kubermaticv1.UserProjectBinding{}
		if err := c.Get(context.Background(), client.ObjectKey{Namespace: metav1.NamespaceAll, Name: a.GetName()}, userProjectBinding); err != nil {
			if kerrors.IsNotFound(err) {
				return []reconcile.Request{}
			}
			utilruntime.HandleError(fmt.Errorf("failed to get userProjectBinding: %v", err))
			return []reconcile.Request{}
		}

		// check only human user bindings
		if strings.HasPrefix(userProjectBinding.Spec.UserEmail, ServiceAcountName) || strings.HasPrefix(userProjectBinding.Spec.UserEmail, MainServiceAccountName) {
			return []reconcile.Request{}
		}

		users := &kubermaticv1.UserList{}
		if err := c.List(context.Background(), users); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list users: %v", err))
			return []reconcile.Request{}
		}

		var humanUser *kubermaticv1.User
		for _, user := range users.Items {
			if strings.EqualFold(user.Spec.Email, userProjectBinding.Spec.UserEmail) {
				humanUser = user.DeepCopy()
				break
			}
		}

		if humanUser == nil {
			utilruntime.HandleError(fmt.Errorf("project %v is bound for not existing user: %v", userProjectBinding.Spec.ProjectID, userProjectBinding.Spec.UserEmail))
			return []reconcile.Request{}
		}

		request := []reconcile.Request{}
		for _, mainSA := range getMainServiceAccountsForHumanUser(users, humanUser).Items {
			{
				request = append(request, reconcile.Request{NamespacedName: types.NamespacedName{Name: mainSA.Name}})
			}
		}
		return request
	})
}

func getMainServiceAccountsForHumanUser(users *kubermaticv1.UserList, humanUser *kubermaticv1.User) *kubermaticv1.UserList {
	resultList := &kubermaticv1.UserList{}
	for _, user := range users.Items {
		if strings.HasPrefix(user.Name, MainServiceAccountName) && hasAnnotation(OwnerAnnotationKey, humanUser.Spec.Email, user) {
			resultList.Items = append(resultList.Items, user)
		}
	}
	return resultList
}

func hasAnnotation(key, value string, user kubermaticv1.User) bool {
	if user.Annotations == nil {
		return false
	}
	if strings.EqualFold(user.Annotations[key], value) {
		return true
	}

	return false
}
