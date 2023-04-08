/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package mlacontroller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/kubernetes"
	"k8c.io/kubermatic/v3/pkg/resources/reconciling"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
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

type ruleGroupSyncReconciler struct {
	seedClient   ctrlruntimeclient.Client
	log          *zap.SugaredLogger
	workerName   string
	recorder     record.EventRecorder
	versions     kubermatic.Versions
	mlaNamespace string
}

var _ cleaner = &ruleGroupSyncReconciler{}

func newRuleGroupSyncReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	versions kubermatic.Versions,
	mlaNamespace string,
) *ruleGroupSyncReconciler {
	return &ruleGroupSyncReconciler{
		seedClient:   mgr.GetClient(),
		log:          log.Named("rulegroup-sync"),
		workerName:   workerName,
		recorder:     mgr.GetEventRecorderFor(ControllerName),
		versions:     versions,
		mlaNamespace: mlaNamespace,
	}
}

func (r *ruleGroupSyncReconciler) Start(ctx context.Context, mgr manager.Manager, workers int) error {
	ctrlOptions := controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: workers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	enqueueRuleGroup := handler.EnqueueRequestsFromMapFunc(func(object ctrlruntimeclient.Object) []reconcile.Request {
		ruleGroup := &kubermaticv1.RuleGroup{}
		nn := types.NamespacedName{
			Name:      object.GetName(),
			Namespace: r.mlaNamespace,
		}
		err = r.seedClient.Get(ctx, nn, ruleGroup)
		if apierrors.IsNotFound(err) {
			return []reconcile.Request{}
		}
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to get rulegroup: %w", err))
		}
		return []reconcile.Request{{NamespacedName: nn}}
	})

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.RuleGroup{}}, enqueueRuleGroup); err != nil {
		return fmt.Errorf("failed to watch RuleGroups: %w", err)
	}

	return nil
}

func (r *ruleGroupSyncReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("rulegroup", request.NamespacedName)
	log.Debug("Processing")

	ruleGroup := &kubermaticv1.RuleGroup{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, ruleGroup); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if !ruleGroup.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, r.handleDeletion(ctx, log, ruleGroup)
	}

	if err := kubernetes.TryAddFinalizer(ctx, r.seedClient, ruleGroup, ruleGroupFinalizer); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
	}

	if err := r.syncClusterNamespaces(ctx, log, ruleGroup, func(seedClient ctrlruntimeclient.Client, ruleGroup *kubermaticv1.RuleGroup, cluster *kubermaticv1.Cluster) error {
		ruleGroupReconcilerFactory := []reconciling.NamedRuleGroupReconcilerFactory{
			ruleGroupReconcilerFactory(ruleGroup, cluster),
		}
		return reconciling.ReconcileRuleGroups(ctx, ruleGroupReconcilerFactory, cluster.Status.NamespaceName, seedClient)
	}); err != nil {
		r.recorder.Event(ruleGroup, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		return reconcile.Result{}, fmt.Errorf("failed to reconcle rulegroup %s: %w", ruleGroup.Name, err)
	}

	return reconcile.Result{}, nil
}

func (r *ruleGroupSyncReconciler) Cleanup(ctx context.Context) error {
	ruleGroupList := &kubermaticv1.RuleGroupList{}
	if err := r.seedClient.List(ctx, ruleGroupList, ctrlruntimeclient.InNamespace(r.mlaNamespace)); err != nil {
		return err
	}

	for _, ruleGroup := range ruleGroupList.Items {
		if err := r.handleDeletion(ctx, r.log, &ruleGroup); err != nil {
			return err
		}
	}

	return nil
}

func (r *ruleGroupSyncReconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, ruleGroup *kubermaticv1.RuleGroup) error {
	if err := r.syncClusterNamespaces(ctx, log, ruleGroup, func(seedClient ctrlruntimeclient.Client, ruleGroup *kubermaticv1.RuleGroup, cluster *kubermaticv1.Cluster) error {
		ruleGroup = &kubermaticv1.RuleGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ruleGroup.Name,
				Namespace: cluster.Status.NamespaceName,
			},
		}
		if err := seedClient.Delete(ctx, ruleGroup); err != nil {
			return ctrlruntimeclient.IgnoreNotFound(err)
		}
		return nil
	}); err != nil {
		return err
	}

	return kubernetes.TryRemoveFinalizer(ctx, r.seedClient, ruleGroup, ruleGroupFinalizer)
}

func (r *ruleGroupSyncReconciler) syncClusterNamespaces(
	ctx context.Context,
	log *zap.SugaredLogger,
	ruleGroup *kubermaticv1.RuleGroup,
	action func(ctrlruntimeclient.Client, *kubermaticv1.RuleGroup, *kubermaticv1.Cluster) error,
) error {
	clusterList := &kubermaticv1.ClusterList{}
	if err := r.seedClient.List(ctx, clusterList); err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	for _, cluster := range clusterList.Items {
		if cluster.Spec.Pause {
			log.Debugw("cluster paused, skipping", "cluster", cluster.Name)
			continue
		}
		if !cluster.DeletionTimestamp.IsZero() {
			log.Debugw("cluster deletion in progress, skipping", "cluster", cluster.Name)
			continue
		}
		if cluster.Status.NamespaceName == "" {
			log.Debugw("cluster namespace not available, skipping", "cluster", cluster.Name)
			continue
		}
		if !mlaEnabled(cluster) {
			log.Debugw("cluster has MLA disabled, skipping", "cluster", cluster.Name)
			continue
		}
		if err := action(r.seedClient, ruleGroup, &cluster); err != nil {
			return fmt.Errorf("failed to sync rulegroup for cluster %s: %w", cluster.Name, err)
		}
	}

	return nil
}

func ruleGroupReconcilerFactory(ruleGroup *kubermaticv1.RuleGroup, cluster *kubermaticv1.Cluster) reconciling.NamedRuleGroupReconcilerFactory {
	return func() (string, reconciling.RuleGroupReconciler) {
		return ruleGroup.Name, func(r *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error) {
			r.Spec = kubermaticv1.RuleGroupSpec{
				RuleGroupType: ruleGroup.Spec.RuleGroupType,
				Cluster: corev1.ObjectReference{
					Name: cluster.Name,
				},
				Data: ruleGroup.Spec.Data,
			}

			return r, nil
		}
	}
}

func mlaEnabled(cluster kubermaticv1.Cluster) bool {
	return cluster.Spec.MLA != nil && (cluster.Spec.MLA.LoggingEnabled || cluster.Spec.MLA.MonitoringEnabled)
}
