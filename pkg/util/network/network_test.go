/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package network

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
)

func TestClusterIPFamily(t *testing.T) {
	tests := []struct {
		name                string
		cluster             *kubermaticv1.Cluster
		wantIPv4OnlyResult  bool
		wantIPv6OnlyResult  bool
		wantDualStackResult bool
	}{
		{
			name: "ipv4-only",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{
							CIDRBlocks: []string{"172.25.0.0/16"},
						},
					},
				},
			},
			wantIPv4OnlyResult:  true,
			wantIPv6OnlyResult:  false,
			wantDualStackResult: false,
		},
		{
			name: "ipv6-only",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{
							CIDRBlocks: []string{"fd00::/104"},
						},
					},
				},
			},
			wantIPv4OnlyResult:  false,
			wantIPv6OnlyResult:  true,
			wantDualStackResult: false,
		},
		{
			name: "dual-stack",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{
							CIDRBlocks: []string{"172.25.0.0/16", "fd00::/104"},
						},
					},
				},
			},
			wantIPv4OnlyResult:  false,
			wantIPv6OnlyResult:  false,
			wantDualStackResult: true,
		},
		{
			name:                "invalid-empty-network-config",
			cluster:             &kubermaticv1.Cluster{},
			wantIPv4OnlyResult:  false,
			wantIPv6OnlyResult:  false,
			wantDualStackResult: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := IsIPv4OnlyCluster(tt.cluster); res != tt.wantIPv4OnlyResult {
				t.Errorf("IsIPv4OnlyCluster result = %v, wantResult %v", res, tt.wantIPv4OnlyResult)
			}
			if res := IsIPv6OnlyCluster(tt.cluster); res != tt.wantIPv6OnlyResult {
				t.Errorf("IsIPv6OnlyCluster result = %v, wantResult %v", res, tt.wantIPv6OnlyResult)
			}
			if res := IsDualStackCluster(tt.cluster); res != tt.wantDualStackResult {
				t.Errorf("IsDualStackCluster result = %v, wantResult %v", res, tt.wantDualStackResult)
			}
		})
	}
}
