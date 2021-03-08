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

	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/vpnsidecar"

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

// DeploymentCreator returns the function to create and update the scheduler deployment
func DeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.SchedulerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.SchedulerDeploymentName
			dep.Labels = resources.BaseAppLabels(name, nil)

			flags := []string{
				"--kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
				// These are used to validate tokens
				"--authentication-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
				"--authorization-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
				// This is used to validate certs
				"--client-ca-file", "/etc/kubernetes/pki/ca/ca.crt",
				// We're going to use the https endpoints for scraping the metrics starting from 1.13. Thus we can deactivate the http endpoint
				"--port", "0",
			}

			// Apply leader election settings
			if lds := data.Cluster().Spec.ComponentsOverride.Scheduler.LeaderElectionSettings.LeaseDurationSeconds; lds != nil {
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
				MatchLabels: resources.BaseAppLabels(name, nil),
			}

			volumes := getVolumes()
			volumeMounts := getVolumeMounts()

			podLabels, err := data.GetPodTemplateLabels(name, volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %v", err)
			}

			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
				Annotations: map[string]string{
					"prometheus.io/path":                  "/metrics",
					"prometheus.io/scrape_with_kube_cert": "true",
					"prometheus.io/port":                  "10259",
				},
			}

			openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
			if err != nil {
				return nil, fmt.Errorf("failed to get openvpn sidecar: %v", err)
			}

			// Configure user cluster DNS resolver for this pod.
			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.Volumes = volumes

			healthAction := &corev1.HTTPGetAction{
				Path:   "/healthz",
				Scheme: corev1.URISchemeHTTPS,
				Port:   intstr.FromInt(10259),
			}

			version, err := data.GetControlPlaneComponentVersion(dep, resources.SchedulerDeploymentName)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				*openvpnSidecar,
				{
					Name:    resources.SchedulerDeploymentName,
					Image:   data.ImageRegistry(resources.RegistryK8SGCR) + "/kube-scheduler:v" + version.String(),
					Command: []string{"/usr/local/bin/kube-scheduler"},
					Args:    flags,
					Env: []corev1.EnvVar{
						// Kubernetes <1.19 did not ship any certificates in their Docker images,
						// so this not only injects _our_ CA bundle, it injects _the only_ CA bundle for 1.17/1.18 clusters.
						{
							Name:  "SSL_CERT_FILE",
							Value: "/etc/kubernetes/pki/ca-bundle/ca-bundle.pem",
						},
					},
					VolumeMounts: volumeMounts,
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: healthAction,
						},
						FailureThreshold: 3,
						PeriodSeconds:    10,
						SuccessThreshold: 1,
						TimeoutSeconds:   15,
					},
					LivenessProbe: &corev1.Probe{
						FailureThreshold: 8,
						Handler: corev1.Handler{
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
				name:                defaultResourceRequirements.DeepCopy(),
				openvpnSidecar.Name: openvpnSidecar.Resources.DeepCopy(),
			}
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, resources.GetOverrides(data.Cluster().Spec.ComponentsOverride), dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}

			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(name, data.Cluster().Name)

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, dep.Spec.Template.Spec, sets.NewString(name))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			dep.Spec.Template.Spec = *wrappedPodSpec

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

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.OpenVPNClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.OpenVPNClientCertificatesSecretName,
				},
			},
		},
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
}
