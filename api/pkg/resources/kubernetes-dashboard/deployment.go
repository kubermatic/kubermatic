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
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("256Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("250m"),
		},
	}
)

const (
	name      = resources.KubernetesDashboardDeploymentName
	imageName = "kubernetesui/dashboard"
	tag       = "v2.0.0-beta4"
	// Namespace used by Dashboard to find required resources.
	namespace = "kubernetes-dashboard"
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
			dep.Labels = resources.BaseAppLabel(name, nil)

			dep.Spec.Replicas = resources.Int32(2)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(name, nil),
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
			dep.Spec.Template.Spec.Containers = getContainers(data)
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

func getContainers(data kubernetesDashboardData) []corev1.Container {
	return []corev1.Container{
		{
			Name:            name,
			Image:           fmt.Sprintf("%s/%s:%s", data.ImageRegistry(resources.RegistryDocker), imageName, tag),
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/dashboard"},
			Args: []string{
				"--kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
				"--namespace", namespace,
				"--enable-insecure-login",
			},
			Resources: defaultResourceRequirements,
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
					ContainerPort: 9090,
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
