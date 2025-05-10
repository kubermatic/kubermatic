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
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

// Anexia CCM is sourced from
// https://github.com/anexia-it/k8s-anexia-ccm/releases

const (
	AnexiaCCMDeploymentName = "anx-cloud-controller-manager"
	anexiaCCMVersion        = "1.5.9"
)

func anexiaDeploymentReconciler(data *resources.TemplateData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (name string, create reconciling.DeploymentReconciler) {
		return AnexiaCCMDeploymentName, func(deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
			deployment.Spec.Replicas = resources.Int32(1)

			deployment.Spec.Template.Spec.AutomountServiceAccountToken = ptr.To(false)
			deployment.Spec.Template.Spec.Volumes = getVolumes(data.IsKonnectivityEnabled(), true)
			deployment.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:  ccmContainerName,
					Image: registry.Must(data.RewriteImage("anx-cr.io/anexia/anx-cloud-controller-manager:" + anexiaCCMVersion)),
					Command: []string{
						"/app/ccm",
						"--cloud-provider=anexia",
						"--cloud-config=/etc/kubernetes/cloud/config",
						fmt.Sprintf("--cluster-name=%s", data.Cluster().Spec.HumanReadableName),
						"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
					},
					Env: append(
						getEnvVars(),
						corev1.EnvVar{
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
						corev1.EnvVar{
							Name:  "ANEXIA_AUTO_DISCOVER_LOAD_BALANCER",
							Value: "true",
						},
					),
					VolumeMounts: getVolumeMounts(true),
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
				},
			}

			return deployment, nil
		}
	}
}
