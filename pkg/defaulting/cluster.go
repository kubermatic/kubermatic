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

	"github.com/imdario/mergo"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/api/v3/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v3/pkg/cni"
	"k8c.io/kubermatic/v3/pkg/provider"
	"k8c.io/kubermatic/v3/pkg/resources"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultEtcdClusterSize = 3
	MinEtcdClusterSize     = 3
	MaxEtcdClusterSize     = 9

	// DefaultNodeAccessNetwork is the default CIDR used for the VPNs
	// transit network through which we route the ControlPlane -> Node/Pod traffic.
	DefaultNodeAccessNetwork = "10.254.0.0/16"
)

var (
	// DefaultClusterComponentSettings are default values that are applied to new clusters
	// upon creation. This is the last source of default values, other mechanisms like the
	// default cluster templates are used earlier.
	DefaultClusterComponentSettings = kubermaticv1.ComponentSettings{
		Apiserver: kubermaticv1.APIServerSettings{
			NodePortRange: resources.DefaultNodePortRange,
			DeploymentSettings: kubermaticv1.DeploymentSettings{
				Replicas: pointer.Int32(DefaultKubernetesApiserverReplicas),
			},
		},
		ControllerManager: kubermaticv1.ControllerSettings{
			DeploymentSettings: kubermaticv1.DeploymentSettings{
				Replicas: pointer.Int32(DefaultKubernetesControllerManagerReplicas),
			},
		},
		Scheduler: kubermaticv1.ControllerSettings{
			DeploymentSettings: kubermaticv1.DeploymentSettings{
				Replicas: pointer.Int32(DefaultKubernetesSchedulerReplicas),
			},
		},
		Etcd: kubermaticv1.EtcdStatefulSetSettings{
			ClusterSize: pointer.Int32(DefaultEtcdClusterSize),
			DiskSize:    mustParseQuantity(DefaultEtcdVolumeSize),
		},
	}
)

func mustParseQuantity(q string) *resource.Quantity {
	parsed := resource.MustParse(q)
	return &parsed
}

