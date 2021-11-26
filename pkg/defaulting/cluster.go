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
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/utils/pointer"
)

var (
	DefaultCNIPluginVersions = map[kubermaticv1.CNIPluginType]string{
		kubermaticv1.CNIPluginTypeCanal:  "v3.20",
		kubermaticv1.CNIPluginTypeCilium: "v1.10",
	}
)

// DefaultCreateClusterSpec defaults the cluster spec when creating a new cluster
func DefaultCreateClusterSpec(spec *kubermaticv1.ClusterSpec, seed *kubermaticv1.Seed, config *operatorv1alpha1.KubermaticConfiguration, cloudProvider provider.CloudProvider) error {
	if cloudProvider != nil {
		if err := cloudProvider.DefaultCloudSpec(&spec.Cloud); err != nil {
			return fmt.Errorf("failed to default cloud spec: %v", err)
		}
	}

	if spec.ExposeStrategy == "" {
		// master level ExposeStrategy is the default
		exposeStrategy := config.Spec.ExposeStrategy
		if seed.Spec.ExposeStrategy != "" {
			exposeStrategy = seed.Spec.ExposeStrategy
		}

		spec.ExposeStrategy = exposeStrategy
	}

	if spec.ComponentsOverride.Etcd.ClusterSize == nil {
		n := int32(kubermaticv1.DefaultEtcdClusterSize)
		spec.ComponentsOverride.Etcd.ClusterSize = &n
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
