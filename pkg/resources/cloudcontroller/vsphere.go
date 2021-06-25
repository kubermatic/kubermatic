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
	"k8c.io/kubermatic/v2/pkg/semver"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	vsphereCPIControllerManager = "vsphere-cloud-controller-manager"
)

var (
	vsphereCPIResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("100Mi"),
			corev1.ResourceCPU:    resource.MustParse("200m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("500m"),
		},
	}
)

func vsphereDeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return vsphereCPIControllerManager, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = vsphereCPIControllerManager
			dep.Labels = resources.BaseAppLabels(vsphereCPIControllerManager, nil)

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(vsphereCPIControllerManager, nil),
			}

			podLabels, err := data.GetPodTemplateLabels(vsphereCPIControllerManager, dep.Spec.Template.Spec.Volumes, map[string]string{
				"component": "cloud-controller-manager",
				"tier":      "control-plane",
			})
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
			dep.Spec.Template.Spec.HostNetwork = true

			version, err := getVsphereCPIVersion(data.Cluster().Spec.Version)
			if err != nil {
				return nil, err
			}
			openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, openvpnClientContainerName)
			if err != nil {
				return nil, fmt.Errorf("failed to get openvpn sidecar: %v", err)
			}
			containers, err := getCPIContainerList(version, data)
			if err != nil {
				return nil, err
			}
			dep.Spec.Template.Spec.Containers = append(containers, *openvpnSidecar)

			dep.Spec.Template.Spec.Volumes = append(getVolumes(), corev1.Volume{
				Name: resources.CloudConfigConfigMapName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: resources.CloudConfigConfigMapName,
						},
					},
				},
			})

			return dep, nil
		}
	}
}

func getCPIContainerList(version string, data *resources.TemplateData) ([]corev1.Container, error) {
	controllerManagerImage := fmt.Sprintf("%s/cloud-provider-vsphere/cpi/release/manager:v%s", data.ImageRegistry(resources.RegistryGCR), version)

	containerList := []corev1.Container{
		{
			Name:  vsphereCPIControllerManager,
			Image: controllerManagerImage,
			Args: []string{
				"--v=2",
				"--cloud-provider=vsphere",
				"--cloud-config=/etc/cloud/config",
				"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
			},
			VolumeMounts: append(getVolumeMounts(), corev1.VolumeMount{
				MountPath: "/etc/cloud",
				Name:      resources.CloudConfigConfigMapName,
			}),
			Resources: vsphereCPIResourceRequirements,
		},
	}

	return containerList, nil
}

const latestVsphereCPIVersion = "1.21.0"

func getVsphereCPIVersion(version semver.Semver) (string, error) {
	if version.Minor() < 18 {
		return "", fmt.Errorf("kubernetes version %s is not supported", version.String())
	}

	switch version.Minor() {
	case 18:
		return "1.18.2", nil
	case 19:
		return "1.19.1", nil
	case 20:
		return "1.20.0", nil
	case 21:
		return latestVsphereCPIVersion, nil
	default:
		return latestVsphereCPIVersion, nil
	}
}

// VsphereSupported checks if this version of Kubernetes is supported
// by our implementation of the external cloud controller.
func VsphereSupported(version semver.Semver) bool {
	if _, err := getVsphereCPIVersion(version); err != nil {
		return false
	}
	return true
}
