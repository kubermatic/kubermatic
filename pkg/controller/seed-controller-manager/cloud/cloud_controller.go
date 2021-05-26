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
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/azure"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/openstack"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
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
		return nil, fmt.Errorf("failed to get seeds: %v", err)
	}
	datacenter, found := seed.Spec.Datacenters[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("couldn't find datacenter %q for cluster %q", cluster.Spec.Cloud.DatacenterName, cluster.Name)
	}
	prov, err := cloud.Provider(datacenter.DeepCopy(), r.getGlobalSecretKeySelectorValue, r.caBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud provider: %v", err)
	}

	if cluster.DeletionTimestamp != nil {
		log.Debug("Cleaning up cloud provider")
		finalizers := sets.NewString(cluster.Finalizers...)
		if finalizers.Has(kubermaticapiv1.InClusterLBCleanupFinalizer) ||
			finalizers.Has(kubermaticapiv1.InClusterPVCleanupFinalizer) ||
			finalizers.Has(kubermaticapiv1.NodeDeletionFinalizer) {
			return &reconcile.Result{RequeueAfter: 5 * time.Second}, nil
		}
		if _, err := prov.CleanUpCloudProvider(cluster, r.updateCluster); err != nil {
			return nil, fmt.Errorf("failed cloud provider cleanup: %v", err)
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

	if cluster.Status.CloudMigrationRevision < awsHarcodedAZMigrationRevision {
		if err := r.migrateAWSMultiAZ(ctx, cluster, prov); err != nil {
			return nil, err
		}
	}

	if _, err := prov.InitializeCloudProvider(cluster, r.updateCluster); err != nil {
		if kerrors.IsConflict(err) {
			// In case of conflict we just re-enqueue the item for later
			// processing without returning an error.
			r.log.Infow("failed to add finalizer to cluster", "error", err)
			return &reconcile.Result{Requeue: true}, nil
		}
		return nil, fmt.Errorf("failed cloud provider init: %v", err)
	}

	if _, err := r.updateCluster(cluster.Name, func(c *kubermaticv1.Cluster) {
		c.Status.ExtendedHealth.CloudProviderInfrastructure = kubermaticv1.HealthStatusUp
	}); err != nil {
		return nil, fmt.Errorf("failed to set cluster health: %v", err)
	}

	return nil, nil
}

func (r *Reconciler) migrateICMP(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, cloudProvider provider.CloudProvider) error {
	switch prov := cloudProvider.(type) {
	case *aws.AmazonEC2:
		if err := prov.AddICMPRulesIfRequired(cluster); err != nil {
			return fmt.Errorf("failed to ensure ICMP rules for cluster %q: %v", cluster.Name, err)
		}
		log.Info("Successfully ensured ICMP rules in security group of cluster")
	case *openstack.Provider:
		if err := prov.AddICMPRulesIfRequired(cluster); err != nil {
			return fmt.Errorf("failed to ensure ICMP rules for cluster %q: %v", cluster.Name, err)
		}
		log.Info("Successfully ensured ICMP rules in security group of cluster")
	case *azure.Azure:
		if err := prov.AddICMPRulesIfRequired(cluster); err != nil {
			return fmt.Errorf("failed to ensure ICMP rules for cluster %q: %v", cluster.Name, err)
		}
		log.Info("Successfully ensured ICMP rules in security group of cluster %q", cluster.Name)
	}

	var err error
	if cluster, err = r.updateCluster(cluster.Name, func(c *kubermaticv1.Cluster) {
		c.Status.CloudMigrationRevision = icmpMigrationRevision
	}); err != nil {
		return fmt.Errorf("failed to update cluster %q after successfully executing its cloudProvider migration: %v",
			cluster.Name, err)
	}

	return nil
}

func (r *Reconciler) migrateAWSMultiAZ(ctx context.Context, cluster *kubermaticv1.Cluster, cloudProvider provider.CloudProvider) error {
	if prov, ok := cloudProvider.(*aws.AmazonEC2); ok {
		if err := prov.MigrateToMultiAZ(cluster, r.updateCluster); err != nil {
			return fmt.Errorf("failed to migrate AWS cluster %q to multi-AZ: %q", cluster.Name, err)
		}
	}

	var err error
	if cluster, err = r.updateCluster(cluster.Name, func(c *kubermaticv1.Cluster) {
		c.Status.CloudMigrationRevision = awsHarcodedAZMigrationRevision
	}); err != nil {
		return fmt.Errorf("failed to update cluster %q after successfully executing its cloudProvider migration: %v",
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
