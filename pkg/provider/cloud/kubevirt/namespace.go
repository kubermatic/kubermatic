/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package kubevirt

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func namespaceReconciler(name string) reconciling.NamedNamespaceReconcilerFactory {
	return func() (string, reconciling.NamespaceReconciler) {
		return name, func(n *corev1.Namespace) (*corev1.Namespace, error) {
			return n, nil
		}
	}
}

// reconcileNamespace reconciles a dedicated namespace in the underlying KubeVirt cluster.
func reconcileNamespace(ctx context.Context, name string, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, client ctrlruntimeclient.Client) (*kubermaticv1.Cluster, error) {
	cluster, err := update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		kuberneteshelper.AddFinalizer(updatedCluster, FinalizerNamespace)
	})
	if err != nil {
		return cluster, err
	}

	creators := []reconciling.NamedNamespaceReconcilerFactory{
		namespaceReconciler(name),
	}

	if err := reconciling.ReconcileNamespaces(ctx, creators, "", client); err != nil {
		return cluster, fmt.Errorf("failed to reconcile Namespace: %w", err)
	}

	return cluster, nil
}

// deleteNamespace deletes the dedicated namespace.
func deleteNamespace(ctx context.Context, name string, client ctrlruntimeclient.Client) error {
	ns := &corev1.Namespace{}
	if err := client.Get(ctx, types.NamespacedName{Name: name}, ns); err != nil {
		return ctrlruntimeclient.IgnoreNotFound(err)
	}

	return client.Delete(ctx, ns)
}
