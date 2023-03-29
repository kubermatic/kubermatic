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

package aws

import (
	"context"

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/provider"
)

func reconcileRegionAnnotation(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, region string) (*kubermaticv1.Cluster, error) {
	if cluster.Annotations[regionAnnotationKey] == region {
		return cluster, nil
	}

	return update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		if cluster.Annotations == nil {
			cluster.Annotations = map[string]string{}
		}
		cluster.Annotations[regionAnnotationKey] = region
	})
}
