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

package auditloggingenforcement

import (
	"context"
	"fmt"
	"reflect"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
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
	ControllerName = "kkp-audit-logging-enforcement-controller"
)

type reconciler struct {
	log                     *zap.SugaredLogger
	workerNameLabelSelector labels.Selector
	recorder                record.EventRecorder
	seedGetter              provider.SeedGetter
	seedClient              ctrlruntimeclient.Client
}

// Add creates a new audit logging enforcement controller.
func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	seedGetter provider.SeedGetter,
	numWorkers int,
) error {
	workerSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return fmt.Errorf("failed to build worker-name selector: %w", err)
	}

	reconciler := &reconciler{
		log:                     log.Named(ControllerName),
		workerNameLabelSelector: workerSelector,
		recorder:                mgr.GetEventRecorderFor(ControllerName),
		seedGetter:              seedGetter,
		seedClient:              mgr.GetClient(),
	}

	// Handler to enqueue all clusters when a Seed is updated
	seedHandler := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
		seed, ok := obj.(*kubermaticv1.Seed)
		if !ok {
			return nil
		}

		log.Debugw("Seed handler triggered", "seed", seed.Name)

		// List all clusters and enqueue those using this seed's datacenters
		clusterList := &kubermaticv1.ClusterList{}
		if err := reconciler.seedClient.List(ctx, clusterList, &ctrlruntimeclient.ListOptions{LabelSelector: workerSelector}); err != nil {
			log.Errorw("Failed to list clusters for seed update", zap.Error(err))
			utilruntime.HandleError(fmt.Errorf("failed to list clusters: %w", err))
			return nil
		}

		log.Debugw("Found clusters for potential reconciliation", "count", len(clusterList.Items))

		var requests []reconcile.Request
		for _, cluster := range clusterList.Items {
			if cluster.Spec.Cloud.DatacenterName == "" {
				continue
			}

			// Check if this cluster's datacenter is in the updated seed
			if _, found := seed.Spec.Datacenters[cluster.Spec.Cloud.DatacenterName]; found {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: cluster.Name,
					},
				})
				log.Debugw("Enqueuing cluster for reconciliation", "cluster", cluster.Name, "datacenter", cluster.Spec.Cloud.DatacenterName)
			}
		}

		log.Debugw("Total clusters enqueued for reconciliation", "count", len(requests))
		return requests
	})

	_, err = builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}, builder.WithPredicates(workerlabel.Predicate(workerName), clusterAuditLoggingPredicate())).
		Watches(&kubermaticv1.Seed{}, seedHandler, builder.WithPredicates(seedAuditLoggingPredicate(reconciler))).
		Build(reconciler)

	return err
}

// clusterAuditLoggingPredicate filters cluster events to only trigger on relevant changes.
func clusterAuditLoggingPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// Always process new clusters
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

			// Reconcile if audit logging configuration changed
			if !reflect.DeepEqual(oldCluster.Spec.AuditLogging, newCluster.Spec.AuditLogging) {
				return true
			}

			// Reconcile if opt-out annotation changed
			if oldCluster.Annotations[kubermaticv1.SkipAuditLoggingEnforcementAnnotation] !=
				newCluster.Annotations[kubermaticv1.SkipAuditLoggingEnforcementAnnotation] {
				return true
			}

			// Reconcile if pause status changed
			if oldCluster.Spec.Pause != newCluster.Spec.Pause {
				return true
			}

			// Reconcile if datacenter changed
			if oldCluster.Spec.Cloud.DatacenterName != newCluster.Spec.Cloud.DatacenterName {
				return true
			}

			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Don't reconcile on deletion
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

