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

package kubestatemetrics

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	defaultResourceRequirements = map[string]*corev1.ResourceRequirements{
		name: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("64Mi"),
				corev1.ResourceCPU:    resource.MustParse("50m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1Gi"),
				corev1.ResourceCPU:    resource.MustParse("150m"),
			},
		},
	}
)

const (
	name      = "kube-state-metrics"
	version   = "v2.12.0"
	tmpVolume = "tmp"
)

// DeploymentReconciler returns the function to create and update the kube-state-metrics deployment.
func DeploymentReconciler(data *resources.TemplateData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.KubeStateMetricsDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			baseLabels := resources.BaseAppLabels(resources.KubeStateMetricsDeploymentName, nil)
			kubernetes.EnsureLabels(dep, baseLabels)

			dep.Spec.Replicas = resources.Int32(1)

			if data.Cluster().Spec.ComponentsOverride.KubeStateMetrics != nil {
				if data.Cluster().Spec.ComponentsOverride.KubeStateMetrics.Replicas != nil {
					dep.Spec.Replicas = data.Cluster().Spec.ComponentsOverride.KubeStateMetrics.Replicas
				}

				if data.Cluster().Spec.ComponentsOverride.KubeStateMetrics.Tolerations != nil {
					dep.Spec.Template.Spec.Tolerations = data.Cluster().Spec.ComponentsOverride.KubeStateMetrics.Tolerations
				}
			}

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				// do not specify a port so that Prometheus automatically
				// scrapes both the metrics and the telemetry endpoints
				"prometheus.io/scrape":                                         "true",
				resources.ClusterLastRestartAnnotation:                         data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
				"cluster-autoscaler.kubernetes.io/safe-to-evict-local-volumes": tmpVolume,
			})

			dep.Spec.Template.Spec.Volumes = getVolumes()

			dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				FSGroup: resources.Int64(65534),
			}

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    name,
					Image:   registry.Must(data.RewriteImage(resources.RegistryK8S + "/kube-state-metrics/kube-state-metrics:" + version)),
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
						{
							Name:      tmpVolume,
							MountPath: "/tmp",
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
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
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
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/",
								Port:   intstr.FromInt(8080),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						FailureThreshold: 3,
						PeriodSeconds:    10,
						SuccessThreshold: 1,
						TimeoutSeconds:   15,
					},
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: resources.Bool(false),
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{
								corev1.Capability("ALL"),
							},
						},
						ReadOnlyRootFilesystem: resources.Bool(true),
						RunAsGroup:             resources.Int64(65534),
						RunAsUser:              resources.Int64(65534),
						RunAsNonRoot:           resources.Bool(true),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
				},
			}

			err := resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defaultResourceRequirements, resources.GetOverrides(data.Cluster().Spec.ComponentsOverride), dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			dep.Spec.Template, err = apiserver.IsRunningWrapper(data, dep.Spec.Template, sets.New(name))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %w", err)
			}

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
		{
			Name: tmpVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}
