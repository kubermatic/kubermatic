package cluster

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
)

const (
	reachableCheckPeriod = 5 * time.Second
)

func (cc *Controller) reconcileCluster(cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	var err error
	// Create the namespace
	if cluster, err = cc.ensureNamespaceExists(cluster); err != nil {
		return nil, err
	}

	// Set the hostname & url
	if cluster, err = cc.syncAddress(cluster); err != nil {
		return nil, err
	}

	// Set default network configuration
	if cluster, err = cc.ensureClusterNetworkDefaults(cluster); err != nil {
		return nil, err
	}

	// Deploy & Update master components
	if err = cc.ensureResourcesAreDeployed(cluster); err != nil {
		return nil, err
	}

	// synchronize cluster.status.health
	if cluster, err = cc.syncHealth(cluster); err != nil {
		return nil, err
	}

	if cluster.Status.Health.Apiserver {
		// Controlling of user-cluster resources
		reachable, err := cc.clusterIsReachable(cluster)
		if err != nil {
			return nil, err
		}

		if !reachable {
			cc.enqueueAfter(cluster, reachableCheckPeriod)
			return cluster, nil
		}

		// Only add the node deletion finalizer when the cluster is actually running
		// Otherwise we fail to delete the nodes and are stuck in a loop
		if !kuberneteshelper.HasFinalizer(cluster, NodeDeletionFinalizer) {
			cluster, err = cc.updateCluster(cluster.Name, func(c *kubermaticv1.Cluster) {
				c.Finalizers = append(c.Finalizers, NodeDeletionFinalizer)
			})
			if err != nil {
				return nil, err
			}
		}

		client, err := cc.userClusterConnProvider.GetClient(cluster)
		if err != nil {
			return nil, err
		}

		// TODO: Move into own controller
		if cluster, err = cc.reconcileUserClusterResources(cluster, client); err != nil {
			return nil, fmt.Errorf("failed to reconcile user cluster resources: %v", err)
		}
	}

	if !cluster.Status.Health.AllHealthy() {
		glog.V(5).Infof("Cluster %q not yet healthy: %+v", cluster.Name, cluster.Status.Health)
		cc.enqueueAfter(cluster, reachableCheckPeriod)
		return cluster, nil
	}

	if cluster.Status.Phase == kubermaticv1.LaunchingClusterStatusPhase {
		cluster, err = cc.updateCluster(cluster.Name, func(c *kubermaticv1.Cluster) {
			c.Status.Phase = kubermaticv1.RunningClusterStatusPhase
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

// ensureClusterNetworkDefaults will apply default cluster network configuration
func (cc *Controller) ensureClusterNetworkDefaults(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	var err error
	if len(c.Spec.ClusterNetwork.Services.CIDRBlocks) == 0 {
		c, err = cc.updateCluster(c.Name, func(c *kubermaticv1.Cluster) {
			c.Spec.ClusterNetwork.Services.CIDRBlocks = []string{"10.10.10.0/24"}
		})
		if err != nil {
			return nil, err
		}
	}

	if len(c.Spec.ClusterNetwork.Pods.CIDRBlocks) == 0 {
		c, err = cc.updateCluster(c.Name, func(c *kubermaticv1.Cluster) {
			c.Spec.ClusterNetwork.Pods.CIDRBlocks = []string{"172.25.0.0/16"}
		})
		if err != nil {
			return nil, err
		}
	}

	if c.Spec.ClusterNetwork.DNSDomain == "" {
		c, err = cc.updateCluster(c.Name, func(c *kubermaticv1.Cluster) {
			c.Spec.ClusterNetwork.DNSDomain = "cluster.local"
		})
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}
