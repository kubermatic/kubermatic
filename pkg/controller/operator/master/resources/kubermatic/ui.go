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

package kubermatic

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func uiPodLabels() map[string]string {
	return map[string]string{
		common.NameLabel: UIDeploymentName,
	}
}

func UIDeploymentReconciler(cfg *kubermaticv1.KubermaticConfiguration, versions kubermatic.Versions) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return UIDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			d.Spec.Replicas = cfg.Spec.UI.Replicas
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: uiPodLabels(),
			}

			kubernetes.EnsureLabels(&d.Spec.Template, d.Spec.Selector.MatchLabels)

			d.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				RunAsNonRoot: ptr.To(true),
				RunAsUser:    ptr.To[int64](65534),
			}

			tag := versions.UIContainerTag
			if cfg.Spec.UI.DockerTag != "" {
				tag = cfg.Spec.UI.DockerTag
			} else if cfg.Spec.UI.DockerTagSuffix != "" {
				tag = fmt.Sprintf("%s-%s", tag, cfg.Spec.UI.DockerTagSuffix)
			}

			volumeMounts := []corev1.VolumeMount{
				{
					MountPath: "/dist/config/",
					Name:      "config",
					ReadOnly:  true,
				},
			}

			if cfg.Spec.UI.ExtraVolumeMounts != nil {
				volumeMounts = append(volumeMounts, cfg.Spec.UI.ExtraVolumeMounts...)
			}

			d.Spec.Template.Spec.SecurityContext = &common.PodSecurityContext

			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "webserver",
					Image:   cfg.Spec.UI.DockerRepository + ":" + tag,
					Command: []string{"dashboard"},
					Env:     common.KubermaticProxyEnvironmentVars(&cfg.Spec.Proxy),
					Ports: []corev1.ContainerPort{
						{
							Name:          "http",
							ContainerPort: 8080,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts:    volumeMounts,
					Resources:       cfg.Spec.UI.Resources,
					SecurityContext: &common.ContainerSecurityContext,
				},
			}

			d.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: uiConfigConfigMapName},
						},
					},
				},
			}

			if cfg.Spec.UI.ExtraVolumes != nil {
				d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, cfg.Spec.UI.ExtraVolumes...)
			}

			return d, nil
		}
	}
}

func UIPDBReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedPodDisruptionBudgetReconcilerFactory {
	return func() (string, reconciling.PodDisruptionBudgetReconciler) {
		return "kubermatic-dashboard", func(pdb *policyv1.PodDisruptionBudget) (*policyv1.PodDisruptionBudget, error) {
			// To prevent the PDB from blocking node rotations, we accept
			// 0 minAvailable if the replica count is only 1.
			// NB: The cfg is defaulted, so Replicas==nil cannot happen.
			minReplicas := intstr.FromInt(1)
			if cfg.Spec.UI.Replicas != nil && *cfg.Spec.UI.Replicas < 2 {
				minReplicas = intstr.FromInt(0)
			}

			pdb.Spec.MinAvailable = &minReplicas
			pdb.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: uiPodLabels(),
			}

			return pdb, nil
		}
	}
}

func UIServiceReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return uiServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Spec.Type = corev1.ServiceTypeNodePort
			s.Spec.Selector = uiPodLabels()

			if len(s.Spec.Ports) < 1 {
				s.Spec.Ports = make([]corev1.ServicePort, 1)
			}

			s.Spec.Ports[0].Port = 80
			s.Spec.Ports[0].TargetPort = intstr.FromInt(8080)
			s.Spec.Ports[0].Protocol = corev1.ProtocolTCP

			return s, nil
		}
	}
}
