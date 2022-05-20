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

package networkdefaults

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8s.io/utils/pointer"
)

func TestOverrideNetworkDefaultsByDefaultingTemplate(t *testing.T) {
	testCases := []struct {
		name                         string
		provider                     kubermaticv1.ProviderType
		templateClusterNetwork       kubermaticv1.ClusterNetworkingConfig
		expectedFinalNetworkDefaults apiv2.NetworkDefaults
	}{
		{
			name:                   "empty cluster network config from template",
			provider:               kubermaticv1.AWSCloudProvider,
			templateClusterNetwork: kubermaticv1.ClusterNetworkingConfig{},
			expectedFinalNetworkDefaults: apiv2.NetworkDefaults{
				IPv4: &apiv2.NetworkDefaultsIPFamily{
					PodsCIDR:                "172.25.0.0/16",
					ServicesCIDR:            "10.240.16.0/20",
					NodeCIDRMaskSize:        24,
					NodePortsAllowedIPRange: "0.0.0.0/0",
				},
				IPv6: &apiv2.NetworkDefaultsIPFamily{
					PodsCIDR:                "fd01::/48",
					ServicesCIDR:            "fd02::/120",
					NodeCIDRMaskSize:        64,
					NodePortsAllowedIPRange: "::/0",
				},
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: true,
			},
		},
		{
			name:                   "empty cluster network config from template kubevirt",
			provider:               kubermaticv1.KubevirtCloudProvider,
			templateClusterNetwork: kubermaticv1.ClusterNetworkingConfig{},
			expectedFinalNetworkDefaults: apiv2.NetworkDefaults{
				IPv4: &apiv2.NetworkDefaultsIPFamily{
					PodsCIDR:                "172.26.0.0/16",
					ServicesCIDR:            "10.241.0.0/20",
					NodeCIDRMaskSize:        24,
					NodePortsAllowedIPRange: "0.0.0.0/0",
				},
				IPv6: &apiv2.NetworkDefaultsIPFamily{
					PodsCIDR:                "fd01::/48",
					ServicesCIDR:            "fd02::/120",
					NodeCIDRMaskSize:        64,
					NodePortsAllowedIPRange: "::/0",
				},
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: true,
			},
		},
		{
			name:                   "empty cluster network config from template hetzner",
			provider:               kubermaticv1.HetznerCloudProvider,
			templateClusterNetwork: kubermaticv1.ClusterNetworkingConfig{},
			expectedFinalNetworkDefaults: apiv2.NetworkDefaults{
				IPv4: &apiv2.NetworkDefaultsIPFamily{
					PodsCIDR:                "172.25.0.0/16",
					ServicesCIDR:            "10.240.16.0/20",
					NodeCIDRMaskSize:        24,
					NodePortsAllowedIPRange: "0.0.0.0/0",
				},
				IPv6: &apiv2.NetworkDefaultsIPFamily{
					PodsCIDR:                "fd01::/48",
					ServicesCIDR:            "fd02::/120",
					NodeCIDRMaskSize:        64,
					NodePortsAllowedIPRange: "::/0",
				},
				ProxyMode:                "iptables",
				NodeLocalDNSCacheEnabled: true,
			},
		},
		{
			name:     "filled cluster network config from template ipv4",
			provider: kubermaticv1.AWSCloudProvider,
			templateClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				IPFamily: "IPv4",
				Pods: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"173.26.0.0/16"},
				},
				Services: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"11.241.17.0/20"},
				},
				ProxyMode: "ipvs",
				IPVS: &kubermaticv1.IPVSConfiguration{
					StrictArp: pointer.Bool(false),
				},
				NodeCIDRMaskSizeIPv4:     pointer.Int32(32),
				NodeLocalDNSCacheEnabled: pointer.Bool(false),
				DNSDomain:                "cluster.local.test",
			},
			expectedFinalNetworkDefaults: apiv2.NetworkDefaults{
				IPv4: &apiv2.NetworkDefaultsIPFamily{
					PodsCIDR:                "173.26.0.0/16",
					ServicesCIDR:            "11.241.17.0/20",
					NodeCIDRMaskSize:        32,
					NodePortsAllowedIPRange: "0.0.0.0/0",
				},
				IPv6: &apiv2.NetworkDefaultsIPFamily{
					PodsCIDR:                "fd01::/48",
					ServicesCIDR:            "fd02::/120",
					NodeCIDRMaskSize:        64,
					NodePortsAllowedIPRange: "::/0",
				},
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: false,
			},
		},
		{
			name:     "filled cluster network config from template dual stack",
			provider: kubermaticv1.KubevirtCloudProvider,
			templateClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				IPFamily: "IPv4+IPv6",
				Pods: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"174.26.0.0/16", "fd02::/48"},
				},
				Services: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"12.241.17.0/20", "fd03::/120"},
				},
				ProxyMode: "proxy-test",
				IPVS: &kubermaticv1.IPVSConfiguration{
					StrictArp: pointer.Bool(false),
				},
				NodeCIDRMaskSizeIPv4:     pointer.Int32(32),
				NodeCIDRMaskSizeIPv6:     pointer.Int32(48),
				NodeLocalDNSCacheEnabled: pointer.Bool(false),
				DNSDomain:                "cluster.local.test",
			},
			expectedFinalNetworkDefaults: apiv2.NetworkDefaults{
				IPv4: &apiv2.NetworkDefaultsIPFamily{
					PodsCIDR:                "174.26.0.0/16",
					ServicesCIDR:            "12.241.17.0/20",
					NodeCIDRMaskSize:        32,
					NodePortsAllowedIPRange: "0.0.0.0/0",
				},
				IPv6: &apiv2.NetworkDefaultsIPFamily{
					PodsCIDR:                "fd02::/48",
					ServicesCIDR:            "fd03::/120",
					NodeCIDRMaskSize:        48,
					NodePortsAllowedIPRange: "::/0",
				},
				ProxyMode:                "proxy-test",
				NodeLocalDNSCacheEnabled: false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			networkDefaults := generateNetworkDefaults(tc.provider)
			networkDefaults = overrideNetworkDefaultsByDefaultingTemplate(networkDefaults, tc.templateClusterNetwork, tc.provider)
			assert.Equal(t, tc.expectedFinalNetworkDefaults, networkDefaults)
		})
	}
}
