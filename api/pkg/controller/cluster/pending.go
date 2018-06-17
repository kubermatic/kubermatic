package cluster

import (
	"fmt"

	"github.com/golang/glog"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

func (cc *Controller) reconcileCluster(cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	var err error
	// Create the namespace
	if err := cc.ensureNamespaceExists(cluster); err != nil {
		return nil, err
	}

	// Setup required infrastructure at cloud provider
	if err := cc.ensureCloudProviderIsInitialize(cluster); err != nil {
		return nil, err
	}

	// Set the hostname & url
	if cluster, err = cc.syncAddress(cluster); err != nil {
		return nil, err
	}

	// Set default network configuration
	if err := cc.ensureClusterNetworkDefaults(cluster); err != nil {
		return nil, err
	}

	// check that all service accounts are created
	if err := cc.ensureCheckServiceAccounts(cluster); err != nil {
		return nil, err
	}

	// check that all roles are created
	if err := cc.ensureRoles(cluster); err != nil {
		return nil, err
	}

	// check that all role bindings are created
	if err := cc.ensureRoleBindings(cluster); err != nil {
		return nil, err
	}

	// check that all role bindings are created
	if err := cc.ensureClusterRoleBindings(cluster); err != nil {
		return nil, err
	}

	// check that all services are available
	if err := cc.ensureServices(cluster); err != nil {
		return nil, err
	}

	// check that all secrets are available
	if err := cc.ensureSecrets(cluster); err != nil {
		return nil, err
	}

	// check that all ConfigMap's are available
	if err := cc.ensureConfigMaps(cluster); err != nil {
		return nil, err
	}

	// check that all deployments are available
	if err := cc.ensureDeployments(cluster); err != nil {
		return nil, err
	}

	// check that all StatefulSet's are created
	if err := cc.ensureStatefulSets(cluster); err != nil {
		return nil, err
	}

	// synchronize cluster.status.health
	if err := cc.syncHealth(cluster); err != nil {
		return nil, err
	}

	if cluster.Status.Health.Apiserver {
		if err := cc.ensureClusterReachable(cluster); err != nil {
			return nil, err
		}

		if err := cc.launchingCreateClusterInfoConfigMap(cluster); err != nil {
			return nil, err
		}

		if err := cc.launchingCreateOpenVPNClientCertificates(cluster); err != nil {
			return nil, err
		}

		if err := cc.launchingCreateOpenVPNConfigMap(cluster); err != nil {
			return nil, err
		}
	}

	if !cluster.Status.Health.AllHealthy() {
		glog.V(5).Infof("Cluster %q not yet healthy: %+v", cluster.Name, cluster.Status.Health)
		return cluster, nil
	}

	if cluster.Status.Phase == kubermaticv1.LaunchingClusterStatusPhase {
		cluster.Status.Phase = kubermaticv1.RunningClusterStatusPhase
	}

	return cluster, nil
}

func (cc *Controller) ensureCloudProviderIsInitialize(cluster *kubermaticv1.Cluster) error {
	_, prov, err := provider.ClusterCloudProvider(cc.cps, cluster)
	if err != nil {
		return err
	}
	if prov == nil {
		return fmt.Errorf("no valid provider specified")
	}

	if cluster, err = prov.InitializeCloudProvider(cluster); err != nil {
		return err
	}

	return nil
}

// ensureClusterNetworkDefaults will apply default cluster network configuration
func (cc *Controller) ensureClusterNetworkDefaults(c *kubermaticv1.Cluster) error {
	if len(c.Spec.ClusterNetwork.Services.CIDRBlocks) == 0 {
		c.Spec.ClusterNetwork.Services.CIDRBlocks = []string{"10.10.10.0/24"}
	}
	if len(c.Spec.ClusterNetwork.Pods.CIDRBlocks) == 0 {
		c.Spec.ClusterNetwork.Pods.CIDRBlocks = []string{"172.25.0.0/16"}
	}
	if c.Spec.ClusterNetwork.DNSDomain == "" {
		c.Spec.ClusterNetwork.DNSDomain = "cluster.local"
	}
	return nil
}
