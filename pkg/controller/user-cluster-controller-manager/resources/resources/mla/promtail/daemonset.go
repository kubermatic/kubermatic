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

package promtail

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
	imageName = "grafana/promtail"
	tag       = "2.1.0"
	appName   = "mla-promtail"

	configVolumeName         = "config"
	configVolumeMountPath    = "/etc/promtail"
	certificatesVolumeName   = "certificates"
	runVolumeName            = "run"
	runVolumeMountPath       = "/run/promtail"
	containerVolumeName      = "containers"
	containerVolumeMountPath = "/var/lib/docker/containers"
	podVolumeName            = "pods"
	podVolumeMountPath       = "/var/log/pods"
	metricsPortName          = "http-metrics"

	promtailNameKey     = "app.kubernetes.io/name"
	promtailInstanceKey = "app.kubernetes.io/instance"
)

var (
	controllerLabels = map[string]string{
		promtailNameKey:     resources.PromtailDaemonSetName,
		promtailInstanceKey: resources.PromtailDaemonSetName,
	}
)

func DaemonSetCreator() reconciling.NamedDaemonSetCreatorGetter {
	return func() (string, reconciling.DaemonSetCreator) {
		return resources.PromtailDaemonSetName, func(ds *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
			ds.Labels = resources.BaseAppLabels(appName, nil)

			ds.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: controllerLabels,
			}

			ds.Spec.Template.ObjectMeta.Labels = controllerLabels
			ds.Spec.Template.Spec.ServiceAccountName = resources.PromtailServiceAccountName
			ds.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				RunAsUser:  pointer.Int64Ptr(0),
				RunAsGroup: pointer.Int64Ptr(0),
			}
			ds.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            "promtail",
					Image:           fmt.Sprintf("%s/%s:%s", resources.RegistryDocker, imageName, tag),
					ImagePullPolicy: corev1.PullAlways,
					Args: []string{
						"-config.file=/etc/promtail/promtail.yaml",
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      configVolumeName,
							MountPath: configVolumeMountPath,
						},
						{
							Name:      certificatesVolumeName,
							MountPath: resources.PromtailClientCertMountPath,
						},
						{
							Name:      runVolumeName,
							MountPath: runVolumeMountPath,
						},
						{
							Name:      containerVolumeName,
							MountPath: containerVolumeMountPath,
							ReadOnly:  true,
						},
						{
							Name:      podVolumeName,
							MountPath: podVolumeMountPath,
							ReadOnly:  true,
						},
					},
					Env: []corev1.EnvVar{
						{
							Name: "HOSTNAME",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "spec.nodeName",
								},
							},
						},
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          metricsPortName,
							ContainerPort: 3101,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: pointer.BoolPtr(false),
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{
								"all",
							},
						},
						ReadOnlyRootFilesystem: pointer.BoolPtr(true),
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/ready",
								Port:   intstr.FromString(metricsPortName),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						FailureThreshold:    5,
						InitialDelaySeconds: 10,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      1,
					},
				},
			}
			ds.Spec.Template.Spec.Tolerations = []corev1.Toleration{
				{
					Effect:   corev1.TaintEffectNoSchedule,
					Key:      "node-role.kubernetes.io/master",
					Operator: corev1.TolerationOpExists,
				},
			}
			hostPathUnset := corev1.HostPathUnset
			ds.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: configVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.PromtailSecretName,
						},
					},
				},
				{
					Name: certificatesVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  resources.PromtailCertificatesSecretName,
							DefaultMode: pointer.Int32Ptr(0400),
						},
					},
				},
				{
					Name: runVolumeName,
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Type: &hostPathUnset,
							Path: runVolumeMountPath,
						},
					},
				},
				{
					Name: containerVolumeName,
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Type: &hostPathUnset,
							Path: containerVolumeMountPath,
						},
					},
				},
				{
					Name: podVolumeName,
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Type: &hostPathUnset,
							Path: podVolumeMountPath,
						},
					},
				},
			}
			return ds, nil
		}
	}
}
