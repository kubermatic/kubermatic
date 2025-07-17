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
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
)

const (
	v130 = "1.30"
	v131 = "1.31"
	v132 = "1.32"
	v133 = "1.33"
)

func getVolumes(isKonnectivityEnabled bool, mountCloudConfig bool) []corev1.Volume {
	vs := []corev1.Volume{
		{
			Name: resources.CloudControllerManagerKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.CloudControllerManagerKubeconfigSecretName,
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
	}
	if mountCloudConfig {
		vs = append(vs, corev1.Volume{
			Name: resources.CloudConfigSeedSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.CloudConfigSeedSecretName,
				},
			},
		})
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

func getVolumeMounts(mountCloudConfig bool) []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{
		{
			Name:      resources.CloudControllerManagerKubeconfigSecretName,
			MountPath: "/etc/kubernetes/kubeconfig",
			ReadOnly:  true,
		},
		{
			Name:      resources.CABundleConfigMapName,
			MountPath: "/etc/kubermatic/certs",
			ReadOnly:  true,
		},
	}

	if mountCloudConfig {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      resources.CloudConfigSeedSecretName,
			MountPath: "/etc/kubernetes/cloud",
			ReadOnly:  true,
		})
	}

	return mounts
}

func getEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "SSL_CERT_FILE",
			Value: "/etc/kubermatic/certs/ca-bundle.pem",
		},
	}
}
