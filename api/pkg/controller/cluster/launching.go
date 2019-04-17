package cluster

import (
	"context"

	"github.com/golang/glog"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// clusterIsReachable checks if the cluster is reachable via its external name
func (r *Reconciler) clusterIsReachable(ctx context.Context, c *kubermaticv1.Cluster) (bool, error) {
	client, err := r.userClusterConnProvider.GetDynamicClient(c)
	if err != nil {
		return false, err
	}

	if err := client.List(ctx, &ctrlruntimeclient.ListOptions{}, &corev1.NamespaceList{}); err != nil {
		glog.V(4).Infof("Cluster %q not yet reachable: %v", c.Name, err)
		return false, nil
	}

	return true, nil
}
