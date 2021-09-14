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

package cloudcontroller

import (
	"net/url"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version"

	corev1 "k8s.io/api/core/v1"
)

// ExternalCloudControllerFeatureSupported checks if the cloud provider supports
// external CCM.
func ExternalCloudControllerFeatureSupported(dc *kubermaticv1.Datacenter, cluster *kubermaticv1.Cluster, incompatibilities ...*version.ProviderIncompatibility) bool {
	switch {
	case cluster.Spec.Cloud.Openstack != nil:
		// When using OpenStack external CCM with Open Telekom Cloud the creation
		// of LBs fail as documented in the issue below:
		// https://github.com/kubernetes/cloud-provider-openstack/issues/960
		// Falling back to the in-tree CloudProvider mitigates the problem, even if
		// not all features are expected to work properly (e.g.
		// `manage-security-groups` should be set to false in cloud config, see
		// https://kubernetes.io/docs/concepts/cluster-administration/cloud-providers/#load-balancer
		// for more details).
		//
		// TODO This is a dirty hack to temporarily support OTC using
		// Openstack provider, remove this when dedicated OTC support is
		// introduced in Kubermatic.
		return !isOTC(dc.Spec.Openstack) && OpenStackCloudControllerSupported(cluster.Spec.Version)

	case cluster.Spec.Cloud.Hetzner != nil:
		return dc.Spec.Hetzner.Network != ""

	case cluster.Spec.Cloud.VSphere != nil:
		supported, err := version.IsSupported(cluster.Spec.Version.Version, kubermaticv1.ProviderVSphere, incompatibilities, version.ExternalCloudProviderCondition)
		if err != nil {
			return false
		}
		return supported

	case cluster.Spec.Cloud.Anexia != nil:
		return true

	case cluster.Spec.Cloud.Kubevirt != nil:
		return true

	default:
		return false
	}
}

// MigrationToExternalCloudControllerSupported checks if the cloud provider supports the migration to the
// external CCM.
func MigrationToExternalCloudControllerSupported(dc *kubermaticv1.Datacenter, cluster *kubermaticv1.Cluster, incompatibilities ...*version.ProviderIncompatibility) bool {
	switch {
	case cluster.Spec.Cloud.Openstack != nil:
		// When using OpenStack external CCM with Open Telekom Cloud the creation
		// of LBs fail as documented in the issue below:
		// https://github.com/kubernetes/cloud-provider-openstack/issues/960
		// Falling back to the in-tree CloudProvider mitigates the problem, even if
		// not all features are expected to work properly (e.g.
		// `manage-security-groups` should be set to false in cloud config, see
		// https://kubernetes.io/docs/concepts/cluster-administration/cloud-providers/#load-balancer
		// for more details).
		//
		// TODO This is a dirty hack to temporarily support OTC using
		// Openstack provider, remove this when dedicated OTC support is
		// introduced in Kubermatic.
		return !isOTC(dc.Spec.Openstack) && OpenStackCloudControllerSupported(cluster.Spec.Version)

	case cluster.Spec.Cloud.VSphere != nil:
		supported, err := version.IsSupported(cluster.Spec.Version.Version, kubermaticv1.ProviderVSphere, incompatibilities, version.ExternalCloudProviderCondition)
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

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.OpenVPNClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.OpenVPNClientCertificatesSecretName,
				},
			},
		},
		{
			Name: resources.CloudControllerManagerKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.CloudControllerManagerKubeconfigSecretName,
				},
			},
		},
	}
}

func getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      resources.CloudControllerManagerKubeconfigSecretName,
			MountPath: "/etc/kubernetes/kubeconfig",
			ReadOnly:  true,
		},
	}
}
