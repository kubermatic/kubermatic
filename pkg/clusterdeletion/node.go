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

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	eviction "github.com/kubermatic/machine-controller/pkg/node/eviction/types"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (d *Deletion) cleanupNodes(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if !kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.NodeDeletionFinalizer) {
		return nil
	}

	userClusterClient, err := d.userClusterClientGetter()
	if err != nil {
		return err
	}

	nodes := &corev1.NodeList{}
	if err := userClusterClient.List(ctx, nodes); err != nil {
		return fmt.Errorf("failed to get user cluster nodes: %v", err)
	}

	// If we delete a cluster, we should disable the eviction on the nodes
	for _, node := range nodes.Items {
		if node.Annotations[eviction.SkipEvictionAnnotationKey] == "true" {
			continue
		}

		oldNode := node.DeepCopy()
		if node.Annotations == nil {
			node.Annotations = map[string]string{}
		}
		node.Annotations[eviction.SkipEvictionAnnotationKey] = "true"
		if err := userClusterClient.Patch(ctx, &node, ctrlruntimeclient.MergeFrom(oldNode)); err != nil {
			return fmt.Errorf("failed to add the annotation '%s=true' to node '%s': %v", eviction.SkipEvictionAnnotationKey, node.Name, err)
		}
	}

	machineDeploymentList := &clusterv1alpha1.MachineDeploymentList{}
	listOpts := &ctrlruntimeclient.ListOptions{Namespace: metav1.NamespaceSystem}
	if err := userClusterClient.List(ctx, machineDeploymentList, listOpts); err != nil && !meta.IsNoMatchError(err) {
		return fmt.Errorf("failed to list MachineDeployments: %v", err)
	}
	if len(machineDeploymentList.Items) > 0 {
		// TODO: Use DeleteCollection once https://github.com/kubernetes-sigs/controller-runtime/issues/344 is resolved
		for _, machineDeployment := range machineDeploymentList.Items {
			if err := userClusterClient.Delete(ctx, &machineDeployment); err != nil {
				return fmt.Errorf("failed to delete MachineDeployment %q: %v", machineDeployment.Name, err)
			}
		}
		// Return here to make sure we don't attempt to delete MachineSets until the MachineDeployment is actually gone
		return nil
	}

	machineSetList := &clusterv1alpha1.MachineSetList{}
	if err := userClusterClient.List(ctx, machineSetList, listOpts); err != nil && !meta.IsNoMatchError(err) {
		return fmt.Errorf("failed to list MachineSets: %v", err)
	}
	if len(machineSetList.Items) > 0 {
		// TODO: Use DeleteCollection once https://github.com/kubernetes-sigs/controller-runtime/issues/344 is resolved
		for _, machineSet := range machineSetList.Items {
			if err := userClusterClient.Delete(ctx, &machineSet); err != nil {
				return fmt.Errorf("failed to delete MachineSet %q: %v", machineSet.Name, err)
			}
		}
		// Return here to make sure we don't attempt to delete Machines until the MachineSet is actually gone
		return nil
	}

	machineList := &clusterv1alpha1.MachineList{}
	if err := userClusterClient.List(ctx, machineList, listOpts); err != nil && !meta.IsNoMatchError(err) {
		return fmt.Errorf("failed to get Machines: %v", err)
	}
	if len(machineList.Items) > 0 {
		// TODO: Use DeleteCollection once https://github.com/kubernetes-sigs/controller-runtime/issues/344 is resolved
		for _, machine := range machineList.Items {
			if err := userClusterClient.Delete(ctx, &machine); err != nil {
				return fmt.Errorf("failed to delete Machine %q: %v", machine.Name, err)
			}
		}

		return nil
	}

	oldCluster := cluster.DeepCopy()
	kuberneteshelper.RemoveFinalizer(cluster, kubermaticapiv1.NodeDeletionFinalizer)
	return d.seedClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}
