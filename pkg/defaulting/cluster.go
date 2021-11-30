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
	"k8c.io/kubermatic/v2/pkg/controller/operator/defaults"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	DefaultCNIPluginVersions = map[kubermaticv1.CNIPluginType]string{
		kubermaticv1.CNIPluginTypeCanal:  "v3.20",
		kubermaticv1.CNIPluginTypeCilium: "v1.10",
	}
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
func DefaultClusterSpec(spec *kubermaticv1.ClusterSpec, template *kubermaticv1.ClusterTemplate, seed *kubermaticv1.Seed, config *operatorv1alpha1.KubermaticConfiguration, cloudProvider provider.CloudProvider) error {
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
		if err := cloudProvider.DefaultCloudSpec(&spec.Cloud); err != nil {
			return fmt.Errorf("failed to default cloud spec: %w", err)
		}
	}

	// set expose strategy
	if spec.ExposeStrategy == "" {
		spec.ExposeStrategy = seed.Spec.ExposeStrategy
	}

	// set provider name
	if spec.Cloud.ProviderName != "" {
		providerName, err := provider.ClusterCloudProviderName(spec.Cloud)
		if err != nil {
			return fmt.Errorf("failed to determine cloud provider: %w", err)
		}

		spec.Cloud.ProviderName = providerName
	}

	// Add default CNI plugin settings if not present.
	if spec.CNIPlugin == nil {
		spec.CNIPlugin = &kubermaticv1.CNIPluginSettings{
			Type:    kubermaticv1.CNIPluginTypeCanal,
			Version: DefaultCNIPluginVersions[kubermaticv1.CNIPluginTypeCanal],
		}
	} else if spec.CNIPlugin.Version == "" {
		spec.CNIPlugin.Version = DefaultCNIPluginVersions[spec.CNIPlugin.Type]
	}

	if len(spec.ClusterNetwork.Services.CIDRBlocks) == 0 {
		if spec.Cloud.Kubevirt != nil {
			// KubeVirt cluster can be provisioned on top of k8s cluster created by KKP
			// thus we have to avoid network collision
			spec.ClusterNetwork.Services.CIDRBlocks = []string{"10.241.0.0/20"}
		} else {
			spec.ClusterNetwork.Services.CIDRBlocks = []string{"10.240.16.0/20"}
		}
	}

	if len(spec.ClusterNetwork.Pods.CIDRBlocks) == 0 {
		if spec.Cloud.Kubevirt != nil {
			spec.ClusterNetwork.Pods.CIDRBlocks = []string{"172.26.0.0/16"}
		} else {
			spec.ClusterNetwork.Pods.CIDRBlocks = []string{"172.25.0.0/16"}
		}
	}

	if spec.ClusterNetwork.DNSDomain == "" {
		spec.ClusterNetwork.DNSDomain = "cluster.local"
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
			spec.ClusterNetwork.IPVS.StrictArp = pointer.BoolPtr(resources.IPVSStrictArp)
		}
	}

	// Network policies for Apiserver are deployed by default
	if _, ok := spec.Features[kubermaticv1.ApiserverNetworkPolicy]; !ok {
		if spec.Features == nil {
			spec.Features = map[string]bool{}
		}
		spec.Features[kubermaticv1.ApiserverNetworkPolicy] = true
	}

	if spec.ClusterNetwork.NodeLocalDNSCacheEnabled == nil {
		spec.ClusterNetwork.NodeLocalDNSCacheEnabled = pointer.BoolPtr(true)
	}

	// Always enable external CCM
	if spec.Cloud.Anexia != nil || spec.Cloud.Kubevirt != nil {
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
