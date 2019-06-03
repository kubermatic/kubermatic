package clusterautoscaler

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
	"k8s.io/apimachinery/pkg/util/intstr"
)

type clusterautoscalerData interface {
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	ImageRegistry(string) string
	Cluster() *kubermaticv1.Cluster
}

// DeploymentCreator returns the function to create and update the cluster-autoscaler deployment
func DeploymentCreator(data clusterautoscalerData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.ClusterAutoscalerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			var tag string
			switch data.Cluster().Spec.Version.Minor() {
			case 14:
				tag = "fe5bee817ad9d37c8ce5e473af201c2f3fdf5b94-1"
			}
			if tag == "" {
				return nil, fmt.Errorf("No matching autoscaler tag found for version %d", data.Cluster().Spec.Version.Minor())
			}

			dep.Name = resources.ClusterAutoscalerDeploymentName
			dep.Labels = resources.BaseAppLabel(resources.ClusterAutoscalerDeploymentName, nil)

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(resources.ClusterAutoscalerDeploymentName, nil),
			}
			dep.Spec.Strategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
			dep.Spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{
				MaxSurge: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 1,
				},
				MaxUnavailable: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 0,
				},
			}

			volumes := []corev1.Volume{
				{
					Name: resources.ClusterAutoscalerKubeconfigSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.ClusterAutoscalerKubeconfigSecretName,
							// We have to make the secret readable for all for now because owner/group cannot be changed.
							// ( upstream proposal: https://github.com/kubernetes/kubernetes/pull/28733 )
							DefaultMode: resources.Int32(resources.DefaultAllReadOnlyMode),
						},
					},
				},
			}
			podLabels, err := data.GetPodTemplateLabels(resources.ClusterAutoscalerDeploymentName,
				volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %v", err)
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
				Annotations: map[string]string{
					"prometheus.io/scrape": "true",
					"prometheus.io/path":   "/metrics",
					"prometheus.io/port":   "8085",
				},
			}

			dep.Spec.Template.Spec.Volumes = volumes

			apiserverIsRunningContainer, err := apiserver.IsRunningInitContainer(data)
			if err != nil {
				return nil, err
			}
			dep.Spec.Template.Spec.InitContainers = []corev1.Container{*apiserverIsRunningContainer}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    resources.ClusterAutoscalerDeploymentName,
					Image:   data.ImageRegistry(resources.RegistryQuay) + "/kubermatic/kubernetes-cluster-autoscaler:" + tag,
					Command: []string{"/cluster-autoscaler"},
					Args: []string{
						"--kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
						"--leader-elect-resource-lock", "configmaps",
						// PercentageUsed treshold. If the current utilization of a node is above this, the CA will never
						// scale it down. Default is 0.5. Increased, because otherwise small nodes never get scaled down
						// because the DS pods on them alone manage to get the utilization above the 0.5 threshold.
						"--scale-down-utilization-threshold", "0.7",
					},
					// This likely won't be enough for bigger clusters, see https://github.com/kubermatic/kubermatic/issues/3568
					// for details on how we want to fix this: https://github.com/kubermatic/kubermatic/issues/3568
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("32Mi"),
							corev1.ResourceCPU:    resource.MustParse("25m"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("64Mi"),
							corev1.ResourceCPU:    resource.MustParse("50m"),
						},
					},
					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/health-check",
								Port:   intstr.FromInt(8085),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						FailureThreshold:    3,
						InitialDelaySeconds: 15,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      15,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.ClusterAutoscalerKubeconfigSecretName,
							MountPath: "/etc/kubernetes/kubeconfig",
							ReadOnly:  true,
						},
					},
				},
			}

			return dep, nil
		}
	}
}
