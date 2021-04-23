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
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8s.io/apimachinery/pkg/api/resource"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	OpenstackCCMDeploymentName = "openstack-cloud-controller-manager"
)

var (
	osResourceRequirements = corev1.ResourceRequirements{
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

func openStackDeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return OpenstackCCMDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = OpenstackCCMDeploymentName
			dep.Labels = resources.BaseAppLabels(OpenstackCCMDeploymentName, nil)

			dep.Spec.Replicas = resources.Int32(1)

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(OpenstackCCMDeploymentName, nil),
			}

			podLabels, err := data.GetPodTemplateLabels(OpenstackCCMDeploymentName, dep.Spec.Template.Spec.Volumes, nil)
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

			openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, openvpnClientContainerName)
			if err != nil {
				return nil, fmt.Errorf("failed to get openvpn sidecar: %v", err)
			}

			version, err := getOSVersion(data.Cluster().Spec.Version)
			if err != nil {
				return nil, err
			}

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

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				*openvpnSidecar,
				{
					Name:    ccmContainerName,
					Image:   data.ImageRegistry(resources.RegistryDocker) + "/k8scloudprovider/openstack-cloud-controller-manager:v" + version,
					Command: []string{"/bin/openstack-cloud-controller-manager"},
					Args:    getOSFlags(data),
					VolumeMounts: append(getVolumeMounts(), corev1.VolumeMount{
						Name:      resources.CloudConfigConfigMapName,
						MountPath: "/etc/kubernetes/cloud",
						ReadOnly:  true,
					}),
				},
			}

			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				ccmContainerName:    osResourceRequirements.DeepCopy(),
				openvpnSidecar.Name: openvpnSidecar.Resources.DeepCopy(),
			}
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}

			return dep, nil
		}
	}
}

func getOSFlags(data *resources.TemplateData) []string {
	flags := []string{
		"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
		"--v=1",
		"--cloud-config=/etc/kubernetes/cloud/config",
		"--cloud-provider=openstack",
	}
	return flags
}

const latestCCMVersion = "1.20.2"

func getOSVersion(version semver.Semver) (string, error) {
	if version.Minor() < 17 {
		return "", fmt.Errorf("Kubernetes version %s is not supported", version.String())
	}

	switch version.Minor() {
	case 17:
		return "1.17.0", nil
	case 18:
		return "1.18.0", nil
	case 19:
		return "1.19.2", nil
	case 20:
		return latestCCMVersion, nil
	case 21:
		return latestCCMVersion, nil
	default:
		return latestCCMVersion, nil
	}
}

// OpenStackCloudControllerSupported checks if this version of Kubernetes is supported
// by our implementation of the external cloud controller.
// This is not called for now, but it's here so we can use it later to automagically
// enable external cloud controller support for supported versions.
func OpenStackCloudControllerSupported(version semver.Semver) bool {
	if _, err := getOSVersion(version); err != nil {
		return false
	}
	return true
}
