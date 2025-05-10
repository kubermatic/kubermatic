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

package cloudcontroller

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

const (
	HetznerCCMDeploymentName = "hcloud-cloud-controller-manager"
	hetznerCCMVersion        = "v1.24.0" // https://github.com/hetznercloud/hcloud-cloud-controller-manager#versioning-policy
)

var (
	hetznerResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("50Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
	}
)

func hetznerDeploymentReconciler(data *resources.TemplateData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return HetznerCCMDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Spec.Replicas = resources.Int32(1)

			var err error
			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err =
				resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			network := data.Cluster().Spec.Cloud.Hetzner.Network
			if network == "" {
				network = data.DC().Spec.Hetzner.Network
			}

			dep.Spec.Template.Spec.AutomountServiceAccountToken = ptr.To(false)
			dep.Spec.Template.Spec.Volumes = getVolumes(data.IsKonnectivityEnabled(), false)
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:  ccmContainerName,
					Image: registry.Must(data.RewriteImage(resources.RegistryDocker + "/hetznercloud/hcloud-cloud-controller-manager:" + hetznerCCMVersion)),
					Command: []string{
						"/bin/hcloud-cloud-controller-manager",
						"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
						"--cloud-provider=hcloud",
						"--allow-untagged-cloud",
						// "false" as we use IPAM in kube-controller-manager
						"--allocate-node-cidrs=false",
						"--configure-cloud-routes=false",
					},
					Env: append(
						getEnvVars(),
						corev1.EnvVar{
							Name: "HCLOUD_TOKEN",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: resources.ClusterCloudCredentialsSecretName,
									},
									Key: resources.HetznerToken,
								},
							},
						},
						corev1.EnvVar{
							Name:  "HCLOUD_NETWORK",
							Value: network,
						},
						corev1.EnvVar{
							// Required since Hetzner CCM v1.11.0.
							// By default, the Hetzner CCM tries to validate is the control plane node
							// attached to the configured Hetzner network. This is causing the Hetzner
							// CCM to crashloopbackoff since the control plane is running on the seed
							// cluster, which might not be a Hetzner cluster.
							// https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/354f8f85714a934ecc9781747a20d13034165c90
							Name:  "HCLOUD_NETWORK_DISABLE_ATTACHED_CHECK",
							Value: "true",
						},
					),
					VolumeMounts: getVolumeMounts(false),
				},
			}

			if data.Cluster().IsDualStack() {
				dep.Spec.Template.Spec.Containers[0].Env = append(dep.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
					Name:  "HCLOUD_INSTANCES_ADDRESS_FAMILY",
					Value: "dualstack",
				})
			}

			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				ccmContainerName: hetznerResourceRequirements.DeepCopy(),
			}

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			return dep, nil
		}
	}
}
