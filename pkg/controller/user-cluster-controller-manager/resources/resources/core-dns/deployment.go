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

package coredns

import (
	"errors"
	"fmt"

	semverlib "github.com/Masterminds/semver/v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/dns"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

var (
	defaultResourceRequirements = map[string]*corev1.ResourceRequirements{
		resources.CoreDNSDeploymentName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("32Mi"),
				corev1.ResourceCPU:    resource.MustParse("50m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("64Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
		},
	}
)

func getReplicas(cluster *kubermaticv1.Cluster) (*int32, error) {
	if coredns := cluster.Spec.ComponentsOverride.CoreDNS; coredns != nil && coredns.Replicas != nil {
		if network := cluster.Spec.ClusterNetwork; network.CoreDNSReplicas != nil && *network.CoreDNSReplicas != *coredns.Replicas {
			return nil, errors.New("both the new spec.componentsOverride.coreDNS.replicas and deprecated spec.clusterNetwork.coreDNSReplicas fields are set")
		}

		return coredns.Replicas, nil
	}

	if network := cluster.Spec.ClusterNetwork; network.CoreDNSReplicas != nil {
		return network.CoreDNSReplicas, nil
	}

	return ptr.To[int32](2), nil
}

// DeploymentReconciler returns the function to create and update the CoreDNS deployment.
func DeploymentReconciler(kubernetesVersion *semverlib.Version, cluster *kubermaticv1.Cluster, imageRewriter registry.ImageRewriter) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.CoreDNSDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Labels = resources.BaseAppLabels(resources.CoreDNSDeploymentName, nil)

			replicas, err := getReplicas(cluster)
			if err != nil {
				return nil, fmt.Errorf("failed to determine CoreDNS replicas: %w", err)
			}

			dep.Spec.Replicas = replicas

			// The Selector is immutable, so we don't change it if it's set. This happens in upgrade cases
			// where coredns is switched from a manifest based addon to a user-cluster-controller-manager resource
			if dep.Spec.Selector == nil {
				dep.Spec.Selector = &metav1.LabelSelector{
					MatchLabels: resources.BaseAppLabels(resources.CoreDNSDeploymentName, map[string]string{
						"app.kubernetes.io/name": "kube-dns",
					}),
				}
			}

			if dep.Spec.Template.Labels == nil {
				// has to be the same as the selector
				dep.Spec.Template.Labels = dep.Spec.Selector.MatchLabels
			}

			iptr := intstr.FromInt(1)
			sptr := intstr.FromString("25%")
			dep.Spec.Strategy = appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &iptr,
					MaxSurge:       &sptr,
				},
			}

			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				resources.ClusterAutoscalerSafeToEvictVolumesAnnotation: "tmp",
			})

			dep.Spec.Template.Spec.PriorityClassName = "system-cluster-critical"
			dep.Spec.Template.Spec.DNSPolicy = corev1.DNSDefault
			dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}

			volumes := getVolumes()
			dep.Spec.Template.Spec.Volumes = volumes

			overrides := resources.GetOverrides(cluster.Spec.ComponentsOverride)

			dep.Spec.Template.Spec.Containers = getContainers(kubernetesVersion, imageRewriter)
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defaultResourceRequirements, overrides, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			if coredns := cluster.Spec.ComponentsOverride.CoreDNS; coredns != nil {
				dep.Spec.Template.Spec.Tolerations = coredns.Tolerations
			}

			dep.Spec.Template.Spec.ServiceAccountName = resources.CoreDNSServiceAccountName

			dep.Spec.Template.Spec.Affinity = &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						{
							Weight: 10,
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: resources.BaseAppLabels(resources.CoreDNSDeploymentName, nil),
								},
								TopologyKey: resources.TopologyKeyHostname,
							},
						},
					},
				},
			}

			return dep, nil
		}
	}
}

func PodDisruptionBudgetReconciler() reconciling.NamedPodDisruptionBudgetReconcilerFactory {
	return func() (string, reconciling.PodDisruptionBudgetReconciler) {
		return resources.CoreDNSPodDisruptionBudgetName, func(pdb *policyv1.PodDisruptionBudget) (*policyv1.PodDisruptionBudget, error) {
			iptr := intstr.FromInt(1)
			pdb.Spec.MinAvailable = &iptr
			pdb.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(resources.CoreDNSDeploymentName, nil),
			}

			return pdb, nil
		}
	}
}

func getContainers(clusterVersion *semverlib.Version, imageRewriter registry.ImageRewriter) []corev1.Container {
	return []corev1.Container{
		{
			Name:            resources.CoreDNSDeploymentName,
			Image:           registry.Must(imageRewriter(fmt.Sprintf("%s/%s", resources.RegistryK8S, dns.CoreDNSImage(clusterVersion)))),
			ImagePullPolicy: corev1.PullIfNotPresent,

			Args: []string{"-conf", "/etc/coredns/Corefile"},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "config-volume",
					MountPath: "/etc/coredns",
					ReadOnly:  true,
				},
				{
					Name:      "tmp",
					MountPath: "/tmp",
				},
			},
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: 53,
					Name:          "dns-tcp",
					Protocol:      corev1.ProtocolTCP,
				},
				{
					ContainerPort: 53,
					Name:          "dns",
					Protocol:      corev1.ProtocolUDP,
				},
				{
					ContainerPort: 9153,
					Name:          "metrics",
					Protocol:      corev1.ProtocolTCP,
				},
			},

			LivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path:   "/health",
						Port:   intstr.FromInt(8080),
						Scheme: corev1.URISchemeHTTP,
					},
				},
				InitialDelaySeconds: 60,
				PeriodSeconds:       10,
				TimeoutSeconds:      5,
				SuccessThreshold:    1,
				FailureThreshold:    5,
			},

			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path:   "/health",
						Port:   intstr.FromInt(8080),
						Scheme: corev1.URISchemeHTTP,
					},
				},
				TimeoutSeconds:   1,
				PeriodSeconds:    10,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			},
			SecurityContext: &corev1.SecurityContext{
				ReadOnlyRootFilesystem:   ptr.To(true),
				AllowPrivilegeEscalation: ptr.To(false),
				Capabilities: &corev1.Capabilities{
					Add: []corev1.Capability{
						"NET_BIND_SERVICE",
					},
					Drop: []corev1.Capability{
						"all",
					},
				},
			},
		},
	}
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "tmp",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "config-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.CoreDNSConfigMapName,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "Corefile",
							Path: "Corefile",
						},
					},
				},
			},
		},
	}
}
