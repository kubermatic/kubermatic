package resources

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	utilpointer "k8s.io/utils/pointer"
)

const (
	openshiftRegistryOperatorName = "openshift-registry-operator"
	// RegistryNamespaceName is the name in which the registry is getting created by the openshift registry operator
	RegistryNamespaceName = "openshift-image-registry"
)

func RegistryOperatorFactory(data openshiftData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return openshiftRegistryOperatorName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			image, err := clusterImageRegistryOperatorImage(data.Cluster().Spec.Version.String(), data.ImageRegistry(""))
			if err != nil {
				return nil, err
			}
			env, err := registryOperatorEnv(data)
			if err != nil {
				return nil, err
			}
			env = append(env, corev1.EnvVar{
				Name:  "KUBECONFIG",
				Value: "/etc/kubernetes/kubeconfig/kubeconfig",
			})

			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(openshiftRegistryOperatorName, nil),
			}
			d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{Name: openshiftImagePullSecretName},
			}
			d.Spec.Template.Spec.AutomountServiceAccountToken = utilpointer.BoolPtr(false)
			d.Spec.Template.Spec.DNSPolicy, d.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}
			d.Spec.Template.Spec.Containers = []corev1.Container{{
				Name:    openshiftRegistryOperatorName,
				Image:   image,
				Env:     env,
				Command: []string{"cluster-image-registry-operator"},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      resources.InternalUserClusterAdminKubeconfigSecretName,
					MountPath: "/etc/kubernetes/kubeconfig",
				}},
			}}
			d.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					// TODO: Properly limit this instead of just using cluster-admin
					Name: resources.InternalUserClusterAdminKubeconfigSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.InternalUserClusterAdminKubeconfigSecretName,
						},
					},
				},
			}
			d.Spec.Template.Labels, err = data.GetPodTemplateLabels(openshiftRegistryOperatorName, d.Spec.Template.Spec.Volumes, nil)
			if err != nil {
				return nil, err
			}
			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, d.Spec.Template.Spec, sets.NewString(openshiftRegistryOperatorName), "Config,imageregistry.operator.openshift.io/v1")
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			d.Spec.Template.Spec = *wrappedPodSpec

			return d, nil
		}
	}
}

func registryOperatorEnv(data openshiftData) ([]corev1.EnvVar, error) {
	openshiftVersion := data.Cluster().Spec.Version.String()
	image, err := dockerRegistryImage(openshiftVersion, data.ImageRegistry(""))
	if err != nil {
		return nil, err
	}

	return []corev1.EnvVar{
		{
			Name:  "RELEASE_VERSION",
			Value: openshiftVersion,
		},
		{
			Name:  "WATCH_NAMESPACE",
			Value: RegistryNamespaceName,
		},
		{
			Name:  "OPERATOR_NAME",
			Value: "cluster-image-registry-operator",
		},
		{
			Name:  "IMAGE",
			Value: image,
		},
	}, nil
}
