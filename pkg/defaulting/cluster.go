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

package defaulting

import (
	"context"
	"fmt"

	"dario.cat/mergo"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/cni"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// DefaultClusterSpec defaults the cluster spec when creating a new cluster.
// Defaults are taken from, in order:
//  1. ClusterTemplate (if given)
//  2. Seed's spec.componentsOverrides
//  3. KubermaticConfiguration's spec.userCluster
//  4. Constants in pkg/controller/operator/defaults
//
// This function assumes that the KubermaticConfiguration has already been defaulted
// (as the KubermaticConfigurationGetter does that automatically), but the Seed
// does not yet need to be defaulted (to the values of the KubermaticConfiguration).
func DefaultClusterSpec(ctx context.Context, spec *kubermaticv1.ClusterSpec, template *kubermaticv1.ClusterTemplate, seed *kubermaticv1.Seed, config *kubermaticv1.KubermaticConfiguration, cloudProvider provider.CloudProvider) error {
	var err error

	// Apply default values to the Seed, just in case.
	if config != nil {
		seed, err = DefaultSeed(seed, config, zap.NewNop().Sugar())
		if err != nil {
			return fmt.Errorf("failed to apply default values to Seed: %w", err)
		}
	}

	// If a ClusterTemplate was configured for the Seed, the caller
	// retrieved it for us already and we can use it as the primary
	// source for defaults.
	if template != nil {
		if err := mergo.Merge(spec, template.Spec); err != nil {
			return fmt.Errorf("failed to apply defaulting template to Cluster spec: %w", err)
		}
	}

	// Checking and applying each field of the ComponentSettings is tedious,
	// so we reuse mergo as well. Even though DefaultComponentSettings is
	// deprecated, we cannot remove its handling here, as the template can
	// be unconfigured (i.e. nil).
	if err := mergo.Merge(&spec.ComponentsOverride, seed.Spec.DefaultComponentSettings); err != nil {
		return fmt.Errorf("failed to apply defaulting template to Cluster spec: %w", err)
	}

	// Give cloud providers a chance to default their spec.
	if cloudProvider != nil {
		if err := cloudProvider.DefaultCloudSpec(ctx, spec); err != nil {
			return fmt.Errorf("failed to default cloud spec: %w", err)
		}
	}

	// set expose strategy
	if spec.ExposeStrategy == "" {
		spec.ExposeStrategy = seed.Spec.ExposeStrategy
	}

	// Though the caller probably had already determined the datacenter
	// to construct the cloud provider instance, we do not take the DC
	// as a parameter, to keep this function's signature at least somewhat
	// short. But to enforce certain settings, we still need to have the DC.
	datacenter, fieldErr := DatacenterForClusterSpec(spec, seed)
	if fieldErr != nil {
		return fieldErr
	}

	// Set the audit logging settings
	if seed.Spec.AuditLogging != nil {
		spec.AuditLogging = new(kubermaticv1.AuditLoggingSettings)
		(*seed.Spec.AuditLogging).DeepCopyInto(spec.AuditLogging)
	}

	// Enforce audit logging
	if datacenter.Spec.EnforceAuditLogging {
		if spec.AuditLogging == nil {
			spec.AuditLogging = &kubermaticv1.AuditLoggingSettings{}
		}
		spec.AuditLogging.Enabled = true
	}

	// Enforce audit webhook backend
	if datacenter.Spec.EnforcedAuditWebhookSettings != nil {
		spec.AuditLogging.WebhookBackend = datacenter.Spec.EnforcedAuditWebhookSettings
	}

	// Enforce PodSecurityPolicy
	if datacenter.Spec.EnforcePodSecurityPolicy {
		spec.UsePodSecurityPolicyAdmissionPlugin = true
	}

	// Ensure provider name matches the given spec
	providerName, err := kubermaticv1helper.ClusterCloudProviderName(spec.Cloud)
	if err != nil {
		return fmt.Errorf("failed to determine cloud provider: %w", err)
	}

	spec.Cloud.ProviderName = providerName

	// Kubernetes dashboard is enabled by default.
	if spec.KubernetesDashboard == nil {
		spec.KubernetesDashboard = &kubermaticv1.KubernetesDashboard{
			Enabled: true,
		}
	}

	// Add default CNI plugin settings if not present.
	if spec.CNIPlugin == nil {
		if spec.Cloud.Edge != nil {
			spec.CNIPlugin = &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCanal,
				Version: cni.GetDefaultCNIPluginVersion(kubermaticv1.CNIPluginTypeCanal),
			}
		} else {
			spec.CNIPlugin = &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCilium,
				Version: cni.GetDefaultCNIPluginVersion(kubermaticv1.CNIPluginTypeCilium),
			}
		}
	} else if spec.CNIPlugin.Version == "" {
		spec.CNIPlugin.Version = cni.GetDefaultCNIPluginVersion(spec.CNIPlugin.Type)
	}

	// default cluster networking parameters
	spec.ClusterNetwork = DefaultClusterNetwork(spec.ClusterNetwork, kubermaticv1.ProviderType(spec.Cloud.ProviderName), spec.ExposeStrategy)
	defaultKubeLBSettings(datacenter, seed, spec)

	return nil
}

