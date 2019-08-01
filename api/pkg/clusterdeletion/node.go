package clusterdeletion

import (
	"context"
	"fmt"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/machine-controller/pkg/node/eviction"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	controllerruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (d *Deletion) cleanupNodes(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if !kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.NodeDeletionFinalizer) {
		return nil
	}

	nodes := &corev1.NodeList{}
	if err := d.userClusterClient.List(ctx, &controllerruntimeclient.ListOptions{}, nodes); err != nil {
		return fmt.Errorf("failed to get user cluster nodes: %v", err)
	}

	// If we delete a cluster, we should disable the eviction on the nodes
	for _, node := range nodes.Items {
		if node.Annotations[eviction.SkipEvictionAnnotationKey] == "true" {
			continue
		}

		err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			// Get latest version of the node to prevent conflict errors
			currentNode := &corev1.Node{}
			if err := d.userClusterClient.Get(ctx, types.NamespacedName{Name: node.Name}, currentNode); err != nil {
				return err
			}
			if currentNode.Annotations == nil {
				currentNode.Annotations = map[string]string{}
			}
			currentNode.Annotations[eviction.SkipEvictionAnnotationKey] = "true"

			return d.userClusterClient.Update(ctx, currentNode)
		})
		if err != nil {
			return fmt.Errorf("failed to add the annotation '%s=true' to node '%s': %v", eviction.SkipEvictionAnnotationKey, node.Name, err)
		}
	}

	machineDeploymentList := &clusterv1alpha1.MachineDeploymentList{}
	listOpts := &controllerruntimeclient.ListOptions{Namespace: metav1.NamespaceSystem}
	if err := d.userClusterClient.List(ctx, listOpts, machineDeploymentList); err != nil {
		return fmt.Errorf("failed to list MachineDeployments: %v", err)
	}
	if len(machineDeploymentList.Items) > 0 {
		// TODO: Use DeleteCollection once https://github.com/kubernetes-sigs/controller-runtime/issues/344 is resolved
		for _, machineDeployment := range machineDeploymentList.Items {
			if err := d.userClusterClient.Delete(ctx, &machineDeployment); err != nil {
				return fmt.Errorf("failed to delete MachineDeployment %q: %v", machineDeployment.Name, err)
			}
		}
		// Return here to make sure we don't attempt to delete MachineSets until the MachineDeployment is actually gone
		return nil
	}

	machineSetList := &clusterv1alpha1.MachineSetList{}
	if err := d.userClusterClient.List(ctx, listOpts, machineSetList); err != nil {
		return fmt.Errorf("failed to list MachineSets: %v", err)
	}
	if len(machineSetList.Items) > 0 {
		// TODO: Use DeleteCollection once https://github.com/kubernetes-sigs/controller-runtime/issues/344 is resolved
		for _, machineSet := range machineSetList.Items {
			if err := d.userClusterClient.Delete(ctx, &machineSet); err != nil {
				return fmt.Errorf("failed to delete MachineSet %q: %v", machineSet.Name, err)
			}
		}
		// Return here to make sure we don't attempt to delete Machines until the MachineSet is actually gone
		return nil
	}

	machineList := &clusterv1alpha1.MachineList{}
	if err := d.userClusterClient.List(ctx, listOpts, machineList); err != nil {
		return fmt.Errorf("failed to get Machines: %v", err)
	}
	if len(machineList.Items) > 0 {
		// TODO: Use DeleteCollection once https://github.com/kubernetes-sigs/controller-runtime/issues/344 is resolved
		for _, machine := range machineList.Items {
			if err := d.userClusterClient.Delete(ctx, &machine); err != nil {
				return fmt.Errorf("failed to delete Machine %q: %v", machine.Name, err)
			}
		}

		return nil
	}

	return d.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
		kuberneteshelper.RemoveFinalizer(c, kubermaticapiv1.NodeDeletionFinalizer)
	})
}
