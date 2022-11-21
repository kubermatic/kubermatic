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

package metricsserver

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/servingcerthelper"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/registry"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	defResourceRequirements = map[string]*corev1.ResourceRequirements{
		resources.MetricsServerDeploymentName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("100Mi"),
				corev1.ResourceCPU:    resource.MustParse("50m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("1"),
			},
		},
	}
)

const (
	servingCertSecretName  = "metrics-server-serving-cert"
	servingCertMountFolder = "/etc/serving-cert"

	imageName = "metrics-server/metrics-server"
	imageTag  = "v0.5.0"
)

// TLSServingCertSecretCreator returns a function to manage the TLS serving cert for the metrics server.
func TLSServingCertSecretCreator(caGetter servingcerthelper.CAGetter) reconciling.NamedSecretCreatorGetter {
	dnsName := "metrics-server.kube-system.svc"
	return servingcerthelper.ServingCertSecretCreator(caGetter, servingCertSecretName, dnsName, []string{dnsName}, nil)
}

// DeploymentCreator returns the function to create and update the metrics server deployment.
func DeploymentCreator(registryWithOverwrite registry.WithOverwriteFunc) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.MetricsServerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.MetricsServerDeploymentName
			dep.Namespace = metav1.NamespaceSystem
			dep.Labels = resources.BaseAppLabels(resources.MetricsServerDeploymentName, nil)

			dep.Spec.Replicas = resources.Int32(2)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(resources.MetricsServerDeploymentName, nil),
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
			dep.Spec.Template.ObjectMeta.Labels = resources.BaseAppLabels(resources.MetricsServerDeploymentName, nil)

			volumes := getVolumes()
			dep.Spec.Template.Spec.Volumes = volumes

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    resources.MetricsServerDeploymentName,
					Image:   fmt.Sprintf("%s/%s:%s", registryWithOverwrite(resources.RegistryK8S), imageName, imageTag),
					Command: []string{"/metrics-server"},
					Args: []string{
						"--kubelet-insecure-tls",
						"--kubelet-use-node-status-port",
						"--metric-resolution", "15s",
						"--kubelet-preferred-address-types", "InternalIP,ExternalIP,Hostname",
						"--v", "1",
						"--tls-cert-file", servingCertMountFolder + "/" + resources.ServingCertSecretKey,
						"--tls-private-key-file", servingCertMountFolder + "/" + resources.ServingCertKeySecretKey,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      servingCertSecretName,
							MountPath: servingCertMountFolder,
							ReadOnly:  true,
						},
					},
				},
			}
			err := resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			dep.Spec.Template.Spec.ServiceAccountName = resources.MetricsServerServiceAccountName

			dep.Spec.Template.Spec.Affinity = &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						{
							Weight: 10,
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: resources.BaseAppLabels(resources.MetricsServerDeploymentName, nil),
								},
								TopologyKey: resources.TopologyKeyHostname,
							},
						},
					},
				},
			}

			dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}

			return dep, nil
		}
	}
}

// PodDisruptionBudgetCreator returns a func to create/update the metrics-server PodDisruptionBudget.
func PodDisruptionBudgetCreator() reconciling.NamedPodDisruptionBudgetCreatorGetter {
	return func() (string, reconciling.PodDisruptionBudgetCreator) {
		return resources.MetricsServerPodDisruptionBudgetName, func(pdb *policyv1beta1.PodDisruptionBudget) (*policyv1beta1.PodDisruptionBudget, error) {
			minAvailable := intstr.FromInt(1)
			pdb.Spec = policyv1beta1.PodDisruptionBudgetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: resources.BaseAppLabels(resources.MetricsServerDeploymentName, nil),
				},
				MinAvailable: &minAvailable,
			}
			return pdb, nil
		}
	}
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: servingCertSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: servingCertSecretName,
				},
			},
		},
	}
}
