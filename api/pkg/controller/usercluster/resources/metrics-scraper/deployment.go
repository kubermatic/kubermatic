package metricsscraper

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("32Mi"),
			corev1.ResourceCPU:    resource.MustParse("50m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("64Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
	}
)

const (
	name      = resources.MetricsScraperDeploymentName
	imageName = "kubernetesui/metrics-scraper"
	tag       = "v1.0.1"
	Namespace = "kubernetes-dashboard"
)

// DeploymentCreator returns the function to create and update the metrics server deployment
func DeploymentCreator() reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return name, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = name
			dep.Labels = resources.BaseAppLabel(name, nil)

			dep.Spec.Replicas = resources.Int32(2)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(name, nil),
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}
			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: resources.BaseAppLabel(name, nil),
			}

			volumes := getVolumes()
			dep.Spec.Template.Spec.Volumes = volumes
			dep.Spec.Template.Spec.Containers = getContainers()
			dep.Spec.Template.Spec.ServiceAccountName = name

			return dep, nil
		}
	}
}

func getContainers() []corev1.Container {
	return []corev1.Container{
		{
			Name:            name,
			Image:           fmt.Sprintf("%s/%s:%s", resources.RegistryDocker, imageName, tag),
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/metrics-sidecar"},
			Resources:       defaultResourceRequirements,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "tmp-volume",
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
				RunAsUser:                &[]int64{1001}[0],
				RunAsGroup:               &[]int64{2001}[0],
				ReadOnlyRootFilesystem:   &[]bool{true}[0],
				AllowPrivilegeEscalation: &[]bool{false}[0],
			},
		},
	}
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "tmp-volume",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}
