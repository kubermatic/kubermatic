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

package kubernetesdashboard

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
)

const (
	apiDeploymentName = "kubernetes-dashboard-api"
	apiContainerName  = "kubernetes-dashboard-api"
	apiServiceName    = "kubernetes-dashboard-api"
)

func APIServiceReconciler() reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return apiServiceName, func(existing *corev1.Service) (*corev1.Service, error) {
			existing.Spec.Ports = []corev1.ServicePort{
				{
					Name:       "api",
					Port:       8000,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8000),
				},
			}

			existing.Spec.Selector = resources.BaseAppLabels(apiDeploymentName, nil)

			return existing, nil
		}
	}
}

func APIDeploymentReconciler(data kubernetesDashboardData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return apiDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			baseLabels := resources.BaseAppLabels(apiDeploymentName, nil)
			kubernetes.EnsureLabels(dep, baseLabels)

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}

			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				resources.ClusterLastRestartAnnotation: data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
				// these volumes should not block the autoscaler from evicting the pod
				resources.ClusterAutoscalerSafeToEvictVolumesAnnotation: "tmp-volume",
			})

			dep.Spec.Template.Spec.AutomountServiceAccountToken = ptr.To(false)

			dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}

			clusterVersion := data.Cluster().Status.Versions.ControlPlane
			if clusterVersion == "" {
				clusterVersion = data.Cluster().Spec.Version
			}

			apiVersion, err := APIVersion(clusterVersion)
			if err != nil {
				return nil, fmt.Errorf("failed to determine API version: %w", err)
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            apiContainerName,
					Image:           registry.Must(data.RewriteImage("docker.io/kubernetesui/dashboard-api:" + apiVersion)),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/dashboard-api"},
					Args: []string{
						"--insecure-bind-address=0.0.0.0",
						"--bind-address=0.0.0.0",
						"--kubeconfig=/opt/kubeconfig/kubeconfig",
						"--metrics-scraper-service-name=kubernetes-dashboard-metrics-scraper",
						fmt.Sprintf("--namespace=%s", resources.KubernetesDashboardNamespace),
					},
					Env: []corev1.EnvVar{
						{
							Name: "CSRF_KEY",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: CSRFSecretName,
									},
									Key: crsfKeyName,
								},
							},
						},
						{
							Name: "GOMAXPROCS",
							ValueFrom: &corev1.EnvVarSource{
								ResourceFieldRef: &corev1.ResourceFieldSelector{
									Resource: "limits.cpu",
									Divisor:  resource.MustParse("1"),
								},
							},
						},
						{
							Name: "GOMEMLIMIT",
							ValueFrom: &corev1.EnvVarSource{
								ResourceFieldRef: &corev1.ResourceFieldSelector{
									Resource: "limits.memory",
									Divisor:  resource.MustParse("1"),
								},
							},
						},
					},
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: ptr.To(false),
						ReadOnlyRootFilesystem:   ptr.To(true),
						RunAsGroup:               ptr.To(int64(2001)),
						RunAsUser:                ptr.To(int64(1001)),
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{
								"ALL",
							},
						},
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "api",
							ContainerPort: 8000,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "kubeconfig",
							MountPath: "/opt/kubeconfig",
						},
						{
							Name:      "tmp-volume",
							MountPath: "/tmp",
						},
					},
				},
			}

			dep.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "kubeconfig",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  KubeconfigSecretName,
							DefaultMode: ptr.To(int32(0644)),
						},
					},
				},
				{
					Name: "tmp-volume",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			}

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defaultResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			dep.Spec.Template, err = apiserver.IsRunningWrapper(data, dep.Spec.Template, sets.New(apiContainerName))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %w", err)
			}

			return dep, nil
		}
	}
}
