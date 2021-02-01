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

package multus

import (
	"fmt"
	"net"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	utilpointer "k8s.io/utils/pointer"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	multusImageName = "nfvpe/multus"
	//TODO(youssefazrak) tbc
	multusImageTag = "stable"
)

// DaemonSetCreator returns the function to create and update the Envoy DaemonSet
func DaemonSetCreator(agentIP net.IP, versions kubermatic.Versions) reconciling.NamedDaemonSetCreatorGetter {
	return func() (string, reconciling.DaemonSetCreator) {
		return resources.MultusDaemonSetName, func(ds *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
			// (youssefazrak) Hardcoding amd64. Other architecture like ARM
			// requires different changes.
			ds.Name = "kube-multus-ds-amd64"
			ds.Namespace = metav1.NamespaceSystem

			labels := map[string]string{"tier": "node", "name": "multus"}
			ds.Labels = resources.BaseAppLabels(resources.MultusDaemonSetName, labels)

			ds.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{"name": "multus"},
			}

			ds.Spec.UpdateStrategy = appsv1.DaemonSetUpdateStrategy{
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
			}

			ds.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: resources.BaseAppLabels(resources.MultusDaemonSetName, labels),
			}

			ds.Spec.Template.Spec.HostNetwork = true

			ds.Spec.Template.Spec.NodeSelector = map[string]string{"kubernetes.io/arch": "amd64"}

			ds.Spec.Template.Spec.Tolerations = []corev1.Toleration{
				{
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				},
			}

			ds.Spec.Template.Spec.ServiceAccountName = resources.MultusServiceAccountName
			ds.Spec.Template.Spec.Containers = getContainers()

			ds.Spec.Template.Spec.TerminationGracePeriodSeconds = utilpointer.Int64Ptr(10)

			volumes := getVolumes()
			ds.Spec.Template.Spec.Volumes = volumes

			return ds, nil
		}
	}
}

func getContainers() []corev1.Container {
	return []corev1.Container{
		{
			Name:            "kube-multus",
			Image:           fmt.Sprintf("%s/%s:%s", resources.RegistryDocker, multusImageName, multusImageTag),
			ImagePullPolicy: corev1.PullIfNotPresent,

			//TODO(youssefazrak) tbc
			Args: []string{"--multus-conf-file", "auto", "--cni-version", "0.3.1"},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("50Mi"),
					corev1.ResourceCPU:    resource.MustParse("100m"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("50Mi"),
					corev1.ResourceCPU:    resource.MustParse("100m"),
				},
			},

			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "cni",
					MountPath: "/host/etc/cni/net.d",
				},
				{
					Name:      "cnibin",
					MountPath: "/host/opt/cni/bin",
				},
				{
					Name:      "multus-cfg",
					MountPath: "/tmp/multus-conf",
				},
			},
			SecurityContext: &corev1.SecurityContext{
				Privileged: utilpointer.BoolPtr(true),
			},
		},
	}
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "cni",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/cni/net.d",
				},
			},
		},
		{
			Name: "cnibin",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/opt/cni/bin",
				},
			},
		},
		{
			Name: "multus-cfg",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: utilpointer.Int32Ptr(corev1.ConfigMapVolumeSourceDefaultMode),
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "multus-cni-config",
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "cni-conf.json",
							Path: "70-multus.conf",
						},
					},
				},
			},
		},
	}
}
