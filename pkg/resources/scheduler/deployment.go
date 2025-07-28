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

package scheduler

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/kubermatic/v2/pkg/resources/vpnsidecar"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("64Mi"),
			corev1.ResourceCPU:    resource.MustParse("20m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("1"),
		},
	}
)

const (
	name = "scheduler"
)

// DeploymentReconciler returns the function to create and update the scheduler deployment.
func DeploymentReconciler(data *resources.TemplateData, dra bool) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.SchedulerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			baseLabels := resources.BaseAppLabels(name, nil)
			kubernetes.EnsureLabels(dep, baseLabels)

			version := data.Cluster().Status.Versions.Scheduler.Semver()
			flags := []string{
				"--kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
				// These are used to validate tokens
				"--authentication-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
				"--authorization-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
				// This is used to validate certs
				"--client-ca-file", "/etc/kubernetes/pki/ca/ca.crt",
				// this can't be passed as two strings as the other parameters
				"--profiling=false",
			}
			if dra {
				flags = append(flags, "--feature-flags=DynamicResourceAllocation=true")
			}

			// Apply leader election settings
			if lds := data.Cluster().Spec.ComponentsOverride.Scheduler.LeaseDurationSeconds; lds != nil {
				flags = append(flags, "--leader-elect-lease-duration", fmt.Sprintf("%ds", *lds))
			}
			if rds := data.Cluster().Spec.ComponentsOverride.Scheduler.LeaderElectionSettings.DeepCopy().RenewDeadlineSeconds; rds != nil {
				flags = append(flags, "--leader-elect-renew-deadline", fmt.Sprintf("%ds", *rds))
			}
			if rps := data.Cluster().Spec.ComponentsOverride.Scheduler.LeaderElectionSettings.DeepCopy().RetryPeriodSeconds; rps != nil {
				flags = append(flags, "--leader-elect-retry-period", fmt.Sprintf("%ds", *rps))
			}

			dep.Spec.Replicas = resources.Int32(1)
			if data.Cluster().Spec.ComponentsOverride.Scheduler.Replicas != nil {
				dep.Spec.Replicas = data.Cluster().Spec.ComponentsOverride.Scheduler.Replicas
			}

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}

			kubernetes.EnsureLabels(&dep.Spec.Template, map[string]string{
				resources.VersionLabel: version.String(),
			})
			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				"prometheus.io/path":                   "/metrics",
				"prometheus.io/scrape_with_kube_cert":  "true",
				"prometheus.io/port":                   "10259",
				resources.ClusterLastRestartAnnotation: data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
			})

			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			var err error
			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			healthAction := &corev1.HTTPGetAction{
				Path:   "/healthz",
				Scheme: corev1.URISchemeHTTPS,
				Port:   intstr.FromInt(10259),
			}

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    resources.SchedulerDeploymentName,
					Image:   registry.Must(data.RewriteImage(resources.RegistryK8S + "/kube-scheduler:v" + version.String())),
					Command: []string{"/usr/local/bin/kube-scheduler"},
					Args:    flags,
					Env: []corev1.EnvVar{
						{
							Name:  "SSL_CERT_FILE",
							Value: "/etc/kubernetes/pki/ca-bundle/ca-bundle.pem",
						},
					},
					VolumeMounts: getVolumeMounts(),
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: healthAction,
						},
						FailureThreshold: 3,
						PeriodSeconds:    10,
						SuccessThreshold: 1,
						TimeoutSeconds:   15,
					},
					LivenessProbe: &corev1.Probe{
						FailureThreshold: 8,
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: healthAction,
						},
						InitialDelaySeconds: 15,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      15,
					},
				},
			}
			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				name: defaultResourceRequirements.DeepCopy(),
			}

			if !data.IsKonnectivityEnabled() {
				openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
				if err != nil {
					return nil, fmt.Errorf("failed to get openvpn sidecar: %w", err)
				}
				dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, *openvpnSidecar)
				defResourceRequirements[openvpnSidecar.Name] = openvpnSidecar.Resources.DeepCopy()
			}

			dep.Spec.Template.Spec.Volumes = getVolumes(data.IsKonnectivityEnabled())

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, resources.GetOverrides(data.Cluster().Spec.ComponentsOverride), dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(name, kubermaticv1.AntiAffinityTypePreferred)

			dep.Spec.Template, err = apiserver.IsRunningWrapper(data, dep.Spec.Template, sets.New(name))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %w", err)
			}

			return dep, nil
		}
	}
}

func getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      resources.SchedulerKubeconfigSecretName,
			MountPath: "/etc/kubernetes/kubeconfig",
			ReadOnly:  true,
		},
		{
			Name:      resources.CASecretName,
			MountPath: "/etc/kubernetes/pki/ca",
			ReadOnly:  true,
		},
		{
			Name:      resources.CABundleConfigMapName,
			MountPath: "/etc/kubernetes/pki/ca-bundle",
			ReadOnly:  true,
		},
	}
}

func getVolumes(isKonnectivityEnabled bool) []corev1.Volume {
	vs := []corev1.Volume{
		{
			Name: resources.CASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.CASecretName,
					Items: []corev1.KeyToPath{
						{
							Path: resources.CACertSecretKey,
							Key:  resources.CACertSecretKey,
						},
					},
				},
			},
		},
		{
			Name: resources.CABundleConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.CABundleConfigMapName,
					},
				},
			},
		},
		{
			Name: resources.SchedulerKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.SchedulerKubeconfigSecretName,
				},
			},
		},
	}
	if !isKonnectivityEnabled {
		vs = append(vs, corev1.Volume{
			Name: resources.OpenVPNClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.OpenVPNClientCertificatesSecretName,
				},
			},
		})
	}
	return vs
}
