package cloudcontroller

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"github.com/kubermatic/kubermatic/api/pkg/resources/vpnsidecar"
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
			dep.Labels = resources.BaseAppLabel(osName, nil)

			dep.Spec.Replicas = resources.Int32(1)

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(osName, nil),
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

			version, err := getOSVersion(data)
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
					Resources:    osResourceRequirements,
					VolumeMounts: osCloudProviderMounts,
				},
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

func getOSVersion(data *resources.TemplateData) (string, error) {
	switch x := data.Cluster().Spec.Version.MajorMinor(); x {
	case "1.16":
		return "1.16.0", nil
	default:
		return "", fmt.Errorf("kubernetes version %s not supported", x)
	}
}

// OpenStackCloudControllerSupported checks if this version of Kubernetes is supported
// by our implementation of the external cloud controller.
// This is not called for now, but it's here so we can use it later to automagically
// enable external cloud controller support for supported versions.
func OpenStackCloudControllerSupported(data *resources.TemplateData) bool {
	if _, err := getOSVersion(data); err != nil {
		return false
	}
	return true
}
