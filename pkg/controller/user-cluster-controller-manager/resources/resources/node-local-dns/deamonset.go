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

package nodelocaldns

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/kubesystem"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	version = "1.25.0"
)

func DaemonSetReconciler(imageRewriter registry.ImageRewriter) reconciling.NamedDaemonSetReconcilerFactory {
	return func() (string, reconciling.DaemonSetReconciler) {
		return resources.NodeLocalDNSDaemonSetName, func(ds *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
			maxUnavailable := intstr.FromString("10%")

			ds.Spec.UpdateStrategy.Type = appsv1.RollingUpdateDaemonSetStrategyType

			// be careful to not override any defaulting a k8s 1.21 with feature gate
			// DaemonSetUpdateSurge might perform on the .MaxSurge field
			if ds.Spec.UpdateStrategy.RollingUpdate == nil {
				ds.Spec.UpdateStrategy.RollingUpdate = &appsv1.RollingUpdateDaemonSet{}
			}
			ds.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable = &maxUnavailable

			labels := resources.BaseAppLabels(resources.NodeLocalDNSDaemonSetName,
				map[string]string{"app.kubernetes.io/name": resources.NodeLocalDNSDaemonSetName})
			if ds.Labels == nil {
				ds.Labels = labels
			}
			ds.Labels[addonManagerModeKey] = reconcileModeValue
			ds.Labels["kubernetes.io/cluster-service"] = "true"

			if ds.Spec.Selector == nil {
				ds.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
			}
			if ds.Spec.Template.Labels == nil {
				ds.Spec.Template.Labels = labels
			}

			ds.Spec.Template.Spec.ServiceAccountName = resources.NodeLocalDNSServiceAccountName
			ds.Spec.Template.Spec.PriorityClassName = "system-cluster-critical"
			ds.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}
			ds.Spec.Template.Spec.HostNetwork = true
			ds.Spec.Template.Spec.DNSPolicy = corev1.DNSDefault
			ds.Spec.Template.Spec.TerminationGracePeriodSeconds = ptr.To[int64](0)
			ds.Spec.Template.Spec.Tolerations = []corev1.Toleration{
				{
					Effect:   corev1.TaintEffectNoSchedule,
					Operator: corev1.TolerationOpExists,
				},
				{
					Effect:   corev1.TaintEffectNoExecute,
					Operator: corev1.TolerationOpExists,
				},
			}

			hostPathType := corev1.HostPathFileOrCreate
			ds.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            "node-cache",
					Image:           registry.Must(imageRewriter(fmt.Sprintf("%s/dns/k8s-dns-node-cache:%s", resources.RegistryK8S, version))),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Args: []string{
						"-localip",
						kubesystem.NodeLocalDNSCacheAddress,
						"-conf",
						"/etc/Corefile",
					},

					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "xtables-lock",
							MountPath: "/run/xtables.lock",
						},
						{
							Name:      "config-volume",
							MountPath: "/etc/coredns",
						},
						{
							Name:      "kube-dns-config",
							MountPath: "/etc/kube-dns",
						},
					},

					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 53,
							HostPort:      53,
							Name:          "dns-tcp",
							Protocol:      corev1.ProtocolTCP,
						},
						{
							ContainerPort: 53,
							HostPort:      53,
							Name:          "dns",
							Protocol:      corev1.ProtocolUDP,
						},
						{
							ContainerPort: 9253,
							HostPort:      9253,
							Name:          "metrics",
							Protocol:      corev1.ProtocolTCP,
						},
					},

					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Host:   kubesystem.NodeLocalDNSCacheAddress,
								Scheme: corev1.URISchemeHTTP,
								Path:   "/health",
								Port:   intstr.FromInt(8080),
							},
						},
						InitialDelaySeconds: 60,
						TimeoutSeconds:      5,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						FailureThreshold:    3,
					},

					SecurityContext: &corev1.SecurityContext{
						Privileged: ptr.To(true),
					},
				},
			}

			ds.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "xtables-lock",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/run/xtables.lock",
							Type: &hostPathType,
						},
					},
				},
				{
					Name: "config-volume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: resources.NodeLocalDNSConfigMapName,
							},
							Items: []corev1.KeyToPath{
								{
									Key:  "Corefile",
									Path: "Corefile.base",
								},
							},
						},
					},
				},
				{
					Name: "kube-dns-config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "kube-dns",
							},
							Optional: ptr.To(true),
						},
					},
				},
			}

			return ds, nil
		}
	}
}
