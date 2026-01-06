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
	"errors"
	"fmt"
	"reflect"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/machine-controller/sdk/providerconfig"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-cloud-controller"
)

// Check if the Reconciler fulfills the interface
// at compile time.
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

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}).
		Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.Name)
	log.Debug("Reconciling")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Add a wrapping here so we can emit an event on error
	result, err := util.ClusterReconcileWrapper(
		ctx,
		r,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionCloudControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, log, cluster)
		},
	)

	if result == nil || err != nil {
		result = &reconcile.Result{}
	}

	if err != nil {
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
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
	prov, err := cloud.Provider(datacenter.DeepCopy(), r.makeGlobalSecretKeySelectorValue(ctx), r.caBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud provider: %w", err)
	}

	if cluster.DeletionTimestamp != nil {
		log.Debug("Cleaning up cloud provider")

		// in-cluster resources, like nodes, are still being cleaned up
		if kuberneteshelper.HasAnyFinalizer(cluster, kubermaticv1.InClusterLBCleanupFinalizer, kubermaticv1.InClusterPVCleanupFinalizer, kubermaticv1.NodeDeletionFinalizer) {
			log.Debug("Cluster still has in-cluster cleanup finalizers, retrying later")
			return &reconcile.Result{RequeueAfter: 5 * time.Second}, nil
		}

		if _, err := prov.CleanUpCloudProvider(ctx, cluster, r.clusterUpdater); err != nil {
			return nil, fmt.Errorf("failed cloud provider cleanup: %w", err)
		}

		return nil, nil
	}

	handleProviderError := func(err error) (*reconcile.Result, error) {
		if apierrors.IsConflict(err) {
			// In case of conflict we just re-enqueue the item for later
			// processing without returning an error.
			log.Infow("failed to run cloud provider", zap.Error(err))
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
			interval = &metav1.Duration{Duration: defaulting.DefaultCloudProviderReconciliationInterval}
		}

		// reconcile if the lastTime isn't set (= first time init or a forced reconciliation) or too long ago
		if last.IsZero() || (interval.Duration > 0 && time.Since(last.Time) >= interval.Duration) || betterProvider.ClusterNeedsReconciling(cluster) {
			log.Info("Reconciling cloud provider for cluster")

			// update metrics
			providerName, _ := kubermaticv1helper.ClusterCloudProviderName(cluster.Spec.Cloud)
			totalProviderReconciliations.WithLabelValues(cluster.Name, providerName).Inc()

			// reconcile
			cluster, err := betterProvider.ReconcileCluster(ctx, cluster, r.clusterUpdater)
			updatedProviderStatus := cluster.Status.ProviderStatus
			if err != nil {
				return handleProviderError(err)
			}
			// remember that we reconciled
			err = util.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
				c.Status.LastProviderReconciliation = metav1.Now()
				c.Status.ProviderStatus = updatedProviderStatus
			})
			if err != nil {
				return nil, fmt.Errorf("failed to set last reconcile timestamp: %w", err)
			}

			// update metrics
			successfulProviderReconciliations.WithLabelValues(cluster.Name, providerName).Inc()
		}
	} else {
		// the provider only offers a one-time init :-(
		cluster, err = prov.InitializeCloudProvider(ctx, cluster, r.clusterUpdater)
		if err != nil {
			return handleProviderError(err)
		}
	}

	if err := util.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.ExtendedHealth.CloudProviderInfrastructure = kubermaticv1.HealthStatusUp
	}); err != nil {
		return nil, fmt.Errorf("failed to set cluster health: %w", err)
	}

	return nil, nil
}

func (r *Reconciler) clusterUpdater(ctx context.Context, name string, modify func(*kubermaticv1.Cluster)) (*kubermaticv1.Cluster, error) {
	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: name}, cluster); err != nil {
		return nil, err
	}

	oldCluster := cluster.DeepCopy()
	modify(cluster)
	if reflect.DeepEqual(oldCluster, cluster) {
		return cluster, nil
	}

	if !reflect.DeepEqual(oldCluster.Status, cluster.Status) {
		return nil, errors.New("updateCluster must not change cluster status")
	}

	// When finalizers were changed, we must force optimistic locking,
	// as we do not exclusively own the metadata.finalizers field and do not
	// want to risk overwriting other finalizers. Labels and annotations are
	// maps and so not affected.
	var opts []ctrlruntimeclient.MergeFromOption
	if !reflect.DeepEqual(oldCluster.Finalizers, cluster.Finalizers) {
		opts = append(opts, ctrlruntimeclient.MergeFromWithOptimisticLock{})
	}

	if err := r.Patch(ctx, cluster, ctrlruntimeclient.MergeFromWithOptions(oldCluster, opts...)); err != nil {
		return nil, err
	}

	return cluster, nil
}

func (r *Reconciler) makeGlobalSecretKeySelectorValue(ctx context.Context) provider.SecretKeySelectorValueFunc {
	return func(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error) {
		return provider.SecretKeySelectorValueFuncFactory(ctx, r)(configVar, key)
	}
}
