/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

const (
	imageName = "prometheus/prometheus"
	tag       = "v2.26.0"
	appName   = "mla-prometheus"

	configVolumeName       = "config-volume"
	configPath             = "/etc/config"
	storageVolumeName      = "storage-volume"
	storagePath            = "/data"
	certificatesVolumeName = "certificates"

	prometheusNameKey     = "app.kubernetes.io/name"
	prometheusInstanceKey = "app.kubernetes.io/instance"

	containerPort = 9090
)

var (
	controllerLabels = map[string]string{
		prometheusNameKey:     resources.UserClusterPrometheusDeploymentName,
		prometheusInstanceKey: resources.UserClusterPrometheusDeploymentName,
	}
)

func DeploymentCreator() reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.UserClusterPrometheusDeploymentName, func(deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
			deployment.Labels = resources.BaseAppLabels(appName, nil)

			deployment.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: controllerLabels,
			}
			deployment.Spec.Replicas = pointer.Int32Ptr(1)
			deployment.Spec.Template.ObjectMeta.Labels = controllerLabels
			deployment.Spec.Template.Spec.ServiceAccountName = resources.UserClusterPrometheusServiceAccountName
			deployment.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				RunAsUser:    pointer.Int64Ptr(65534),
				RunAsGroup:   pointer.Int64Ptr(65534),
				FSGroup:      pointer.Int64Ptr(65534),
				RunAsNonRoot: pointer.BoolPtr(true),
			}
			deployment.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            "prometheus-server-configmap-reload",
					Image:           fmt.Sprintf("%s/%s:%s", resources.RegistryQuay, imageName, tag),
					ImagePullPolicy: corev1.PullAlways,
					Args: []string{
						fmt.Sprintf("--config.file=%s/prometheus.yaml", configPath),
						"--storage.tsdb.retention.time=15d",
						fmt.Sprintf("--storage.tsdb.path=%s", storagePath),
						"--web.console.libraries=/etc/prometheus/console_libraries",
						"--web.console.templates=/etc/prometheus/consoles",
						"--web.enable-lifecycle",
					},
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: containerPort,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      configVolumeName,
							MountPath: configPath,
						},
						{
							Name:      certificatesVolumeName,
							MountPath: resources.UserClusterPrometheusClientCertMountPath,
						},
						{
							Name:      storageVolumeName,
							MountPath: storagePath,
							SubPath:   "",
						},
					},
					LivenessProbe: &corev1.Probe{
						PeriodSeconds:       5,
						TimeoutSeconds:      4,
						FailureThreshold:    3,
						InitialDelaySeconds: 30,
						SuccessThreshold:    1,
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/-/healthy",
								Port:   intstr.FromInt(containerPort),
								Scheme: corev1.URISchemeHTTP,
							},
						},
					},
					ReadinessProbe: &corev1.Probe{
						PeriodSeconds:       5,
						TimeoutSeconds:      4,
						FailureThreshold:    3,
						InitialDelaySeconds: 30,
						SuccessThreshold:    1,
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/-/ready",
								Port:   intstr.FromInt(containerPort),
								Scheme: corev1.URISchemeHTTP,
							},
						},
					},
				},
			}
			deployment.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: configVolumeName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: resources.UserClusterPrometheusConfigMapName,
							},
						},
					},
				},
				{
					Name: certificatesVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  resources.UserClusterPrometheusCertificatesSecretName,
							DefaultMode: pointer.Int32Ptr(0400),
						},
					},
				},
				{
					Name: storageVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			}
			return deployment, nil
		}
	}
}
