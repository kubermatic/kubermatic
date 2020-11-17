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

package rolecloner

import (
	"context"
	"fmt"
	"reflect"

	"go.uber.org/zap"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// This controller duplicate roles with label component=userClusterRole for all namespaces
	controllerName = "clone_role_controller"
)

type reconciler struct {
	ctx      context.Context
	log      *zap.SugaredLogger
	client   ctrlruntimeclient.Client
	recorder record.EventRecorder
}

func Add(ctx context.Context, log *zap.SugaredLogger, mgr manager.Manager) error {
	log = log.Named(controllerName)

	r := &reconciler{
		ctx:      ctx,
		log:      log,
		client:   mgr.GetClient(),
		recorder: mgr.GetEventRecorderFor(controllerName),
	}
	c, err := controller.New(controllerName, mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %v", err)
	}

	// Watch for changes to Roles and Namespaces
	if err = c.Watch(&source.Kind{Type: &rbacv1.Role{}}, enqueueTemplateRoles(mgr.GetClient())); err != nil {
		return fmt.Errorf("failed to establish watch for the Roles %v", err)
	}
	if err = c.Watch(&source.Kind{Type: &corev1.Namespace{}}, enqueueTemplateRoles(mgr.GetClient())); err != nil {
		return fmt.Errorf("failed to establish watch for the Namespace %v", err)
	}

	return nil
}

func (r *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("Role", request.Name)
	log.Debug("Reconciling")

	role := &rbacv1.Role{}
	if err := r.client.Get(r.ctx, request.NamespacedName, role); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("role not found, returning")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get role: %v", err)
	}

	err := r.reconcile(log, role)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(role, corev1.EventTypeWarning, "CloningRoleFailed", err.Error())
	}
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(log *zap.SugaredLogger, role *rbacv1.Role) error {

	namespaces := []string{}
	namespaceList := &corev1.NamespaceList{}
	if err := r.client.List(r.ctx, namespaceList); err != nil {
		return fmt.Errorf("failed to get namespaces: %v", err)
	}

	for _, n := range namespaceList.Items {
		// This NS is the authoritative source of roles we configure
		if n.Name == v1.NamespaceSystem {
			continue
		}
		// No point in trying to create something in a deleted namespace
		if n.DeletionTimestamp != nil {
			log.Debugf("Skipping namespace %s", n.Name)
			continue
		}
		namespaces = append(namespaces, n.Name)
	}

	return r.reconcileRoles(log, role, namespaces)
}

func (r *reconciler) reconcileRoles(log *zap.SugaredLogger, oldRole *rbacv1.Role, namespaces []string) error {
	if oldRole.DeletionTimestamp != nil {
		if !kuberneteshelper.HasFinalizer(oldRole, kubermaticapiv1.UserClusterRoleCleanupFinalizer) {
			return nil
		}
		for _, namespace := range namespaces {
			if err := r.client.Delete(r.ctx, &rbacv1.Role{
				ObjectMeta: v1.ObjectMeta{
					Name:      oldRole.Name,
					Namespace: namespace,
				},
			}); err != nil {
				if kerrors.IsNotFound(err) {
					continue
				}
				return fmt.Errorf("failed to delete role: %v", err)
			}
		}

		kuberneteshelper.RemoveFinalizer(oldRole, kubermaticapiv1.UserClusterRoleCleanupFinalizer)
		if err := r.client.Update(r.ctx, oldRole); err != nil {
			return fmt.Errorf("failed to update role: %v", err)
		}
		return nil
	}

	if !kuberneteshelper.HasFinalizer(oldRole, kubermaticapiv1.UserClusterRoleCleanupFinalizer) {
		kuberneteshelper.AddFinalizer(oldRole, kubermaticapiv1.UserClusterRoleCleanupFinalizer)
		if err := r.client.Update(r.ctx, oldRole); err != nil {
			return fmt.Errorf("failed to update role: %v", err)
		}
	}

	for _, namespace := range namespaces {
		log := log.With("namespace", namespace)
		wasCreated := false
		role := &rbacv1.Role{}
		if err := r.client.Get(r.ctx, ctrlruntimeclient.ObjectKey{
			Namespace: namespace,
			Name:      oldRole.Name,
		}, role); err != nil {
			if kerrors.IsNotFound(err) {
				log.Debug("role not found, creating")
				newRole := &rbacv1.Role{
					ObjectMeta: v1.ObjectMeta{
						Name:      oldRole.Name,
						Namespace: namespace,
						Labels:    oldRole.Labels,
					},
					Rules: oldRole.Rules,
				}
				if err := r.client.Create(r.ctx, newRole); err != nil {
					return fmt.Errorf("failed to create role: %v", err)
				}
				wasCreated = true
			} else {
				return fmt.Errorf("failed to get role: %v", err)
			}
		}

		// update only existing roles, not already created
		if !wasCreated {
			if !reflect.DeepEqual(role.Rules, oldRole.Rules) {
				log.Debug("Role was changed, updating")
				role.Rules = oldRole.Rules
				if err := r.client.Update(r.ctx, role); err != nil {
					return fmt.Errorf("failed to update role: %v", err)
				}
			}
		}

	}

	return nil
}

// enqueueTemplateRoles enqueues the roles from kube-system namespace and special label component=userClusterRole
func enqueueTemplateRoles(client ctrlruntimeclient.Client) *handler.EnqueueRequestsFromMapFunc {
	return &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
		roleList := &rbacv1.RoleList{}
		if err := client.List(context.Background(), roleList, ctrlruntimeclient.MatchingLabels{handlercommon.UserClusterComponentKey: handlercommon.UserClusterRoleComponentValue}, ctrlruntimeclient.InNamespace(v1.NamespaceSystem)); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list Roles: %v", err))
			return []reconcile.Request{}
		}

		request := []reconcile.Request{}
		for _, role := range roleList.Items {
			request = append(request, reconcile.Request{NamespacedName: types.NamespacedName{Name: role.Name, Namespace: role.Namespace}})
		}
		return request
	})}
}
