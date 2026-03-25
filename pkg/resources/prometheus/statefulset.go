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

package prometheus

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	name = "prometheus"
	tag  = "v2.51.1"

	volumeConfigName = "config"
	volumeDataName   = "data"
)

var defaultResourceRequirements = map[string]*corev1.ResourceRequirements{
	name: {
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("256Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("1Gi"),
			corev1.ResourceCPU:    resource.MustParse("500m"),
		},
	},
}

// StatefulSetReconciler returns the function to reconcile the Prometheus StatefulSet.
func StatefulSetReconciler(data *resources.TemplateData) reconciling.NamedStatefulSetReconcilerFactory {
	return func() (string, reconciling.StatefulSetReconciler) {
		return resources.PrometheusStatefulSetName, func(existing *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
			var set *appsv1.StatefulSet
			if existing != nil {
				set = existing
			} else {
				set = &appsv1.StatefulSet{}
			}

			requiredBaseLabels := map[string]string{"cluster": data.Cluster().Name}
			baseLabels := resources.BaseAppLabels(name, requiredBaseLabels)

			kubernetes.EnsureLabels(set, baseLabels)
			set.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}

			set.Spec.Replicas = resources.Int32(1)
			override := data.Cluster().Spec.ComponentsOverride.Prometheus
			if override.Replicas != nil {
				set.Spec.Replicas = override.Replicas
			}
			set.Spec.Template.Spec.Tolerations = override.Tolerations

			set.Spec.UpdateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType

			kubernetes.EnsureAnnotations(&set.Spec.Template, map[string]string{
				resources.ClusterLastRestartAnnotation: data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
				// these volumes should not block the autoscaler from evicting the pod
				resources.ClusterAutoscalerSafeToEvictVolumesAnnotation: volumeDataName,
			})

			set.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyAlways
			set.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				FSGroup:      resources.Int64(2000),
				RunAsNonRoot: resources.Bool(true),
				RunAsUser:    resources.Int64(1000),
			}
			set.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}
			set.Spec.Template.Spec.ServiceAccountName = resources.PrometheusServiceAccountName
			// We don't persist data, so there's no need for a graceful shutdown.
			// The faster restart time is preferable
			set.Spec.Template.Spec.TerminationGracePeriodSeconds = resources.Int64(0)

			set.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:  resources.PrometheusStatefulSetName,
					Image: registry.Must(data.RewriteImage(resources.RegistryQuay + "/prometheus/prometheus:" + tag)),
					Args: []string{
						"--config.file=/etc/prometheus/config/prometheus.yaml",
						"--storage.tsdb.path=/var/prometheus/data",
						"--storage.tsdb.min-block-duration=15m",
						"--storage.tsdb.max-block-duration=30m",
						"--storage.tsdb.retention.time=1h",
						"--web.enable-lifecycle",
						"--storage.tsdb.no-lockfile",
						"--web.route-prefix=/",
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "web",
							ContainerPort: 9090,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      volumeConfigName,
							MountPath: "/etc/prometheus/config",
							ReadOnly:  true,
						},
						{
							Name:      volumeDataName,
							MountPath: "/var/prometheus/data",
						},
						{
							Name:      resources.ApiserverEtcdClientCertificateSecretName,
							MountPath: "/etc/etcd/pki/client",
							ReadOnly:  true,
						},
						{
							Name:      resources.PrometheusApiserverClientCertificateSecretName,
							MountPath: "/etc/kubernetes",
							ReadOnly:  true,
						},
					},
					LivenessProbe: &corev1.Probe{
						PeriodSeconds:       5,
						TimeoutSeconds:      3,
						FailureThreshold:    10,
						InitialDelaySeconds: 30,
						SuccessThreshold:    1,
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/-/healthy",
								Port:   intstr.FromString("web"),
								Scheme: corev1.URISchemeHTTP,
							},
						},
					},
					ReadinessProbe: &corev1.Probe{
						PeriodSeconds:       5,
						TimeoutSeconds:      3,
						FailureThreshold:    6,
						InitialDelaySeconds: 5,
						SuccessThreshold:    1,
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/-/ready",
								Port:   intstr.FromString("web"),
								Scheme: corev1.URISchemeHTTP,
							},
						},
					},
				},
			}

			set.Spec.Template.Spec.Volumes = getVolumes()

			err := resources.SetResourceRequirements(set.Spec.Template.Spec.Containers, defaultResourceRequirements, resources.GetOverrides(data.Cluster().Spec.ComponentsOverride), set.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			return set, nil
		}
	}
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: volumeConfigName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.PrometheusConfigConfigMapName,
					},
				},
			},
		},
		{
			Name: volumeDataName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: resources.ApiserverEtcdClientCertificateSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.ApiserverEtcdClientCertificateSecretName,
				},
			},
		},
		{
			Name: resources.PrometheusApiserverClientCertificateSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.PrometheusApiserverClientCertificateSecretName,
				},
			},
		},
	}
}
