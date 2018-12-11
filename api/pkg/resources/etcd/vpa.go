package etcd

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	autoscalingv1beta "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta1"
)

// VerticalPodAutoscaler returns a VerticalPodAutoscaler which can be applied to the etcd StatefulSet
func VerticalPodAutoscaler(data *resources.TemplateData, existing *autoscalingv1beta.VerticalPodAutoscaler) (*autoscalingv1beta.VerticalPodAutoscaler, error) {
	baseLabels := getBasePodLabels(data.Cluster())

	return resources.GetVerticalPodAutoscaler(name, baseLabels)(data, existing)
}
