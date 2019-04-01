package cluster

import (
	"context"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"k8s.io/client-go/kubernetes"
)

func (r *Reconciler) reconcileUserClusterResources(ctx context.Context, cluster *kubermaticv1.Cluster, client kubernetes.Interface) error {
	return r.launchingCreateOpenVPNClientCertificates(ctx, cluster)
}
