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

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

func DaemonSetCreator() reconciling.NamedDaemonSetCreatorGetter {
	return func() (string, reconciling.DaemonSetCreator) {
		return resources.NodeLocalDNSDaemonSetName, func(ds *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
			sptr := intstr.FromString("10%")
			ds.Spec.UpdateStrategy = appsv1.DaemonSetUpdateStrategy{
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: &sptr,
				},
			}
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
			if ds.Spec.Template.ObjectMeta.Labels == nil {
				ds.Spec.Template.ObjectMeta.Labels = labels
			}

			ds.Spec.Template.Spec.ServiceAccountName = resources.NodeLocalDNSServiceAccountName
			ds.Spec.Template.Spec.PriorityClassName = "system-cluster-critical"
			ds.Spec.Template.Spec.HostNetwork = true
			ds.Spec.Template.Spec.DNSPolicy = corev1.DNSDefault
			ds.Spec.Template.Spec.TerminationGracePeriodSeconds = pointer.Int64Ptr(0)

			patchTolerations(&ds.Spec.Template.Spec)

			hostPathType := corev1.HostPathFileOrCreate
			ds.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            "node-cache",
					ImagePullPolicy: corev1.PullAlways,
					Image:           fmt.Sprintf("%s/k8s-dns-node-cache:1.15.7", resources.RegistryK8SGCR),
					Args: []string{
						"-localip",
						"169.254.20.10",
						"-conf",
						"/etc/coredns/Corefile",
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
							ContainerPort: 9153,
							HostPort:      9153,
							Name:          "metrics",
							Protocol:      corev1.ProtocolTCP,
						},
					},

					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Host:   "169.254.20.10",
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
						Privileged: pointer.BoolPtr(true),
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
									Path: "Corefile",
								},
							},
						},
					},
				},
			}

			return ds, nil
		}
	}
}

// patchTolerations ensures that a toleration for the CriticalAddonsOnly taint
// exists, but pays attention to not overwrite any potential changes the user
// has made.
func patchTolerations(podSpec *corev1.PodSpec) {
	exists := false
	defaultTolerationKey := "CriticalAddonsOnly"

	for _, toleration := range podSpec.Tolerations {
		if toleration.Key == defaultTolerationKey {
			exists = true
			break
		}
	}

	if !exists {
		podSpec.Tolerations = append(podSpec.Tolerations, corev1.Toleration{
			Key:      defaultTolerationKey,
			Operator: corev1.TolerationOpExists,
		})
	}
}