func defaultKubeLBSettings(datacenter *kubermaticv1.Datacenter, seed *kubermaticv1.Seed, spec *kubermaticv1.ClusterSpec) {
	var enableGatewayAPI *bool
	// If KubeLB is enforced, enable it.
	if datacenter.Spec.KubeLB != nil && datacenter.Spec.KubeLB.Enforced {
		if spec.KubeLB == nil {
			spec.KubeLB = &kubermaticv1.KubeLB{
				Enabled: true,
			}
		} else {
			spec.KubeLB.Enabled = true
		}
	}

	if spec.KubeLB != nil {
		if spec.KubeLB.EnableGatewayAPI == nil {
			if seed.Spec.KubeLB != nil && seed.Spec.KubeLB.EnableGatewayAPI != nil {
				enableGatewayAPI = seed.Spec.KubeLB.EnableGatewayAPI
			}

			if datacenter.Spec.KubeLB != nil && datacenter.Spec.KubeLB.EnableGatewayAPI != nil {
				enableGatewayAPI = datacenter.Spec.KubeLB.EnableGatewayAPI
			}

			if enableGatewayAPI != nil {
				spec.KubeLB.EnableGatewayAPI = enableGatewayAPI
			}
		}

		if datacenter.Spec.KubeLB != nil {
			if datacenter.Spec.KubeLB.UseLoadBalancerClass && spec.KubeLB.UseLoadBalancerClass == nil {
				spec.KubeLB.UseLoadBalancerClass = ptr.To(true)
			}
		}
	}
}

// GetDefaultingClusterTemplate returns the ClusterTemplate that is referenced by the Seed.
// Note that this can return nil if no template is configured yet (this is not considered
// an error).
func GetDefaultingClusterTemplate(ctx context.Context, client ctrlruntimeclient.Reader, seed *kubermaticv1.Seed) (*kubermaticv1.ClusterTemplate, error) {
	if seed.Spec.DefaultClusterTemplate == "" {
		return nil, nil
	}

	tpl := kubermaticv1.ClusterTemplate{}
	key := types.NamespacedName{Namespace: seed.Namespace, Name: seed.Spec.DefaultClusterTemplate}
	if err := client.Get(ctx, key, &tpl); err != nil {
		return nil, fmt.Errorf("failed to get ClusterTemplate: %w", err)
	}

	if scope := tpl.Labels["scope"]; scope != kubermaticv1.SeedTemplateScope {
		return nil, fmt.Errorf("invalid scope of default cluster template, is %q but must be %q", scope, kubermaticv1.SeedTemplateScope)
	}

	return &tpl, nil
}

