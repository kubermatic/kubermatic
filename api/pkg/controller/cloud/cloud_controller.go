package cloud

import (
	"context"
	"fmt"
	"time"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/aws"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/azure"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/openstack"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	"github.com/golang/glog"

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
	recorder   record.EventRecorder
	seedGetter provider.SeedGetter
}

func Add(mgr manager.Manager, numWorkers int, seedGetter provider.SeedGetter, clusterPredicates predicate.Predicate) error {
	reconciler := &Reconciler{
		Client:     mgr.GetClient(),
		recorder:   mgr.GetRecorder(ControllerName),
		seedGetter: seedGetter,
	}

	c, err := controller.New(ControllerName, mgr,
		controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return err
	}
	return c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}, clusterPredicates)
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Add a wrapping here so we can emit an event on error
	result, err := r.reconcile(ctx, cluster)
	if result == nil {
		result = &reconcile.Result{}
	}
	if err != nil {
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError", "%v", err)
		glog.Errorf("error reconciling cluster %s: %v", cluster.Name, err)
		return *result, err
	}
	_, err = r.updateCluster(cluster.Name, func(c *kubermaticv1.Cluster) {
		c.Status.ExtendedHealth.CloudProviderInfrastructure = kubermaticv1.HealthStatusUp
	})
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	if cluster.Spec.Pause {
		glog.V(4).Infof("skipping paused cluster %s", cluster.Name)
		return nil, nil
	}

	glog.V(4).Infof("syncing cluster %s", cluster.Name)
	seed, err := r.seedGetter()
	if err != nil {
		return nil, err
	}
	datacenter, found := seed.Spec.Datacenters[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("couldn't find datacentrer %q for cluster %q", cluster.Spec.Cloud.DatacenterName, cluster.Name)
	}
	prov, err := cloud.Provider(datacenter.DeepCopy())
	if err != nil {
		return nil, err
	}

	if cluster.DeletionTimestamp != nil {
		finalizers := sets.NewString(cluster.Finalizers...)
		if finalizers.Has(kubermaticapiv1.InClusterLBCleanupFinalizer) ||
			finalizers.Has(kubermaticapiv1.InClusterPVCleanupFinalizer) ||
			finalizers.Has(kubermaticapiv1.NodeDeletionFinalizer) {
			return &reconcile.Result{RequeueAfter: 5 * time.Second}, nil
		}
		_, err := prov.CleanUpCloudProvider(cluster, r.getGlobalSecretKeySelectorValue)
		return nil, err
	}

	// We do the migration inside the controller because it has a decent potential to fail (e.G. due
	// to invalid credentials) and may take some time and we do not want to block the startup just
	// because one cluster can not be migrated
	if cluster.Status.CloudMigrationRevision < icmpMigrationRevision {
		if err := r.migrateICMP(ctx, cluster, prov); err != nil {
			return nil, err
		}
	}

	if cluster.Status.CloudMigrationRevision < awsHarcodedAZMigrationRevision {
		if err := r.migrateAWSMultiAZ(ctx, cluster, prov); err != nil {
			return nil, err
		}
	}

	_, err = prov.InitializeCloudProvider(cluster, r.updateCluster, r.getGlobalSecretKeySelectorValue)
	return nil, err
}

func (r *Reconciler) migrateICMP(ctx context.Context, cluster *kubermaticv1.Cluster, cloudProvider provider.CloudProvider) error {
	switch provider := cloudProvider.(type) {
	case *aws.AmazonEC2:
		if err := provider.AddICMPRulesIfRequired(cluster); err != nil {
			return fmt.Errorf("failed to ensure ICMP rules for cluster %q: %v", cluster.Name, err)
		}
		glog.Infof("Successfully ensured ICMP rules in security group of cluster %q", cluster.Name)
	case *openstack.Provider:
		if err := provider.AddICMPRulesIfRequired(cluster); err != nil {
			return fmt.Errorf("failed to ensure ICMP rules for cluster %q: %v", cluster.Name, err)
		}
		glog.Infof("Successfully ensured ICMP rules in security group of cluster %q", cluster.Name)
	case *azure.Azure:
		if err := provider.AddICMPRulesIfRequired(cluster); err != nil {
			return fmt.Errorf("failed to ensure ICMP rules for cluster %q: %v", cluster.Name, err)
		}
		glog.Infof("Successfully ensured ICMP rules in security group of cluster %q", cluster.Name)
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
	if awsprovider, ok := cloudProvider.(*aws.AmazonEC2); ok {

		if err := awsprovider.MigrateToMultiAZ(cluster); err != nil {
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
