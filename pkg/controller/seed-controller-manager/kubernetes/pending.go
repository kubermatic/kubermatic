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

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
)

const (
	reachableCheckPeriod = 5 * time.Second
)

func (r *Reconciler) reconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// Create the namespace
	if err := r.ensureNamespaceExists(ctx, cluster); err != nil {
		return nil, err
	}

	// Set default network configuration
	if err := r.ensureClusterNetworkDefaults(ctx, cluster); err != nil {
		return nil, err
	}

	// Apply etcdLauncher flag
	if err := r.ensureEtcdLauncherFeatureFlag(ctx, cluster); err != nil {
		return nil, err
	}

	// Deploy & Update master components for Kubernetes
	if err := r.ensureResourcesAreDeployed(ctx, cluster); err != nil {
		return nil, err
	}

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
			err = r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
				kuberneteshelper.AddFinalizer(c, kubermaticapiv1.NodeDeletionFinalizer)
			})
			if err != nil {
				return nil, err
			}
		}

	}

	return &reconcile.Result{}, nil
}

// ensureClusterNetworkDefaults will apply default cluster network configuration
func (r *Reconciler) ensureClusterNetworkDefaults(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	var modifiers []func(*kubermaticv1.Cluster)

	if len(cluster.Spec.ClusterNetwork.Services.CIDRBlocks) == 0 {
		setServiceNetwork := func(c *kubermaticv1.Cluster) {
			c.Spec.ClusterNetwork.Services.CIDRBlocks = []string{"10.240.16.0/20"}
		}
		modifiers = append(modifiers, setServiceNetwork)
	}

	if len(cluster.Spec.ClusterNetwork.Pods.CIDRBlocks) == 0 {
		setPodNetwork := func(c *kubermaticv1.Cluster) {
			c.Spec.ClusterNetwork.Pods.CIDRBlocks = []string{"172.25.0.0/16"}
		}
		modifiers = append(modifiers, setPodNetwork)
	}

	if cluster.Spec.ClusterNetwork.DNSDomain == "" {
		setDNSDomain := func(c *kubermaticv1.Cluster) {
			c.Spec.ClusterNetwork.DNSDomain = "cluster.local"
		}
		modifiers = append(modifiers, setDNSDomain)
	}

	if cluster.Spec.ClusterNetwork.ProxyMode == "" {
		setProxyMode := func(c *kubermaticv1.Cluster) {
			c.Spec.ClusterNetwork.ProxyMode = resources.IPVSProxyMode
		}
		modifiers = append(modifiers, setProxyMode)
	}

	return r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
		for _, modify := range modifiers {
			modify(c)
		}
	})
}

// ensureEtcdLauncherFeatureFlag will apply seed controller etcdLauncher setting on the cluster level
func (r *Reconciler) ensureEtcdLauncherFeatureFlag(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	return r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
		if r.features.EtcdLauncher { // enabled at the controller level
			// we only modify the cluster feature flag if it's not explicitly set, regardless if the value
			_, set := c.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher]
			if !set {
				if c.Spec.Features == nil {
					c.Spec.Features = make(map[string]bool)
				}
				c.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher] = true
			}
		}
	})
}
