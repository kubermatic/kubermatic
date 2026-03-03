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

package defaulting

import (
	"testing"

	"github.com/stretchr/testify/assert"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/utils/ptr"
)

func TestDefaultClusterNetwork(t *testing.T) {
	testCases := []struct {
		name                string
		spec                *kubermaticv1.ClusterSpec
		expectedChangedSpec *kubermaticv1.ClusterSpec
	}{
		{
			name: "empty spec ipv4",
			spec: &kubermaticv1.ClusterSpec{},
			expectedChangedSpec: &kubermaticv1.ClusterSpec{
				ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
					IPFamily: "IPv4",
					Pods: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"172.25.0.0/16"},
					},
					Services: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"10.240.16.0/20"},
					},
					ProxyMode: "ipvs",
					IPVS: &kubermaticv1.IPVSConfiguration{
						StrictArp: ptr.To(true),
					},
					NodeCIDRMaskSizeIPv4:     ptr.To[int32](24),
					NodeLocalDNSCacheEnabled: ptr.To(true),
					DNSDomain:                "cluster.local",
				},
			},
		},
		{
			name: "empty spec dual stack",
			spec: &kubermaticv1.ClusterSpec{
				ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
					IPFamily: "IPv4+IPv6",
				},
			},
			expectedChangedSpec: &kubermaticv1.ClusterSpec{
				ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
					IPFamily: "IPv4+IPv6",
					Pods: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"172.25.0.0/16", "fd01::/48"},
					},
					Services: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"10.240.16.0/20", "fd02::/108"},
					},
					ProxyMode: "ipvs",
					IPVS: &kubermaticv1.IPVSConfiguration{
						StrictArp: ptr.To(true),
					},
					NodeCIDRMaskSizeIPv4:     ptr.To[int32](24),
					NodeCIDRMaskSizeIPv6:     ptr.To[int32](64),
					NodeLocalDNSCacheEnabled: ptr.To(true),
					DNSDomain:                "cluster.local",
				},
			},
		},
		{
			name: "empty spec detect dual stack",
			spec: &kubermaticv1.ClusterSpec{
				ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
					Pods: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"174.27.0.0/16", "fd05::/48"},
					},
				},
			},
			expectedChangedSpec: &kubermaticv1.ClusterSpec{
				ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
					IPFamily: "IPv4+IPv6",
					Pods: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"174.27.0.0/16", "fd05::/48"},
					},
					Services: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"10.240.16.0/20", "fd02::/108"},
					},
					ProxyMode: "ipvs",
					IPVS: &kubermaticv1.IPVSConfiguration{
						StrictArp: ptr.To(true),
					},
					NodeCIDRMaskSizeIPv4:     ptr.To[int32](24),
					NodeCIDRMaskSizeIPv6:     ptr.To[int32](64),
					NodeLocalDNSCacheEnabled: ptr.To(true),
					DNSDomain:                "cluster.local",
				},
			},
		},
		{
			name: "empty spec dual stack kubevirt",
			spec: &kubermaticv1.ClusterSpec{
				Cloud: kubermaticv1.CloudSpec{
					ProviderName: "kubevirt",
				},
				ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
					IPFamily: "IPv4+IPv6",
				},
			},
			expectedChangedSpec: &kubermaticv1.ClusterSpec{
				Cloud: kubermaticv1.CloudSpec{
					ProviderName: "kubevirt",
				},
				ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
					IPFamily: "IPv4+IPv6",
					Pods: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"172.26.0.0/16", "fd01::/48"},
					},
					Services: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"10.241.0.0/20", "fd02::/108"},
					},
					ProxyMode: "ipvs",
					IPVS: &kubermaticv1.IPVSConfiguration{
						StrictArp: ptr.To(true),
					},
					NodeCIDRMaskSizeIPv4:     ptr.To[int32](24),
					NodeCIDRMaskSizeIPv6:     ptr.To[int32](64),
					NodeLocalDNSCacheEnabled: ptr.To(true),
					DNSDomain:                "cluster.local",
				},
			},
		},
		{
			name: "prefilled spec ipv4",
			spec: &kubermaticv1.ClusterSpec{
				ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
					IPFamily: "IPv4",
					Pods: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"173.26.0.0/16"},
					},
					Services: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"11.241.17.0/20"},
					},
					ProxyMode: "ipvs",
					IPVS: &kubermaticv1.IPVSConfiguration{
						StrictArp: ptr.To(false),
					},
					NodeCIDRMaskSizeIPv4:     ptr.To[int32](32),
					NodeLocalDNSCacheEnabled: ptr.To(false),
					DNSDomain:                "cluster.local.test",
				},
			},
			expectedChangedSpec: &kubermaticv1.ClusterSpec{
				ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
					IPFamily: "IPv4",
					Pods: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"173.26.0.0/16"},
					},
					Services: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"11.241.17.0/20"},
					},
					ProxyMode: "ipvs",
					IPVS: &kubermaticv1.IPVSConfiguration{
						StrictArp: ptr.To(false),
					},
					NodeCIDRMaskSizeIPv4:     ptr.To[int32](32),
					NodeLocalDNSCacheEnabled: ptr.To(false),
					DNSDomain:                "cluster.local.test",
				},
			},
		},
		{
			name: "prefilled spec dual stack",
			spec: &kubermaticv1.ClusterSpec{
				ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
					IPFamily: "IPv4+IPv6",
					Pods: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"173.26.0.0/16", "fd02::/48"},
					},
					Services: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"11.241.17.0/20", "fd03::/120"},
					},
					ProxyMode: "proxy-test",
					IPVS: &kubermaticv1.IPVSConfiguration{
						StrictArp: ptr.To(false),
					},
					NodeCIDRMaskSizeIPv4:     ptr.To[int32](32),
					NodeCIDRMaskSizeIPv6:     ptr.To[int32](48),
					NodeLocalDNSCacheEnabled: ptr.To(false),
					DNSDomain:                "cluster.local.test",
				},
			},
			expectedChangedSpec: &kubermaticv1.ClusterSpec{
				ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
					IPFamily: "IPv4+IPv6",
					Pods: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"173.26.0.0/16", "fd02::/48"},
					},
					Services: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"11.241.17.0/20", "fd03::/120"},
					},
					ProxyMode: "proxy-test",
					IPVS: &kubermaticv1.IPVSConfiguration{
						StrictArp: ptr.To(false),
					},
					NodeCIDRMaskSizeIPv4:     ptr.To[int32](32),
					NodeCIDRMaskSizeIPv6:     ptr.To[int32](48),
					NodeLocalDNSCacheEnabled: ptr.To(false),
					DNSDomain:                "cluster.local.test",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.spec.ClusterNetwork = DefaultClusterNetwork(tc.spec.ClusterNetwork, kubermaticv1.ProviderType(tc.spec.Cloud.ProviderName), tc.spec.ExposeStrategy)
			assert.Equal(t, tc.expectedChangedSpec, tc.spec)
		})
	}
}

