/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package eventratelimitenforcement

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// ControllerName is the name of this controller.
	ControllerName = "kkp-event-rate-limit-enforcement-controller"
)

type reconciler struct {
	log                     *zap.SugaredLogger
	workerNameLabelSelector labels.Selector
	recorder                events.EventRecorder
	configGetter            provider.KubermaticConfigurationGetter
	seedClient              ctrlruntimeclient.Client
}

// Add creates a new event rate limit enforcement controller.
func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	configGetter provider.KubermaticConfigurationGetter,
	numWorkers int,
) error {
	workerSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return fmt.Errorf("failed to build worker-name selector: %w", err)
	}

	reconciler := &reconciler{
		log:                     log.Named(ControllerName),
		workerNameLabelSelector: workerSelector,
		recorder:                mgr.GetEventRecorder(ControllerName),
		configGetter:            configGetter,
		seedClient:              mgr.GetClient(),
	}

	// Handler to enqueue all clusters when a KubermaticConfiguration is updated
	configHandler := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, _ ctrlruntimeclient.Object) []reconcile.Request {
		clusterList := &kubermaticv1.ClusterList{}
		if err := reconciler.seedClient.List(ctx, clusterList, &ctrlruntimeclient.ListOptions{LabelSelector: workerSelector}); err != nil {
			log.Errorw("Failed to list clusters for config update", zap.Error(err))
			utilruntime.HandleError(fmt.Errorf("failed to list clusters: %w", err))
			return nil
		}

		requests := make([]reconcile.Request, 0, len(clusterList.Items))
		for _, cluster := range clusterList.Items {
			requests = append(requests, reconcile.Request{
				NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(&cluster),
			})
		}

		return requests
	})

	_, err = builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}, builder.WithPredicates(workerlabel.Predicate(workerName), clusterEventRateLimitPredicate())).
		Watches(&kubermaticv1.KubermaticConfiguration{}, configHandler, builder.WithPredicates(configEventRateLimitPredicate())).
		Build(reconciler)

	return err
}

// clusterEventRateLimitPredicate filters cluster events to only trigger on relevant changes.
func clusterEventRateLimitPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldCluster, ok := e.ObjectOld.(*kubermaticv1.Cluster)
			if !ok {
				return false
			}
			newCluster, ok := e.ObjectNew.(*kubermaticv1.Cluster)
			if !ok {
				return false
			}

			// Reconcile if UseEventRateLimitAdmissionPlugin changed
			if oldCluster.Spec.UseEventRateLimitAdmissionPlugin != newCluster.Spec.UseEventRateLimitAdmissionPlugin {
				return true
			}

			// Reconcile if EventRateLimitConfig changed
			if !equality.Semantic.DeepEqual(oldCluster.Spec.EventRateLimitConfig, newCluster.Spec.EventRateLimitConfig) {
				return true
			}

			// Reconcile if pause status changed
			if oldCluster.Spec.Pause != newCluster.Spec.Pause {
				return true
			}

			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

// configEventRateLimitPredicate filters KubermaticConfiguration events to only trigger on EventRateLimit changes.
func configEventRateLimitPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			config, ok := e.Object.(*kubermaticv1.KubermaticConfiguration)
			if !ok {
				return false
			}
			return config.Spec.UserCluster.AdmissionPlugins != nil && config.Spec.UserCluster.AdmissionPlugins.EventRateLimit != nil
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldConfig, ok := e.ObjectOld.(*kubermaticv1.KubermaticConfiguration)
			if !ok {
				return false
			}
			newConfig, ok := e.ObjectNew.(*kubermaticv1.KubermaticConfiguration)
			if !ok {
				return false
			}

			return !equality.Semantic.DeepEqual(
				getEventRateLimitConfig(oldConfig),
				getEventRateLimitConfig(newConfig),
			)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func getEventRateLimitConfig(config *kubermaticv1.KubermaticConfiguration) *kubermaticv1.EventRateLimitPluginConfiguration {
	if config.Spec.UserCluster.AdmissionPlugins == nil {
		return nil
	}
	return config.Spec.UserCluster.AdmissionPlugins.EventRateLimit
}

// Reconcile enforces EventRateLimit configuration from KubermaticConfiguration to User Clusters.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.Name)
	log.Debug("Reconciling")

	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, cluster); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	err := r.reconcile(ctx, log, cluster)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Eventf(cluster, nil, corev1.EventTypeWarning, "EventRateLimitEnforcementError", "Reconciling", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	// Skip if cluster is being deleted
	if !cluster.DeletionTimestamp.IsZero() {
		return nil
	}

	// Skip if cluster is paused
	if cluster.Spec.Pause {
		return nil
	}

	config, err := r.configGetter(ctx)
	if err != nil {
		return fmt.Errorf("failed to get KubermaticConfiguration: %w", err)
	}

	eventRateLimitConfig := getEventRateLimitConfig(config)
	if eventRateLimitConfig == nil || eventRateLimitConfig.Enforced == nil || !*eventRateLimitConfig.Enforced {
		return nil
	}

	// Enforcement is active: check if plugin is already in desired state
	needsPluginEnabled := !cluster.Spec.UseEventRateLimitAdmissionPlugin
	needsConfigUpdate := eventRateLimitConfig.DefaultConfig != nil &&
		!equality.Semantic.DeepEqual(cluster.Spec.EventRateLimitConfig, eventRateLimitConfig.DefaultConfig)

	if !needsPluginEnabled && !needsConfigUpdate {
		return nil
	}

	oldCluster := cluster.DeepCopy()

	if needsPluginEnabled {
		cluster.Spec.UseEventRateLimitAdmissionPlugin = true
	}

	if needsConfigUpdate {
		cluster.Spec.EventRateLimitConfig = eventRateLimitConfig.DefaultConfig.DeepCopy()
	}

	if err := r.seedClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
		return fmt.Errorf("failed to update cluster EventRateLimit configuration: %w", err)
	}

	log.Info("EventRateLimit configuration enforced from KubermaticConfiguration")
	r.recorder.Eventf(cluster, nil, corev1.EventTypeNormal, "EventRateLimitEnforced", "Reconciling",
		"EventRateLimit configuration enforced from KubermaticConfiguration")

	return nil
}