// DefaultClusterSpec defaults the cluster spec when creating a new cluster.
// Defaults are taken from, in order:
//  1. ClusterTemplate (if given)
//  3. KubermaticConfiguration's spec.userCluster
//  4. Constants in pkg/controller/operator/defaults
//
// This function assumes that the KubermaticConfiguration has already been defaulted
// (as the KubermaticConfigurationGetter does that automatically).
func DefaultClusterSpec(ctx context.Context, spec *kubermaticv1.ClusterSpec, template *kubermaticv1.ClusterTemplate, config *kubermaticv1.KubermaticConfiguration, datacenterGetter provider.DatacenterGetter, cloudProvider provider.CloudProvider) error {
	var err error

	// If a ClusterTemplate was configured for the Seed, the caller
	// retrieved it for us already and we can use it as the primary
	// source for defaults.
	if template != nil {
		if err := mergo.Merge(spec, template.Spec); err != nil {
			return fmt.Errorf("failed to apply defaulting template to Cluster spec: %w", err)
		}
	}

	// Checking and applying each field of the ComponentSettings is tedious,
	// so we re-use mergo as well. The default template can be unconfigured (i.e. nil)
	// and we still want sensible defaults persisted in the Cluster object,
	// so we resort to hardcoded defaults here.
	if err := mergo.Merge(&spec.ComponentsOverride, DefaultClusterComponentSettings); err != nil {
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
		spec.ExposeStrategy = config.Spec.ExposeStrategy
	}

	// Though the caller probably had already determined the datacenter
	// to construct the cloud provider instance, we do not take the DC
	// as a parameter, to keep this function's signature at least somewhat
	// short. But to enforce certain settings, we still need to have the DC.
	datacenter, fieldErr := DatacenterForClusterSpec(ctx, spec, datacenterGetter)
	if fieldErr != nil {
		return fieldErr
	}

	// Enforce audit logging
	if datacenter.Spec.EnforceAuditLogging {
		spec.AuditLogging = &kubermaticv1.AuditLoggingSettings{
			Enabled: true,
		}
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

	// OSM is enabled by default.
	if spec.EnableOperatingSystemManager == nil {
		spec.EnableOperatingSystemManager = pointer.Bool(true)
	}

	// Add default CNI plugin settings if not present.
	if spec.CNIPlugin == nil {
		spec.CNIPlugin = &kubermaticv1.CNIPluginSettings{
			Type:    kubermaticv1.CNIPluginTypeCanal,
			Version: cni.GetDefaultCNIPluginVersion(kubermaticv1.CNIPluginTypeCanal),
		}
	} else if spec.CNIPlugin.Version == "" {
		spec.CNIPlugin.Version = cni.GetDefaultCNIPluginVersion(spec.CNIPlugin.Type)
	}

	// default cluster networking parameters
	spec.ClusterNetwork = DefaultClusterNetwork(spec.ClusterNetwork, spec.Cloud.ProviderName, spec.ExposeStrategy)

	return nil
}

// GetDefaultingClusterTemplate returns the ClusterTemplate that is referenced by the Seed.
// Note that this can return nil if no template is configured yet (this is not considered
// an error).
func GetDefaultingClusterTemplate(ctx context.Context, client ctrlruntimeclient.Reader, config *kubermaticv1.KubermaticConfiguration) (*kubermaticv1.ClusterTemplate, error) {
	if config.Spec.UserCluster.DefaultTemplate == "" {
		return nil, nil
	}

	tpl := &kubermaticv1.ClusterTemplate{}
	key := types.NamespacedName{Name: config.Spec.UserCluster.DefaultTemplate}
	if err := client.Get(ctx, key, tpl); err != nil {
		return nil, fmt.Errorf("failed to get ClusterTemplate: %w", err)
	}

	return tpl, nil
}

func DatacenterForClusterSpec(ctx context.Context, spec *kubermaticv1.ClusterSpec, datacenterGetter provider.DatacenterGetter) (*kubermaticv1.Datacenter, *field.Error) {
	datacenterName := spec.Cloud.DatacenterName
	if datacenterName == "" {
		return nil, field.Required(field.NewPath("spec", "cloud", "dc"), "no datacenter name specified")
	}

	datacenter, err := datacenterGetter(ctx, datacenterName)
	if err != nil {
		return nil, field.Invalid(field.NewPath("spec", "cloud", "dc"), datacenterName, "invalid datacenter name")
	}

	return datacenter, nil
}

func DefaultClusterNetwork(specClusterNetwork kubermaticv1.ClusterNetworkingConfig, provider kubermaticv1.CloudProvider, exposeStrategy kubermaticv1.ExposeStrategy) kubermaticv1.ClusterNetworkingConfig {
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
		specClusterNetwork.NodeCIDRMaskSizeIPv4 = pointer.Int32(resources.DefaultNodeCIDRMaskSizeIPv4)
	}
	if specClusterNetwork.NodeCIDRMaskSizeIPv6 == nil && specClusterNetwork.Pods.HasIPv6CIDR() {
		specClusterNetwork.NodeCIDRMaskSizeIPv6 = pointer.Int32(resources.DefaultNodeCIDRMaskSizeIPv6)
	}

	if specClusterNetwork.ProxyMode == "" {
		specClusterNetwork.ProxyMode = resources.GetDefaultProxyMode(provider)
	}

	if specClusterNetwork.ProxyMode == resources.IPVSProxyMode {
		if specClusterNetwork.IPVS == nil {
			specClusterNetwork.IPVS = &kubermaticv1.IPVSConfiguration{}
		}
		if specClusterNetwork.IPVS.StrictArp == nil {
			specClusterNetwork.IPVS.StrictArp = pointer.Bool(true)
		}
	}

	if specClusterNetwork.NodeLocalDNSCacheEnabled == nil {
		specClusterNetwork.NodeLocalDNSCacheEnabled = pointer.Bool(resources.DefaultNodeLocalDNSCacheEnabled)
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
