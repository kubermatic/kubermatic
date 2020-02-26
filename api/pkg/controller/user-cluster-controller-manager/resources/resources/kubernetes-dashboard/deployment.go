package kubernetesdashboard

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
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
	scraperTag       = "v1.0.3"
)

// DeploymentCreator returns the function to create and update the dashboard-metrics-scraper deployment
func DeploymentCreator() reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return scraperName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = scraperName
			dep.Labels = resources.BaseAppLabels(scraperName, nil)

			dep.Spec.Replicas = resources.Int32(2)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(scraperName, nil),
			}
			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: resources.BaseAppLabels(scraperName, nil),
			}

			volumes := getVolumes()
			dep.Spec.Template.Spec.Volumes = volumes

			dep.Spec.Template.Spec.Containers = getContainers()
			err := resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defaultResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}

			dep.Spec.Template.Spec.ServiceAccountName = scraperName

			return dep, nil
		}
	}
}

func getContainers() []corev1.Container {
	return []corev1.Container{
		{
			Name:            scraperName,
			Image:           fmt.Sprintf("%s/%s:%s", resources.RegistryDocker, scraperImageName, scraperTag),
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/metrics-sidecar"},
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
				RunAsUser:                pointer.Int64Ptr(1001),
				RunAsGroup:               pointer.Int64Ptr(2001),
				ReadOnlyRootFilesystem:   pointer.BoolPtr(true),
				AllowPrivilegeEscalation: pointer.BoolPtr(false),
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
