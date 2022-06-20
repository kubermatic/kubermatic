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
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/vpnsidecar"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	HetznerCCMDeploymentName = "hcloud-cloud-controller-manager"
	hetznerCCMVersion        = "v1.12.1"
)

var (
	hetznerResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("50Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
	}
)

func hetznerDeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return HetznerCCMDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Labels = resources.BaseAppLabels(HetznerCCMDeploymentName, nil)

			dep.Spec.Replicas = resources.Int32(1)

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(HetznerCCMDeploymentName, nil),
			}

			podLabels, err := data.GetPodTemplateLabels(HetznerCCMDeploymentName, dep.Spec.Template.Spec.Volumes, nil)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
			}

			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err =
				resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			f := false
			dep.Spec.Template.Spec.AutomountServiceAccountToken = &f

			credentials, err := resources.GetCredentials(data)
			if err != nil {
				return nil, fmt.Errorf("failed to get credentials: %w", err)
			}

			network := data.Cluster().Spec.Cloud.Hetzner.Network
			if network == "" {
				network = data.DC().Spec.Hetzner.Network
			}

			dep.Spec.Template.Spec.Volumes = getVolumes(data.IsKonnectivityEnabled())

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:  ccmContainerName,
					Image: data.ImageRegistry(resources.RegistryDocker) + "/hetznercloud/hcloud-cloud-controller-manager:" + hetznerCCMVersion,
					Command: []string{
						"/bin/hcloud-cloud-controller-manager",
						"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
						"--cloud-provider=hcloud",
						"--allow-untagged-cloud",
						// "false" as we use IPAM in kube-controller-manager
						"--allocate-node-cidrs=false",
					},
					Env: []corev1.EnvVar{
						{
							Name: "NODE_NAME",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "spec.nodeName",
								},
							},
						},
						{
							Name:  "HCLOUD_TOKEN",
							Value: credentials.Hetzner.Token,
						},
						{
							Name:  "HCLOUD_NETWORK",
							Value: network,
						},
						{
							// Required since Hetzner CCM v1.11.0.
							// By default, the Hetzner CCM tries to validate is the control plane node
							// attached to the configured Hetzner network. This is causing the Hetzner
							// CCM to crashloopbackoff since the control plane is running on the seed
							// cluster, which might not be a Hetzner cluster.
							// https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/354f8f85714a934ecc9781747a20d13034165c90
							Name:  "HCLOUD_NETWORK_DISABLE_ATTACHED_CHECK",
							Value: "true",
						},
					},
					VolumeMounts: getVolumeMounts(),
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

			if !data.IsKonnectivityEnabled() {
				openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, openvpnClientContainerName)
				if err != nil {
					return nil, fmt.Errorf("failed to get openvpn sidecar: %w", err)
				}
				dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, *openvpnSidecar)
				defResourceRequirements[openvpnSidecar.Name] = openvpnSidecar.Resources.DeepCopy()
			}

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			return dep, nil
		}
	}
}
