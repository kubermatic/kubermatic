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

package clusterrolelabeler

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// This controller adds special label for build-it cluster roles to make them visible in the API
	controllerName = "cluster_role_label_controller"
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
		return fmt.Errorf("failed to create controller: %v", err)
	}

	// Watch for changes to ClusterRoles
	if err = c.Watch(&source.Kind{Type: &rbacv1.ClusterRole{}}, &handler.EnqueueRequestForObject{}, predicateutil.ByName("cluster-admin", "view", "edit")); err != nil {
		return fmt.Errorf("failed to establish watch for the ClusterRoles %v", err)
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

	log := r.log.With("ClusterRole", request.Name)
	log.Debug("Reconciling")

	clusterRole := &rbacv1.ClusterRole{}
	if err := r.client.Get(ctx, request.NamespacedName, clusterRole); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("cluster role not found, returning")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get cluster role: %v", err)
	}

	err = r.reconcile(ctx, log, clusterRole)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(clusterRole, corev1.EventTypeWarning, "AddingLabelFailed", err.Error())
	}
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, clusterRole *rbacv1.ClusterRole) error {
	oldClusterRole := clusterRole.DeepCopy()
	if clusterRole.Labels == nil {
		clusterRole.Labels = map[string]string{}
	}

	if value, ok := clusterRole.Labels[handlercommon.UserClusterComponentKey]; ok {
		if value == handlercommon.UserClusterRoleComponentValue {
			log.Debug("label ", handlercommon.UserClusterRoleLabelSelector, " exists, not updating cluster role: ", clusterRole.Name)
			return nil
		}
	}

	clusterRole.Labels[handlercommon.UserClusterComponentKey] = handlercommon.UserClusterRoleComponentValue

	if err := r.client.Patch(ctx, clusterRole, ctrlruntimeclient.MergeFrom(oldClusterRole)); err != nil {
		return fmt.Errorf("failed to update cluster role: %v", err)
	}

	return nil
}
