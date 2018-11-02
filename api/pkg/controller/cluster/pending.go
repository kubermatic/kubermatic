package cluster

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
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

	// Setup required infrastructure at cloud provider
	if err = cc.ensureCloudProviderIsInitialized(cluster); err != nil {
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
		if !kuberneteshelper.HasFinalizer(cluster, nodeDeletionFinalizer) {
			cluster, err = cc.updateCluster(cluster.Name, func(c *kubermaticv1.Cluster) {
				c.Finalizers = append(c.Finalizers, nodeDeletionFinalizer)
			})
			if err != nil {
				return nil, err
			}
		}

		if err = cc.launchingCreateClusterInfoConfigMap(cluster); err != nil {
			return nil, err
		}

		if err = cc.launchingCreateOpenVPNClientCertificates(cluster); err != nil {
			return nil, err
		}

		client, err := cc.userClusterConnProvider.GetClient(cluster)
		if err != nil {
			return nil, err
		}

		if len(cluster.Spec.MachineNetworks) > 0 {
			if err = cc.userClusterEnsureInitializerConfiguration(cluster, client); err != nil {
				return nil, err
			}
		}

		if err = cc.userClusterEnsureRoles(cluster, client); err != nil {
			return nil, err
		}

		if err = cc.userClusterEnsureConfigMaps(cluster, client); err != nil {
			return nil, err
		}

		if err = cc.userClusterEnsureRoleBindings(cluster, client); err != nil {
			return nil, err
		}

		if err = cc.userClusterEnsureClusterRoles(cluster, client); err != nil {
			return nil, err
		}

		if err = cc.userClusterEnsureClusterRoleBindings(cluster, client); err != nil {
			return nil, err
		}

		if err = cc.userClusterEnsureCustomResourceDefinitions(cluster); err != nil {
			return nil, err
		}

		if err = cc.userClusterEnsureAPIServices(cluster); err != nil {
			return nil, err
		}

		if err = cc.userClusterEnsureServices(cluster); err != nil {
			return nil, err
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

func (cc *Controller) ensureCloudProviderIsInitialized(cluster *kubermaticv1.Cluster) error {
	_, prov, err := provider.ClusterCloudProvider(cc.cps, cluster)
	if err != nil {
		return err
	}
	if prov == nil {
		return fmt.Errorf("no valid provider specified")
	}

	if _, err = prov.InitializeCloudProvider(cluster, cc.updateCluster); err != nil {
		return err
	}

	return nil
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
