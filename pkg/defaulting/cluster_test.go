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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
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

func TestDefaultAuditLogging(t *testing.T) {
	const dcName = "audit-test-dc"

	seedAuditConfig := &kubermaticv1.AuditLoggingSettings{
		Enabled:      true,
		PolicyPreset: kubermaticv1.AuditPolicyRecommended,
	}

	webhookSettings := &kubermaticv1.AuditWebhookBackendSettings{
		AuditWebhookConfig: &corev1.SecretReference{
			Name:      "audit-webhook-secret",
			Namespace: "kube-system",
		},
	}

	makeSeed := func(auditLogging *kubermaticv1.AuditLoggingSettings, enforce bool, webhookBackend *kubermaticv1.AuditWebhookBackendSettings) *kubermaticv1.Seed {
		return &kubermaticv1.Seed{
			Spec: kubermaticv1.SeedSpec{
				AuditLogging: auditLogging,
				Datacenters: map[string]kubermaticv1.Datacenter{
					dcName: {
						Spec: kubermaticv1.DatacenterSpec{
							Fake:                         &kubermaticv1.DatacenterSpecFake{},
							EnforceAuditLogging:          enforce,
							EnforcedAuditWebhookSettings: webhookBackend,
						},
					},
				},
			},
		}
	}

	makeSpec := func(auditLogging *kubermaticv1.AuditLoggingSettings) *kubermaticv1.ClusterSpec {
		return &kubermaticv1.ClusterSpec{
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: dcName,
				ProviderName:   string(kubermaticv1.FakeCloudProvider),
				Fake:           &kubermaticv1.FakeCloudSpec{Token: "test"},
			},
			AuditLogging: auditLogging,
		}
	}

	testCases := []struct {
		name                 string
		spec                 *kubermaticv1.ClusterSpec
		seed                 *kubermaticv1.Seed
		annotations          map[string]string
		expectedAuditLogging *kubermaticv1.AuditLoggingSettings
	}{
		{
			name: "enforcement on with seed config: cluster gets seed config with Enabled=true",
			spec: makeSpec(nil),
			seed: makeSeed(seedAuditConfig, true, nil),
			expectedAuditLogging: &kubermaticv1.AuditLoggingSettings{
				Enabled:      true,
				PolicyPreset: kubermaticv1.AuditPolicyRecommended,
			},
		},
		{
			name: "enforcement on with nil seed config: cluster gets bare Enabled=true",
			spec: makeSpec(nil),
			seed: makeSeed(nil, true, nil),
			expectedAuditLogging: &kubermaticv1.AuditLoggingSettings{
				Enabled: true,
			},
		},
		{
			name:                 "enforcement off with seed config: cluster audit logging untouched",
			spec:                 makeSpec(nil),
			seed:                 makeSeed(seedAuditConfig, false, nil),
			expectedAuditLogging: nil,
		},
		{
			name:                 "enforcement off without seed config: cluster audit logging untouched",
			spec:                 makeSpec(nil),
			seed:                 makeSeed(nil, false, nil),
			expectedAuditLogging: nil,
		},
		{
			name: "enforcement off, cluster has own audit logging: preserved",
			spec: makeSpec(&kubermaticv1.AuditLoggingSettings{
				Enabled:      true,
				PolicyPreset: kubermaticv1.AuditPolicyMinimal,
			}),
			seed: makeSeed(seedAuditConfig, false, nil),
			expectedAuditLogging: &kubermaticv1.AuditLoggingSettings{
				Enabled:      true,
				PolicyPreset: kubermaticv1.AuditPolicyMinimal,
			},
		},
		{
			name: "opt-out annotation set: cluster audit logging untouched despite enforcement",
			spec: makeSpec(nil),
			seed: makeSeed(seedAuditConfig, true, nil),
			annotations: map[string]string{
				kubermaticv1.SkipAuditLoggingEnforcementAnnotation: "true",
			},
			expectedAuditLogging: nil,
		},
		{
			name: "EnforcedAuditWebhookSettings with nil spec.AuditLogging: no nil dereference",
			spec: makeSpec(nil),
			seed: makeSeed(nil, false, webhookSettings),
			expectedAuditLogging: &kubermaticv1.AuditLoggingSettings{
				WebhookBackend: webhookSettings,
			},
		},
		{
			name: "enforcement on with EnforcedAuditWebhookSettings: both applied",
			spec: makeSpec(nil),
			seed: makeSeed(seedAuditConfig, true, webhookSettings),
			expectedAuditLogging: &kubermaticv1.AuditLoggingSettings{
				Enabled:        true,
				PolicyPreset:   kubermaticv1.AuditPolicyRecommended,
				WebhookBackend: webhookSettings,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := DefaultConfiguration(&kubermaticv1.KubermaticConfiguration{}, zap.NewNop().Sugar())
			if err != nil {
				t.Fatalf("DefaultConfiguration returned error: %v", err)
			}
			err = DefaultClusterSpec(context.Background(), tc.spec, tc.annotations, nil, tc.seed, config, nil)
			if err != nil {
				t.Fatalf("DefaultClusterSpec returned error: %v", err)
			}
			assert.Equal(t, tc.expectedAuditLogging, tc.spec.AuditLogging)
		})
	}
}
