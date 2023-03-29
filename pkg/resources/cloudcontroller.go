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

package resources

import (
	"net/url"

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	"k8c.io/api/v2/pkg/semver"
	"k8c.io/kubermatic/v3/pkg/version"
)

// ExternalCloudControllerFeatureSupported checks if the cloud provider supports
// external CCM. The clusterVersion has to be specified, depending on whether you
// want to verify against the spec'ed (desired) version or the current version
// in the ClusterStatus.
func ExternalCloudControllerFeatureSupported(dc *kubermaticv1.Datacenter, cloudSpec *kubermaticv1.CloudSpec, clusterVersion semver.Semver, incompatibilities ...*version.ProviderIncompatibility) bool {
	switch cloudSpec.ProviderName {
	case kubermaticv1.CloudProviderOpenStack:
		// When using OpenStack external CCM with Open Telekom Cloud the creation
		// of LBs fail as documented in the issue below:
		// https://github.com/kubernetes/cloud-provider-openstack/issues/960
		// Falling back to the in-tree CloudProvider mitigates the problem, even if
		// not all features are expected to work properly (e.g.
		// `manage-security-groups` should be set to false in cloud config).
		//
		// TODO This is a dirty hack to temporarily support OTC using
		// OpenStack provider, remove this when dedicated OTC support is
		// introduced in Kubermatic.
		return !isOTC(dc.Spec.OpenStack)

	case kubermaticv1.CloudProviderHetzner:
		if cloudSpec.Hetzner.Network == "" && dc.Spec.Hetzner.Network == "" {
			return false
		}

		fallthrough

	case kubermaticv1.CloudProviderAWS,
		kubermaticv1.CloudProviderAnexia,
		kubermaticv1.CloudProviderAzure,
		kubermaticv1.CloudProviderDigitalocean,
		kubermaticv1.CloudProviderKubeVirt,
		kubermaticv1.CloudProviderVSphere:
		supported, err := version.IsSupported(clusterVersion.Semver(), cloudSpec.ProviderName, incompatibilities, kubermaticv1.ConditionExternalCloudProvider)
		if err != nil {
			return false
		}
		return supported

	default:
		return false
	}
}

// MigrationToExternalCloudControllerSupported checks if the cloud provider supports the migration to the
// external CCM.
func MigrationToExternalCloudControllerSupported(dc *kubermaticv1.Datacenter, cluster *kubermaticv1.Cluster, incompatibilities ...*version.ProviderIncompatibility) bool {
	// External CCM must only be enabled when the cluster has reached a given
	// version, so this check must depend on the status if possible.
	v := cluster.Status.Versions.ControlPlane
	if v == "" {
		v = cluster.Spec.Version
	}

	switch t := cluster.Spec.Cloud.ProviderName; t {
	case kubermaticv1.CloudProviderOpenStack:
		// When using OpenStack external CCM with Open Telekom Cloud the creation
		// of LBs fail as documented in the issue below:
		// https://github.com/kubernetes/cloud-provider-openstack/issues/960
		// Falling back to the in-tree CloudProvider mitigates the problem, even if
		// not all features are expected to work properly (e.g.
		// `manage-security-groups` should be set to false in cloud config).
		//
		// TODO This is a dirty hack to temporarily support OTC using
		// OpenStack provider, remove this when dedicated OTC support is
		// introduced in Kubermatic.
		return !isOTC(dc.Spec.OpenStack)

	case kubermaticv1.CloudProviderAWS,
		kubermaticv1.CloudProviderVSphere,
		kubermaticv1.CloudProviderAzure:
		supported, err := version.IsSupported(v.Semver(), t, incompatibilities, kubermaticv1.ConditionExternalCloudProvider)
		if err != nil {
			return false
		}
		return supported

	default:
		return false
	}
}

// ExternalCloudControllerClusterName checks if the ClusterFeatureCCMClusterName is supported
// for the cloud provider.
func ExternalCloudControllerClusterName(cloudSpec *kubermaticv1.CloudSpec) bool {
	switch cloudSpec.ProviderName {
	case kubermaticv1.CloudProviderOpenStack, kubermaticv1.CloudProviderAzure, kubermaticv1.CloudProviderAWS, kubermaticv1.CloudProviderKubeVirt:
		return true
	default:
		return false
	}
}

// isOTC returns `true` if the OpenStack Datacenter uses OTC (i.e.
// Open Telekom Cloud), `false` otherwise.
func isOTC(dc *kubermaticv1.DatacenterSpecOpenStack) bool {
	u, err := url.Parse(dc.AuthURL)
	if err != nil {
		return false
	}
	return u.Host == "iam.eu-de.otc.t-systems.com"
}
