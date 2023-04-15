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
	"bytes"
	"context"
	"fmt"
	"reflect"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/mla-controller/cortex"
	"k8c.io/kubermatic/v3/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v3/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v3/pkg/kubernetes"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ruleGroupFinalizer = "kubermatic.k8c.io/rule-group"
)

// Manage rule groups that will be used to generate alerts.
type ruleGroupReconciler struct {
	seedClient           ctrlruntimeclient.Client
	log                  *zap.SugaredLogger
	workerName           string
	recorder             record.EventRecorder
	versions             kubermatic.Versions
	mlaNamespace         string
	cortexClientProvider cortex.ClientProvider
}

var _ cleaner = &ruleGroupReconciler{}

func newRuleGroupReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	versions kubermatic.Versions,
	mlaNamespace string,
	cortexClientProvider cortex.ClientProvider,
) *ruleGroupReconciler {
	return &ruleGroupReconciler{
		seedClient:           mgr.GetClient(),
		log:                  log.Named("rulegroup"),
		workerName:           workerName,
		recorder:             mgr.GetEventRecorderFor(ControllerName),
		versions:             versions,
		mlaNamespace:         mlaNamespace,
		cortexClientProvider: cortexClientProvider,
	}
}

func (r *ruleGroupReconciler) Start(ctx context.Context, mgr manager.Manager, workers int) error {
	ctrlOptions := controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: workers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	ruleGroupPredicate := predicateutil.Factory(func(o ctrlruntimeclient.Object) bool {
		// We don't want to enqueue RuleGroup objects in mla namespace since those are regarded as rulegroup template,
		// and will be rolled out to cluster namespaces in rulegroup_sync_controller.
		if o.GetNamespace() == r.mlaNamespace {
			return false
		}

		// If the cluster name in empty, we just ignore the ruleGroup.
		ruleGroup := o.(*kubermaticv1.RuleGroup)
		return ruleGroup.Spec.Cluster.Name != ""
	})

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.RuleGroup{}}, &handler.EnqueueRequestForObject{}, ruleGroupPredicate); err != nil {
		return fmt.Errorf("failed to watch RuleGroups: %w", err)
	}

	enqueueRuleGroupsForCluster := handler.EnqueueRequestsFromMapFunc(func(object ctrlruntimeclient.Object) []reconcile.Request {
		cluster := object.(*kubermaticv1.Cluster)
		if cluster.Status.NamespaceName == "" {
			return nil
		}

		ruleGroupList := &kubermaticv1.RuleGroupList{}
		if err := r.seedClient.List(ctx, ruleGroupList, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName)); err != nil {
			r.log.Errorw("Failed to list ruleGroups for cluster", zap.Error(err), "cluster", cluster.Name)
			utilruntime.HandleError(fmt.Errorf("failed to list ruleGroups: %w", err))
			return nil
		}

		var requests []reconcile.Request
		for _, ruleGroup := range ruleGroupList.Items {
			requests = append(requests, reconcile.Request{
				NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(&ruleGroup),
			})
		}

		return requests
	})

	clusterPredicate := predicate.Funcs{
		// For Update event, only trigger reconciliation when MonitoringEnabled or LoggingEnabled changes.
		UpdateFunc: func(event event.UpdateEvent) bool {
			oldCluster := event.ObjectOld.(*kubermaticv1.Cluster)
			newCluster := event.ObjectNew.(*kubermaticv1.Cluster)
			oldMonitoringEnabled := oldCluster.Spec.MLA != nil && oldCluster.Spec.MLA.MonitoringEnabled
			newMonitoringEnabled := newCluster.Spec.MLA != nil && newCluster.Spec.MLA.MonitoringEnabled
			oldLoggingEnabled := oldCluster.Spec.MLA != nil && oldCluster.Spec.MLA.LoggingEnabled
			newLoggingEnabled := newCluster.Spec.MLA != nil && newCluster.Spec.MLA.LoggingEnabled
			return (oldMonitoringEnabled != newMonitoringEnabled) || (oldLoggingEnabled != newLoggingEnabled)
		},
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, enqueueRuleGroupsForCluster, clusterPredicate); err != nil {
		return fmt.Errorf("failed to watch Clusters: %w", err)
	}

	return nil
}

