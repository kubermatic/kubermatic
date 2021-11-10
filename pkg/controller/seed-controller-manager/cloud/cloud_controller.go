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

package cloud

import (
	"context"
	"crypto/x509"
	"fmt"
	"reflect"
	"time"

	"go.uber.org/zap"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/defaults"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/azure"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/openstack"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kubermatic_cloud_controller"
	// icmpMigrationRevision is the migration revision that will be set on the cluster after its
	// security group was migrated to contain allow rules for ICMP
	icmpMigrationRevision = 1
	// awsHarcodedAZMigrationRevision is the migration revision for moving AWS clusters away from
	// hardcoded AZs and Subnets towards multi-AZ support.
	awsHarcodedAZMigrationRevision = 2
	// currentMigrationRevision describes the current migration revision. If this is set on the
	// cluster, certain migrations won't get executed. This must never be decremented.
	CurrentMigrationRevision = awsHarcodedAZMigrationRevision
)

// Check if the Reconciler fulfills the interface
// at compile time
var _ reconcile.Reconciler = &Reconciler{}

type Reconciler struct {
	ctrlruntimeclient.Client

	log        *zap.SugaredLogger
	recorder   record.EventRecorder
	seedGetter provider.SeedGetter
	workerName string
	versions   kubermatic.Versions
	caBundle   *x509.CertPool
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	seedGetter provider.SeedGetter,
	workerName string,
	versions kubermatic.Versions,
	caBundle *x509.CertPool,
) error {
	reconciler := &Reconciler{
		Client:     mgr.GetClient(),
		log:        log.Named(ControllerName),
		recorder:   mgr.GetEventRecorderFor(ControllerName),
		seedGetter: seedGetter,
		workerName: workerName,
		versions:   versions,
		caBundle:   caBundle,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return err
	}
	return c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{})
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	log = log.With("cluster", cluster.Name)

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionCloudControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, log, cluster)
		},
	)
	if result == nil {
		result = &reconcile.Result{}
	}
	if err != nil {
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		log.Errorw("Reconciling failed", zap.Error(err))
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	seed, err := r.seedGetter()
	if err != nil {
		return nil, fmt.Errorf("failed to get seeds: %w", err)
	}
	datacenter, found := seed.Spec.Datacenters[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("couldn't find datacenter %q for cluster %q", cluster.Spec.Cloud.DatacenterName, cluster.Name)
	}
	prov, err := cloud.Provider(datacenter.DeepCopy(), r.getGlobalSecretKeySelectorValue, r.caBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud provider: %w", err)
	}

	if cluster.DeletionTimestamp != nil {
		log.Debug("Cleaning up cloud provider")

		// in-cluster resources, like nodes, are still being cleaned up
		if kuberneteshelper.HasAnyFinalizer(cluster, kubermaticapiv1.InClusterLBCleanupFinalizer, kubermaticapiv1.InClusterPVCleanupFinalizer, kubermaticapiv1.NodeDeletionFinalizer) {
			log.Debug("Cluster still has in-cluster cleanup finalizers, retrying later")
			return &reconcile.Result{RequeueAfter: 5 * time.Second}, nil
		}

		if _, err := prov.CleanUpCloudProvider(cluster, r.updateCluster); err != nil {
			return nil, fmt.Errorf("failed cloud provider cleanup: %w", err)
		}

		return nil, nil
	}

	// We do the migration inside the controller because it has a decent potential to fail (e.G. due
	// to invalid credentials) and may take some time and we do not want to block the startup just
	// because one cluster can not be migrated
	if cluster.Status.CloudMigrationRevision < icmpMigrationRevision {
		if err := r.migrateICMP(ctx, log, cluster, prov); err != nil {
			return nil, err
		}
	}

	handleProviderError := func(err error) (*reconcile.Result, error) {
		if kerrors.IsConflict(err) {
			// In case of conflict we just re-enqueue the item for later
			// processing without returning an error.
			r.log.Infow("failed to run cloud provider", zap.Error(err))
			return &reconcile.Result{Requeue: true}, nil
		}

		return nil, fmt.Errorf("failed cloud provider init: %w", err)
	}

	// initialize the cloud provider resources that need to exist for the cluster to function;
	// this will only create the resources once, and becomes a NOP after the first call.
	//
	// Some providers support ongoing reconciliations, where missing or broken resources can
	// automatically be fixed; to prevent excessive API calls to cloud providers, this
	// reconciliation is only triggered after a certain amount of time has passed since the
	// last time, configured by the admin.
	//
	// To prevent reconciling right after initialization (would cause a bunch of unneeded API
	// calls), we distinguish early between the providers and use _only_ reconciling when
	// provider implements it.
	if betterProvider, ok := prov.(provider.ReconcilingCloudProvider); ok {
		last := cluster.Status.LastProviderReconciliation

		// default the interval to a safe value
		interval := datacenter.Spec.ProviderReconciliationInterval
		if interval == nil || interval.Duration == 0 {
			interval = &metav1.Duration{Duration: defaults.DefaultCloudProviderReconciliationInterval}
		}

		// reconcile if the lastTime isn't set (= first time init or a forced reconciliation) or too long ago
		if last == nil || (interval.Duration > 0 && time.Since(last.Time) >= interval.Duration) {
			log.Info("Reconciling cloud provider for cluster")

			// update metrics
			providerName, _ := resources.GetCloudProviderName(cluster.Spec.Cloud)
			totalProviderReconciliations.WithLabelValues(cluster.Name, providerName).Inc()

			// reconcile
			cluster, err = betterProvider.ReconcileCluster(cluster, r.updateCluster)
			if err != nil {
				return handleProviderError(err)
			}

			// remember that we reconciled
			cluster, err = r.updateCluster(cluster.Name, func(c *kubermaticv1.Cluster) {
				now := v1.Now()
				c.Status.LastProviderReconciliation = &now
			})
			if err != nil {
				return nil, fmt.Errorf("failed to set last reconcile timestamp: %w", err)
			}

			// update metrics
			successfulProviderReconciliations.WithLabelValues(cluster.Name, providerName).Inc()
		}
	} else {
		// the provider only offers a one-time init :-(
		cluster, err = prov.InitializeCloudProvider(cluster, r.updateCluster)
		if err != nil {
			return handleProviderError(err)
		}
	}

	if _, err := r.updateCluster(cluster.Name, func(c *kubermaticv1.Cluster) {
		c.Status.ExtendedHealth.CloudProviderInfrastructure = kubermaticv1.HealthStatusUp
	}); err != nil {
		return nil, fmt.Errorf("failed to set cluster health: %w", err)
	}

	return nil, nil
}

