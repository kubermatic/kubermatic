package kubestatemetrics

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("12Mi"),
			corev1.ResourceCPU:    resource.MustParse("10m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("1Gi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
	}
)

const (
	name    = "kube-state-metrics"
	version = "v1.7.2"
)

// DeploymentCreator returns the function to create and update the kube-state-metrics deployment
func DeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.KubeStateMetricsDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.KubeStateMetricsDeploymentName
			dep.Labels = resources.BaseAppLabel(name, nil)

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(name, nil),
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			volumes := getVolumes()
			podLabels, err := data.GetPodTemplateLabels(name, volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %v", err)
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
				Annotations: map[string]string{
					// do not specify a port so that Prometheus automatically
					// scrapes both the metrics and the telemetry endpoints
					"prometheus.io/scrape": "true",
				},
			}

			dep.Spec.Template.Spec.Volumes = volumes

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    name,
					Image:   data.ImageRegistry(resources.RegistryQuay) + "/coreos/kube-state-metrics:" + version,
					Command: []string{"/kube-state-metrics"},
					Args: []string{
						"--kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
						"--port", "8080",
						"--telemetry-port", "8081",
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.KubeStateMetricsKubeconfigSecretName,
							MountPath: "/etc/kubernetes/kubeconfig",
							ReadOnly:  true,
						},
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: 8080,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          "telemetry",
							ContainerPort: 8081,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Resources: defaultResourceRequirements,
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Port:   intstr.FromInt(8080),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						FailureThreshold: 3,
						PeriodSeconds:    10,
						SuccessThreshold: 1,
						TimeoutSeconds:   15,
					},
				},
			}

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, dep.Spec.Template.Spec, sets.NewString(name))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			dep.Spec.Template.Spec = *wrappedPodSpec

			return dep, nil
		}
	}
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.KubeStateMetricsKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.KubeStateMetricsKubeconfigSecretName,
				},
			},
		},
	}
}
