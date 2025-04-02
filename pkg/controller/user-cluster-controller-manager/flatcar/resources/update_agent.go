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

package resources

import (
	nodelabelerapi "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/node-labeler/api"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	AgentDaemonSetName = "flatcar-linux-update-agent"
)

var (
	daemonSetMaxUnavailable = intstr.FromInt(1)
	hostPathType            = corev1.HostPathUnset
)

func AgentDaemonSetReconciler(imageRewriter registry.ImageRewriter) reconciling.NamedDaemonSetReconcilerFactory {
	var rootUser int64 = 0

	return func() (string, reconciling.DaemonSetReconciler) {
		return AgentDaemonSetName, func(ds *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
			ds.Spec.UpdateStrategy.Type = appsv1.RollingUpdateDaemonSetStrategyType

			// be careful to not override any defaulting a k8s 1.21 with feature gate
			// DaemonSetUpdateSurge might perform on the .MaxSurge field
			if ds.Spec.UpdateStrategy.RollingUpdate == nil {
				ds.Spec.UpdateStrategy.RollingUpdate = &appsv1.RollingUpdateDaemonSet{}
			}
			ds.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable = &daemonSetMaxUnavailable

			// We broke compatibility with upstream in #5875 and instead of performing a migration,
			// we simply keep the changed labels.
			labels := map[string]string{"app.kubernetes.io/name": AgentDaemonSetName}

			ds.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
			ds.Spec.Template.Labels = labels

			// The agent should only run on Flatcar nodes
			ds.Spec.Template.Spec.NodeSelector = map[string]string{nodelabelerapi.DistributionLabelKey: nodelabelerapi.FlatcarLabelValue}

			ds.Spec.Template.Spec.ServiceAccountName = agentServiceAccountName

			ds.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "update-agent",
					Image:   operatorImage(imageRewriter),
					Command: []string{"/bin/update-agent"},
					SecurityContext: &corev1.SecurityContext{
						RunAsUser: &rootUser,
					},
					Env: []corev1.EnvVar{
						{
							Name: "UPDATE_AGENT_NODE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									APIVersion: "v1",
									FieldPath:  "spec.nodeName",
								},
							},
						},
						{
							Name: "POD_NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									APIVersion: "v1",
									FieldPath:  "metadata.namespace",
								},
							},
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "var-run-dbus",
							MountPath: "/var/run/dbus",
						},
						{
							Name:      "etc-flatcar",
							MountPath: "/etc/flatcar",
						},
						{
							Name:      "usr-share-flatcar",
							MountPath: "/usr/share/flatcar",
						},
						{
							Name:      "etc-os-release",
							MountPath: "/etc/os-release",
						},
					},
				},
			}

			// This does not match upstream because in KKP, user clusters have no control-plane nodes.
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
					Name: "var-run-dbus",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/var/run/dbus",
							Type: &hostPathType,
						},
					},
				},
				{
					Name: "etc-flatcar",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/etc/flatcar",
							Type: &hostPathType,
						},
					},
				},
				{
					Name: "usr-share-flatcar",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/usr/share/flatcar",
							Type: &hostPathType,
						},
					},
				},
				{
					Name: "etc-os-release",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/etc/os-release",
							Type: &hostPathType,
						},
					},
				},
			}

			return ds, nil
		}
	}
}