func (r *ruleGroupReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("rulegroup", request.NamespacedName)
	log.Debug("Processing")

	ruleGroup := &kubermaticv1.RuleGroup{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, ruleGroup); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if ruleGroup.Spec.Cluster.Name == "" {
		return reconcile.Result{}, nil
	}

	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClient.Get(ctx, types.NamespacedName{Name: ruleGroup.Spec.Cluster.Name}, cluster); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}

	log = log.With("cluster", ruleGroup.Spec.Cluster.Name)

	// Add a wrapping here so we can emit an event on error
	result, err := util.ClusterReconcileWrapper(
		ctx,
		r.seedClient,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionMLAControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return nil, r.reconcile(ctx, log, cluster, ruleGroup)
		},
	)
	if err != nil {
		r.log.Errorw("Failed to reconcile cluster", "cluster", cluster.Name, zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *ruleGroupReconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, ruleGroup *kubermaticv1.RuleGroup) error {
	if !ruleGroup.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, log, ruleGroup)
	}

	mlaEnabled := cluster.Spec.MLA != nil && (cluster.Spec.MLA.MonitoringEnabled || cluster.Spec.MLA.LoggingEnabled)
	if !cluster.DeletionTimestamp.IsZero() || !mlaEnabled {
		// If this cluster is being deleted, or MLA is disabled for this cluster, we just delete this `RuleGroup`,
		// and the clean up of `RuleGroup` will be triggered in the next reconciliation loop.
		if err := r.seedClient.Delete(ctx, ruleGroup); err != nil {
			return ctrlruntimeclient.IgnoreNotFound(err)
		}

		return nil
	}

	if err := kubernetes.TryAddFinalizer(ctx, r.seedClient, ruleGroup, ruleGroupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	if err := r.ensureRuleGroup(ctx, log, ruleGroup); err != nil {
		return fmt.Errorf("failed to create rule group: %w", err)
	}

	return nil
}

func (r *ruleGroupReconciler) Cleanup(ctx context.Context, log *zap.SugaredLogger) error {
	ruleGroupList := &kubermaticv1.RuleGroupList{}
	if err := r.seedClient.List(ctx, ruleGroupList); err != nil {
		return err
	}

	for _, ruleGroup := range ruleGroupList.Items {
		groupLog := log.With("rulegroup", ruleGroup.Name, "cluster", ruleGroup.Spec.Cluster.Name)

		if err := r.handleDeletion(ctx, groupLog, &ruleGroup); err != nil {
			return fmt.Errorf("failed to handle deletion: %w", err)
		}
		if err := r.seedClient.Delete(ctx, &ruleGroup); err != nil {
			return err
		}
	}

	return nil
}

func (r *ruleGroupReconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, ruleGroup *kubermaticv1.RuleGroup) error {
	cClient := r.cortexClientProvider()

	log.Infow("Deleting rule group", "type", ruleGroup.Spec.RuleGroupType)
	if err := cClient.DeleteRuleGroupConfiguration(ctx, ruleGroup.Spec.Cluster.Name, ruleGroup.Spec.RuleGroupType, ruleGroup.Name); err != nil {
		return fmt.Errorf("failed to delete Cortex rule group: %w", err)
	}

	return kubernetes.TryRemoveFinalizer(ctx, r.seedClient, ruleGroup, ruleGroupFinalizer)
}

func (r *ruleGroupReconciler) ensureRuleGroup(ctx context.Context, log *zap.SugaredLogger, ruleGroup *kubermaticv1.RuleGroup) error {
	expectedRuleGroup := map[string]interface{}{}
	decoder := yaml.NewDecoder(bytes.NewReader(ruleGroup.Spec.Data))
	if err := decoder.Decode(&expectedRuleGroup); err != nil {
		return fmt.Errorf("unable to unmarshal expected rule group: %w", err)
	}

	currentRuleGroup, err := r.getCurrentRuleGroup(ctx, ruleGroup)
	if err != nil {
		return err
	}

	if reflect.DeepEqual(currentRuleGroup, expectedRuleGroup) {
		return nil
	}

	cClient := r.cortexClientProvider()

	log.Infow("Upserting rule group", "type", ruleGroup.Spec.RuleGroupType)
	return cClient.SetRuleGroupConfiguration(ctx, ruleGroup.Spec.Cluster.Name, ruleGroup.Spec.RuleGroupType, ruleGroup.Name, ruleGroup.Spec.Data)
}

func (r *ruleGroupReconciler) getCurrentRuleGroup(ctx context.Context, ruleGroup *kubermaticv1.RuleGroup) (map[string]interface{}, error) {
	cClient := r.cortexClientProvider()

	ruleConfig, err := cClient.GetRuleGroupConfiguration(ctx, ruleGroup.Spec.Cluster.Name, ruleGroup.Spec.RuleGroupType, ruleGroup.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete Cortex rule group: %w", err)
	}
	if ruleConfig == nil {
		return nil, nil
	}

	config := map[string]interface{}{}
	decoder := yaml.NewDecoder(bytes.NewReader(ruleConfig))
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode rule group: %w", err)
	}

	return config, nil
}
