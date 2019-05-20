package cluster

import (
	"context"
	"time"

	"github.com/golang/glog"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
)

const (
	reachableCheckPeriod = 5 * time.Second
)

func (r *Reconciler) reconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// Create the namespace
	if err := r.ensureNamespaceExists(ctx, cluster); err != nil {
		return nil, err
	}

	// Set default network configuration
	if err := r.ensureClusterNetworkDefaults(ctx, cluster); err != nil {
		return nil, err
	}

	// Deploy & Update master components for Kubernetes
	if err := r.ensureResourcesAreDeployed(ctx, cluster); err != nil {
		return nil, err
	}

	// synchronize cluster.status.health for Kubernetes clusters
	if err := r.syncHealth(ctx, cluster); err != nil {
		return nil, err
	}

	if cluster.Status.Health.Apiserver {
		// Controlling of user-cluster resources
		reachable, err := r.clusterIsReachable(ctx, cluster)
		if err != nil {
			return nil, err
		}

		if !reachable {
			return &reconcile.Result{RequeueAfter: reachableCheckPeriod}, nil
		}

		// Only add the node deletion finalizer when the cluster is actually running
		// Otherwise we fail to delete the nodes and are stuck in a loop
		if !kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.NodeDeletionFinalizer) {
			err = r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
				kuberneteshelper.AddFinalizer(c, kubermaticapiv1.NodeDeletionFinalizer)
			})
			if err != nil {
				return nil, err
			}
		}

	}

	if !cluster.Status.Health.AllHealthy() {
		glog.V(4).Infof("Cluster %q not yet healthy: %+v", cluster.Name, cluster.Status.Health)
		return &reconcile.Result{RequeueAfter: reachableCheckPeriod}, nil
	}

	if cluster.Status.Phase == kubermaticv1.LaunchingClusterStatusPhase {
		err := r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			c.Status.Phase = kubermaticv1.RunningClusterStatusPhase
		})
		if err != nil {
			return nil, err
		}
	}

	return &reconcile.Result{}, nil
}

// ensureClusterNetworkDefaults will apply default cluster network configuration
func (r *Reconciler) ensureClusterNetworkDefaults(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	var modifiers []func(*kubermaticv1.Cluster)

	if len(cluster.Spec.ClusterNetwork.Services.CIDRBlocks) == 0 {
		setServiceNetwork := func(c *kubermaticv1.Cluster) {
			c.Spec.ClusterNetwork.Services.CIDRBlocks = []string{"10.10.10.0/24"}
		}
		modifiers = append(modifiers, setServiceNetwork)
	}

	if len(cluster.Spec.ClusterNetwork.Pods.CIDRBlocks) == 0 {
		setPodNetwork := func(c *kubermaticv1.Cluster) {
			c.Spec.ClusterNetwork.Pods.CIDRBlocks = []string{"172.25.0.0/16"}
		}
		modifiers = append(modifiers, setPodNetwork)
	}

	if cluster.Spec.ClusterNetwork.DNSDomain == "" {
		setDNSDomain := func(c *kubermaticv1.Cluster) {
			c.Spec.ClusterNetwork.DNSDomain = "cluster.local"
		}
		modifiers = append(modifiers, setDNSDomain)
	}

	return r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
		for _, modify := range modifiers {
			modify(c)
		}
	})
}
