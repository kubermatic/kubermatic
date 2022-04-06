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
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/defaults"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version/cni"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// DefaultClusterSpec defaults the cluster spec when creating a new cluster.
// Defaults are taken from, in order:
//  1. ClusterTemplate (if given)
//  2. Seed's spec.componentsOverrides
//  3. KubermaticConfiguration's spec.userCluster
//  4. Constants in pkg/controller/operator/defaults
// This function assumes that the KubermaticConfiguration has already been defaulted
// (as the KubermaticConfigurationGetter does that automatically), but the Seed
// does not yet need to be defaulted (to the values of the KubermaticConfiguration).
func DefaultClusterSpec(ctx context.Context, spec *kubermaticv1.ClusterSpec, template *kubermaticv1.ClusterTemplate, seed *kubermaticv1.Seed, config *kubermaticv1.KubermaticConfiguration, cloudProvider provider.CloudProvider) error {
	var err error

	// Apply default values to the Seed, just in case.
	if config != nil {
		seed, err = defaults.DefaultSeed(seed, config, zap.NewNop().Sugar())
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
	// so we re-use mergo as well. Even though DefaultComponentSettings is
	// deprecated, we cannot remove its handling here, as the template can
	// be unconfigured (i.e. nil).
	if err := mergo.Merge(&spec.ComponentsOverride, seed.Spec.DefaultComponentSettings); err != nil {
		return fmt.Errorf("failed to apply defaulting template to Cluster spec: %w", err)
	}

	// Give cloud providers a chance to default their spec.
	if cloudProvider != nil {
		if err := cloudProvider.DefaultCloudSpec(ctx, &spec.Cloud); err != nil {
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
	providerName, err := provider.ClusterCloudProviderName(spec.Cloud)
	if err != nil {
		return fmt.Errorf("failed to determine cloud provider: %w", err)
	}

	spec.Cloud.ProviderName = providerName

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
	defaultClusterNetwork(spec)

	// Always enable external CCM
	if spec.Cloud.Anexia != nil || spec.Cloud.Kubevirt != nil {
		if spec.Features == nil {
			spec.Features = map[string]bool{}
		}
		spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] = true
	}

	return nil
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

func defaultClusterNetwork(spec *kubermaticv1.ClusterSpec) {
	if len(spec.ClusterNetwork.Services.CIDRBlocks) == 0 {
		if spec.Cloud.Kubevirt != nil {
			// KubeVirt cluster can be provisioned on top of k8s cluster created by KKP
			// thus we have to avoid network collision
			spec.ClusterNetwork.Services.CIDRBlocks = []string{resources.DefaultClusterServicesCIDRKubeVirt}
		} else {
			spec.ClusterNetwork.Services.CIDRBlocks = []string{resources.DefaultClusterServicesCIDR}
		}
	}
	if len(spec.ClusterNetwork.Pods.CIDRBlocks) == 0 {
		if spec.Cloud.Kubevirt != nil {
			spec.ClusterNetwork.Pods.CIDRBlocks = []string{resources.DefaultClusterPodsCIDRKubeVirt}
		} else {
			spec.ClusterNetwork.Pods.CIDRBlocks = []string{resources.DefaultClusterPodsCIDR}
		}
	}

	if spec.ClusterNetwork.NodeCIDRMaskSizeIPv4 == nil && spec.ClusterNetwork.Pods.HasIPv4CIDR() {
		spec.ClusterNetwork.NodeCIDRMaskSizeIPv4 = pointer.Int32(resources.DefaultNodeCIDRMaskSizeIPv4)
	}
	if spec.ClusterNetwork.NodeCIDRMaskSizeIPv6 == nil && spec.ClusterNetwork.Pods.HasIPv6CIDR() {
		spec.ClusterNetwork.NodeCIDRMaskSizeIPv6 = pointer.Int32(resources.DefaultNodeCIDRMaskSizeIPv6)
	}

	if spec.ClusterNetwork.ProxyMode == "" {
		// IPVS causes issues with Hetzner's LoadBalancers, which should
		// be addressed via https://github.com/kubernetes/enhancements/pull/1392
		if spec.Cloud.Hetzner != nil {
			spec.ClusterNetwork.ProxyMode = resources.IPTablesProxyMode
		} else {
			spec.ClusterNetwork.ProxyMode = resources.IPVSProxyMode
		}
	}

	if spec.ClusterNetwork.IPVS != nil {
		if spec.ClusterNetwork.IPVS.StrictArp == nil {
			spec.ClusterNetwork.IPVS.StrictArp = pointer.BoolPtr(true)
		}
	}

	if spec.ClusterNetwork.NodeLocalDNSCacheEnabled == nil {
		spec.ClusterNetwork.NodeLocalDNSCacheEnabled = pointer.BoolPtr(true)
	}

	if spec.ClusterNetwork.DNSDomain == "" {
		spec.ClusterNetwork.DNSDomain = "cluster.local"
	}
}
