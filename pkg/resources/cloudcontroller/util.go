package cloudcontroller

import (
	"net/url"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
)

// ExternalCloudControllerFeatureSupported checks if the
func ExternalCloudControllerFeatureSupported(dc *kubermaticv1.Datacenter, cluster *kubermaticv1.Cluster) bool {
	if dc.Spec.Openstack != nil {
		// When using OpenStack external CCM with Open Telekom Cloud the creation
		// of LBs fail as documented in the issue below:
		// https://github.com/kubernetes/cloud-provider-openstack/issues/960
		// Falling back to the in-tree CloudProvider mitigates the problem, even if
		// not all features are expected to work properly (e.g.
		// `manage-security-groups` should be set to false in cloud config, see
		// https://kubernetes.io/docs/concepts/cluster-administration/cloud-providers/#load-balancer
		// for more details).
		//
		// TODO(irozzo) This is a dirty hack to temporarily support OTC using
		// Openstack provider, remove this when dedicated OTC support is
		// introduced in Kubermatic.
		return !isOTC(dc.Spec.Openstack) && OpenStackCloudControllerSupported(cluster.Spec.Version)
	}

	if dc.Spec.Hetzner != nil {
		return cluster.Spec.Version.Minor() >= 18
	}

	return false
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
