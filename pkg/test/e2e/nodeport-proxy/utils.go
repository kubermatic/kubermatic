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

package nodeportproxy

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// function used to extract port info
type extractPortFunc func(corev1.ServicePort) int32

func extractPort(svc *corev1.Service, extract extractPortFunc) sets.Int32 {
	res := sets.NewInt32()
	for _, p := range svc.Spec.Ports {
		val := extract(p)
		if val != 0 {
			res.Insert(val)
		}
	}
	return res
}

// ExtractNodePorts returns the set of node ports extracted from the given
// Service.
func ExtractNodePorts(svc *corev1.Service) sets.Int32 {
	return extractPort(svc,
		func(p corev1.ServicePort) int32 { return p.NodePort })
}

// ExtractPorts returns the set of ports extracted from the given
// Service.
func ExtractPorts(svc *corev1.Service) sets.Int32 {
	return extractPort(svc,
		func(p corev1.ServicePort) int32 { return p.Port })
}

// FindExposingNodePort returns the node port associated to the given target
// port.
func FindExposingNodePort(svc *corev1.Service, targetPort int32) int32 {
	for _, p := range svc.Spec.Ports {
		if p.TargetPort.IntVal == targetPort {
			return p.NodePort
		}
	}
	return 0
}
