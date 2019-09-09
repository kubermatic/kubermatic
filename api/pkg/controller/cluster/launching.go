package cluster

import (
	"context"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// clusterIsReachable checks if the cluster is reachable via its external name
func (r *Reconciler) clusterIsReachable(ctx context.Context, c *kubermaticv1.Cluster) (bool, error) {
	client, err := r.userClusterConnProvider.GetClient(c)
	if err != nil {
		return false, err
	}

	if err := client.List(ctx, &ctrlruntimeclient.ListOptions{}, &corev1.NamespaceList{}); err != nil {
		r.log.Infow("Cluster not yet reachable", "cluster", c.Name, "error", err)
		return false, nil
	}

	return true, nil
}
