package cluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"

	"github.com/kubermatic/machine-controller/pkg/node/eviction"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	controllerruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	InClusterPVCleanupFinalizer = "kubermatic.io/cleanup-in-cluster-pv"
	InClusterLBCleanupFinalizer = "kubermatic.io/cleanup-in-cluster-lb"
	deletedLBAnnotationName     = "kubermatic.io/cleaned-up-loadbalancers"
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
	if kuberneteshelper.HasFinalizer(c, NodeDeletionFinalizer) {
		return c, nil
	}

	// Delete Volumes and LB's inside the user cluster
	return cc.cleanupInClusterResources(c)
}

func (cc *Controller) cleanupInClusterResources(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	shouldDeleteLBs := kuberneteshelper.HasFinalizer(c, InClusterLBCleanupFinalizer)
	shouldDeletePVs := kuberneteshelper.HasFinalizer(c, InClusterPVCleanupFinalizer)

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
			// Need to change the scope so the inline func in the updateCluster call always has the service from the current iteration
			service := service
			if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
				if err := client.CoreV1().Services(service.Namespace).Delete(service.Name, &metav1.DeleteOptions{}); err != nil {
					return nil, fmt.Errorf("failed to delete Service '%s/%s' from user cluster: %v", service.Namespace, service.Name, err)
				}
				deletedSomeResource = true
				c, err = cc.updateCluster(c.Name, func(cluster *kubermaticv1.Cluster) {
					if cluster.Annotations == nil {
						cluster.Annotations = map[string]string{}
					}
					cluster.Annotations[deletedLBAnnotationName] = cluster.Annotations[deletedLBAnnotationName] + fmt.Sprintf(",%s", string(service.UID))
				})
				if err != nil {
					return nil, fmt.Errorf("failed to update cluster when trying to add UID of deleted LoadBalancer: %v", err)
				}
				// Wait for the update to appear in the lister as we use the data from the lister later on to verify if the LoadBalancers
				// are gone
				if err := wait.Poll(10*time.Millisecond, 5*time.Second, func() (bool, error) {
					clusterFromLister, err := cc.clusterLister.Get(c.Name)
					if err != nil {
						return false, err
					}
					if strings.Contains(clusterFromLister.Annotations[deletedLBAnnotationName], string(service.UID)) {
						return true, nil
					}
					return false, nil
				}); err != nil {
					return nil, fmt.Errorf("failed to wait for deletedLBAnnotation to appear in the lister: %v", err)
				}
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

	// If we deleted something it is implied that there was still something left. Just return
	// here so the finalizers stay, it will make the cluster controller requeue us after a delay
	// This also means that we may end up issuing multiple DELETE calls against the same ressource
	// if cleaning up takes some time, but that shouldn't cause any harm
	// We also need to return when something was deleted so the checkIfAllLoadbalancersAreGone
	// call gets an updated version of the cluster from the lister
	if deletedSomeResource {
		return c, nil
	}

	lbsAreGone, err := cc.checkIfAllLoadbalancersAreGone(c)
	if err != nil {
		return nil, fmt.Errorf("failed to check if all Loadbalancers are gone: %v", err)
	}
	// Return so we check again later
	if !lbsAreGone {
		return c, nil
	}

	return cc.updateCluster(c.Name, func(c *kubermaticv1.Cluster) {
		c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, InClusterLBCleanupFinalizer)
		c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, InClusterPVCleanupFinalizer)
	})
}

func (cc *Controller) deletingNodeCleanup(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	if !kuberneteshelper.HasFinalizer(c, NodeDeletionFinalizer) {
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
		c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, NodeDeletionFinalizer)
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

// checkIfAllLoadbalancersAreGone checks if all the services of type LoadBalancer were successfully
// deleted. The in-tree cloud providers do this without a finalizer and only after the service
// object is gone from the API, the only way to check is to wait for the relevant event
func (cc *Controller) checkIfAllLoadbalancersAreGone(c *kubermaticv1.Cluster) (bool, error) {
	// This check is only required for in-tree cloud provider that support LoadBalancers
	// TODO once we start external cloud controllers for one of these three: Make this check
	// a bit smarter, external cloud controllers will most likely not emit the event we wait for
	if c.Spec.Cloud.AWS == nil && c.Spec.Cloud.Azure == nil && c.Spec.Cloud.Openstack == nil {
		return true, nil
	}

	// We only need to wait for this if there were actually services of type Loadbalancer deleted
	if c.Annotations[deletedLBAnnotationName] == "" {
		return true, nil
	}

	deletedLoadBalancers := sets.NewString(strings.Split(strings.TrimPrefix(c.Annotations[deletedLBAnnotationName], ","), ",")...)

	// Kubernetes gives no guarantees at all about events, it is possible we don't get the event
	// so bail out after 2h
	if c.DeletionTimestamp.UTC().Add(2 * time.Hour).Before(time.Now().UTC()) {
		staleLBs.WithLabelValues(c.Name).Set(float64(deletedLoadBalancers.Len()))
		return true, nil
	}

	userClusterDynamicClient, err := cc.userClusterConnProvider.GetDynamicClient(c)
	if err != nil {
		return false, fmt.Errorf("failed to get dynamic client for user cluster: %v", err)
	}
	for deletedLB := range deletedLoadBalancers {
		selector := fields.OneTermEqualSelector("involvedObject.uid", deletedLB)
		events := &corev1.EventList{}
		if err := userClusterDynamicClient.List(context.Background(), &controllerruntimeclient.ListOptions{FieldSelector: selector}, events); err != nil {
			return false, fmt.Errorf("failed to get service events: %v", err)
		}
		for _, event := range events.Items {
			if event.Reason == "DeletedLoadBalancer" {
				deletedLoadBalancers.Delete(deletedLB)
			}
		}

	}
	c, err = cc.updateCluster(c.Name, func(cluster *kubermaticv1.Cluster) {
		if deletedLoadBalancers.Len() > 0 {
			cluster.Annotations[deletedLBAnnotationName] = strings.Join(deletedLoadBalancers.List(), ",")
		} else {
			delete(cluster.Annotations, deletedLBAnnotationName)
		}
	})
	if err != nil {
		return false, fmt.Errorf("failed to update cluster: %v", err)
	}
	if deletedLoadBalancers.Len() > 0 {
		return false, nil
	}
	return true, nil
}
