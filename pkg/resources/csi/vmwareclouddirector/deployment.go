/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

// Original source: https://github.com/vmware/cloud-director-named-disk-csi-driver/blob/1.5.0/manifests/csi-controller.yaml

package vmwareclouddirector

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	csiVersion = "1.5.0"
)

var (
	hostPathType            = corev1.HostPathDirectoryOrCreate
	mountPropagationHostToC = corev1.MountPropagationHostToContainer
)

// DeploymentsReconcilers returns the CSI controller Deployments for VMware Cloud Director.
func DeploymentsReconcilers(data *resources.TemplateData) []reconciling.NamedDeploymentReconcilerFactory {
	creators := []reconciling.NamedDeploymentReconcilerFactory{
		ControllerDeploymentReconciler(data),
	}
	return creators
}

// ControllerDeploymentReconciler returns the CSI controller Deployment for VMware Cloud Director.
func ControllerDeploymentReconciler(data *resources.TemplateData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (name string, create reconciling.DeploymentReconciler) {
		return resources.VMwareCloudDirectorCSIControllerName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			volumes := []corev1.Volume{
				{
					Name: "socket-dir",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "pods-probe-dir",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/dev",
							Type: &hostPathType,
						},
					},
				},
				{
					Name: "pv-dir",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/var/lib/kubelet/plugins/kubernetes.io/csi",
							Type: &hostPathType,
						},
					},
				},
				{
					Name: resources.CSICloudConfigSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.CSICloudConfigSecretName,
						},
					},
				},
				{
					Name: "vcloud-basic-auth-volume",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.VMwareCloudDirectorCSISecretName,
						},
					},
				},
				{
					Name: resources.VMwareCloudDirectorCSIKubeconfigSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.VMwareCloudDirectorCSIKubeconfigSecretName,
						},
					},
				},
			}
			dep.Name = resources.OperatingSystemManagerDeploymentName
			dep.Labels = resources.BaseAppLabels(resources.VMwareCloudDirectorCSIControllerName, nil)

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(resources.VMwareCloudDirectorCSIControllerName, nil),
			}
			dep.Spec.Template.Spec.Volumes = volumes

			podLabels, err := data.GetPodTemplateLabels(resources.VMwareCloudDirectorCSIControllerName, volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %w", err)
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
			}
			dep.Spec.Template.Spec.ServiceAccountName = resources.VMwareCloudDirectorCSIServiceAccountName
			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            "csi-attacher",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Image:           registry.Must(data.RewriteImage("registry.k8s.io/sig-storage/csi-attacher:v4.3.0")),
					Args: []string{
						"--csi-address=$(ADDRESS)",
						"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
						"--timeout=180s",
						"--v=5",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "ADDRESS",
							Value: "unix:///var/lib/csi/sockets/pluginproxy/csi.sock",
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "socket-dir",
							MountPath: "/var/lib/csi/sockets/pluginproxy/",
						},
						{
							Name:      resources.VMwareCloudDirectorCSIKubeconfigSecretName,
							MountPath: "/etc/kubernetes/kubeconfig",
							ReadOnly:  true,
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("10m"),
							corev1.ResourceMemory: resource.MustParse("24Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
					},
				},
				{
					Name:            "csi-provisioner",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Image:           registry.Must(data.RewriteImage("registry.k8s.io/sig-storage/csi-provisioner:v2.2.2")),
					Args: []string{
						"--csi-address=$(ADDRESS)",
						"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
						"--default-fstype=ext4",
						"--timeout=300s",
						"--v=5",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "ADDRESS",
							Value: "unix:///var/lib/csi/sockets/pluginproxy/csi.sock",
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "socket-dir",
							MountPath: "/var/lib/csi/sockets/pluginproxy/",
						},
						{
							Name:      resources.VMwareCloudDirectorCSIKubeconfigSecretName,
							MountPath: "/etc/kubernetes/kubeconfig",
							ReadOnly:  true,
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("10m"),
							corev1.ResourceMemory: resource.MustParse("24Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
					},
				},
				{
					Name:            "vcd-csi-plugin",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Image:           registry.Must(data.RewriteImage("projects.registry.vmware.com/vmware-cloud-director/cloud-director-named-disk-csi-driver:" + csiVersion)),
					Command:         []string{"/opt/vcloud/bin/cloud-director-named-disk-csi-driver"},
					Args: []string{
						"--endpoint=$(CSI_ENDPOINT)",
						"--cloud-config=/etc/kubernetes/vcloud/config",
						"--v=5",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "CSI_ENDPOINT",
							Value: "unix:///var/lib/csi/sockets/pluginproxy/csi.sock",
						},
						{
							Name: "NODE_ID",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "spec.nodeName",
								},
							},
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "socket-dir",
							MountPath: "/var/lib/csi/sockets/pluginproxy/",
						},
						{
							Name:             "pods-probe-dir",
							MountPath:        "/dev",
							MountPropagation: &mountPropagationHostToC,
						},
						{
							Name:             "pv-dir",
							MountPath:        "/var/lib/kubelet/plugins/kubernetes.io/csi",
							MountPropagation: &mountPropagationHostToC,
						},
						{
							Name:      resources.CSICloudConfigSecretName,
							MountPath: "/etc/kubernetes/vcloud",
						},
						{
							Name:      "vcloud-basic-auth-volume",
							MountPath: "/etc/kubernetes/vcloud/basic-auth",
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("10m"),
							corev1.ResourceMemory: resource.MustParse("24Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
				},
			}
			return dep, nil
		}
	}
}
