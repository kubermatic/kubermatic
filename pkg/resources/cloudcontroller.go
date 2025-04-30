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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/version"
)

// ExternalCloudControllerFeatureSupported checks if the cloud provider supports
// external CCM. The clusterVersion has to be specified, depending on whether you
// want to verify against the spec'ed (desired) version or the current version
// in the ClusterStatus.
func ExternalCloudControllerFeatureSupported(dc *kubermaticv1.Datacenter, cloudSpec *kubermaticv1.CloudSpec, clusterVersion semver.Semver, incompatibilities ...*version.ProviderIncompatibility) bool {
	switch t := kubermaticv1.ProviderType(cloudSpec.ProviderName); t {
	case kubermaticv1.OpenstackCloudProvider:
		// When using OpenStack external CCM with Open Telekom Cloud the creation
		// of LBs fail as documented in the issue below:
		// https://github.com/kubernetes/cloud-provider-openstack/issues/960
		// Falling back to the in-tree CloudProvider mitigates the problem, even if
		// not all features are expected to work properly (e.g.
		// `manage-security-groups` should be set to false in cloud config).
		//
		// TODO This is a dirty hack to temporarily support OTC using
		// Openstack provider, remove this when dedicated OTC support is
		// introduced in Kubermatic.
		return !isOTC(dc.Spec.Openstack)

	case kubermaticv1.HetznerCloudProvider:
		if cloudSpec.Hetzner.Network == "" && dc.Spec.Hetzner.Network == "" {
			return false
		}

		fallthrough

	case kubermaticv1.AWSCloudProvider,
		kubermaticv1.AnexiaCloudProvider,
		kubermaticv1.AzureCloudProvider,
		kubermaticv1.GCPCloudProvider,
		kubermaticv1.DigitaloceanCloudProvider,
		kubermaticv1.KubevirtCloudProvider,
		kubermaticv1.VSphereCloudProvider:
		supported, err := version.IsSupported(clusterVersion.Semver(), t, incompatibilities, kubermaticv1.ExternalCloudProviderCondition)
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

	switch t := kubermaticv1.ProviderType(cluster.Spec.Cloud.ProviderName); t {
	case kubermaticv1.OpenstackCloudProvider:
		// When using OpenStack external CCM with Open Telekom Cloud the creation
		// of LBs fail as documented in the issue below:
		// https://github.com/kubernetes/cloud-provider-openstack/issues/960
		// Falling back to the in-tree CloudProvider mitigates the problem, even if
		// not all features are expected to work properly (e.g.
		// `manage-security-groups` should be set to false in cloud config).
		//
		// TODO This is a dirty hack to temporarily support OTC using
		// Openstack provider, remove this when dedicated OTC support is
		// introduced in Kubermatic.
		return !isOTC(dc.Spec.Openstack)

	case kubermaticv1.AWSCloudProvider,
		kubermaticv1.VSphereCloudProvider,
		kubermaticv1.AzureCloudProvider,
		kubermaticv1.GCPCloudProvider:
		supported, err := version.IsSupported(v.Semver(), t, incompatibilities, kubermaticv1.ExternalCloudProviderCondition)
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
	switch kubermaticv1.ProviderType(cloudSpec.ProviderName) {
	case kubermaticv1.OpenstackCloudProvider,
		kubermaticv1.AzureCloudProvider,
		kubermaticv1.AWSCloudProvider,
		kubermaticv1.GCPCloudProvider,
		kubermaticv1.KubevirtCloudProvider:
		return true
	default:
		return false
	}
}

// isOTC returns `true` if the OpenStack Datacenter uses OTC (i.e.
// Open Telekom Cloud), `false` otherwise.
func isOTC(dc *kubermaticv1.DatacenterSpecOpenstack) bool {
	u, err := url.Parse(dc.AuthURL)
	if err != nil {
		return false
	}
	return u.Host == "iam.eu-de.otc.t-systems.com"
}
