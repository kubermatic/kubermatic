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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
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
func (r *Reconciler) cleanupCluster(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	err := r.deletingNodeCleanup(cluster)
	if err != nil {
		return err
	}

	// If we still have nodes, we must not cleanup other infrastructure at the cloud provider
	if kuberneteshelper.HasFinalizer(cluster, NodeDeletionFinalizer) {
		return nil
	}

	// Delete Volumes and LB's inside the user cluster
	return r.cleanupInClusterResources(ctx, cluster)
}

func (r *Reconciler) cleanupInClusterResources(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	shouldDeleteLBs := kuberneteshelper.HasFinalizer(cluster, InClusterLBCleanupFinalizer)
	shouldDeletePVs := kuberneteshelper.HasFinalizer(cluster, InClusterPVCleanupFinalizer)

	// If no relevant finalizer exists, directly return
	if !shouldDeleteLBs && !shouldDeletePVs {
		return nil
	}

	client, err := r.userClusterConnProvider.GetClient(cluster)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %v", err)
	}

	// We'll set this to true in case we deleted something. This is meant to requeue as long as all resources are really gone
	// We'll use it for LB's and PV's as well, so the Kubernetes controller manager does the cleanup of all resources in parallel
	var deletedSomeResource bool

	if shouldDeleteLBs {
		serviceList, err := client.CoreV1().Services(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list Service's from user cluster: %v", err)
		}

		for _, service := range serviceList.Items {
			// Need to change the scope so the inline func in the updateCluster call always has the service from the current iteration
			service := service
			if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
				if err := client.CoreV1().Services(service.Namespace).Delete(service.Name, &metav1.DeleteOptions{}); err != nil {
					return fmt.Errorf("failed to delete Service '%s/%s' from user cluster: %v", service.Namespace, service.Name, err)
				}
				deletedSomeResource = true
				err = r.updateCluster(ctx, cluster, func(cluster *kubermaticv1.Cluster) {
					if cluster.Annotations == nil {
						cluster.Annotations = map[string]string{}
					}
					cluster.Annotations[deletedLBAnnotationName] = cluster.Annotations[deletedLBAnnotationName] + fmt.Sprintf(",%s", string(service.UID))
				})
				if err != nil {
					return fmt.Errorf("failed to update cluster when trying to add UID of deleted LoadBalancer: %v", err)
				}
				// Wait for the update to appear in the lister as we use the data from the lister later on to verify if the LoadBalancers
				// are gone
				if err := wait.Poll(10*time.Millisecond, 5*time.Second, func() (bool, error) {
					latestCluster := &kubermaticv1.Cluster{}
					if err := r.Get(ctx, types.NamespacedName{Name: cluster.Name}, latestCluster); err != nil {
						return false, err
					}
					if strings.Contains(latestCluster.Annotations[deletedLBAnnotationName], string(service.UID)) {
						return true, nil
					}
					return false, nil
				}); err != nil {
					return fmt.Errorf("failed to wait for deletedLBAnnotation to appear in the lister: %v", err)
				}
			}
		}
	}

	if shouldDeletePVs {
		// Delete PVC's
		pvcList, err := client.CoreV1().PersistentVolumeClaims(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list services from user cluster: %v", err)
		}

		for _, pvc := range pvcList.Items {
			if err := client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(pvc.Name, &metav1.DeleteOptions{}); err != nil {
				return fmt.Errorf("failed to delete PVC '%s/%s' from user cluster: %v", pvc.Namespace, pvc.Name, err)
			}
			deletedSomeResource = true
		}

		// Delete PV's
		pvList, err := client.CoreV1().PersistentVolumes().List(metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list services from user cluster: %v", err)
		}

		for _, pv := range pvList.Items {
			if err := client.CoreV1().PersistentVolumes().Delete(pv.Name, &metav1.DeleteOptions{}); err != nil {
				return fmt.Errorf("failed to delete PV '%s' from user cluster: %v", pv.Name, err)
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
		return nil
	}

	lbsAreGone, err := r.checkIfAllLoadbalancersAreGone(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to check if all Loadbalancers are gone: %v", err)
	}
	// Return so we check again later
	if !lbsAreGone {
		return nil
	}

	return r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
		c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, InClusterLBCleanupFinalizer)
		c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, InClusterPVCleanupFinalizer)
	})
}

