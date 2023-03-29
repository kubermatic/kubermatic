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

package kubernetes

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func HasClusters(ctx context.Context, client ctrlruntimeclient.Client, projectID string) (bool, error) {
	clusterList := &kubermaticv1.ClusterList{}

	if err := client.List(ctx,
		clusterList,
		&ctrlruntimeclient.ListOptions{Limit: 1},
		ctrlruntimeclient.MatchingLabels{kubermaticv1.ProjectIDLabelKey: projectID}); err != nil {
		return false, fmt.Errorf("failed to list clusters: %w", err)
	}

	return len(clusterList.Items) > 0, nil
}

func HasExternalClusters(ctx context.Context, client ctrlruntimeclient.Client, projectID string) (bool, error) {
	extClusterList := &kubermaticv1.ExternalClusterList{}

	if err := client.List(ctx,
		extClusterList,
		&ctrlruntimeclient.ListOptions{Limit: 1},
		ctrlruntimeclient.MatchingLabels{kubermaticv1.ProjectIDLabelKey: projectID}); err != nil {
		return false, fmt.Errorf("failed to list external clusters: %w", err)
	}

	return len(extClusterList.Items) > 0, nil
}
