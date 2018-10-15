package cluster

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"github.com/kubermatic/machine-controller/pkg/node/eviction"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
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

	return c, nil
}

func (cc *Controller) deletingNodeCleanup(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	if !kuberneteshelper.HasFinalizer(c, nodeDeletionFinalizer) {
		return c, nil
	}

	userClusterCoreClient, err := cc.userClusterConnProvider.GetClient(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster client: %v", err)
	}

	nodes, err := userClusterCoreClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster nodes: %v", err)
	}

	// If we delete a cluster, we should disable the eviction on the nodes
	for _, node := range nodes.Items {
		if node.Annotations[eviction.SkipEvictionAnnotationKey] == "true" {
			continue
		}

		err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			// Get latest version of the node to prevent conflict errors
			currentNode, err := userClusterCoreClient.CoreV1().Nodes().Get(node.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			if currentNode.Annotations == nil {
				currentNode.Annotations = map[string]string{}
			}
			node.Annotations[eviction.SkipEvictionAnnotationKey] = "true"

			currentNode, err = userClusterCoreClient.CoreV1().Nodes().Update(&node)
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add the annotation '%s=true' to node '%s': %v", eviction.SkipEvictionAnnotationKey, node.Name, err)
		}
	}

	machineClient, err := cc.userClusterConnProvider.GetMachineClient(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster machine client: %v", err)
	}

	machineList, err := machineClient.ClusterV1alpha1().Machines(metav1.NamespaceSystem).List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster machines: %v", err)
	}
	if len(machineList.Items) > 0 {
		if err = machineClient.ClusterV1alpha1().Machines(metav1.NamespaceSystem).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{}); err != nil {
			return nil, fmt.Errorf("failed to delete cluster machines: %v", err)
		}

		return c, nil
	}

	c, err = cc.updateCluster(c.Name, func(c *kubermaticv1.Cluster) {
		c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, nodeDeletionFinalizer)
	})
	if err != nil {
		return nil, err
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