// seedAuditLoggingPredicate filters seed events to only trigger on audit logging changes for the specific seed.
func seedAuditLoggingPredicate(r *reconciler) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// Only process the seed this controller is responsible for
			seed, ok := e.Object.(*kubermaticv1.Seed)
			if !ok {
				return false
			}

			// Get our seed name lazily
			ourSeed, err := r.seedGetter()
			if err != nil {
				r.log.Debugw("Seed predicate CreateFunc: failed to get our seed", zap.Error(err))
				return false
			}

			if ourSeed.Name != seed.Name {
				r.log.Debugw("Seed predicate CreateFunc: skipping different seed",
					"ourSeed", ourSeed.Name, "eventSeed", seed.Name)
				return false
			}

			// Process new seeds with audit logging configuration
			result := seed.Spec.AuditLogging != nil
			r.log.Debugw("Seed predicate CreateFunc", "seed", seed.Name, "allowed", result)
			return result
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldSeed, ok := e.ObjectOld.(*kubermaticv1.Seed)
			if !ok {
				return false
			}
			newSeed, ok := e.ObjectNew.(*kubermaticv1.Seed)
			if !ok {
				return false
			}

			// Get our seed name lazily
			ourSeed, err := r.seedGetter()
			if err != nil {
				r.log.Debugw("Seed predicate UpdateFunc: failed to get our seed", zap.Error(err))
				return false
			}

			if ourSeed.Name != oldSeed.Name {
				r.log.Debugw("Seed predicate UpdateFunc: skipping different seed",
					"ourSeed", ourSeed.Name, "eventSeed", oldSeed.Name)
				return false
			}

			// Trigger if audit logging configuration changed
			// Use field-by-field comparison instead of reflect.DeepEqual to handle omitempty tags correctly
			auditLoggingChanged := auditLoggingSettingsChanged(oldSeed.Spec.AuditLogging, newSeed.Spec.AuditLogging)

			// Debug: Always log the comparison details to understand what's happening
			r.log.Debugw("Seed predicate UpdateFunc: comparing audit logging configs",
				"seed", newSeed.Name,
				"oldConfigPtr", fmt.Sprintf("%p", oldSeed.Spec.AuditLogging),
				"newConfigPtr", fmt.Sprintf("%p", newSeed.Spec.AuditLogging),
				"oldConfigNil", oldSeed.Spec.AuditLogging == nil,
				"newConfigNil", newSeed.Spec.AuditLogging == nil,
				"oldConfig", oldSeed.Spec.AuditLogging,
				"newConfig", newSeed.Spec.AuditLogging,
				"changed", auditLoggingChanged)

			if auditLoggingChanged {
				r.log.Infow("Seed predicate UpdateFunc: audit logging config changed",
					"seed", newSeed.Name,
					"oldConfig", oldSeed.Spec.AuditLogging,
					"newConfig", newSeed.Spec.AuditLogging)
				return true
			}

			// Trigger if any datacenter's enforceAuditLogging flag changed
			for dcName, newDC := range newSeed.Spec.Datacenters {
				oldDC, exists := oldSeed.Spec.Datacenters[dcName]
				if !exists || oldDC.Spec.EnforceAuditLogging != newDC.Spec.EnforceAuditLogging {
					r.log.Infow("Seed predicate UpdateFunc: datacenter enforceAuditLogging changed",
						"seed", newSeed.Name,
						"datacenter", dcName,
						"oldValue", oldDC.Spec.EnforceAuditLogging,
						"newValue", newDC.Spec.EnforceAuditLogging)
					return true
				}
			}

			r.log.Debugw("Seed predicate UpdateFunc: no relevant changes", "seed", newSeed.Name)
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

// auditLoggingSettingsChanged compares two AuditLoggingSettings field-by-field.
// This avoids issues with reflect.DeepEqual and omitempty JSON tags where
// false values are omitted during serialization, making {} appear equal to {enabled: false}.
func auditLoggingSettingsChanged(old, current *kubermaticv1.AuditLoggingSettings) bool {
	// If both are nil, no change
	if old == nil && current == nil {
		return false
	}
	// If one is nil and the other is not, there's a change
	if (old == nil) != (current == nil) {
		return true
	}

	// Compare individual fields explicitly
	if old.Enabled != current.Enabled {
		return true
	}
	if old.PolicyPreset != current.PolicyPreset {
		return true
	}
	// Use reflect.DeepEqual for complex nested structs (SidecarSettings, WebhookBackend)
	if !reflect.DeepEqual(old.SidecarSettings, current.SidecarSettings) {
		return true
	}
	if !reflect.DeepEqual(old.WebhookBackend, current.WebhookBackend) {
		return true
	}

	return false
}

