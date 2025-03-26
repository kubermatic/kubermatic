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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
)

var (
	defaultResourceRequirements = map[string]*corev1.ResourceRequirements{
		resources.KubernetesDashboardDeploymentName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("256Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("250m"),
			},
		},
	}
)

const (
	name      = resources.KubernetesDashboardDeploymentName
	imageName = "kubernetesui/dashboard"
	// Namespace used by dashboard to find required resources.
	Namespace        = "kubernetes-dashboard"
	ContainerPort    = 9090
	AppLabel         = resources.AppLabelKey + "=" + name
	tmpVolumeName    = "tmp-volume"
	dashboardVersion = "v2.7.0"
)

// kubernetesDashboardData is the data needed to construct the Kubernetes Dashboard components.
type kubernetesDashboardData interface {
	Cluster() *kubermaticv1.Cluster
	RewriteImage(string) (string, error)
}

// DeploymentReconciler returns the function to create and update the Kubernetes Dashboard deployment.
func DeploymentReconciler(data kubernetesDashboardData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return name, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			baseLabels := resources.BaseAppLabels(name, nil)
			kubernetes.EnsureLabels(dep, baseLabels)

			dep.Spec.Replicas = resources.Int32(2)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}

			containers, err := getContainers(data, dep.Spec.Template.Spec.Containers)
			if err != nil {
				return nil, err
			}

			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				resources.ClusterLastRestartAnnotation: data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
				// these volumes should not block the autoscaler from evicting the pod
				resources.ClusterAutoscalerSafeToEvictVolumesAnnotation: tmpVolumeName,
			})

			dep.Spec.Template.Spec.Volumes = getVolumes()
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}
			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
			dep.Spec.Template.Spec.Containers = containers
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defaultResourceRequirements, nil, dep.Annotations)
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

func getContainers(data kubernetesDashboardData, existingContainers []corev1.Container) ([]corev1.Container, error) {
	// We must do some hoops there because SecurityContext.RunAsGroup
	// does not exit in all Kubernetes versions. We must keep it if it
	// exists but never set it ourselves. The APIServer defaults
	// RunAsGroup to the RunAsUser setting
	securityContext := &corev1.SecurityContext{}
	if len(existingContainers) == 1 && existingContainers[0].SecurityContext != nil {
		securityContext = existingContainers[0].SecurityContext
	}
	securityContext.RunAsUser = ptr.To[int64](1001)
	securityContext.ReadOnlyRootFilesystem = ptr.To(true)
	securityContext.AllowPrivilegeEscalation = ptr.To(false)

	tag, err := DashboardVersion(data.Cluster().Status.Versions.ControlPlane)
	if err != nil {
		return nil, err
	}

	return []corev1.Container{{
		Name:            name,
		Image:           registry.Must(data.RewriteImage(fmt.Sprintf("%s/%s:%s", resources.RegistryDocker, imageName, tag))),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/dashboard"},
		Args: []string{
			"--kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
			"--namespace", Namespace,
			"--enable-insecure-login",
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      resources.KubernetesDashboardKubeconfigSecretName,
				MountPath: "/etc/kubernetes/kubeconfig",
				ReadOnly:  true,
			}, {
				Name:      tmpVolumeName,
				MountPath: "/tmp",
			},
		},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: ContainerPort,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		SecurityContext: securityContext,
	}}, nil
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.KubernetesDashboardKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.KubernetesDashboardKubeconfigSecretName,
				},
			},
		}, {
			Name: tmpVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}

func DashboardVersion(clusterVersion semver.Semver) (string, error) {
	// check the GitHub releases for find compat info on the dashboard:
	// https://github.com/kubernetes/dashboard/releases

	switch clusterVersion.MajorMinor() {
	case "1.28":
		return dashboardVersion, nil // not officially marked as supported
	case "1.29":
		return dashboardVersion, nil // not officially marked as supported
	case "1.30":
		return dashboardVersion, nil // not officially marked as supported
	case "1.31":
		return dashboardVersion, nil // not officially marked as supported
	case "1.32":
		return dashboardVersion, nil // not officially marked as supported
	default:
		return "", fmt.Errorf("no compatible version defined for Kubernetes %q", clusterVersion.MajorMinor())
	}
}
