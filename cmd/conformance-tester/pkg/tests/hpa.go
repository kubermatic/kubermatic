package tests

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TestHPA verifies that the Horizontal Pod Autoscaler (HPA) can scale a Deployment
// based on CPU utilization. It creates a Deployment and HPA, simulates CPU load,
// waits for the HPA to scale up the Deployment, and cleans up resources afterwards.
func TestHPA(ctx context.Context, client ctrlruntimeclient.Client) error {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hpa-test",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "hpa-test"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "hpa-test"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:    "busybox",
						Image:   "busybox",
						Command: []string{"sh", "-c", "while true; do :; done"},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("10m"),
								corev1.ResourceMemory: resource.MustParse("16Mi"),
							},
						},
					}},
				},
			},
		},
	}
	if err := client.Create(ctx, deployment); err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}
	defer client.Delete(ctx, deployment)

	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hpa-test",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       "hpa-test",
				APIVersion: "apps/v1",
			},
			MinReplicas: pointer.Int32Ptr(1),
			MaxReplicas: 3,
			Metrics: []autoscalingv2.MetricSpec{{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: corev1.ResourceCPU,
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.UtilizationMetricType,
						AverageUtilization: pointer.Int32Ptr(1),
					},
				},
			}},
		},
	}
	if err := client.Create(ctx, hpa); err != nil {
		return fmt.Errorf("failed to create HPA: %w", err)
	}
	defer client.Delete(ctx, hpa)

	// Wait for HPA to scale up (polling for increased replicas)
	err := wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
		dep := &appsv1.Deployment{}
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: "hpa-test", Namespace: "default"}, dep); err != nil {
			return false, err
		}
		if dep.Status.Replicas > 1 {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("HPA did not scale up deployment: %w", err)
	}

	return nil
}
