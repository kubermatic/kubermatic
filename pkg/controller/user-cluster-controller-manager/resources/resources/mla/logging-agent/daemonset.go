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

package loggingagent

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	imageName     = "grafana/agent"
	imageTag      = "v0.29.0"
	appName       = "mla-logging-agent"
	containerName = "grafana-agent"

	reloaderImageName = "prometheus-operator/prometheus-config-reloader"
	reloaderTag       = "v0.60.1"

	configVolumeName         = "config"
	configVolumeMountPath    = "/etc/agent"
	configFileName           = "agent.yaml"
	certificatesVolumeName   = "certificates"
	runVolumeName            = "run"
	runVolumeMountPath       = "/run/grafana-agent"
	containerVolumeName      = "containers"
	containerVolumeMountPath = "/var/lib/docker/containers"
	podVolumeName            = "pods"
	podVolumeMountPath       = "/var/log/pods"
	metricsPortName          = "http-metrics"
	containerPort            = 3101
)

var (
	controllerLabels = map[string]string{
		common.NameLabel:      resources.MLALoggingAgentDaemonSetName,
		common.InstanceLabel:  resources.MLALoggingAgentDaemonSetName,
		common.ComponentLabel: resources.MLAComponentName,
	}
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("64Mi"),
			corev1.ResourceCPU:    resource.MustParse("50m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("128Mi"),
			corev1.ResourceCPU:    resource.MustParse("200m"),
		},
	}
)

func DaemonSetReconciler(overrides *corev1.ResourceRequirements, imageRewriter registry.ImageRewriter) reconciling.NamedDaemonSetReconcilerFactory {
	return func() (string, reconciling.DaemonSetReconciler) {
		return resources.MLALoggingAgentDaemonSetName, func(ds *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
			ds.Labels = resources.BaseAppLabels(appName, nil)

			ds.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: controllerLabels,
			}

			ds.Spec.Template.Labels = controllerLabels
			ds.Spec.Template.Spec.ServiceAccountName = resources.MLALoggingAgentServiceAccountName
			ds.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				RunAsUser:  ptr.To[int64](0),
				RunAsGroup: ptr.To[int64](0),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}
			ds.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            containerName,
					Image:           registry.Must(imageRewriter(fmt.Sprintf("%s:%s", imageName, imageTag))),
					ImagePullPolicy: corev1.PullAlways,
					Args: []string{
						fmt.Sprintf("-config.file=%s/%s", configVolumeMountPath, configFileName),
						fmt.Sprintf("-server.http.address=0.0.0.0:%d", containerPort),
						"-disable-reporting",
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      configVolumeName,
							MountPath: configVolumeMountPath,
						},
						{
							Name:      certificatesVolumeName,
							MountPath: resources.MLALoggingAgentClientCertMountPath,
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
							ContainerPort: containerPort,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: ptr.To(false),
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{
								"all",
							},
						},
						ReadOnlyRootFilesystem: ptr.To(true),
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/-/ready",
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
				{
					Name:            "prometheus-config-reloader",
					Image:           registry.Must(imageRewriter(fmt.Sprintf("%s/%s:%s", resources.RegistryQuay, reloaderImageName, reloaderTag))),
					ImagePullPolicy: corev1.PullAlways,
					Args: []string{
						// Full usage of prometheus-config-reloader:
						// https://github.com/prometheus-operator/prometheus-operator/blob/v0.49.0/cmd/prometheus-config-reloader/main.go#L72-L108
						"--listen-address=:8080",
						"--watch-interval=10s",
						fmt.Sprintf("--config-file=%s/%s", configVolumeMountPath, configFileName),
						fmt.Sprintf("--reload-url=http://localhost:%d/-/reload", containerPort),
					},
					Env: []corev1.EnvVar{
						{
							Name: "POD_NAME",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.name",
								},
							},
						},
						{
							Name: "HOSTNAME",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "spec.nodeName",
								},
							},
						},
						{
							Name:  "SHARD",
							Value: "0",
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      configVolumeName,
							MountPath: configVolumeMountPath,
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("10Mi"),
							corev1.ResourceCPU:    resource.MustParse("10m"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("100Mi"),
							corev1.ResourceCPU:    resource.MustParse("100m"),
						},
					},
				},
			}
			ds.Spec.Template.Spec.Tolerations = []corev1.Toleration{
				{
					Effect:   corev1.TaintEffectNoSchedule,
					Key:      "node-role.kubernetes.io/master",
					Operator: corev1.TolerationOpExists,
				},
				{
					Effect:   corev1.TaintEffectNoSchedule,
					Key:      "node-role.kubernetes.io/control-plane",
					Operator: corev1.TolerationOpExists,
				},
				{
					Effect:   corev1.TaintEffectNoSchedule,
					Operator: corev1.TolerationOpExists,
				},
				{
					Effect:   corev1.TaintEffectNoExecute,
					Operator: corev1.TolerationOpExists,
				},
			}
			hostPathUnset := corev1.HostPathUnset
			ds.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: configVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.MLALoggingAgentSecretName,
						},
					},
				},
				{
					Name: certificatesVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  resources.MLALoggingAgentCertificatesSecretName,
							DefaultMode: ptr.To[int32](0400),
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

			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				containerName: defaultResourceRequirements.DeepCopy(),
			}
			var overridesRequirements map[string]*corev1.ResourceRequirements
			if overrides != nil {
				overridesRequirements = map[string]*corev1.ResourceRequirements{
					containerName: overrides.DeepCopy(),
				}
			}
			if err := resources.SetResourceRequirements(ds.Spec.Template.Spec.Containers, defResourceRequirements, overridesRequirements, ds.Annotations); err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}
			return ds, nil
		}
	}
}
