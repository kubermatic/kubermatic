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
	"fmt"
	"net/url"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/vpnsidecar"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8s.io/apimachinery/pkg/api/resource"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	osName = "openstack-cloud-controller-manager"
)

var (
	osResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("100Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("500m"),
		},
	}
)

func openStackDeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return osName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = osName
			dep.Labels = resources.BaseAppLabels(osName, nil)

			dep.Spec.Replicas = resources.Int32(1)

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(osName, nil),
			}

			dep.Spec.Template.Spec.Volumes = getOSVolumes()

			podLabels, err := data.GetPodTemplateLabels(osName, dep.Spec.Template.Spec.Volumes, nil)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
			}

			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err =
				resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			f := false
			dep.Spec.Template.Spec.AutomountServiceAccountToken = &f

			openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
			if err != nil {
				return nil, fmt.Errorf("failed to get openvpn sidecar: %v", err)
			}

			osCloudProviderMounts := []corev1.VolumeMount{
				{
					Name:      resources.CloudControllerManagerKubeconfigSecretName,
					MountPath: "/etc/kubernetes/kubeconfig",
					ReadOnly:  true,
				},
				{
					Name:      resources.CloudConfigConfigMapName,
					MountPath: "/etc/kubernetes/cloud",
					ReadOnly:  true,
				},
			}

			version, err := getOSVersion(data.Cluster().Spec.Version)
			if err != nil {
				return nil, err
			}
			flags := getOSFlags(data)

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				*openvpnSidecar,
				{
					Name:         osName,
					Image:        data.ImageRegistry(resources.RegistryDocker) + "/k8scloudprovider/openstack-cloud-controller-manager:v" + version,
					Command:      []string{"/bin/openstack-cloud-controller-manager"},
					Args:         flags,
					VolumeMounts: osCloudProviderMounts,
				},
			}
			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				osName:              osResourceRequirements.DeepCopy(),
				openvpnSidecar.Name: openvpnSidecar.Resources.DeepCopy(),
			}
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}

			return dep, nil
		}
	}
}

func getOSVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.CloudConfigConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.CloudConfigConfigMapName,
					},
				},
			},
		},
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

func getOSFlags(data *resources.TemplateData) []string {
	flags := []string{
		"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
		"--v=1",
		"--cloud-config=/etc/kubernetes/cloud/config",
		"--cloud-provider=openstack",
	}
	return flags
}

func getOSVersion(version semver.Semver) (string, error) {
	if version.Major() < 17 {
		return "", fmt.Errorf("Kubernetes version %s is not supported", version.String())
	}

	switch version.Minor() {
	case 17:
		return "1.17.0", nil
	case 18:
		return "1.18.0", nil
	case 19:
		return "1.19.2", nil
	default:
		return "1.19.2", nil
	}
}

// ExternalCloudControllerFeatureSupported checks if the
func ExternalCloudControllerFeatureSupported(dc *kubermaticv1.Datacenter, cluster *kubermaticv1.Cluster) bool {
	if cluster.Spec.Cloud.Openstack == nil {
		return false
	}
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
	if dc.Spec.Openstack != nil && isOTC(dc.Spec.Openstack) {
		return false
	}
	return OpenStackCloudControllerSupported(cluster.Spec.Version)
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

// OpenStackCloudControllerSupported checks if this version of Kubernetes is supported
// by our implementation of the external cloud controller.
// This is not called for now, but it's here so we can use it later to automagically
// enable external cloud controller support for supported versions.
func OpenStackCloudControllerSupported(version semver.Semver) bool {
	if _, err := getOSVersion(version); err != nil {
		return false
	}
	return true
}
