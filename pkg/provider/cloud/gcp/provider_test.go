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

package gcp

import (
	"testing"

	"google.golang.org/api/compute/v1"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

func TestIsClusterRoute(t *testing.T) {
	testCluster := &kubermaticv1.Cluster{
		Spec: kubermaticv1.ClusterSpec{
			ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				Pods: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{
						"172.25.0.0/16",
					},
				},
			},
		},
	}
	tests := []struct {
		name           string
		cluster        *kubermaticv1.Cluster
		route          *compute.Route
		isClusterRoute bool
	}{
		{
			name:    "route is a cluster route",
			cluster: testCluster,
			route: &compute.Route{
				DestRange: "172.25.0.0/24",
			},
			isClusterRoute: true,
		},
		{
			name:    "route is not a cluster route",
			cluster: testCluster,
			route: &compute.Route{
				DestRange: "172.26.0.0/24",
			},
			isClusterRoute: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if clusterRoute, err := isClusterRoute(test.cluster, test.route); err != nil || clusterRoute != test.isClusterRoute {
				t.Fatalf("failed to check if route belongs to the cluster. got: %v, want: %v, err: %v", clusterRoute, test.isClusterRoute, err)
			}
		})
	}
}
