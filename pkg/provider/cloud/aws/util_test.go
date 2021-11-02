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

package aws

import (
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func makeCluster(cloudSpec *kubermaticv1.AWSCloudSpec) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		Spec: kubermaticv1.ClusterSpec{
			Cloud: kubermaticv1.CloudSpec{
				AWS: cloudSpec,
			},
		},
	}
}

func testClusterUpdater(cluster *kubermaticv1.Cluster) provider.ClusterUpdater {
	return func(clusterName string, patcher func(*kubermaticv1.Cluster), opts ...provider.UpdaterOption) (*kubermaticv1.Cluster, error) {
		patcher(cluster)
		return cluster, nil
	}
}
