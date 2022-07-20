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

package roleclonercontroller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// This controller duplicate roles with label component=userClusterRole for all namespaces.
	controllerName = "kkp-role-cloner-controller"

	// cleanupFinalizer indicates that user cluster role still need cleanup.
	cleanupFinalizer = "kubermatic.k8c.io/user-cluster-role"
)

type reconciler struct {
	log             *zap.SugaredLogger
	client          ctrlruntimeclient.Client
	recorder        record.EventRecorder
	clusterIsPaused userclustercontrollermanager.IsPausedChecker
}

func Add(ctx context.Context, log *zap.SugaredLogger, mgr manager.Manager, clusterIsPaused userclustercontrollermanager.IsPausedChecker) error {
	log = log.Named(controllerName)

	r := &reconciler{
		log:             log,
		client:          mgr.GetClient(),
		recorder:        mgr.GetEventRecorderFor(controllerName),
		clusterIsPaused: clusterIsPaused,
	}
	c, err := controller.New(controllerName, mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	// enqueues the roles from kube-system namespace and special label component=userClusterRole.
	eventHandler := handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		roleList := &rbacv1.RoleList{}
		if err := r.client.List(ctx, roleList, ctrlruntimeclient.MatchingLabels{handlercommon.UserClusterComponentKey: handlercommon.UserClusterRoleComponentValue}, ctrlruntimeclient.InNamespace(metav1.NamespaceSystem)); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list Roles: %w", err))
			return []reconcile.Request{}
		}

		request := []reconcile.Request{}
		for _, role := range roleList.Items {
			request = append(request, reconcile.Request{NamespacedName: types.NamespacedName{Name: role.Name, Namespace: role.Namespace}})
		}

		return request
	})

	// Watch for changes to Roles and Namespaces
	if err = c.Watch(&source.Kind{Type: &rbacv1.Role{}}, eventHandler); err != nil {
		return fmt.Errorf("failed to establish watch for Roles: %w", err)
	}
	if err = c.Watch(&source.Kind{Type: &corev1.Namespace{}}, eventHandler); err != nil {
		return fmt.Errorf("failed to establish watch for Namespaces: %w", err)
	}

	return nil
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

	log := r.log.With("Role", request.Name)
	log.Debug("Reconciling")

	role := &rbacv1.Role{}
	if err := r.client.Get(ctx, request.NamespacedName, role); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("role not found, returning")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get role: %w", err)
	}

	err = r.reconcile(ctx, log, role)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(role, corev1.EventTypeWarning, "CloningRoleFailed", err.Error())
	}
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, role *rbacv1.Role) error {
	namespaces := []string{}
	namespaceList := &corev1.NamespaceList{}
	if err := r.client.List(ctx, namespaceList); err != nil {
		return fmt.Errorf("failed to get namespaces: %w", err)
	}

	for _, n := range namespaceList.Items {
		// This NS is the authoritative source of roles we configure
		if n.Name == metav1.NamespaceSystem {
			continue
		}
		// No point in trying to create something in a deleted namespace
		if n.DeletionTimestamp != nil {
			log.Debugf("Skipping namespace %s", n.Name)
			continue
		}
		namespaces = append(namespaces, n.Name)
	}

	return r.reconcileRoles(ctx, log, role, namespaces)
}

func (r *reconciler) reconcileRoles(ctx context.Context, log *zap.SugaredLogger, oldRole *rbacv1.Role, namespaces []string) error {
	if oldRole.DeletionTimestamp != nil {
		if kuberneteshelper.HasFinalizer(oldRole, cleanupFinalizer) {
			for _, namespace := range namespaces {
				if err := r.client.Delete(ctx, &rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      oldRole.Name,
						Namespace: namespace,
					},
				}); err != nil {
					if apierrors.IsNotFound(err) {
						continue
					}
					return fmt.Errorf("failed to delete Role in namespace %q: %w", namespace, err)
				}
			}
		}

		return kuberneteshelper.TryRemoveFinalizer(ctx, r.client, oldRole, cleanupFinalizer)
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.client, oldRole, cleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	for _, namespace := range namespaces {
		creatorGetters := []reconciling.NamedRoleCreatorGetter{
			func() (name string, create func(*rbacv1.Role) (*rbacv1.Role, error)) {
				return oldRole.Name, func(r *rbacv1.Role) (*rbacv1.Role, error) {
					r.Rules = oldRole.Rules
					r.Labels = oldRole.Labels

					return r, nil
				}
			},
		}

		if err := reconciling.ReconcileRoles(ctx, creatorGetters, namespace, r.client); err != nil {
			return fmt.Errorf("failed to reconcile Role in namespace %q: %w", namespace, err)
		}
	}

	return nil
}
