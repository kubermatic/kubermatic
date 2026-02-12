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

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// This controller adds special label for build-it cluster roles to make them visible in the API.
	controllerName = "kkp-cluster-role-label-synchronizer"
)

type reconciler struct {
	log             *zap.SugaredLogger
	client          ctrlruntimeclient.Client
	recorder        events.EventRecorder
	clusterIsPaused userclustercontrollermanager.IsPausedChecker
}

func Add(ctx context.Context, log *zap.SugaredLogger, mgr manager.Manager, clusterIsPaused userclustercontrollermanager.IsPausedChecker) error {
	log = log.Named(controllerName)

	r := &reconciler{
		log:             log,
		client:          mgr.GetClient(),
		recorder:        mgr.GetEventRecorder(controllerName),
		clusterIsPaused: clusterIsPaused,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(controllerName).
		For(&rbacv1.ClusterRole{}, builder.WithPredicates(predicateutil.ByName("cluster-admin", "view", "edit"))).
		Build(r)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("clusterrole", request.Name)
	log.Debug("Reconciling")

	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

	clusterRole := &rbacv1.ClusterRole{}
	if err := r.client.Get(ctx, request.NamespacedName, clusterRole); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("cluster role not found, returning")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get cluster role: %w", err)
	}

	err = r.reconcile(ctx, log, clusterRole)
	if err != nil {
		r.recorder.Eventf(clusterRole, nil, corev1.EventTypeWarning, "AddingLabelFailed", "Reconciling", err.Error())
	}
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, clusterRole *rbacv1.ClusterRole) error {
	if clusterRole.Labels[userclustercontrollermanager.UserClusterComponentKey] == userclustercontrollermanager.UserClusterRoleComponentValue {
		return nil
	}

	oldClusterRole := clusterRole.DeepCopy()
	if clusterRole.Labels == nil {
		clusterRole.Labels = map[string]string{}
	}

	clusterRole.Labels[userclustercontrollermanager.UserClusterComponentKey] = userclustercontrollermanager.UserClusterRoleComponentValue

	log.Infow("Labeling ClusterRole", "label", userclustercontrollermanager.UserClusterComponentKey, "value", userclustercontrollermanager.UserClusterRoleComponentValue)

	if err := r.client.Patch(ctx, clusterRole, ctrlruntimeclient.MergeFrom(oldClusterRole)); err != nil {
		return fmt.Errorf("failed to update cluster role: %w", err)
	}

	return nil
}
