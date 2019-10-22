package resources

import (
	"fmt"
	"strconv"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	utilpointer "k8s.io/utils/pointer"
)

var (
	openshiftNetworkOperatorResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("50Mi"),
			corev1.ResourceCPU:    resource.MustParse("10m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("200Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
	}
)

const openshiftNetworkOperatorDeploymentName = "openshift-network-operator"

func OpenshiftNetworkOperatorCreatorFactory(data openshiftData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return openshiftNetworkOperatorDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			image, err := clusterNetworkOperatorImage(data.Cluster().Spec.Version.String())
			if err != nil {
				return nil, err
			}
			env, err := openshiftNetworkOperatorEnv(data.Cluster().Spec.Version.String())
			if err != nil {
				return nil, err
			}
			// This gets injected into the created pods, so it must match the usercluster
			env = append(env,
				corev1.EnvVar{
					Name:  "KUBERNETES_SERVICE_HOST",
					Value: data.Cluster().Address.IP,
				},
				corev1.EnvVar{
					Name:  "KUBERNETES_SERVICE_PORT",
					Value: strconv.Itoa(int(data.Cluster().Address.Port)),
				})

			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(openshiftNetworkOperatorDeploymentName, nil),
			}
			d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{Name: openshiftImagePullSecretName},
			}
			d.Spec.Template.Spec.AutomountServiceAccountToken = utilpointer.BoolPtr(false)
			// Configure user cluster DNS resolver for this pod.
			d.Spec.Template.Spec.DNSPolicy, d.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}
			d.Spec.Template.Spec.Containers = []corev1.Container{{
				Name:  "network-operator",
				Image: image,
				Env: append(env, corev1.EnvVar{
					Name:  "KUBECONFIG",
					Value: "/etc/kubernetes/kubeconfig/kubeconfig",
				}),
				Command: []string{
					"/usr/bin/cluster-network-operator",
					"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
				},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      resources.InternalUserClusterAdminKubeconfigSecretName,
					MountPath: "/etc/kubernetes/kubeconfig",
				}},
				Resources: *openshiftNetworkOperatorResourceRequirements.DeepCopy(),
			}}
			d.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: resources.InternalUserClusterAdminKubeconfigSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.InternalUserClusterAdminKubeconfigSecretName,
						},
					},
				},
			}

			d.Spec.Template.Labels, err = data.GetPodTemplateLabels(openshiftNetworkOperatorDeploymentName, d.Spec.Template.Spec.Volumes, nil)
			if err != nil {
				return nil, err
			}

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, d.Spec.Template.Spec, sets.NewString("network-operator"), "Network,operator.openshift.io/v1")
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			d.Spec.Template.Spec = *wrappedPodSpec

			return d, nil
		}
	}
}

func openshiftNetworkOperatorEnv(openshiftVersion string) ([]corev1.EnvVar, error) {
	nodeImageValue, err := nodeImage(openshiftVersion)
	if err != nil {
		return nil, err
	}
	hypershiftImageValue, err := hypershiftImage(openshiftVersion)
	if err != nil {
		return nil, err
	}
	multusCniImageValue, err := multusCniImage(openshiftVersion)
	if err != nil {
		return nil, err
	}
	containerNetworkingPluginsSupportedImageValue, err := containerNetworkingPluginsSupportedImage(openshiftVersion)
	if err != nil {
		return nil, err
	}
	containerNetworkingPluginsUnsupportedImageValue, err := containerNetworkingPluginsUnsupportedImage(openshiftVersion)
	if err != nil {
		return nil, err
	}
	sriovCniImageValue, err := sriovCniImage(openshiftVersion)
	if err != nil {
		return nil, err
	}
	sriovNetworkDevicePluginImageValue, err := sriovNetworkDevicePluginImage(openshiftVersion)
	if err != nil {
		return nil, err
	}
	ovnKubernetesImageValue, err := ovnKubernetesImage(openshiftVersion)
	if err != nil {
		return nil, err
	}

	return []corev1.EnvVar{
		{Name: "RELEASE_VERSION", Value: openshiftVersion},
		{Name: "NODE_IMAGE", Value: nodeImageValue},
		{Name: "HYPERSHIFT_IMAGE", Value: hypershiftImageValue},
		{Name: "MULTUS_IMAGE", Value: multusCniImageValue},
		{Name: "CNI_PLUGINS_SUPPORTED_IMAGE", Value: containerNetworkingPluginsSupportedImageValue},
		{Name: "CNI_PLUGINS_UNSUPPORTED_IMAGE", Value: containerNetworkingPluginsUnsupportedImageValue},
		{Name: "SRIOV_CNI_IMAGE", Value: sriovCniImageValue},
		{Name: "SRIOV_DEVICE_PLUGIN_IMAGE", Value: sriovNetworkDevicePluginImageValue},
		{Name: "OVN_IMAGE", Value: ovnKubernetesImageValue},
	}, nil
}