func DatacenterForClusterSpec(spec *kubermaticv1.ClusterSpec, seed *kubermaticv1.Seed) (*kubermaticv1.Datacenter, *field.Error) {
	datacenterName := spec.Cloud.DatacenterName
	if datacenterName == "" {
		return nil, field.Required(field.NewPath("spec", "cloud", "dc"), "no datacenter name specified")
	}

	for dcName, dc := range seed.Spec.Datacenters {
		if dcName == datacenterName {
			return &dc, nil
		}
	}

	return nil, field.Invalid(field.NewPath("spec", "cloud", "dc"), datacenterName, "invalid datacenter name")
}

func DefaultClusterNetwork(specClusterNetwork kubermaticv1.ClusterNetworkingConfig, provider kubermaticv1.ProviderType, exposeStrategy kubermaticv1.ExposeStrategy) kubermaticv1.ClusterNetworkingConfig {
	if specClusterNetwork.IPFamily == "" {
		if len(specClusterNetwork.Pods.CIDRBlocks) < 2 {
			// single / no pods CIDR means IPv4-only (IPv6-only is not supported yet and not allowed by cluster validation)
			specClusterNetwork.IPFamily = kubermaticv1.IPFamilyIPv4
		} else {
			// more than one pods CIDR means dual-stack (multiple IPv4 CIDRs are not allowed by cluster validation)
			specClusterNetwork.IPFamily = kubermaticv1.IPFamilyDualStack
		}
	}

	if len(specClusterNetwork.Pods.CIDRBlocks) == 0 {
		if specClusterNetwork.IPFamily == kubermaticv1.IPFamilyDualStack {
			specClusterNetwork.Pods.CIDRBlocks = []string{resources.GetDefaultPodCIDRIPv4(provider), resources.DefaultClusterPodsCIDRIPv6}
		} else {
			specClusterNetwork.Pods.CIDRBlocks = []string{resources.GetDefaultPodCIDRIPv4(provider)}
		}
	}
	if len(specClusterNetwork.Services.CIDRBlocks) == 0 {
		if specClusterNetwork.IPFamily == kubermaticv1.IPFamilyDualStack {
			specClusterNetwork.Services.CIDRBlocks = []string{resources.GetDefaultServicesCIDRIPv4(provider), resources.DefaultClusterServicesCIDRIPv6}
		} else {
			specClusterNetwork.Services.CIDRBlocks = []string{resources.GetDefaultServicesCIDRIPv4(provider)}
		}
	}

	if specClusterNetwork.NodeCIDRMaskSizeIPv4 == nil && specClusterNetwork.Pods.HasIPv4CIDR() {
		specClusterNetwork.NodeCIDRMaskSizeIPv4 = ptr.To[int32](resources.DefaultNodeCIDRMaskSizeIPv4)
	}
	if specClusterNetwork.NodeCIDRMaskSizeIPv6 == nil && specClusterNetwork.Pods.HasIPv6CIDR() {
		specClusterNetwork.NodeCIDRMaskSizeIPv6 = ptr.To[int32](resources.DefaultNodeCIDRMaskSizeIPv6)
	}

	if specClusterNetwork.ProxyMode == "" {
		specClusterNetwork.ProxyMode = resources.GetDefaultProxyMode(provider)
	}

	if specClusterNetwork.ProxyMode == resources.IPVSProxyMode {
		if specClusterNetwork.IPVS == nil {
			specClusterNetwork.IPVS = &kubermaticv1.IPVSConfiguration{}
		}
		if specClusterNetwork.IPVS.StrictArp == nil {
			specClusterNetwork.IPVS.StrictArp = ptr.To(true)
		}
	}

	if specClusterNetwork.NodeLocalDNSCacheEnabled == nil {
		specClusterNetwork.NodeLocalDNSCacheEnabled = ptr.To(resources.DefaultNodeLocalDNSCacheEnabled)
	}

	if specClusterNetwork.DNSDomain == "" {
		specClusterNetwork.DNSDomain = "cluster.local"
	}

	if exposeStrategy == kubermaticv1.ExposeStrategyTunneling {
		if specClusterNetwork.TunnelingAgentIP == "" {
			specClusterNetwork.TunnelingAgentIP = resources.DefaultTunnelingAgentIP
		}
	}

	return specClusterNetwork
}
