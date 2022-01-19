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

package kubernetes

import (
	"context"
	"time"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	reachableCheckPeriod = 5 * time.Second
)

func (r *Reconciler) reconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// Create the namespace
	namespace, err := r.ensureNamespaceExists(ctx, cluster)
	if err != nil {
		return nil, err
	}

	if !kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.EtcdBackupConfigCleanupFinalizer) {
		res, err := r.AddFinalizers(ctx, cluster, kubermaticapiv1.EtcdBackupConfigCleanupFinalizer)
		if err != nil {
			return nil, err
		}
		if !res.IsZero() {
			return res, nil
		}
	}

	// Apply etcdLauncher flag
	if err := r.ensureEtcdLauncherFeatureFlag(ctx, cluster); err != nil {
		return nil, err
	}

	// Deploy & Update master components for Kubernetes
	res, err := r.ensureResourcesAreDeployed(ctx, cluster, namespace)
	if err != nil {
		return nil, err
	}
	if !res.IsZero() {
		return res, nil
	}

	var finalizers []string
	if cluster.Status.ExtendedHealth.Apiserver == kubermaticv1.HealthStatusUp {
		// Controlling of user-cluster resources
		reachable, err := r.clusterIsReachable(ctx, cluster)
		if err != nil {
			return nil, err
		}

		if !reachable {
			return &reconcile.Result{RequeueAfter: reachableCheckPeriod}, nil
		}

		// Only add the node deletion finalizer when the cluster is actually running
		// Otherwise we fail to delete the nodes and are stuck in a loop
		if !kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.NodeDeletionFinalizer) {
			finalizers = append(finalizers, kubermaticapiv1.NodeDeletionFinalizer)
		}

	}

	if !kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.KubermaticConstraintCleanupFinalizer) {
		finalizers = append(finalizers, kubermaticapiv1.KubermaticConstraintCleanupFinalizer)
	}

	if len(finalizers) > 0 {
		return r.AddFinalizers(ctx, cluster, finalizers...)
	}

	return &reconcile.Result{}, nil
}

// ensureEtcdLauncherFeatureFlag will apply seed controller etcdLauncher setting on the cluster level
func (r *Reconciler) ensureEtcdLauncherFeatureFlag(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	return r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
		if r.features.EtcdLauncher { // enabled at the controller level
			// we only modify the cluster feature flag if it's not explicitly set, regardless of the value
			if _, set := c.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher]; !set {
				if c.Spec.Features == nil {
					c.Spec.Features = make(map[string]bool)
				}
				c.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher] = true
			}
		}
	})
}
