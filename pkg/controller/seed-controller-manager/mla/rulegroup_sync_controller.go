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

package mla

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

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
	ctrlruntimeclient.Client
	log                     *zap.SugaredLogger
	workerName              string
	recorder                record.EventRecorder
	versions                kubermatic.Versions
	ruleGroupSyncController *ruleGroupSyncController
}

func newRuleGroupSyncReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	ruleGroupSyncController *ruleGroupSyncController,
) error {
	log = log.Named(ControllerName)
	client := mgr.GetClient()

	reconciler := &ruleGroupSyncReconciler{
		Client:                  client,
		log:                     log,
		workerName:              workerName,
		recorder:                mgr.GetEventRecorderFor(ControllerName),
		versions:                versions,
		ruleGroupSyncController: ruleGroupSyncController,
	}

	ctrlOptions := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	enqueueRuleGroup := handler.EnqueueRequestsFromMapFunc(func(object ctrlruntimeclient.Object) []reconcile.Request {
		ruleGroup := &kubermaticv1.RuleGroup{}
		nn := types.NamespacedName{
			Name:      object.GetName(),
			Namespace: reconciler.ruleGroupSyncController.mlaNamespace,
		}
		err = client.Get(context.Background(), nn, ruleGroup)
		if apierrors.IsNotFound(err) {
			return []reconcile.Request{}
		}
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to get rulegroup: %w", err))
		}
		return []reconcile.Request{{NamespacedName: nn}}
	})
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.RuleGroup{}}, enqueueRuleGroup); err != nil {
		return fmt.Errorf("failed to watch RuleGroup: %w", err)
	}
	return nil
}

func (r *ruleGroupSyncReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("rulegroup", request)
	log.Debug("Processing")

	ruleGroup := &kubermaticv1.RuleGroup{}
	if err := r.Get(ctx, request.NamespacedName, ruleGroup); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if !ruleGroup.DeletionTimestamp.IsZero() {
		if err := r.ruleGroupSyncController.handleDeletion(ctx, log, ruleGroup); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to delete ruleGroup: %w", err)
		}
		return reconcile.Result{}, nil
	}

	if err := kubernetes.TryAddFinalizer(ctx, r, ruleGroup, ruleGroupFinalizer); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
	}

	if err := r.ruleGroupSyncController.syncClusterNS(ctx, log, ruleGroup, func(seedClient ctrlruntimeclient.Client, ruleGroup *kubermaticv1.RuleGroup, cluster *kubermaticv1.Cluster) error {
		ruleGroupCreatorGetter := []reconciling.NamedKubermaticV1RuleGroupCreatorGetter{
			ruleGroupCreatorGetter(ruleGroup, cluster),
		}
		return reconciling.ReconcileKubermaticV1RuleGroups(ctx, ruleGroupCreatorGetter, cluster.Status.NamespaceName, seedClient)
	}); err != nil {
		r.recorder.Event(ruleGroup, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		return reconcile.Result{}, fmt.Errorf("failed to reconcle rulegroup %s: %w", ruleGroup.Name, err)
	}
	return reconcile.Result{}, nil
}

type ruleGroupSyncController struct {
	ctrlruntimeclient.Client
	mlaNamespace string
	log          *zap.SugaredLogger
}

func newRuleGroupSyncController(
	client ctrlruntimeclient.Client,
	log *zap.SugaredLogger,
	mlaNamespace string,
) *ruleGroupSyncController {
	return &ruleGroupSyncController{
		Client:       client,
		mlaNamespace: mlaNamespace,
		log:          log,
	}
}

func (r *ruleGroupSyncController) CleanUp(ctx context.Context) error {
	ruleGroupList := &kubermaticv1.RuleGroupList{}
	if err := r.List(ctx, ruleGroupList, ctrlruntimeclient.InNamespace(r.mlaNamespace)); err != nil {
		return err
	}
	for _, ruleGroup := range ruleGroupList.Items {
		if err := r.handleDeletion(ctx, r.log, &ruleGroup); err != nil {
			return err
		}
	}
	return nil
}

func (r *ruleGroupSyncController) handleDeletion(ctx context.Context, log *zap.SugaredLogger, ruleGroup *kubermaticv1.RuleGroup) error {
	if err := r.syncClusterNS(ctx, log, ruleGroup, func(seedClient ctrlruntimeclient.Client, ruleGroup *kubermaticv1.RuleGroup, cluster *kubermaticv1.Cluster) error {
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

	return kubernetes.TryRemoveFinalizer(ctx, r, ruleGroup, ruleGroupFinalizer)
}

func (r *ruleGroupSyncController) syncClusterNS(
	ctx context.Context,
	log *zap.SugaredLogger,
	ruleGroup *kubermaticv1.RuleGroup,
	action func(seedClient ctrlruntimeclient.Client, ruleGroup *kubermaticv1.RuleGroup, cluster *kubermaticv1.Cluster) error) error {
	clusterList := &kubermaticv1.ClusterList{}
	if err := r.List(ctx, clusterList); err != nil {
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
			log.Debugw("cluster have mla disabled, skipping", "cluster", cluster.Name)
			continue
		}
		if err := action(r.Client, ruleGroup, &cluster); err != nil {
			return fmt.Errorf("failed to sync rulegroup for cluster %s: %w", cluster.Name, err)
		}
	}
	return nil
}

func ruleGroupCreatorGetter(ruleGroup *kubermaticv1.RuleGroup, cluster *kubermaticv1.Cluster) reconciling.NamedKubermaticV1RuleGroupCreatorGetter {
	return func() (string, reconciling.KubermaticV1RuleGroupCreator) {
		return ruleGroup.Name, func(r *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error) {
			r.Name = ruleGroup.Name
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
