package resources

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	autoscalingv1beta1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta1"
)

// GetVerticalPodAutoscaler returns a function to create a generic VerticalPodAutoscaler with the given name and selector
func GetVerticalPodAutoscaler(name string, labels map[string]string) VerticalPodAutoscalerCreator {
	return func(existing *autoscalingv1beta1.VerticalPodAutoscaler) (*autoscalingv1beta1.VerticalPodAutoscaler, error) {
		var pdb *autoscalingv1beta1.VerticalPodAutoscaler
		if existing != nil {
			pdb = existing
		} else {
			pdb = &autoscalingv1beta1.VerticalPodAutoscaler{}
		}

		pdb.Name = name

		pdb.Spec = autoscalingv1beta1.VerticalPodAutoscalerSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
		}

		return pdb, nil
	}
}
