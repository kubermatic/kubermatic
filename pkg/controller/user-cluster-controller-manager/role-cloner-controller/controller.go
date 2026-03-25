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
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
	recorder        events.EventRecorder
	clusterIsPaused userclustercontrollermanager.IsPausedChecker
}

func Add(log *zap.SugaredLogger, mgr manager.Manager, clusterIsPaused userclustercontrollermanager.IsPausedChecker) error {
	log = log.Named(controllerName)

	r := &reconciler{
		log:             log,
		client:          mgr.GetClient(),
		recorder:        mgr.GetEventRecorder(controllerName),
		clusterIsPaused: clusterIsPaused,
	}

	// enqueues the roles from kube-system namespace and special label component=userClusterRole.
	eventHandler := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		roleList := &rbacv1.RoleList{}
		if err := r.client.List(ctx, roleList, ctrlruntimeclient.MatchingLabels{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue}, ctrlruntimeclient.InNamespace(metav1.NamespaceSystem)); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list Roles: %w", err))
			return []reconcile.Request{}
		}

		request := []reconcile.Request{}
		for _, role := range roleList.Items {
			request = append(request, reconcile.Request{NamespacedName: types.NamespacedName{Name: role.Name, Namespace: role.Namespace}})
		}

		return request
	})

	_, err := builder.ControllerManagedBy(mgr).
		Named(controllerName).
		Watches(&rbacv1.Role{}, eventHandler).
		Watches(&corev1.Namespace{}, eventHandler).
		Build(r)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("role", request.Name)
	log.Debug("Reconciling")

	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

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
		r.recorder.Eventf(role, nil, corev1.EventTypeWarning, "CloningRoleFailed", "Reconciling", err.Error())
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
			log.Debugw("Skipping terminating namespace", "namespace", n.Name)
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
		creatorGetters := []reconciling.NamedRoleReconcilerFactory{
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
