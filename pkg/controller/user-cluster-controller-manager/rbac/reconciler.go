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

package rbacusercluster

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconciler reconciles Cluster Role and Cluster Role Binding objects.
type reconciler struct {
	ctrlruntimeclient.Client

	logger                     *zap.SugaredLogger
	rLock                      *sync.Mutex
	clusterIsPaused            userclustercontrollermanager.IsPausedChecker
	reconciledSuccessfullyOnce bool
}

// Reconcile makes changes in response to ClusterRole and ClusterRoleBinding related changes.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	err := r.reconcile(ctx, request)
	if err != nil {
		r.logger.Errorw("Reconciling failed", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, request reconcile.Request) error {
	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		return nil
	}

	err = r.ensureRBACClusterRole(ctx, request.Name)
	if err != nil {
		return err
	}

	err = r.ensureRBACClusterRoleBinding(ctx, request.Name)
	if err != nil {
		return err
	}

	r.rLock.Lock()
	defer r.rLock.Unlock()
	r.reconciledSuccessfullyOnce = true

	return nil
}

func (r *reconciler) ensureRBACClusterRole(ctx context.Context, resourceName string) error {
	creator, err := newClusterRoleCreator(resourceName)
	if err != nil {
		return fmt.Errorf("failed to init ClusterRole creator: %w", err)
	}

	if err := reconciling.ReconcileClusterRoles(ctx, []reconciling.NamedClusterRoleReconcilerFactory{creator}, "", r); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoles: %w", err)
	}

	return nil
}

func (r *reconciler) ensureRBACClusterRoleBinding(ctx context.Context, resourceName string) error {
	creator, err := newClusterRoleBindingCreator(resourceName)
	if err != nil {
		return fmt.Errorf("failed to init ClusterRole creator: %w", err)
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, []reconciling.NamedClusterRoleBindingReconcilerFactory{creator}, "", r); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %w", err)
	}

	return nil
}
