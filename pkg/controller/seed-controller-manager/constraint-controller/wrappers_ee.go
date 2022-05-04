//go:build ee

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

package seedconstraintsynchronizer

import (
	"context"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	eeutil "k8c.io/kubermatic/v2/pkg/ee/constraint-controller"
)

func (r *reconciler) filterClustersForConstraint(ctx context.Context, constraint *kubermaticv1.Constraint, clusterList *kubermaticv1.ClusterList) ([]kubermaticv1.Cluster, []kubermaticv1.Cluster, error) {
	return eeutil.FilterClustersForConstraint(ctx, r.seedClient, constraint, clusterList)
}
