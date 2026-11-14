/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package mla

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// LoadInput builds the Input for an existing user cluster: it loads the
// Cluster, its Project and a client for the user cluster.
func LoadInput(ctx context.Context, seedClient ctrlruntimeclient.Client, seedConfig *rest.Config, log *zap.SugaredLogger, clusterName, grafanaAdminEmail string) (Input, error) {
	clusterJig := jig.NewClusterJig(seedClient, log).WithExistingCluster(clusterName)

	cluster, err := clusterJig.Cluster(ctx)
	if err != nil {
		return Input{}, err
	}

	projectName := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
	if projectName == "" {
		return Input{}, fmt.Errorf("cluster %s has no %s label", clusterName, kubermaticv1.ProjectIDLabelKey)
	}

	project := &kubermaticv1.Project{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: projectName}, project); err != nil {
		return Input{}, fmt.Errorf("failed to get project %s: %w", projectName, err)
	}

	userClient, err := clusterJig.ClusterClient(ctx)
	if err != nil {
		return Input{}, fmt.Errorf("failed to get user cluster client: %w", err)
	}

	return Input{
		SeedClient:        seedClient,
		SeedConfig:        seedConfig,
		UserClient:        userClient,
		Cluster:           cluster,
		Project:           project,
		GrafanaAdminEmail: grafanaAdminEmail,
		Log:               log,
	}, nil
}
