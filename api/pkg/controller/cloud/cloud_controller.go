package cloud

import (
	"context"
	"errors"
	"fmt"
	"time"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/aws"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/azure"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/openstack"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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
	// cluster, certain migrations wont get executed. This must never be decremented.
	CurrentMigrationRevision = awsHarcodedAZMigrationRevision
)

// Check if the Reconciler fullfills the interface
// at compile time
var _ reconcile.Reconciler = &Reconciler{}

type Reconciler struct {
	client.Client
	log        *zap.SugaredLogger
	recorder   record.EventRecorder
	seedGetter provider.SeedGetter
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	seedGetter provider.SeedGetter,
	clusterPredicates predicate.Predicate,
) error {
	reconciler := &Reconciler{
		Client:     mgr.GetClient(),
		log:        log.Named(ControllerName),
		recorder:   mgr.GetEventRecorderFor(ControllerName),
		seedGetter: seedGetter,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return err
	}
	return c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}, clusterPredicates)
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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

	if cluster.Spec.Pause {
		log.Debug("Skipping because the cluster is paused")
		return reconcile.Result{}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := r.reconcile(ctx, log, cluster)
	if result == nil {
		result = &reconcile.Result{}
	}
	if err != nil {
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError", "%v", err)
		log.Errorw("Reconciling failed", zap.Error(err))
		return *result, err
	}
	_, err = r.updateCluster(cluster.Name, func(c *kubermaticv1.Cluster) {
		c.Status.ExtendedHealth.CloudProviderInfrastructure = kubermaticv1.HealthStatusUp
	})
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	seed, err := r.seedGetter()
	if err != nil {
		return nil, err
	}
	datacenter, found := seed.Spec.Datacenters[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("couldn't find datacentrer %q for cluster %q", cluster.Spec.Cloud.DatacenterName, cluster.Name)
	}
	prov, err := cloud.Provider(datacenter.DeepCopy(), r.getGlobalSecretKeySelectorValue)
	if err != nil {
		return nil, err
	}

	if cluster.DeletionTimestamp != nil {
		log.Debug("Cleaning up cloud provider")
		finalizers := sets.NewString(cluster.Finalizers...)
		if finalizers.Has(kubermaticapiv1.InClusterLBCleanupFinalizer) ||
			finalizers.Has(kubermaticapiv1.InClusterPVCleanupFinalizer) ||
			finalizers.Has(kubermaticapiv1.NodeDeletionFinalizer) {
			return &reconcile.Result{RequeueAfter: 5 * time.Second}, nil
		}
		_, err := prov.CleanUpCloudProvider(cluster, r.updateCluster)
		return nil, err
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

	_, err = prov.InitializeCloudProvider(cluster, r.updateCluster)
	return nil, err
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

func (r *Reconciler) updateCluster(name string, modify func(*kubermaticv1.Cluster)) (*kubermaticv1.Cluster, error) {
	cluster := &kubermaticv1.Cluster{}
	return cluster, retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := r.Get(context.Background(), types.NamespacedName{Name: name}, cluster); err != nil {
			return err
		}
		modify(cluster)
		return r.Update(context.Background(), cluster)
	})
}

func (r *Reconciler) getGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error) {
	if configVar == nil {
		return "", errors.New("configVar is nil")
	}
	if configVar.Name == "" || configVar.Namespace == "" {
		return "", fmt.Errorf("both name and namespace must be specified in the secret key selector")
	}
	if key == "" {
		return "", fmt.Errorf("key cannot be empty")
	}

	secret := &corev1.Secret{}
	namespacedName := types.NamespacedName{Namespace: configVar.Namespace, Name: configVar.Name}
	if err := r.Get(context.Background(), namespacedName, secret); err != nil {
		return "", fmt.Errorf("error retrieving secret %q from namespace %q: %v", configVar.Name, configVar.Namespace, err)
	}

	if val, ok := secret.Data[key]; ok {
		return string(val), nil
	}
	return "", fmt.Errorf("secret %q in namespace %q has no key %q", configVar.Name, configVar.Namespace, key)
}
