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

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
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
	tag       = "v2.0.0-rc3"
	// Namespace used by Dashboard to find required resources.
	Namespace     = "kubernetes-dashboard"
	ContainerPort = 9090
	AppLabel      = resources.AppLabelKey + "=" + name
)

// kubernetesDashboardData is the data needed to construct the Kubernetes Dashboard components
type kubernetesDashboardData interface {
	Cluster() *kubermaticv1.Cluster
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	ImageRegistry(string) string
}

// DeploymentCreator returns the function to create and update the Kubernetes Dashboard deployment
func DeploymentCreator(data kubernetesDashboardData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return name, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = name
			dep.Labels = resources.BaseAppLabels(name, nil)

			dep.Spec.Replicas = resources.Int32(2)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(name, nil),
			}

			volumes := getVolumes()
			podLabels, err := data.GetPodTemplateLabels(name, volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %v", err)
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
			}

			dep.Spec.Template.Spec.Volumes = volumes
			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
			dep.Spec.Template.Spec.Containers = getContainers(data, dep.Spec.Template.Spec.Containers)
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defaultResourceRequirements, nil, dep.Annotations)
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

func getContainers(data kubernetesDashboardData, existingContainers []corev1.Container) []corev1.Container {
	// We must do some hoops there because SecurityContext.RunAsGroup
	// does not exit in all Kubernetes versions. We must keep it if it
	// exists but never set it ourselves. The APIServer defaults
	// RunAsGroup to the RunAsUser setting
	securityContext := &corev1.SecurityContext{}
	if len(existingContainers) == 1 && existingContainers[0].SecurityContext != nil {
		securityContext = existingContainers[0].SecurityContext
	}
	securityContext.RunAsUser = pointer.Int64Ptr(1001)
	securityContext.ReadOnlyRootFilesystem = pointer.BoolPtr(true)
	securityContext.AllowPrivilegeEscalation = pointer.BoolPtr(false)
	return []corev1.Container{{
		Name:            name,
		Image:           fmt.Sprintf("%s/%s:%s", data.ImageRegistry(resources.RegistryDocker), imageName, tag),
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
				Name:      "tmp-volume",
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
	}}
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
			Name: "tmp-volume",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}
