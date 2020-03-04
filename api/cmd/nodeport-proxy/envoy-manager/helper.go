package main

import (
	"fmt"

	envoycorev2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// hasher returns a static string so changes apply to all nodes
type hasher struct{}

// ID returns a static string as all envoy nodes should receive the same config
func (h hasher) ID(node *envoycorev2.Node) string {
	return envoyNodeName
}

func ServiceKey(service *corev1.Service) string {
	return fmt.Sprintf("%s/%s", service.Namespace, service.Name)
}

func PodIsReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func getMatchingPodPort(servicePort corev1.ServicePort, pod *corev1.Pod) int32 {
	if servicePort.TargetPort.Type == intstr.Int {
		return servicePort.TargetPort.IntVal
	}

	for _, container := range pod.Spec.Containers {
		for _, podPort := range container.Ports {
			if servicePort.TargetPort.Type == intstr.String && servicePort.TargetPort.StrVal == podPort.Name {
				return podPort.ContainerPort
			}
		}
	}

	return 0
}
