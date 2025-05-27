/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
)

const (
	// clusterRoleBindingsCleanupFinalizer indicates that the cluster ClusterRoleBindings on the seed cluster need cleanup.
	// This finalizer is deprecated and should not be used anymore since we migrated to using owner references for cleanup.
	clusterRoleBindingsCleanupFinalizer = "kubermatic.k8c.io/cleanup-cluster-role-bindings"
)

// cleanupClusterRoleBindings is deprecated and should be removed in KKP 2.20+, because
// nowadays we use owner references for cleanup and this manual step is not needed anymore.
func (d *Deletion) cleanupClusterRoleBindings(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	return kuberneteshelper.TryRemoveFinalizer(ctx, d.seedClient, cluster, clusterRoleBindingsCleanupFinalizer)
}
