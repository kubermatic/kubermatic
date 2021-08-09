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
	"errors"
	"fmt"
	"net/http"
	"sync"

	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/util"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName     = "rbac-user-cluster-controller"
	ResourceOwnerName  = "system:kubermatic:owners"
	ResourceEditorName = "system:kubermatic:editors"
	ResourceViewerName = "system:kubermatic:viewers"
)

var mapFn = handler.EnqueueRequestsFromMapFunc(func(o ctrlruntimeclient.Object) []reconcile.Request {
	return []reconcile.Request{
		{NamespacedName: types.NamespacedName{
			Name:      ResourceOwnerName,
			Namespace: "",
		}},
		{NamespacedName: types.NamespacedName{
			Name:      ResourceEditorName,
			Namespace: "",
		}},
		{NamespacedName: types.NamespacedName{
			Name:      ResourceViewerName,
			Namespace: "",
		}},
	}
})

// Add creates a new RBAC generator controller that is responsible for creating Cluster Roles and Cluster Role Bindings
// for groups: `owners`, `editors` and `viewers``
func Add(mgr manager.Manager, registerReconciledCheck func(name string, check healthz.Checker) error, clusterIsPaused util.IsPausedChecker) error {
	reconcile := &reconcileRBAC{Client: mgr.GetClient(), rLock: &sync.Mutex{}, clusterIsPaused: clusterIsPaused}

	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: reconcile})
	if err != nil {
		return err
	}

	// Watch for changes to ClusterRoles
	if err = c.Watch(&source.Kind{Type: &rbacv1.ClusterRole{}}, mapFn); err != nil {
		return err
	}
	// Watch for changes to ClusterRoleBindings
	if err = c.Watch(&source.Kind{Type: &rbacv1.ClusterRoleBinding{}}, mapFn); err != nil {
		return err
	}

	// A very simple but limited way to express the first successful reconciling to the seed cluster
	return registerReconciledCheck(fmt.Sprintf("%s-%s", controllerName, "reconciled_successfully_once"), func(_ *http.Request) error {
		reconcile.rLock.Lock()
		defer reconcile.rLock.Unlock()

		if !reconcile.reconciledSuccessfullyOnce {
			return errors.New("no successful reconciliation so far")
		}
		return nil
	})
}

// reconcileRBAC reconciles Cluster Role and Cluster Role Binding objects
type reconcileRBAC struct {
	ctrlruntimeclient.Client

	rLock                      *sync.Mutex
	clusterIsPaused            util.IsPausedChecker
	reconciledSuccessfullyOnce bool
}

// Reconcile makes changes in response to Cluster Role and Cluster Role Binding related changes
func (r *reconcileRBAC) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

	rdr := reconciler{client: r.Client}

	if err := rdr.Reconcile(ctx, request.Name); err != nil {
		klog.Errorf("RBAC reconciliation failed: %v", err)
		return reconcile.Result{}, err
	}

	r.rLock.Lock()
	defer r.rLock.Unlock()
	r.reconciledSuccessfullyOnce = true
	return reconcile.Result{}, nil
}
