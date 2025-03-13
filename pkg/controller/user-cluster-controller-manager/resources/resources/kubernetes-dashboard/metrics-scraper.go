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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	seedresources "k8c.io/kubermatic/v2/pkg/resources/kubernetes-dashboard"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	// msServiceAccountName is the name of the user coming from kubeconfig cert.
	msServiceAccountName = "kubernetes-dashboard-metrics-scraper"
	// msDeploymentName is the name of dashboard-metrics-scraper deployment.
	msDeploymentName = "kubernetes-dashboard-metrics-scraper"
	// msContainerName is the name of the primary container in the dashboard-metrics-scraper deployment.
	msContainerName = "kubernetes-dashboard-metrics-scraper"
	// msServiceName is the name of dashboard-metrics-scraper service.
	msServiceName = "kubernetes-dashboard-metrics-scraper"
	// msClusterRoleName is the name of the role for the dashboard-metrics-scraper.
	msClusterRoleName = "system:kubernetes-dashboard-metrics-scraper"
	// msClusterRoleBindingName is the name of the role binding for the dashboard-metrics-scraper.
	msClusterRoleBindingName = "system:kubernetes-dashboard-metrics-scraper"
)

var (
	defaultResourceRequirements = map[string]*corev1.ResourceRequirements{
		msContainerName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("200Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("400Mi"),
				corev1.ResourceCPU:    resource.MustParse("250m"),
			},
		},
	}
)

func MetricsScraperServiceAccountReconciler() reconciling.NamedServiceAccountReconcilerFactory {
	return func() (string, reconciling.ServiceAccountReconciler) {
		return msServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			sa.Labels = resources.BaseAppLabels(msDeploymentName, nil)
			return sa, nil
		}
	}
}

func MetricsScraperClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return msClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = resources.BaseAppLabels(msDeploymentName, nil)

			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"metrics.k8s.io"},
					Resources: []string{"pods", "nodes"},
					Verbs:     []string{"get", "list", "watch"},
				},
			}

			return cr, nil
		}
	}
}

func MetricsScraperClusterRoleBindingReconciler(namespace string) reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return msClusterRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = resources.BaseAppLabels(msDeploymentName, nil)

			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     msClusterRoleName,
			}

			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      msServiceAccountName,
					Namespace: namespace,
				},
			}

			return crb, nil
		}
	}
}

func MetricsScraperServiceReconciler(ipFamily kubermaticv1.IPFamily) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return msServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Labels = resources.BaseAppLabels(msDeploymentName, nil)
			s.Spec.Selector = s.Labels

			s.Spec.Ports = []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       8000,
					TargetPort: intstr.FromInt(8000),
				},
			}

			if ipFamily == kubermaticv1.IPFamilyDualStack {
				dsPolicy := corev1.IPFamilyPolicyPreferDualStack
				s.Spec.IPFamilyPolicy = &dsPolicy
			}

			return s, nil
		}
	}
}

func MetricsScraperDeploymentReconciler(cluster *kubermaticv1.Cluster, imageRewriter registry.ImageRewriter) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return msDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			baseLabels := resources.BaseAppLabels(msDeploymentName, nil)
			kubernetes.EnsureLabels(dep, baseLabels)

			dep.Spec.Replicas = resources.Int32(2)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}

			clusterVersion := cluster.Status.Versions.ControlPlane
			if clusterVersion == "" {
				clusterVersion = cluster.Spec.Version
			}

			// for convenience reasons, this version is kept together with all the others in the seed package
			msVersion, err := seedresources.MetricsScraperVersion(clusterVersion)
			if err != nil {
				return nil, fmt.Errorf("failed to determine version: %w", err)
			}

			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				// these volumes should not block the autoscaler from evicting the pod
				resources.ClusterAutoscalerSafeToEvictVolumesAnnotation: "tmp-volume",
			})
			kubernetes.EnsureLabels(&dep.Spec.Template, baseLabels)

			dep.Spec.Template.Spec.ServiceAccountName = msServiceAccountName
			dep.Spec.Template.Spec.AutomountServiceAccountToken = ptr.To(true)

			dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				RunAsNonRoot: ptr.To(true),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            msContainerName,
					Image:           registry.Must(imageRewriter("docker.io/kubernetesui/dashboard-metrics-scraper:" + msVersion)),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Env: []corev1.EnvVar{
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
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/",
								Port:   intstr.FromInt(8000),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						InitialDelaySeconds: 30,
						TimeoutSeconds:      30,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						FailureThreshold:    3,
					},
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 8000,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "tmp-volume",
							MountPath: "/tmp",
						},
					},
				},
			}

			dep.Spec.Template.Spec.Volumes = []corev1.Volume{
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

			return dep, nil
		}
	}
}
