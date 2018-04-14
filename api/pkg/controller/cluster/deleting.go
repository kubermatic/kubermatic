package cluster

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// cleanupCluster is the function which handles clusters in the deleting phase.
// It is responsible for cleaning up a cluster (right now: deleting nodes, deleting cloud-provider infrastructure)
// If this function does not return a pointer to a cluster or a error, the cluster is deleted.
func (cc *Controller) cleanupCluster(c *kubermaticv1.Cluster) error {
	stillHasNodes, err := cc.deletingNodeCleanup(c)
	if err != nil {
		return err
	}

	if stillHasNodes {
		return nil
	}

	if err := cc.deletingCloudProviderCleanup(c); err != nil {
		return err
	}

	if err := cc.deletingNamespaceCleanup(c); err != nil {
		return err
	}

	return cc.deletingClusterResource(c)
}

func (cc *Controller) deletingNodeCleanup(c *kubermaticv1.Cluster) (bool, error) {
	if !kuberneteshelper.HasFinalizer(c, nodeDeletionFinalizer) {
		return false, nil
	}

	machineClient, err := cc.userClusterConnProvider.GetMachineClient(c)
	if err != nil {
		return true, fmt.Errorf("failed to get cluster machine client: %v", err)
	}

	machineList, err := machineClient.MachineV1alpha1().Machines().List(metav1.ListOptions{})
	if err != nil {
		return true, fmt.Errorf("failed to get cluster machines: %v", err)
	}
	if len(machineList.Items) > 0 {
		if err := machineClient.MachineV1alpha1().Machines().DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{}); err != nil {
			return true, fmt.Errorf("failed to delete cluster machines: %v", err)
		}

		return true, nil
	}

	clusterClient, err := cc.userClusterConnProvider.GetClient(c)
	if err != nil {
		return true, fmt.Errorf("failed to get cluster client: %v", err)
	}

	nodes, err := clusterClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return true, fmt.Errorf("failed to get cluster nodes: %v", err)
	}

	if len(nodes.Items) == 0 {
		c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, nodeDeletionFinalizer)
		return false, nil
	}

	err = clusterClient.CoreV1().Nodes().DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
	if err != nil {
		return true, fmt.Errorf("failed to delete nodes: %v", err)
	}

	return true, nil
}

func (cc *Controller) deletingCloudProviderCleanup(c *kubermaticv1.Cluster) error {
	if !kuberneteshelper.HasFinalizer(c, cloudProviderCleanupFinalizer) {
		return nil
	}

	_, cp, err := provider.ClusterCloudProvider(cc.cps, c)
	if err != nil {
		return err
	}

	if err = cp.CleanUpCloudProvider(c.Spec.Cloud); err != nil {
		return err
	}

	c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, cloudProviderCleanupFinalizer)
	return nil
}

func (cc *Controller) deletingNamespaceCleanup(c *kubermaticv1.Cluster) error {
	if !kuberneteshelper.HasFinalizer(c, namespaceDeletionFinalizer) {
		return nil
	}

	ns, err := cc.NamespaceLister.Get(c.Status.NamespaceName)
	// Only delete finalizer if namespace is really gone
	if err != nil {
		if errors.IsNotFound(err) {
			c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, namespaceDeletionFinalizer)
			return nil
		}
		return err
	}

	if ns.DeletionTimestamp == nil {
		return cc.kubeClient.CoreV1().Namespaces().Delete(c.Status.NamespaceName, &metav1.DeleteOptions{})
	}

	return nil
}

// deletingClusterResource deletes the cluster resource. Needed since Finalizers are broken in 1.7.
func (cc *Controller) deletingClusterResource(c *kubermaticv1.Cluster) error {
	if len(c.Finalizers) != 0 {
		return nil
	}

	if err := cc.kubermaticClient.KubermaticV1().Clusters().Delete(c.Name, &metav1.DeleteOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return nil
}
