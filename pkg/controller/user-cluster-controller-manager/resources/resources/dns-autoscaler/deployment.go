package dnsautoscaler

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
		resources.CoreDNSDeploymentName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("10Mi"),
				corev1.ResourceCPU:    resource.MustParse("20m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("100Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
		},
	}
)

// DeploymentCreator returns the function to create and update the dns autoscaler deployment.
func DeploymentCreator() reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.DNSAutoscalerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.DNSAutoscalerDeploymentName
			dep.Namespace = metav1.NamespaceSystem
			dep.Labels = resources.BaseAppLabels(resources.DNSAutoscalerDeploymentName, nil)

			dep.Spec.Replicas = resources.Int32(1)
			// The Selector is immutable, so we don't change it if it's set. This happens in upgrade cases
			// where dns autoscaler is switched from a manifest based addon to a user-cluster-controller-manager resource
			if dep.Spec.Selector == nil {
				dep.Spec.Selector = &metav1.LabelSelector{
					MatchLabels: resources.BaseAppLabels(resources.DNSAutoscalerDeploymentName,
						map[string]string{"app.kubernetes.io/name": "dns-autoscaler"}),
				}
			}

			// has to be the same as the selector
			if dep.Spec.Template.ObjectMeta.Labels == nil {
				dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
					Labels: resources.BaseAppLabels(resources.DNSAutoscalerDeploymentName,
						map[string]string{"app.kubernetes.io/name": "dns-autoscaler"}),
				}
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "autoscaler",
					Image:   fmt.Sprintf("%s/cluster-proportional-autoscaler-amd64:1.6.0", resources.RegistryK8SGCR),
					Command: []string{"/cluster-proportional-autoscaler", "--namespace=kube-system", "--configmap=dns-autoscaler", "--target=deployment/coredns", "--logtostderr=true", "--v=2"},
				},
			}

			err := resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defaultResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}

			dep.Spec.Template.Spec.ServiceAccountName = resources.DNSAutoscalerServicaAccountName

			return dep, nil
		}
	}
}
