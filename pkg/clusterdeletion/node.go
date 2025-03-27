/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clusterdeletion

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	nodetypes "k8c.io/machine-controller/sdk/node"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (d *Deletion) cleanupNodes(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if !kuberneteshelper.HasFinalizer(cluster, kubermaticv1.NodeDeletionFinalizer) {
		return nil
	}

	if cluster.Status.NamespaceName == "" {
		return kuberneteshelper.TryRemoveFinalizer(ctx, d.seedClient, cluster, kubermaticv1.NodeDeletionFinalizer)
	}

	userClusterClient, err := d.userClusterClientGetter()
	if err != nil {
		return err
	}

	nodes := &corev1.NodeList{}
	if err := userClusterClient.List(ctx, nodes); err != nil {
		return fmt.Errorf("failed to get user cluster nodes: %w", err)
	}

	// If we delete a cluster, we should disable the eviction on the nodes
	for _, node := range nodes.Items {
		if node.Annotations[nodetypes.SkipEvictionAnnotationKey] == "true" {
			continue
		}

		oldNode := node.DeepCopy()
		if node.Annotations == nil {
			node.Annotations = map[string]string{}
		}
		node.Annotations[nodetypes.SkipEvictionAnnotationKey] = "true"
		if err := userClusterClient.Patch(ctx, &node, ctrlruntimeclient.MergeFrom(oldNode)); err != nil {
			return fmt.Errorf("failed to add the annotation '%s=true' to node '%s': %w", nodetypes.SkipEvictionAnnotationKey, node.Name, err)
		}
	}

	listOpts := ctrlruntimeclient.InNamespace(metav1.NamespaceSystem)

	machineDeploymentList := &clusterv1alpha1.MachineDeploymentList{}
	if err := userClusterClient.List(ctx, machineDeploymentList, listOpts); err != nil && !meta.IsNoMatchError(err) {
		return fmt.Errorf("failed to list MachineDeployments: %w", err)
	}

	if len(machineDeploymentList.Items) > 0 {
		if err = userClusterClient.DeleteAllOf(ctx, &clusterv1alpha1.MachineDeployment{}, listOpts); err != nil {
			return fmt.Errorf("failed to delete MachineDeployments: %w", err)
		}

		// Return here to make sure we don't attempt to delete MachineSets until the MachineDeployment is actually gone
		d.recorder.Eventf(cluster, corev1.EventTypeNormal, "NodeCleanup", "Waiting for %d MachineDeployment(s) to be destroyed.", len(machineDeploymentList.Items))
		return nil
	}

	machineSetList := &clusterv1alpha1.MachineSetList{}
	if err := userClusterClient.List(ctx, machineSetList, listOpts); err != nil && !meta.IsNoMatchError(err) {
		return fmt.Errorf("failed to list MachineSets: %w", err)
	}

	if len(machineSetList.Items) > 0 {
		if err = userClusterClient.DeleteAllOf(ctx, &clusterv1alpha1.MachineSet{}, listOpts); err != nil {
			return fmt.Errorf("failed to delete MachineSets: %w", err)
		}

		// Return here to make sure we don't attempt to delete Machines until the MachineSet is actually gone
		d.recorder.Eventf(cluster, corev1.EventTypeNormal, "NodeCleanup", "Waiting for %d MachineSet(s) to be destroyed.", len(machineSetList.Items))
		return nil
	}

	machineList := &clusterv1alpha1.MachineList{}
	if err := userClusterClient.List(ctx, machineList, listOpts); err != nil && !meta.IsNoMatchError(err) {
		return fmt.Errorf("failed to get Machines: %w", err)
	}

	if len(machineList.Items) > 0 {
		if err = userClusterClient.DeleteAllOf(ctx, &clusterv1alpha1.Machine{}, listOpts); err != nil {
			return fmt.Errorf("failed to delete Machines: %w", err)
		}

		d.recorder.Eventf(cluster, corev1.EventTypeNormal, "NodeCleanup", "Waiting for %d Machine(s) to be destroyed.", len(machineList.Items))
		return nil
	}

	d.recorder.Event(cluster, corev1.EventTypeNormal, "NodeCleanup", "Cleanup has been completed, all machines have been destroyed.")

	return kuberneteshelper.TryRemoveFinalizer(ctx, d.seedClient, cluster, kubermaticv1.NodeDeletionFinalizer)
}
