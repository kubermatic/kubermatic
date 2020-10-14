package kubernetes

import (
	"context"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/resources/address"
)

// syncAddress will set the all address relevant fields on the cluster
func (r *Reconciler) syncAddress(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seed *kubermaticv1.Seed) error {
	var err error
	//TODO(mrIncompetent): The token should be moved out of Address. But maybe we rather implement another auth-handling? Like openid-connect?
	if cluster.Address.AdminToken == "" {
		// Generate token according to https://kubernetes.io/docs/admin/bootstrap-tokens/#token-format
		err = r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			c.Address.AdminToken = kubernetes.GenerateToken()
		})
		if err != nil {
			return err
		}
		r.log.Infow("Created admin token for cluster", "cluster", cluster.Name)
	}

	modifiers, err := address.NewModifiersBuilder(log).
		Cluster(cluster).
		Client(r.Client).
		ExternalURL(r.externalURL).
		Seed(seed).
		Build(ctx)
	if err != nil {
		return err
	}
	if len(modifiers) > 0 {
		if err := r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			for _, modifier := range modifiers {
				modifier(c)
			}
		}); err != nil {
			return err
		}
	}

	return nil
}
