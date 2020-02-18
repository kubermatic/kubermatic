package resources

import (
	"fmt"

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

const (
	openshiftDNSOperatorDeploymentName = "openshift-dns-operator"
	openshiftDNSOperatorContainerName  = "dns-operator"
)

var (
	openshiftDNSOperatorResourceRequirements = map[string]*corev1.ResourceRequirements{
		openshiftDNSOperatorContainerName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("50Mi"),
				corev1.ResourceCPU:    resource.MustParse("10m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("200Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
		},
	}
)

func OpenshiftDNSOperatorFactory(data openshiftData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return openshiftDNSOperatorDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {

			image, err := clusterDnsOperatorImage(data.Cluster().Spec.Version.String(), data.ImageRegistry(""))
			if err != nil {
				return nil, err
			}

			env, err := openshiftDNSOperatorEnv(data)
			if err != nil {
				return nil, err
			}
			env = append(env, corev1.EnvVar{
				Name:  "KUBECONFIG",
				Value: "/etc/kubernetes/kubeconfig/kubeconfig",
			})

			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(openshiftDNSOperatorDeploymentName, nil),
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
				Name:  openshiftDNSOperatorContainerName,
				Image: image,
				Env: append(env, corev1.EnvVar{
					Name:  "KUBECONFIG",
					Value: "/etc/kubernetes/kubeconfig/kubeconfig",
				}),
				Command: []string{"dns-operator"},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      resources.InternalUserClusterAdminKubeconfigSecretName,
					MountPath: "/etc/kubernetes/kubeconfig",
				}},
			}}
			err = resources.SetResourceRequirements(d.Spec.Template.Spec.Containers, openshiftDNSOperatorResourceRequirements, nil, d.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}
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

			d.Spec.Template.Labels, err = data.GetPodTemplateLabels(openshiftDNSOperatorDeploymentName, d.Spec.Template.Spec.Volumes, nil)
			if err != nil {
				return nil, err
			}

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, d.Spec.Template.Spec, sets.NewString(openshiftDNSOperatorContainerName), "DNS,operator.openshift.io/v1", "Network,config.openshift.io/v1")
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			d.Spec.Template.Spec = *wrappedPodSpec

			return d, nil
		}
	}
}

func openshiftDNSOperatorEnv(data openshiftData) ([]corev1.EnvVar, error) {
	openshiftVersion := data.Cluster().Spec.Version.String()
	cliImageValue, err := cliImage(openshiftVersion, data.ImageRegistry(""))
	if err != nil {
		return nil, err
	}
	coreDNSImageValue, err := corednsImage(openshiftVersion, data.ImageRegistry(""))
	if err != nil {
		return nil, err
	}

	return []corev1.EnvVar{
		{Name: "RELEASE_VERSION", Value: openshiftVersion},
		{Name: "IMAGE", Value: coreDNSImageValue},
		{Name: "OPENSHIFT_CLI_IMAGE", Value: cliImageValue},
	}, nil
}
