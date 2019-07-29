package resources

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	DeploymentName = "container-linux-update-operator"

	// Used as NodeSelector on the Agent & Operator. Ensures that we only run those components on ContainerLinux nodes
	NodeSelectorLabelKey   = "kubernetes.io/uses-container-linux"
	NodeSelectorLabelValue = "true"
)

var (
	deploymentReplicas       int32 = 1
	deploymentMaxSurge             = intstr.FromInt(1)
	deploymentMaxUnavailable       = intstr.FromString("25%")
)

type GetImageRegistry func(reg string) string

func DeploymentCreator(getRegistry GetImageRegistry) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return DeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Spec.Replicas = &deploymentReplicas

			dep.Spec.Strategy = appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       &deploymentMaxSurge,
					MaxUnavailable: &deploymentMaxUnavailable,
				},
			}

			labels := map[string]string{"app": "container-linux-update-operator"}
			dep.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
			dep.Spec.Template.ObjectMeta.Labels = labels
			dep.Spec.Template.Spec.ServiceAccountName = ServiceAccountName

			// The operator should only run on ContainerLinux nodes
			dep.Spec.Template.Spec.NodeSelector = map[string]string{NodeSelectorLabelKey: NodeSelectorLabelValue}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "update-operator",
					Image:   getRegistry(resources.RegistryQuay) + "/coreos/container-linux-update-operator:v0.7.0",
					Command: []string{"/bin/update-operator"},
					Env: []corev1.EnvVar{
						{
							Name: "POD_NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									APIVersion: "v1",
									FieldPath:  "metadata.namespace",
								},
							},
						},
					},
				},
			}

			return dep, nil
		}
	}
}
