package etcd

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	autoscalingv1beta "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta1"
)

// VerticalPodAutoscalerCreator returns the function to reconcile the etcd VerticalPodAutoscaler resource
func VerticalPodAutoscalerCreator(data *resources.TemplateData) resources.VerticalPodAutoscalerCreator {
	return func(existing *autoscalingv1beta.VerticalPodAutoscaler) (*autoscalingv1beta.VerticalPodAutoscaler, error) {
		baseLabels := getBasePodLabels(data.Cluster())

		return resources.GetVerticalPodAutoscaler(name, baseLabels)(existing)
	}

}
