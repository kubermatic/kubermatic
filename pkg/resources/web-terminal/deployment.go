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

package webterminal

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	defaultResourceRequirements = map[string]*corev1.ResourceRequirements{
		name: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("256Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1Gi"),
				corev1.ResourceCPU:    resource.MustParse("250m"),
			},
		},
	}
)

const (
	name               = "web-terminal"
	webTerminalStorage = "web-terminal-storage"
)

// DeploymentCreator returns the function to create WEB terminal deployment.
func DeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return name, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = name
			dep.Labels = resources.BaseAppLabels(name, nil)

			dep.Spec.Replicas = resources.Int32(1)

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(name, nil),
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			var err error
			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			version := data.Cluster().Status.Versions.Apiserver.Semver()
			volumes := getVolumes()

			podLabels, err := data.GetPodTemplateLabels(name, volumes, map[string]string{
				resources.VersionLabel: version.String(),
			})
			if err != nil {
				return nil, err
			}
			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels:      podLabels,
				Annotations: map[string]string{},
			}

			dep.Spec.Template.Spec.Volumes = volumes

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    name,
					Image:   data.ImageRegistry(resources.RegistryQuay) + "/kubermatic/web-terminal:0.2.0",
					Command: []string{"/bin/bash", "-c", "--"},
					Args:    []string{"while true; do sleep 30; done;"},
					Env: []corev1.EnvVar{
						{
							Name:  "KUBECONFIG",
							Value: "/etc/kubernetes/kubeconfig/kubeconfig",
						},
						{
							Name:  "PS1",
							Value: "\\$ ",
						},
					},
					VolumeMounts: getVolumeMounts(),
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: resources.Bool(false),
					},
				},
			}

			dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				RunAsUser:  resources.Int64(1000),
				RunAsGroup: resources.Int64(3000),
				FSGroup:    resources.Int64(2000),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defaultResourceRequirements, resources.GetOverrides(data.Cluster().Spec.ComponentsOverride), dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			return dep, nil
		}
	}
}

func getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      resources.WEBTerminalKubeconfigSecretName,
			MountPath: "/etc/kubernetes/kubeconfig",
			ReadOnly:  true,
		},
		{
			Name:      webTerminalStorage,
			ReadOnly:  false,
			MountPath: "/data/terminal",
		},
	}
}

func getVolumes() []corev1.Volume {
	vs := []corev1.Volume{
		{
			Name: resources.WEBTerminalKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.AdminKubeconfigSecretName,
				},
			},
		},
		{
			Name: webTerminalStorage,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		},
	}
	return vs
}
