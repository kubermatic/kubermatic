package cluster

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/azure"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/openstack"

	"github.com/kubermatic/machine-controller/pkg/node/eviction"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

const (
	inClusterPVCleanupFinalizer = "kubermatic.io/cleanup-in-cluster-pv"
	inClusterLBCleanupFinalizer = "kubermatic.io/cleanup-in-cluster-lb"
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

	// Delete Volumes and LB's inside the user cluster
	if c, err = cc.cleanupInClusterResources(c); err != nil {
		return c, err
	}

	// Do not run the cloud provider cleanup until we finished the PV & LB cleanup.
	// Otherwise we risk deleting those resources as well
	if kuberneteshelper.HasFinalizer(c, inClusterLBCleanupFinalizer) || kuberneteshelper.HasFinalizer(c, inClusterPVCleanupFinalizer) {
		return c, nil
	}

	if c, err = cc.deletingCloudProviderCleanup(c); err != nil {
		return c, err
	}

	return c, nil
}

func (cc *Controller) cleanupInClusterResources(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	shouldDeleteLBs := kuberneteshelper.HasFinalizer(c, inClusterLBCleanupFinalizer)
	shouldDeletePVs := kuberneteshelper.HasFinalizer(c, inClusterPVCleanupFinalizer)

	// If no relevant finalizer exists, directly return
	if !shouldDeleteLBs && !shouldDeletePVs {
		return c, nil
	}

	client, err := cc.userClusterConnProvider.GetClient(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes client: %v", err)
	}

	// We'll set this to true in case we deleted something. This is meant to requeue as long as all resources are really gone
	// We'll use it for LB's and PV's as well, so the Kubernetes controller manager does the cleanup of all resources in parallel
	var deletedSomeResource bool

	if shouldDeleteLBs {
		serviceList, err := client.CoreV1().Services(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list Service's from user cluster: %v", err)
		}

		for _, service := range serviceList.Items {
			if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
				if err := client.CoreV1().Services(service.Namespace).Delete(service.Name, &metav1.DeleteOptions{}); err != nil {
					return nil, fmt.Errorf("failed to delete Service '%s/%s' from user cluster: %v", service.Namespace, service.Name, err)
				}
				deletedSomeResource = true
			}
		}
	}

	if shouldDeletePVs {
		// Delete PVC's
		pvcList, err := client.CoreV1().PersistentVolumeClaims(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list services from user cluster: %v", err)
		}

		for _, pvc := range pvcList.Items {
			if err := client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(pvc.Name, &metav1.DeleteOptions{}); err != nil {
				return nil, fmt.Errorf("failed to delete PVC '%s/%s' from user cluster: %v", pvc.Namespace, pvc.Name, err)
			}
			deletedSomeResource = true
		}

		// Delete PV's
		pvList, err := client.CoreV1().PersistentVolumes().List(metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list services from user cluster: %v", err)
		}

		for _, pv := range pvList.Items {
			if err := client.CoreV1().PersistentVolumes().Delete(pv.Name, &metav1.DeleteOptions{}); err != nil {
				return nil, fmt.Errorf("failed to delete PV '%s' from user cluster: %v", pv.Name, err)
			}
			deletedSomeResource = true
		}
	}

	if deletedSomeResource {
		return c, nil
	}

	c, err = cc.updateCluster(c.Name, func(c *kubermaticv1.Cluster) {
		// In case we should keep the LB's, we must remove some cloud provider specific finalizers.
		// Otherwise we might break the LB
		if !shouldDeleteLBs {
			// OpenStack
			c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, openstack.OldNetworkCleanupFinalizer)
			c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, openstack.NetworkCleanupFinalizer)
			c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, openstack.SubnetCleanupFinalizer)
			c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, openstack.RouterCleanupFinalizer)
			c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, openstack.RouterSubnetLinkCleanupFinalizer)

			// Azure
			c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, azure.FinalizerRouteTable)
			c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, azure.FinalizerSubnet)
			c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, azure.FinalizerSubnet)
			c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, azure.FinalizerVNet)
			c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, azure.FinalizerResourceGroup)
			c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, azure.FinalizerAvailabilitySet)
		}

		if !shouldDeletePVs {
			// Azure
			c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, azure.FinalizerResourceGroup)
		}

		c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, inClusterLBCleanupFinalizer)
		c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, inClusterPVCleanupFinalizer)
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (cc *Controller) deletingNodeCleanup(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	if !kuberneteshelper.HasFinalizer(c, nodeDeletionFinalizer) {
		return c, nil
	}

	userClusterCoreClient, err := cc.userClusterConnProvider.GetClient(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get user cluster client: %v", err)
	}

	nodes, err := userClusterCoreClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get user cluster nodes: %v", err)
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
		return nil, fmt.Errorf("failed to get machine client: %v", err)
	}

	machineDeploymentList, err := machineClient.ClusterV1alpha1().MachineDeployments(metav1.NamespaceSystem).List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list MachineDeployments: %v", err)
	}
	if len(machineDeploymentList.Items) > 0 {
		if err := machineClient.ClusterV1alpha1().MachineDeployments(metav1.NamespaceSystem).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{}); err != nil {
			return nil, fmt.Errorf("failed to delete MachineDeployments: %v", err)
		}
		// Return here to make sure we don't attempt to delete MachineSets until the MachineDeployment is actually gone
		return c, nil
	}

	machineSetList, err := machineClient.ClusterV1alpha1().MachineSets(metav1.NamespaceSystem).List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list MachineSets: %v", err)
	}
	if len(machineSetList.Items) > 0 {
		if err := machineClient.ClusterV1alpha1().MachineSets(metav1.NamespaceSystem).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{}); err != nil {
			return nil, fmt.Errorf("failed to delete MachineSets: %v", err)
		}
		// Return here to make sure we don't attempt to delete Machines until the MachineSet is actually gone
		return c, nil
	}

	machineList, err := machineClient.ClusterV1alpha1().Machines(metav1.NamespaceSystem).List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Machines: %v", err)
	}
	if len(machineList.Items) > 0 {
		if err = machineClient.ClusterV1alpha1().Machines(metav1.NamespaceSystem).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{}); err != nil {
			return nil, fmt.Errorf("failed to delete Machines: %v", err)
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