// Reconcile enforces audit logging configuration from Seed to User Clusters.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.Name)
	log.Debug("Reconciling audit logging enforcement")

	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, cluster); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	err := r.reconcile(ctx, cluster, log)
	if err != nil {
		r.recorder.Event(cluster, corev1.EventTypeWarning, "AuditLoggingEnforcementError", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster, log *zap.SugaredLogger) error {
	// Skip if cluster is being deleted
	if !cluster.DeletionTimestamp.IsZero() {
		log.Debug("Cluster is being deleted, skipping")
		return nil
	}

	// Skip if cluster is paused
	if cluster.Spec.Pause {
		log.Debug("Cluster is paused, skipping")
		return nil
	}

	// Skip if cluster has opt-out annotation
	if cluster.Annotations[kubermaticv1.SkipAuditLoggingEnforcementAnnotation] == "true" {
		log.Debug("Cluster has opt-out annotation, skipping enforcement")
		return nil
	}

	// Get the seed for this cluster
	if cluster.Spec.Cloud.DatacenterName == "" {
		log.Debug("Cluster has no datacenter name, skipping")
		return nil
	}

	seed, err := r.seedGetter()
	if err != nil {
		return fmt.Errorf("failed to get seed: %w", err)
	}

	// Get the datacenter to check enforcement flag
	datacenter, found := seed.Spec.Datacenters[cluster.Spec.Cloud.DatacenterName]
	if !found {
		log.Debug("Cluster's datacenter not found in seed, skipping")
		return nil
	}

	// Skip enforcement if seed has no audit logging configuration
	// This prevents accidentally removing audit logging from clusters when the seed config is empty
	seedAuditLogging := seed.Spec.AuditLogging
	if seedAuditLogging == nil {
		log.Debug("Seed has no audit logging configuration, skipping enforcement")
		return nil
	}

	// Determine the desired audit logging configuration based on enforcement flag
	var desiredAuditLogging *kubermaticv1.AuditLoggingSettings
	if datacenter.Spec.EnforceAuditLogging {
		// When enforcement is enabled, copy the seed's audit logging configuration
		desiredAuditLogging = seedAuditLogging.DeepCopy()
	} else {
		// When enforcement is disabled, explicitly disable audit logging
		desiredAuditLogging = &kubermaticv1.AuditLoggingSettings{
			Enabled: false,
		}
	}

	clusterAuditLogging := cluster.Spec.AuditLogging

	if reflect.DeepEqual(desiredAuditLogging, clusterAuditLogging) {
		log.Debug("Cluster audit logging configuration already matches desired state, no update needed")
		return nil
	}

	// Update cluster's audit logging configuration
	if datacenter.Spec.EnforceAuditLogging {
		log.Infow("Enforcing audit logging configuration from seed", "seed", seed.Name)
	} else {
		log.Infow("Disabling audit logging as enforcement is disabled for datacenter", "datacenter", cluster.Spec.Cloud.DatacenterName)
	}

	oldCluster := cluster.DeepCopy()
	cluster.Spec.AuditLogging = desiredAuditLogging

	if err := r.seedClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
		return fmt.Errorf("failed to update cluster audit logging configuration: %w", err)
	}

	if datacenter.Spec.EnforceAuditLogging {
		r.recorder.Eventf(cluster, corev1.EventTypeNormal, "AuditLoggingEnforced",
			"Audit logging configuration enforced from seed %s", seed.Name)
	} else {
		r.recorder.Eventf(cluster, corev1.EventTypeNormal, "AuditLoggingDisabled",
			"Audit logging disabled as enforcement is disabled for datacenter %s", cluster.Spec.Cloud.DatacenterName)
	}

	log.Info("Successfully updated audit logging configuration")
	return nil
}