func (r *Reconciler) migrateICMP(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, cloudProvider provider.CloudProvider) error {
	switch prov := cloudProvider.(type) {
	case *openstack.Provider:
		if err := prov.AddICMPRulesIfRequired(cluster); err != nil {
			return fmt.Errorf("failed to ensure ICMP rules for cluster %q: %w", cluster.Name, err)
		}
		log.Info("Successfully ensured ICMP rules in security group of cluster")
	case *azure.Azure:
		if err := prov.AddICMPRulesIfRequired(cluster); err != nil {
			return fmt.Errorf("failed to ensure ICMP rules for cluster %q: %w", cluster.Name, err)
		}
		log.Info("Successfully ensured ICMP rules in security group of cluster %q", cluster.Name)
	}

	var err error
	if cluster, err = r.updateCluster(cluster.Name, func(c *kubermaticv1.Cluster) {
		c.Status.CloudMigrationRevision = icmpMigrationRevision
	}); err != nil {
		return fmt.Errorf("failed to update cluster %q after successfully executing its cloudProvider migration: %w",
			cluster.Name, err)
	}

	return nil
}

func (r *Reconciler) updateCluster(name string, modify func(*kubermaticv1.Cluster), options ...provider.UpdaterOption) (*kubermaticv1.Cluster, error) {
	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(context.Background(), types.NamespacedName{Name: name}, cluster); err != nil {
		return nil, err
	}
	oldCluster := cluster.DeepCopy()
	modify(cluster)
	if reflect.DeepEqual(oldCluster, cluster) {
		return cluster, nil
	}
	opts := (&provider.UpdaterOptions{}).Apply(options...)
	var patch ctrlruntimeclient.Patch
	if opts.OptimisticLock {
		patch = ctrlruntimeclient.MergeFromWithOptions(oldCluster, ctrlruntimeclient.MergeFromWithOptimisticLock{})
	} else {
		patch = ctrlruntimeclient.MergeFrom(oldCluster)
	}
	return cluster, r.Patch(context.Background(), cluster, patch)
}

func (r *Reconciler) getGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error) {
	return provider.SecretKeySelectorValueFuncFactory(context.Background(), r.Client)(configVar, key)
}
