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

package konnectivity

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// DeploymentCreator returns the function to create and update konnectivity agent deployment in seed-cluster.
func DeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		const (
			name    = "k8s-artifacts-prod/kas-network-proxy/proxy-agent"
			version = "v0.0.24"
		)

		return resources.KonnectivityDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			labels := map[string]string{"app": resources.KonnectivityDeploymentName}
			dep.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
			dep.Spec.Template.ObjectMeta.Labels = labels
			dep.Spec.Replicas = intPtr(1)

			metricServerAddr := fmt.Sprintf("metrics-server.cluster-%s.svc.cluster.local", data.Cluster().Name)

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            resources.KonnectivityAgentContainer,
					Image:           fmt.Sprintf("%s/%s:%s", data.ImageRegistry(resources.RegistryUSGCR), name, version),
					ImagePullPolicy: corev1.PullAlways,
					Command:         []string{"/proxy-agent"},
					Args: []string{
						"--logtostderr=true",
						"-v=100",
						fmt.Sprintf("--agent-id=metrics-server.cluster-%s.svc.cluster.local", data.Cluster().Name),
						fmt.Sprintf("--agent-identifiers=host=%s&host=%s:443", metricServerAddr, metricServerAddr),
						fmt.Sprintf("--ca-cert=/var/run/secrets/certs/%s", resources.KonnectivityStolenAgentTokenNameCert),
						fmt.Sprintf("--proxy-server-host=konnectivity-server.%s", data.Cluster().Address.ExternalName),
						"--proxy-server-port=6443",
						"--admin-server-port=8133",
						"--health-server-port=8134",
						fmt.Sprintf("--service-account-token-path=/var/run/secrets/tokens/%s", resources.KonnectivityStolenAgentTokenNameToken),
					},
					Resources: corev1.ResourceRequirements{},
					VolumeMounts: []corev1.VolumeMount{
						{
							MountPath: "/var/run/secrets/certs/",
							Name:      "konnectivity-agent-ca",
						},
						{
							MountPath: "/var/run/secrets/tokens",
							Name:      "konnectivity-agent-ca",
						},
					},
					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/healthz",
								Port: intstr.IntOrString{
									Type:   intstr.Int,
									IntVal: 8134,
								},
							},
						},
						InitialDelaySeconds: 15,
						TimeoutSeconds:      15,
					},
				},
			}

			dep.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "konnectivity-agent-ca",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  resources.KonnectivityStolenAgentTokenSecretName,
							DefaultMode: intPtr(420),
						},
					},
				},
			}

			return dep, nil
		}
	}
}

func intPtr(i int32) *int32 {
	return &i
}
