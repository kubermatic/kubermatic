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

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
)

// clusterIsReachable checks if the cluster is reachable via its external name
func (r *Reconciler) clusterIsReachable(ctx context.Context, c *kubermaticv1.Cluster) (bool, error) {
	client, err := r.userClusterConnProvider.GetClient(c)
	if err != nil {
		return false, err
	}

	if err := client.List(ctx, &corev1.NamespaceList{}); err != nil {
		r.log.Debugw("Cluster not yet reachable", "cluster", c.Name, zap.Error(err))
		return false, nil
	}

	return true, nil
}