func TestDefaultEventRateLimitPlugin(t *testing.T) {
	testCases := []struct {
		name                                     string
		spec                                     *kubermaticv1.ClusterSpec
		config                                   *kubermaticv1.KubermaticConfiguration
		expectedUseEventRateLimitAdmissionPlugin bool
		expectedAdmissionPlugins                 []string
		expectedConfig                           *kubermaticv1.EventRateLimitConfig
	}{
		{
			name:                                     "nil config - no change",
			spec:                                     &kubermaticv1.ClusterSpec{},
			config:                                   nil,
			expectedUseEventRateLimitAdmissionPlugin: false,
			expectedAdmissionPlugins:                 nil,
			expectedConfig:                           nil,
		},
		{
			name: "config without admission plugins - no change",
			spec: &kubermaticv1.ClusterSpec{},
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{},
				},
			},
			expectedUseEventRateLimitAdmissionPlugin: false,
			expectedAdmissionPlugins:                 nil,
			expectedConfig:                           nil,
		},
		{
			name: "enforced - enables plugin",
			spec: &kubermaticv1.ClusterSpec{},
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
							EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
								Enforced: ptr.To(true),
							},
						},
					},
				},
			},
			expectedUseEventRateLimitAdmissionPlugin: true,
			expectedAdmissionPlugins:                 nil,
			expectedConfig:                           nil,
		},
		{
			name: "enabled - enables plugin",
			spec: &kubermaticv1.ClusterSpec{},
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
							EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
								Enabled: ptr.To(true),
							},
						},
					},
				},
			},
			expectedUseEventRateLimitAdmissionPlugin: true,
			expectedAdmissionPlugins:                 nil,
			expectedConfig:                           nil,
		},
		{
			name: "enabled=false - does not enable plugin",
			spec: &kubermaticv1.ClusterSpec{},
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
							EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
								Enabled: ptr.To(false),
							},
						},
					},
				},
			},
			expectedUseEventRateLimitAdmissionPlugin: false,
			expectedAdmissionPlugins:                 nil,
			expectedConfig:                           nil,
		},
		{
			name: "user already enabled via dedicated field - no change",
			spec: &kubermaticv1.ClusterSpec{
				UseEventRateLimitAdmissionPlugin: true,
			},
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
							EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
								Enabled: ptr.To(false),
							},
						},
					},
				},
			},
			expectedUseEventRateLimitAdmissionPlugin: true,
			expectedAdmissionPlugins:                 nil,
			expectedConfig:                           nil,
		},
		{
			name: "already enabled via admissionPlugins - no change",
			spec: &kubermaticv1.ClusterSpec{
				AdmissionPlugins: []string{resources.EventRateLimitAdmissionPlugin},
			},
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
							EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
								Enabled: ptr.To(true),
							},
						},
					},
				},
			},
			expectedUseEventRateLimitAdmissionPlugin: false,
			expectedAdmissionPlugins:                 []string{resources.EventRateLimitAdmissionPlugin},
			expectedConfig:                           nil,
		},
		{
			name: "enforced with plugin already in admissionPlugins - sets boolean too",
			spec: &kubermaticv1.ClusterSpec{
				AdmissionPlugins: []string{resources.EventRateLimitAdmissionPlugin},
			},
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
							EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
								Enforced: ptr.To(true),
							},
						},
					},
				},
			},
			expectedUseEventRateLimitAdmissionPlugin: true,
			expectedAdmissionPlugins:                 []string{resources.EventRateLimitAdmissionPlugin},
			expectedConfig:                           nil,
		},
		{
			name: "enabled with default config - applies config",
			spec: &kubermaticv1.ClusterSpec{},
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
							EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
								Enabled: ptr.To(true),
								DefaultConfig: &kubermaticv1.EventRateLimitConfig{
									Server: &kubermaticv1.EventRateLimitConfigItem{
										QPS:   50,
										Burst: 100,
									},
								},
							},
						},
					},
				},
			},
			expectedUseEventRateLimitAdmissionPlugin: true,
			expectedAdmissionPlugins:                 nil,
			expectedConfig: &kubermaticv1.EventRateLimitConfig{
				Server: &kubermaticv1.EventRateLimitConfigItem{
					QPS:   50,
					Burst: 100,
				},
			},
		},
		{
			name: "user config not overwritten",
			spec: &kubermaticv1.ClusterSpec{
				UseEventRateLimitAdmissionPlugin: true,
				EventRateLimitConfig: &kubermaticv1.EventRateLimitConfig{
					Namespace: &kubermaticv1.EventRateLimitConfigItem{
						QPS:   25,
						Burst: 50,
					},
				},
			},
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
							EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
								Enabled: ptr.To(true),
								DefaultConfig: &kubermaticv1.EventRateLimitConfig{
									Server: &kubermaticv1.EventRateLimitConfigItem{
										QPS:   50,
										Burst: 100,
									},
								},
							},
						},
					},
				},
			},
			expectedUseEventRateLimitAdmissionPlugin: true,
			expectedAdmissionPlugins:                 nil,
			expectedConfig: &kubermaticv1.EventRateLimitConfig{
				Namespace: &kubermaticv1.EventRateLimitConfigItem{
					QPS:   25,
					Burst: 50,
				},
			},
		},
		{
			name: "enforced with config overwrites user config",
			spec: &kubermaticv1.ClusterSpec{
				UseEventRateLimitAdmissionPlugin: true,
				EventRateLimitConfig: &kubermaticv1.EventRateLimitConfig{
					Namespace: &kubermaticv1.EventRateLimitConfigItem{
						QPS:   25,
						Burst: 50,
					},
				},
			},
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
							EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
								Enforced: ptr.To(true),
								DefaultConfig: &kubermaticv1.EventRateLimitConfig{
									Server: &kubermaticv1.EventRateLimitConfigItem{
										QPS:   50,
										Burst: 100,
									},
								},
							},
						},
					},
				},
			},
			expectedUseEventRateLimitAdmissionPlugin: true,
			expectedAdmissionPlugins:                 nil,
			expectedConfig: &kubermaticv1.EventRateLimitConfig{
				Server: &kubermaticv1.EventRateLimitConfigItem{
					QPS:   50,
					Burst: 100,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defaultEventRateLimitPlugin(tc.spec, tc.config)
			assert.Equal(t, tc.expectedUseEventRateLimitAdmissionPlugin, tc.spec.UseEventRateLimitAdmissionPlugin)
			assert.Equal(t, tc.expectedAdmissionPlugins, tc.spec.AdmissionPlugins)
			assert.Equal(t, tc.expectedConfig, tc.spec.EventRateLimitConfig)
		})
	}
}
