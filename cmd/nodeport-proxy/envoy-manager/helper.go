/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"

	envoycorev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// hasher returns a static string so changes apply to all nodes
type hasher struct{}

// ID returns a static string as all envoy nodes should receive the same config
func (h hasher) ID(node *envoycorev3.Node) string {
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
