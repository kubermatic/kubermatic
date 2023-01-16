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

package cloudcontroller

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/vpnsidecar"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const AnexiaCCMDeploymentName = "anx-cloud-controller-manager"

func anexiaDeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	return func() (name string, create reconciling.DeploymentCreator) {
		return AnexiaCCMDeploymentName, func(deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
			deployment.Labels = resources.BaseAppLabels(AnexiaCCMDeploymentName, nil)
			deployment.Spec.Replicas = resources.Int32(1)

			deployment.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(AnexiaCCMDeploymentName, nil),
			}

			podLabels, err := data.GetPodTemplateLabels(AnexiaCCMDeploymentName, deployment.Spec.Template.Spec.Volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("unable to get pod template labels: %w", err)
			}

			deployment.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
			}

			f := false
			deployment.Spec.Template.Spec.AutomountServiceAccountToken = &f

			deployment.Spec.Template.Spec.Volumes = append(getVolumes(data.IsKonnectivityEnabled()),
				corev1.Volume{
					Name: resources.CloudConfigConfigMapName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: resources.CloudConfigConfigMapName,
							},
						},
					},
				})

			deployment.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:  ccmContainerName,
					Image: data.ImageRegistry(resources.RegistryAnexia) + "/anexia/anx-cloud-controller-manager:1.5.1",
					Command: []string{
						"/app/ccm",
						"--cloud-provider=anexia",
						"--cloud-config=/etc/kubernetes/cloud/config",
						fmt.Sprintf("--cluster-name=%s", data.Cluster().Spec.HumanReadableName),
						"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
					},
					Env: []corev1.EnvVar{
						{
							Name: "ANEXIA_TOKEN",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: resources.ClusterCloudCredentialsSecretName,
									},
									Key: resources.AnexiaToken,
								},
							},
						},
						{
							Name:  "ANEXIA_AUTO_DISCOVER_LOAD_BALANCER",
							Value: "true",
						},
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "http",
							ContainerPort: 8080,
							Protocol:      "TCP",
						},
					},
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Port:   intstr.FromString("http"),
								Scheme: "HTTPS",
							},
						},
						InitialDelaySeconds: 5,
						TimeoutSeconds:      10,
						PeriodSeconds:       20,
						SuccessThreshold:    1,
						FailureThreshold:    3,
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Port:   intstr.FromString("http"),
								Scheme: "HTTPS",
							},
						},
						InitialDelaySeconds: 5,
						TimeoutSeconds:      10,
						PeriodSeconds:       20,
						SuccessThreshold:    1,
						FailureThreshold:    3,
					},
					VolumeMounts: append(getVolumeMounts(), corev1.VolumeMount{
						Name:      resources.CloudConfigConfigMapName,
						MountPath: "/etc/kubernetes/cloud",
						ReadOnly:  true,
					}),
				},
			}

			if !data.IsKonnectivityEnabled() {
				openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, openvpnClientContainerName)
				if err != nil {
					return nil, fmt.Errorf("failed to get openvpn sidecar: %w", err)
				}
				deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, *openvpnSidecar)
			}

			if err != nil {
				return nil, err
			}
			return deployment, nil
		}
	}
}
