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
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

func uiPodLabels() map[string]string {
	return map[string]string{
		common.NameLabel: uiDeploymentName,
	}
}

func UIDeploymentCreator(cfg *operatorv1alpha1.KubermaticConfiguration, versions kubermatic.Versions) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return uiDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			d.Spec.Replicas = cfg.Spec.UI.Replicas
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: uiPodLabels(),
			}

			d.Spec.Template.Labels = d.Spec.Selector.MatchLabels

			d.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				RunAsNonRoot: pointer.BoolPtr(true),
				RunAsUser:    pointer.Int64Ptr(65534),
			}

			tag := versions.UI
			if cfg.Spec.UI.DockerTag != "" {
				tag = cfg.Spec.UI.DockerTag
			}

			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:  "webserver",
					Image: cfg.Spec.UI.DockerRepository + ":" + tag,
					Env:   common.ProxyEnvironmentVars(cfg),
					Ports: []corev1.ContainerPort{
						{
							Name:          "http",
							ContainerPort: 8080,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							MountPath: "/dist/config/",
							Name:      "config",
							ReadOnly:  true,
						},
					},
					Resources: cfg.Spec.UI.Resources,
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

			return d, nil
		}
	}
}

func UIPDBCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedPodDisruptionBudgetCreatorGetter {
	return func() (string, reconciling.PodDisruptionBudgetCreator) {
		return "kubermatic-dashboard", func(pdb *policyv1beta1.PodDisruptionBudget) (*policyv1beta1.PodDisruptionBudget, error) {
			min := intstr.FromInt(1)

			pdb.Spec.MinAvailable = &min
			pdb.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: uiPodLabels(),
			}

			return pdb, nil
		}
	}
}

func UIServiceCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
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
