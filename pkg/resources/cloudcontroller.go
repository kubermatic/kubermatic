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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/version"
)

const (
	v121 = "1.21"
	v122 = "1.22"
	v123 = "1.23"
	v124 = "1.24"

	v1240 = "1.24.0"
)

// ExternalCloudControllerFeatureSupported checks if the cloud provider supports
// external CCM. The clusterVersion has to be specified, depending on whether you
// want to verify against the spec'ed (desired) version or the current version
// in the ClusterStatus.
func ExternalCloudControllerFeatureSupported(dc *kubermaticv1.Datacenter, cloudSpec *kubermaticv1.CloudSpec, clusterVersion semver.Semver, incompatibilities ...*version.ProviderIncompatibility) bool {
	switch {
	case cloudSpec.Openstack != nil:
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

	case cloudSpec.Hetzner != nil:
		if cloudSpec.Hetzner.Network == "" && dc.Spec.Hetzner.Network == "" {
			return false
		}

		fallthrough

	case cloudSpec.AWS != nil:
		fallthrough

	case cloudSpec.Anexia != nil:
		fallthrough

	case cloudSpec.Azure != nil:
		fallthrough

	case cloudSpec.Kubevirt != nil:
		fallthrough

	case cloudSpec.VSphere != nil:
		supported, err := version.IsSupported(clusterVersion.Semver(), kubermaticv1.ProviderType(cloudSpec.ProviderName), incompatibilities, kubermaticv1.ExternalCloudProviderCondition)
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

	switch {
	case cluster.Spec.Cloud.Openstack != nil:
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

	case cluster.Spec.Cloud.VSphere != nil:
		fallthrough

	case cluster.Spec.Cloud.Azure != nil:
		supported, err := version.IsSupported(v.Semver(), kubermaticv1.AzureCloudProvider, incompatibilities, kubermaticv1.ExternalCloudProviderCondition)
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
func ExternalCloudControllerClusterName(cluster *kubermaticv1.Cluster) bool {
	switch {
	case cluster.Spec.Cloud.Openstack != nil:
		return true
	case cluster.Spec.Cloud.Azure != nil:
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
