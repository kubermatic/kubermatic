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

package clusterdeletion

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (d *Deletion) cleanupNamespace(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	if !kuberneteshelper.HasFinalizer(cluster, kubermaticv1.NamespaceCleanupFinalizer) {
		return nil
	}

	// It can happen that the namespace is correctly created, but the status update failed.
	// In this case we replicate the cluster controller's defaulting behaviour to catch the
	// default namespace name.
	namespace := cluster.Status.NamespaceName
	if namespace == "" {
		namespace = kubernetesprovider.NamespaceName(cluster.Name)
	}

	// check if the namespace still exists
	ns := &corev1.Namespace{}
	ns.Name = namespace

	err := d.seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(ns), ns)
	if ctrlruntimeclient.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to check for cluster namespace: %w", err)
	}

	// namespace could still be retrieved
	if err == nil {
		if ns.DeletionTimestamp == nil {
			log.Infow("deleting cluster namespace", "namespace", ns.Name)
			if err := d.seedClient.Delete(ctx, ns); ctrlruntimeclient.IgnoreNotFound(err) != nil {
				return fmt.Errorf("failed to delete cluster namespace: %w", err)
			}
		}

		d.recorder.Eventf(cluster, nil, corev1.EventTypeNormal, "ClusterNamespaceCleanup", "Reconciling", "Cluster namespace is still terminating, some resources might be blocked by finalizers.")
		return nil
	}

	// Removing the NamespaceName from the Cluster will make all other controllers
	// instantly stop reconciling this one, without even checking its DeletionTimestamp.
	if cluster.Status.NamespaceName != "" {
		err = util.UpdateClusterStatus(ctx, d.seedClient, cluster, func(c *kubermaticv1.Cluster) {
			c.Status.NamespaceName = ""
		})
		if err != nil {
			return fmt.Errorf("failed to update cluster namespace status: %w", err)
		}
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, d.seedClient, cluster, kubermaticv1.NamespaceCleanupFinalizer)
}
