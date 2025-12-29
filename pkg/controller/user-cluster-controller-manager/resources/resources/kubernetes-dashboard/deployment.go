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

package kubernetesdashboard

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

var (
	defaultResourceRequirements = map[string]*corev1.ResourceRequirements{
		scraperName: {
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

const (
	scraperName      = resources.MetricsScraperDeploymentName
	scraperImageName = "kubernetesui/metrics-scraper"
	scraperTag       = "v1.0.8"
	tmpVolumeName    = "tmp-volume"
)

// DeploymentReconciler returns the function to create and update the dashboard-metrics-scraper deployment.
func DeploymentReconciler(imageRewriter registry.ImageRewriter) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return scraperName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			baseLabels := resources.BaseAppLabels(scraperName, nil)
			kubernetes.EnsureLabels(dep, baseLabels)

			dep.Spec.Replicas = resources.Int32(2)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}

			kubernetes.EnsureLabels(&dep.Spec.Template, baseLabels)
			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				// these volumes should not block the autoscaler from evicting the pod
				resources.ClusterAutoscalerSafeToEvictVolumesAnnotation: tmpVolumeName,
			})

			volumes := getVolumes()
			dep.Spec.Template.Spec.Volumes = volumes

			dep.Spec.Template.Spec.Containers = getContainers(imageRewriter)
			err := resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defaultResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			dep.Spec.Template.Spec.ServiceAccountName = scraperName

			dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}

			return dep, nil
		}
	}
}

func getContainers(imageRewriter registry.ImageRewriter) []corev1.Container {
	return []corev1.Container{
		{
			Name:            scraperName,
			Image:           registry.Must(imageRewriter(fmt.Sprintf("%s:%s", scraperImageName, scraperTag))),
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/metrics-sidecar"},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      tmpVolumeName,
					MountPath: "/tmp",
				},
			},
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: 8000,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			SecurityContext: &corev1.SecurityContext{
				RunAsUser:                ptr.To[int64](1001),
				RunAsGroup:               ptr.To[int64](2001),
				RunAsNonRoot:             ptr.To(true),
				ReadOnlyRootFilesystem:   ptr.To(true),
				AllowPrivilegeEscalation: ptr.To(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
				},
			},
		},
	}
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: tmpVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}
