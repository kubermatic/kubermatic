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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version"

	corev1 "k8s.io/api/core/v1"
)

const (
	v121 = "1.21"
	v122 = "1.22"
	v123 = "1.23"
	v124 = "1.24"

	v1240 = "1.24.0"
)

// ExternalCloudControllerFeatureSupported checks if the cloud provider supports
// external CCM.
func ExternalCloudControllerFeatureSupported(dc *kubermaticv1.Datacenter, cluster *kubermaticv1.Cluster, incompatibilities ...*version.ProviderIncompatibility) bool {
	// This function is called during cluster creation and at that time, the
	// cluster status might not have been initially set yet, so we must ensure
	// to fallback to the spec version.
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
		return !isOTC(dc.Spec.Openstack) && OpenStackCloudControllerSupported(v)

	case cluster.Spec.Cloud.Hetzner != nil:
		return cluster.Spec.Cloud.Hetzner.Network != "" || dc.Spec.Hetzner.Network != ""

	case cluster.Spec.Cloud.VSphere != nil:
		supported, err := version.IsSupported(v.Semver(), kubermaticv1.VSphereCloudProvider, incompatibilities, kubermaticv1.ExternalCloudProviderCondition)
		if err != nil {
			return false
		}
		return supported

	case cluster.Spec.Cloud.AWS != nil:
		return true

	case cluster.Spec.Cloud.Anexia != nil:
		return true

	case cluster.Spec.Cloud.Kubevirt != nil:
		return true

	case cluster.Spec.Cloud.Azure != nil:
		return AzureCloudControllerSupported(v)

	default:
		return false
	}
}

// MigrationToExternalCloudControllerSupported checks if the cloud provider supports the migration to the
// external CCM.
func MigrationToExternalCloudControllerSupported(dc *kubermaticv1.Datacenter, cluster *kubermaticv1.Cluster, incompatibilities ...*version.ProviderIncompatibility) bool {
	// This function is called during cluster creation and at that time, the
	// cluster status might not have been initially set yet, so we must ensure
	// to fallback to the spec version.
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
		return !isOTC(dc.Spec.Openstack) && OpenStackCloudControllerSupported(v)

	case cluster.Spec.Cloud.VSphere != nil:
		supported, err := version.IsSupported(v.Semver(), kubermaticv1.VSphereCloudProvider, incompatibilities, kubermaticv1.ExternalCloudProviderCondition)
		if err != nil {
			return false
		}
		return supported

	case cluster.Spec.Cloud.Azure != nil:
		versionSupported, err := version.IsSupported(v.Semver(), kubermaticv1.AzureCloudProvider, incompatibilities, kubermaticv1.ExternalCloudProviderCondition)
		if err != nil {
			return false
		}
		return versionSupported && AzureCloudControllerSupported(v)

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

func getVolumes(isKonnectivityEnabled bool) []corev1.Volume {
	vs := []corev1.Volume{
		{
			Name: resources.CloudControllerManagerKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.CloudControllerManagerKubeconfigSecretName,
				},
			},
		},
	}
	if !isKonnectivityEnabled {
		vs = append(vs, corev1.Volume{
			Name: resources.OpenVPNClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.OpenVPNClientCertificatesSecretName,
				},
			},
		})
	}
	return vs
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
