package metricsserver

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
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("32Mi"),
			corev1.ResourceCPU:    resource.MustParse("25m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("150m"),
		},
	}

	certDirSize = resource.MustParse("1Mi")
)

const (
	name = "metrics-server"

	tag = "v0.5.0"
)

// DeploymentCreator returns the function to create and update the metrics server deployment
func DeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.MetricsServerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.MetricsServerDeploymentName
			dep.Labels = resources.BaseAppLabel(name, nil)

			dep.Spec.Replicas = resources.Int32(2)
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

			volumes := getVolumes()
			podLabels, err := data.GetPodTemplateLabels(name, volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %v", err)
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
			}

			dep.Spec.Template.Spec.Volumes = volumes

			apiserverIsRunningContainer, err := apiserver.IsRunningInitContainer(data)
			if err != nil {
				return nil, err
			}
			dep.Spec.Template.Spec.InitContainers = []corev1.Container{*apiserverIsRunningContainer}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            name,
					Image:           data.ImageRegistry(resources.RegistryDocker) + "/directxman12/k8s-prometheus-adapter-amd64:" + tag,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/adapter"},
					Args: []string{
						"--secure-port", "6443",
						"--logtostderr",
						"--v", "2",
						"--cert-dir", "/etc/adapter-certs",
						"--prometheus-url", "http://prometheus:9090/",
						"--metrics-relist-interval", "1m",
						"--config", "/etc/adapter/config.yaml",
						"--lister-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
						"--authentication-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
						"--authorization-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
					},
					Resources: defaultResourceRequirements,
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Port:   intstr.FromInt(6443),
								Scheme: "HTTPS",
							},
						},
						FailureThreshold: 3,
						PeriodSeconds:    5,
						SuccessThreshold: 1,
						TimeoutSeconds:   15,
					},
					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Port:   intstr.FromInt(6443),
								Scheme: "HTTPS",
							},
						},
						InitialDelaySeconds: 60,
						FailureThreshold:    8,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      30,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.MetricsServerKubeconfigSecretName,
							MountPath: "/etc/kubernetes/kubeconfig",
							ReadOnly:  true,
						},
						{
							Name:      resources.MetricsServerConfigConfigMapName,
							MountPath: "/etc/adapter",
							ReadOnly:  true,
						},
						{
							Name:      "cert-dir",
							MountPath: "/etc/adapter-certs",
							ReadOnly:  false,
						},
					},
				},
			}

			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(name, data.Cluster().Name)

			return dep, nil
		}
	}
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.MetricsServerKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.MetricsServerKubeconfigSecretName,
					// We have to make the secret readable for all for now because owner/group cannot be changed.
					// ( upstream proposal: https://github.com/kubernetes/kubernetes/pull/28733 )
					DefaultMode: resources.Int32(resources.DefaultAllReadOnlyMode),
				},
			},
		},
		{
			Name: resources.MetricsServerConfigConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.MetricsServerConfigConfigMapName,
					},
					DefaultMode: resources.Int32(resources.DefaultAllReadOnlyMode),
				},
			},
		},
		{
			Name: "cert-dir",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: &certDirSize,
				},
			},
		},
	}
}
