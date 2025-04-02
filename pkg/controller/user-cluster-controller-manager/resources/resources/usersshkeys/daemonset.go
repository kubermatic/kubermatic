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

package usersshkeys

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	daemonSetName = "user-ssh-keys-agent"
	dockerImage   = "kubermatic/user-ssh-keys-agent"
)

var (
	daemonSetMaxUnavailable = intstr.FromInt(1)
	hostPathType            = corev1.HostPathDirectoryOrCreate
)

func DaemonSetReconciler(versions kubermatic.Versions, imageRewriter registry.ImageRewriter) reconciling.NamedDaemonSetReconcilerFactory {
	return func() (string, reconciling.DaemonSetReconciler) {
		return daemonSetName, func(ds *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
			ds.Spec.UpdateStrategy.Type = appsv1.RollingUpdateDaemonSetStrategyType

			// be careful to not override any defaulting a k8s 1.21 with feature gate
			// DaemonSetUpdateSurge might perform on the .MaxSurge field
			if ds.Spec.UpdateStrategy.RollingUpdate == nil {
				ds.Spec.UpdateStrategy.RollingUpdate = &appsv1.RollingUpdateDaemonSet{}
			}
			ds.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable = &daemonSetMaxUnavailable

			labels := map[string]string{"app": "user-ssh-keys-agent"}
			ds.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
			ds.Spec.Template.Labels = labels

			ds.Spec.Template.Spec.ServiceAccountName = serviceAccountName

			ds.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            daemonSetName,
					ImagePullPolicy: corev1.PullAlways,
					Image:           registry.Must(imageRewriter(fmt.Sprintf("%s/%s:%s", resources.RegistryQuay, dockerImage, versions.KubermaticContainerTag))),
					Command:         []string{fmt.Sprintf("/usr/local/bin/%v", daemonSetName)},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "root",
							MountPath: "/root",
						},
						{
							Name:      "home",
							MountPath: "/home",
						},
					},
				},
			}

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

			ds.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "root",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/root",
							Type: &hostPathType,
						},
					},
				},
				{
					Name: "home",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/home",
							Type: &hostPathType,
						},
					},
				},
			}

			ds.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}

			return ds, nil
		}
	}
}
