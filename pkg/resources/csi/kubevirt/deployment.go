/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package kubevirt

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	csiVersion = "v0.4.4"
)

// DeploymentsReconcilers returns the CSI controller Deployments for KubeVirt.
func DeploymentsReconcilers(data *resources.TemplateData) []reconciling.NamedDeploymentReconcilerFactory {
	creators := []reconciling.NamedDeploymentReconcilerFactory{
		ControllerDeploymentReconciler(data),
	}
	return creators
}

// ControllerDeploymentReconciler returns the CSI controller Deployment for KubeVirt.
func ControllerDeploymentReconciler(data *resources.TemplateData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (name string, create reconciling.DeploymentReconciler) {
		return resources.KubeVirtCSIControllerName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			kubeVirtInfraNamespace := data.Cluster().Status.NamespaceName
			if data.DC().Spec.Kubevirt != nil && data.DC().Spec.Kubevirt.NamespacedMode != nil && data.DC().Spec.Kubevirt.NamespacedMode.Enabled {
				kubeVirtInfraNamespace = data.DC().Spec.Kubevirt.NamespacedMode.Namespace
			}
			version := data.Cluster().Status.Versions.ControllerManager.Semver()
			volumes := []corev1.Volume{
				{
					Name: "socket-dir",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "tenantcluster",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.AdminKubeconfigSecretName,
						},
					},
				},
				{
					Name: "infracluster",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.KubeVirtCSISecretName,
						},
					},
				},
			}

			d.Labels = resources.BaseAppLabels(resources.KubeVirtCSIControllerName, nil)

			d.Spec.Replicas = ptr.To[int32](1)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			}

			kubernetes.EnsureLabels(&d.Spec.Template, map[string]string{
				resources.VersionLabel: version.String(),
			})
			kubernetes.EnsureAnnotations(&d.Spec.Template, map[string]string{
				resources.ClusterLastRestartAnnotation: data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
				// these volumes should not block the autoscaler from evicting the pod
				resources.ClusterAutoscalerSafeToEvictVolumesAnnotation: "socket-dir",
			})

			var err error
			d.Spec.Template.Spec.DNSPolicy, d.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}
			d.Spec.Template.Spec.ServiceAccountName = resources.KubeVirtCSIServiceAccountName
			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            "csi-driver",
					ImagePullPolicy: corev1.PullAlways,
					Image:           registry.Must(data.RewriteImage("quay.io/kubermatic/kubevirt-csi-driver:" + csiVersion)),
					Args: []string{
						"--endpoint=$(CSI_ENDPOINT)",
						fmt.Sprintf("--infra-cluster-namespace=%s", kubeVirtInfraNamespace),
						fmt.Sprintf("--infra-cluster-labels=%s", fmt.Sprintf("cluster-name=%s", data.Cluster().Name)),
						"--infra-cluster-kubeconfig=/var/run/secrets/infracluster/kubeconfig",
						"--tenant-cluster-kubeconfig=/var/run/secrets/tenantcluster/kubeconfig",
						"--run-node-service=false",
						"--run-controller-service=true",
						"--v=5",
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "healthz",
							ContainerPort: 10301,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "CSI_ENDPOINT",
							Value: "unix:///var/lib/csi/sockets/pluginproxy/csi.sock",
						},
						{
							Name: "KUBE_NODE_NAME",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "spec.nodeName",
								},
							},
						},
						{
							Name: "INFRACLUSTER_NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: resources.KubeVirtCSIConfigMapName,
									},
									Key: resources.KubeVirtCSINamespaceKey,
								},
							},
						},
						{
							Name: "INFRACLUSTER_LABELS",
							ValueFrom: &corev1.EnvVarSource{
								ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: resources.KubeVirtCSIConfigMapName,
									},
									Key: resources.KubeVirtCSIClusterLabelKey,
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
							Name:      "tenantcluster",
							MountPath: "/var/run/secrets/tenantcluster",
						},
						{
							Name:      "infracluster",
							MountPath: "/var/run/secrets/infracluster",
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("10m"),
							corev1.ResourceMemory: resource.MustParse("50Mi"),
						},
					},
				},
				{
					Name:            "csi-provisioner",
					ImagePullPolicy: corev1.PullAlways,
					Image:           registry.Must(data.RewriteImage("quay.io/openshift/origin-csi-external-provisioner:4.20.0")),
					Args: []string{
						"--csi-address=$(ADDRESS)",
						"--default-fstype=ext4",
						"--kubeconfig=/var/run/secrets/tenantcluster/kubeconfig",
						"--v=5",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "ADDRESS",
							Value: "/var/lib/csi/sockets/pluginproxy/csi.sock",
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "socket-dir",
							MountPath: "/var/lib/csi/sockets/pluginproxy/",
						},
						{
							Name:      "tenantcluster",
							MountPath: "/var/run/secrets/tenantcluster",
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("10m"),
							corev1.ResourceMemory: resource.MustParse("50Mi"),
						},
					},
				},
				{
					Name:            "csi-attacher",
					ImagePullPolicy: corev1.PullAlways,
					Image:           registry.Must(data.RewriteImage("quay.io/openshift/origin-csi-external-attacher:4.20.0")),
					Args: []string{
						"--csi-address=$(ADDRESS)",
						"--kubeconfig=/var/run/secrets/tenantcluster/kubeconfig",
						"--v=5",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "ADDRESS",
							Value: "/var/lib/csi/sockets/pluginproxy/csi.sock",
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "socket-dir",
							MountPath: "/var/lib/csi/sockets/pluginproxy/",
						},
						{
							Name:      "tenantcluster",
							MountPath: "/var/run/secrets/tenantcluster",
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("10m"),
							corev1.ResourceMemory: resource.MustParse("50Mi"),
						},
					},
				},
				{
					Name:            "csi-liveness-probe",
					ImagePullPolicy: corev1.PullAlways,
					Image:           registry.Must(data.RewriteImage("quay.io/openshift/origin-csi-livenessprobe:4.20.0")),
					Args: []string{
						"--csi-address=/csi/csi.sock",
						"--probe-timeout=3s",
						"--health-port=10301",
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "socket-dir",
							MountPath: "/csi",
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("10m"),
							corev1.ResourceMemory: resource.MustParse("50Mi"),
						},
					},
				},
			}
			d.Spec.Template.Spec.Volumes = volumes
			return d, nil
		}
	}
}
