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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/address"
)

// syncAddress will set the all address relevant fields on the cluster.
func (r *Reconciler) syncAddress(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seed *kubermaticv1.Seed) error {
	var err error
	// TODO(mrIncompetent): The token should be moved out of Address. But maybe we rather implement another auth-handling? Like openid-connect?
	if cluster.Status.Address.AdminToken == "" {
		// Generate token according to https://kubernetes.io/docs/admin/bootstrap-tokens/#token-format
		err = util.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
			c.Status.Address.AdminToken = kubernetes.GenerateToken()
		})
		if err != nil {
			return err
		}
		log.Infow("Created admin token for cluster")
	}

	b := address.NewModifiersBuilder(log)
	modifiers, err := b.
		Cluster(cluster).
		Client(r).
		ExternalURL(r.externalURL).
		Seed(seed).
		Build(ctx)
	if err != nil {
		return err
	}

	if len(modifiers) > 0 {
		err = util.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
			for _, modifier := range modifiers {
				modifier(c)
			}
		})
	}

	return err
}
