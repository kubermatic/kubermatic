package cluster

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
