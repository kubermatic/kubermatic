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
	switch openshiftVersion {
	case openshiftVersion419:
		return []corev1.EnvVar{
			{Name: "RELEASE_VERSION", Value: "4.1.9"},
			{Name: "NODE_IMAGE", Value: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:472dd90bc413a9bcb99be23f7296763468ebbeb985c10b26d1c44c4b04f57a77"},
			{Name: "HYPERSHIFT_IMAGE", Value: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:86255c4efe6bbc141a0f41444f863bbd5cd832ffca21d2b737a4f9c225ed00ad"},
			{Name: "MULTUS_IMAGE", Value: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:6766e62f61307e7c5a187f61d33b99ba90390b2f43351f591bb8da951915ce04"},
			{Name: "CNI_PLUGINS_SUPPORTED_IMAGE", Value: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:473d03cbfa265d2a6def817f8ec5bd1c6536d3e39cf8c2f8223dd41ed2bd4541"},
			{Name: "CNI_PLUGINS_UNSUPPORTED_IMAGE", Value: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:d7c6701150c7ad12fc6dd26f2c6b093da5e9e3b43dea89196a77da1c6ef6904b"},
			{Name: "SRIOV_CNI_IMAGE", Value: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:9d332f4b42997f917fa7660d85975c579ee4abe354473acbd45fc2a093b12e3b"},
			{Name: "SRIOV_DEVICE_PLUGIN_IMAGE", Value: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:21c668c419662bf1a5c1f38d55f6ab20b4e22b807d076f927efb1ac954beed60"},
			{Name: "OVN_IMAGE", Value: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:81088a1f27ff88e7e4a65dd3ca47513aad76bfbfc44af359887baa1d3fa60eba"},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported version %q", openshiftVersion)
	}
}
