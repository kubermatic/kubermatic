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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/vpnsidecar"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	KubeVirtCCMDeploymentName = "kubevirt-cloud-controller-manager"
	KubeVirtCCMTag            = "v0.0.8"
)

var (
	kvResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("100Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("500m"),
		},
	}
)

func kubevirtDeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return KubeVirtCCMDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = KubeVirtCCMDeploymentName
			dep.Labels = resources.BaseAppLabels(KubeVirtCCMDeploymentName, nil)

			dep.Spec.Replicas = resources.Int32(1)

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(KubeVirtCCMDeploymentName, nil),
			}

			podLabels, err := data.GetPodTemplateLabels(KubeVirtCCMDeploymentName, dep.Spec.Template.Spec.Volumes, nil)
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

			dep.Spec.Template.Spec.AutomountServiceAccountToken = pointer.BoolPtr(false)

			dep.Spec.Template.Spec.Volumes = append(getVolumes(data.IsKonnectivityEnabled()), corev1.Volume{
				Name: resources.CloudConfigConfigMapName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: resources.CloudConfigConfigMapName,
						},
					},
				},
			})

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    ccmContainerName,
					Image:   data.ImageRegistry(resources.RegistryQuay) + "/kubermatic/kubevirt-cloud-controller-manager:" + KubeVirtCCMTag,
					Command: []string{"/bin/kubevirt-cloud-controller-manager"},
					Args:    getKVFlags(data),
					VolumeMounts: append(getVolumeMounts(), corev1.VolumeMount{
						Name:      resources.CloudConfigConfigMapName,
						MountPath: "/etc/kubernetes/cloud",
						ReadOnly:  true,
					}),
				},
			}

			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				ccmContainerName: kvResourceRequirements.DeepCopy(),
			}

			if !data.IsKonnectivityEnabled() {
				openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, openvpnClientContainerName)
				if err != nil {
					return nil, fmt.Errorf("failed to get openvpn sidecar: %v", err)
				}
				dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, *openvpnSidecar)
				defResourceRequirements[openvpnSidecar.Name] = openvpnSidecar.Resources.DeepCopy()
			}

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}
			return dep, nil
		}
	}
}

func getKVFlags(data *resources.TemplateData) []string {
	flags := []string{
		"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
		"--cloud-config=/etc/kubernetes/cloud/config",
		"--cloud-provider=kubevirt",
	}
	if data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureCCMClusterName] {
		flags = append(flags, "--cluster-name", data.Cluster().Name)
	}
	return flags
}
