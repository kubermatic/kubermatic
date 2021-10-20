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

// DeploymentCreator returns function to create/update deployment for konnectivity agents in user cluster.
func DeploymentCreator(clusterHostname string, registryWithOverwrite func(string) string) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		const (
			name    = "k8s-artifacts-prod/kas-network-proxy/proxy-agent"
			version = "v0.0.24"
		)

		return resources.KonnectivityDeploymentName, func(ds *appsv1.Deployment) (*appsv1.Deployment, error) {
			labels := resources.BaseAppLabels(resources.KonnectivityDeploymentName, nil)
			ds.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
			ds.Spec.Template.ObjectMeta.Labels = labels
			replicas := int32(1)
			ds.Spec.Replicas = &replicas
			ds.Spec.Template.Spec.ServiceAccountName = resources.KonnectivityServiceAccountName
			ds.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            resources.KonnectivityAgentContainer,
					Image:           fmt.Sprintf("%s/%s:%s", registryWithOverwrite(resources.RegistryUSGCR), name, version),
					ImagePullPolicy: corev1.PullAlways,
					Command:         []string{"/proxy-agent"},
					Args: []string{
						"--logtostderr=true",
						"-v=100",
						"--agent-identifiers=default-route=true",
						"--ca-cert=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
						fmt.Sprintf("--proxy-server-host=konnectivity-server.%s", clusterHostname),
						"--proxy-server-port=6443",
						"--admin-server-port=8133",
						"--health-server-port=8134",
						fmt.Sprintf("--service-account-token-path=/var/run/secrets/tokens/%s", resources.KonnectivityAgentToken),
					},
					Resources: corev1.ResourceRequirements{},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.KonnectivityAgentToken,
							MountPath: "/var/run/secrets/tokens",
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
			ds.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: resources.KonnectivityAgentToken,
					VolumeSource: corev1.VolumeSource{
						Projected: &corev1.ProjectedVolumeSource{
							Sources: []corev1.VolumeProjection{
								{
									ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
										Audience: resources.KonnectivityClusterRoleBindingUsername, // TODO(pratik): what?
										Path:     resources.KonnectivityAgentToken,
									},
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