func (r *Reconciler) deletingNodeCleanup(cluster *kubermaticv1.Cluster) error {
	if !kuberneteshelper.HasFinalizer(cluster, NodeDeletionFinalizer) {
		return nil
	}

	client, err := r.userClusterConnProvider.GetDynamicClient(cluster)
	if err != nil {
		return fmt.Errorf("failed to get user cluster client: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := &corev1.NodeList{}
	if err := client.List(ctx, &controllerruntimeclient.ListOptions{}, nodes); err != nil {
		return fmt.Errorf("failed to get user cluster nodes: %v", err)
	}

	// If we delete a cluster, we should disable the eviction on the nodes
	for _, node := range nodes.Items {
		if node.Annotations[eviction.SkipEvictionAnnotationKey] == "true" {
			continue
		}

		err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			// Get latest version of the node to prevent conflict errors
			currentNode := &corev1.Node{}
			if err := client.Get(ctx, types.NamespacedName{Name: node.Name}, currentNode); err != nil {
				return err
			}
			if currentNode.Annotations == nil {
				currentNode.Annotations = map[string]string{}
			}
			node.Annotations[eviction.SkipEvictionAnnotationKey] = "true"

			return client.Update(ctx, currentNode)
		})
		if err != nil {
			return fmt.Errorf("failed to add the annotation '%s=true' to node '%s': %v", eviction.SkipEvictionAnnotationKey, node.Name, err)
		}
	}

	machineDeploymentList := &clusterv1alpha1.MachineDeploymentList{}
	listOpts := &controllerruntimeclient.ListOptions{Namespace: metav1.NamespaceSystem}
	if err := client.List(ctx, listOpts, machineDeploymentList); err != nil {
		return fmt.Errorf("failed to list MachineDeployments: %v", err)
	}
	if len(machineDeploymentList.Items) > 0 {
		// TODO: Use DeleteCollection once https://github.com/kubernetes-sigs/controller-runtime/issues/344 is resolved
		for _, machineDeployment := range machineDeploymentList.Items {
			if err := client.Delete(ctx, &machineDeployment); err != nil {
				return fmt.Errorf("failed to delete MachineDeployment %q: %v", machineDeployment.Name, err)
			}
		}
		// Return here to make sure we don't attempt to delete MachineSets until the MachineDeployment is actually gone
		return nil
	}

	machineSetList := &clusterv1alpha1.MachineSetList{}
	if err = client.List(ctx, listOpts, machineSetList); err != nil {
		return fmt.Errorf("failed to list MachineSets: %v", err)
	}
	if len(machineSetList.Items) > 0 {
		// TODO: Use DeleteCollection once https://github.com/kubernetes-sigs/controller-runtime/issues/344 is resolved
		for _, machineSet := range machineSetList.Items {
			if err := client.Delete(ctx, &machineSet); err != nil {
				return fmt.Errorf("failed to delete MachineSet %q: %v", machineSet.Namespace, err)
			}
		}
		// Return here to make sure we don't attempt to delete Machines until the MachineSet is actually gone
		return nil
	}

	machineList := &clusterv1alpha1.MachineList{}
	if err := client.List(ctx, listOpts, machineList); err != nil {
		return fmt.Errorf("failed to get Machines: %v", err)
	}
	if len(machineList.Items) > 0 {
		// TODO: Use DeleteCollection once https://github.com/kubernetes-sigs/controller-runtime/issues/344 is resolved
		for _, machine := range machineList.Items {
			if err := client.Delete(ctx, &machine); err != nil {
				return fmt.Errorf("failed to delete Machine %q: %v", machine.Name, err)
			}
		}

		return nil
	}

	return r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
		c.Finalizers = kuberneteshelper.RemoveFinalizer(c.Finalizers, NodeDeletionFinalizer)
	})
}

// checkIfAllLoadbalancersAreGone checks if all the services of type LoadBalancer were successfully
// deleted. The in-tree cloud providers do this without a finalizer and only after the service
// object is gone from the API, the only way to check is to wait for the relevant event
func (r *Reconciler) checkIfAllLoadbalancersAreGone(ctx context.Context, cluster *kubermaticv1.Cluster) (bool, error) {
	// This check is only required for in-tree cloud provider that support LoadBalancers
	// TODO once we start external cloud controllers for one of these three: Make this check
	// a bit smarter, external cloud controllers will most likely not emit the event we wait for
	if cluster.Spec.Cloud.AWS == nil && cluster.Spec.Cloud.Azure == nil && cluster.Spec.Cloud.Openstack == nil {
		return true, nil
	}

	// We only need to wait for this if there were actually services of type Loadbalancer deleted
	if cluster.Annotations[deletedLBAnnotationName] == "" {
		return true, nil
	}

	deletedLoadBalancers := sets.NewString(strings.Split(strings.TrimPrefix(cluster.Annotations[deletedLBAnnotationName], ","), ",")...)

	// Kubernetes gives no guarantees at all about events, it is possible we don't get the event
	// so bail out after 2h
	if cluster.DeletionTimestamp.UTC().Add(2 * time.Hour).Before(time.Now().UTC()) {
		staleLBs.WithLabelValues(cluster.Name).Set(float64(deletedLoadBalancers.Len()))
		return true, nil
	}

	userClusterDynamicClient, err := r.userClusterConnProvider.GetDynamicClient(cluster)
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
	err = r.updateCluster(ctx, cluster, func(cluster *kubermaticv1.Cluster) {
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
