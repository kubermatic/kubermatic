package machinecontroller

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	controllerResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("32Mi"),
			corev1.ResourceCPU:    resource.MustParse("25m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("2"),
		},
	}
)

const (
	name = "machine-controller"

	tag = "v0.10.0"
)

// Deployment returns the machine-controller Deployment
func Deployment(data resources.DeploymentDataProvider, existing *appsv1.Deployment) (*appsv1.Deployment, error) {
	var dep *appsv1.Deployment
	if existing != nil {
		dep = existing
	} else {
		dep = &appsv1.Deployment{}
	}

	dep.Name = resources.MachineControllerDeploymentName
	dep.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

	dep.Labels = resources.BaseAppLabel(name, nil)

	dep.Spec.Replicas = resources.Int32(1)
	dep.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: resources.BaseAppLabel(name, nil),
	}
	dep.Spec.Strategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
	dep.Spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{
		MaxSurge: &intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: 1,
		},
		MaxUnavailable: &intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: 0,
		},
	}
	dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

	volumes := []corev1.Volume{getKubeconfigVolume()}
	podLabels, err := data.GetPodTemplateLabels(name, volumes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create pod labels: %v", err)
	}

	dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Labels: podLabels,
		Annotations: map[string]string{
			"prometheus.io/scrape": "true",
			"prometheus.io/path":   "/metrics",
			"prometheus.io/port":   "8085",
		},
	}

	dep.Spec.Template.Spec.Volumes = volumes

	apiserverIsRunningContainer, err := apiserver.IsRunningInitContainer(data)
	if err != nil {
		return nil, err
	}
	dep.Spec.Template.Spec.InitContainers = []corev1.Container{*apiserverIsRunningContainer}

	clusterDNSIP, err := resources.UserClusterDNSResolverIP(data.Cluster())
	if err != nil {
		return nil, err
	}
	dep.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:            name,
			Image:           data.ImageRegistry(resources.RegistryDocker) + "/kubermatic/machine-controller:" + tag,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/usr/local/bin/machine-controller"},
			Args: []string{
				"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
				"-logtostderr",
				"-v", "4",
				"-cluster-dns", clusterDNSIP,
				"-internal-listen-address", "0.0.0.0:8085",
			},
			Env:                      getEnvVars(data),
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			Resources:                controllerResourceRequirements,
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/ready",
						Port: intstr.FromInt(8085),
					},
				},
				FailureThreshold: 3,
				PeriodSeconds:    10,
				SuccessThreshold: 1,
				TimeoutSeconds:   15,
			},
			LivenessProbe: &corev1.Probe{
				FailureThreshold: 8,
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/live",
						Port: intstr.FromInt(8085),
					},
				},
				InitialDelaySeconds: 15,
				PeriodSeconds:       10,
				SuccessThreshold:    1,
				TimeoutSeconds:      15,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      resources.MachineControllerKubeconfigSecretName,
					MountPath: "/etc/kubernetes/kubeconfig",
					ReadOnly:  true,
				},
			},
		},
	}

	return dep, nil
}

func getKubeconfigVolume() corev1.Volume {
	return corev1.Volume{
		Name: resources.MachineControllerKubeconfigSecretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: resources.MachineControllerKubeconfigSecretName,
				// We have to make the secret readable for all for now because owner/group cannot be changed.
				// ( upstream proposal: https://github.com/kubernetes/kubernetes/pull/28733 )
				DefaultMode: resources.Int32(resources.DefaultAllReadOnlyMode),
			},
		},
	}
}

func getEnvVars(data resources.DeploymentDataProvider) []corev1.EnvVar {
	var vars []corev1.EnvVar
	if data.Cluster().Spec.Cloud.AWS != nil {
		vars = append(vars, corev1.EnvVar{Name: "AWS_ACCESS_KEY_ID", Value: data.Cluster().Spec.Cloud.AWS.AccessKeyID})
		vars = append(vars, corev1.EnvVar{Name: "AWS_SECRET_ACCESS_KEY", Value: data.Cluster().Spec.Cloud.AWS.SecretAccessKey})
	}
	if data.Cluster().Spec.Cloud.Openstack != nil {
		vars = append(vars, corev1.EnvVar{Name: "OS_AUTH_URL", Value: data.DC().Spec.Openstack.AuthURL})
		vars = append(vars, corev1.EnvVar{Name: "OS_USER_NAME", Value: data.Cluster().Spec.Cloud.Openstack.Username})
		vars = append(vars, corev1.EnvVar{Name: "OS_PASSWORD", Value: data.Cluster().Spec.Cloud.Openstack.Password})
		vars = append(vars, corev1.EnvVar{Name: "OS_DOMAIN_NAME", Value: data.Cluster().Spec.Cloud.Openstack.Domain})
		vars = append(vars, corev1.EnvVar{Name: "OS_TENANT_NAME", Value: data.Cluster().Spec.Cloud.Openstack.Tenant})
	}
	if data.Cluster().Spec.Cloud.Hetzner != nil {
		vars = append(vars, corev1.EnvVar{Name: "HZ_TOKEN", Value: data.Cluster().Spec.Cloud.Hetzner.Token})
	}
	if data.Cluster().Spec.Cloud.Digitalocean != nil {
		vars = append(vars, corev1.EnvVar{Name: "DO_TOKEN", Value: data.Cluster().Spec.Cloud.Digitalocean.Token})
	}
	if data.Cluster().Spec.Cloud.VSphere != nil {
		vars = append(vars, corev1.EnvVar{Name: "VSPHERE_ADDRESS", Value: data.DC().Spec.VSphere.Endpoint})
		vars = append(vars, corev1.EnvVar{Name: "VSPHERE_USERNAME", Value: data.Cluster().Spec.Cloud.VSphere.InfraManagementUser.Username})
		vars = append(vars, corev1.EnvVar{Name: "VSPHERE_PASSWORD", Value: data.Cluster().Spec.Cloud.VSphere.InfraManagementUser.Password})
	}
	return vars
}
