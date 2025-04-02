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
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	defResourceRequirements = map[string]*corev1.ResourceRequirements{
		resources.MetricsServerDeploymentName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("200Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
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
	imageTag  = "v0.7.0"

	servingPort = 10250
)

// TLSServingCertSecretReconciler returns a function to manage the TLS serving cert for the metrics server.
func TLSServingCertSecretReconciler(caGetter servingcerthelper.CAGetter) reconciling.NamedSecretReconcilerFactory {
	dnsName := "metrics-server.kube-system.svc"
	return servingcerthelper.ServingCertSecretReconciler(caGetter, servingCertSecretName, dnsName, []string{dnsName}, nil)
}

// DeploymentReconciler returns the function to create and update the metrics server deployment.
func DeploymentReconciler(imageRewriter registry.ImageRewriter) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
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
			dep.Spec.Template.Labels = resources.BaseAppLabels(resources.MetricsServerDeploymentName, nil)

			volumes := getVolumes()
			dep.Spec.Template.Spec.Volumes = volumes

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    resources.MetricsServerDeploymentName,
					Image:   registry.Must(imageRewriter(fmt.Sprintf("%s/%s:%s", resources.RegistryK8S, imageName, imageTag))),
					Command: []string{"/metrics-server"},
					Args: []string{
						"--kubelet-insecure-tls",
						"--kubelet-use-node-status-port",
						"--secure-port", fmt.Sprintf("%d", servingPort),
						"--metric-resolution", "15s",
						"--kubelet-preferred-address-types", "InternalIP,ExternalIP,Hostname",
						"--v", "1",
						"--tls-cert-file", servingCertMountFolder + "/" + resources.ServingCertSecretKey,
						"--tls-private-key-file", servingCertMountFolder + "/" + resources.ServingCertKeySecretKey,
					},
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: servingPort,
							Name:          "https",
							Protocol:      corev1.ProtocolTCP,
						},
					},
					// Do not define a readiness probe, as the metrics-server will only get ready
					// when it has scraped a node or pod at least once, which might never happen in
					// clusters without nodes. An unready metrics-server would prevent the
					// SeedResourcesUpToDate condition to become true.
					// ReadinessProbe: nil,
					LivenessProbe: &corev1.Probe{
						FailureThreshold: 3,
						PeriodSeconds:    10,
						SuccessThreshold: 1,
						TimeoutSeconds:   1,
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/livez",
								Port:   intstr.FromString("https"),
								Scheme: corev1.URISchemeHTTPS,
							},
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      servingCertSecretName,
							MountPath: servingCertMountFolder,
							ReadOnly:  true,
						},
					},
					SecurityContext: &corev1.SecurityContext{
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{"ALL"},
						},
						AllowPrivilegeEscalation: resources.Bool(false),
						ReadOnlyRootFilesystem:   resources.Bool(true),
						RunAsNonRoot:             resources.Bool(true),
						RunAsUser:                resources.Int64(1000),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
				},
			}
			err := resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			dep.Spec.Template.Spec.ServiceAccountName = resources.MetricsServerServiceAccountName
			dep.Spec.Template.Spec.PriorityClassName = "system-cluster-critical"

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

// PodDisruptionBudgetReconciler returns a func to create/update the metrics-server PodDisruptionBudget.
func PodDisruptionBudgetReconciler() reconciling.NamedPodDisruptionBudgetReconcilerFactory {
	return func() (string, reconciling.PodDisruptionBudgetReconciler) {
		return resources.MetricsServerPodDisruptionBudgetName, func(pdb *policyv1.PodDisruptionBudget) (*policyv1.PodDisruptionBudget, error) {
			minAvailable := intstr.FromInt(1)
			pdb.Spec = policyv1.PodDisruptionBudgetSpec{
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
