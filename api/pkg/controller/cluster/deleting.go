package cluster

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// cleanupCluster is the function which handles clusters in the deleting phase.
// It is responsible for cleaning up a cluster (right now: deleting nodes, deleting cloud-provider infrastructure)
// If this function does not return a pointer to a cluster or a error, the cluster is deleted.
func (cc *Controller) cleanupCluster(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	c, err := cc.deletingNodeCleanup(c)
	if err != nil {
		return nil, err
	}

	// If we still have nodes, we must not cleanup other infrastructure at the cloud provider
	if kuberneteshelper.HasFinalizer(c, nodeDeletionFinalizer) {
		return c, nil
	}

	if c, err = cc.deletingCloudProviderCleanup(c); err != nil {
		return c, err
	}

	// update metrics since we're not handling this cluster anymore
	removeClusterFromMetrics(c)

	return c, nil
}

func (cc *Controller) deletingNodeCleanup(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	if !kuberneteshelper.HasFinalizer(c, nodeDeletionFinalizer) {
		return c, nil
	}

	machineClient, err := cc.userClusterConnProvider.GetMachineClient(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster machine client: %v", err)
	}

	machineList, err := machineClient.MachineV1alpha1().Machines().List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster machines: %v", err)
	}
	if len(machineList.Items) > 0 {
		if err := machineClient.MachineV1alpha1().Machines().DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{}); err != nil {
			return nil, fmt.Errorf("failed to delete cluster machines: %v", err)
		}

		return c, nil
	}

	clusterClient, err := cc.userClusterConnProvider.GetClient(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster client: %v", err)
	}

	nodes, err := clusterClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster nodes: %v", err)
	}

	if len(nodes.Items) == 0 {
		c, err = cc.updateCluster(c.Name, func(c *kubermaticv1.Cluster) {
			c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, nodeDeletionFinalizer)
		})
		if err != nil {
			return nil, err
		}
		return c, nil
	}

	err = clusterClient.CoreV1().Nodes().DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to delete nodes: %v", err)
	}

	return c, nil
}

func (cc *Controller) deletingCloudProviderCleanup(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	_, cp, err := provider.ClusterCloudProvider(cc.cps, c)
	if err != nil {
		return nil, err
	}

	if c, err = cp.CleanUpCloudProvider(c, cc.updateCluster); err != nil {
		return nil, err
	}

	return c, nil
}
