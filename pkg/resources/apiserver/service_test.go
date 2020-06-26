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

package apiserver

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestExternalServiceCreatorRequiresExposeStrategy(t *testing.T) {
	testCases := []struct {
		name           string
		exposeStrategy corev1.ServiceType
		errExpected    bool
	}{
		{
			name:           "NodePort is accepted as exposeStrategy",
			exposeStrategy: corev1.ServiceTypeNodePort,
		},
		{
			name:           "LoadBalancer is accepted as exposeStrategy",
			exposeStrategy: corev1.ServiceTypeLoadBalancer,
		},
		{
			name:        "Empty is not accepted as exposeStrategy",
			errExpected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, creator := ExternalServiceCreator(tc.exposeStrategy)()
			_, err := creator(&corev1.Service{})
			if (err != nil) != tc.errExpected {
				t.Errorf("Expected err: %t, but got err %v", tc.errExpected, err)
			}
		})
	}
}

func TestExternalServiceCreatorSetsPort(t *testing.T) {
	testCases := []struct {
		name               string
		inService          *corev1.Service
		expectedPort       int32
		expectedTargetPort intstr.IntOrString
	}{
		{
			name: "Empty LoadBalancer service, port 443",
			inService: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
				},
			},
			expectedPort:       int32(443),
			expectedTargetPort: intstr.FromInt(443),
		},
		{
			name: "Empty NodePort service, port 443",
			inService: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
				},
			},
			expectedPort:       int32(443),
			expectedTargetPort: intstr.FromInt(443),
		},
		{
			name: "NodePort service with allocation, allocation is used everywhere",
			inService: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
					Ports: []corev1.ServicePort{
						{
							Name:       "my-fancy-port",
							Port:       int32(8080),
							TargetPort: intstr.FromInt(8080),
							Protocol:   corev1.ProtocolUDP,
							NodePort:   int32(32000),
						},
					},
				},
			},
			expectedPort:       int32(32000),
			expectedTargetPort: intstr.FromInt(32000),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, creator := ExternalServiceCreator(tc.inService.Spec.Type)()
			svc, err := creator(tc.inService)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if portlen := len(svc.Spec.Ports); portlen != 1 {
				t.Fatalf("Expected exactly one port, got %d", portlen)
			}
			if svc.Spec.Ports[0].Name != "secure" {
				t.Errorf("expected port name to be 'secure', was %q", svc.Spec.Ports[0].Name)
			}
			if svc.Spec.Ports[0].Protocol != corev1.ProtocolTCP {
				t.Errorf("Expected error to be %q but was %q", corev1.ProtocolTCP, svc.Spec.Ports[0].Protocol)
			}
			if svc.Spec.Ports[0].Port != tc.expectedPort {
				t.Errorf("Expected port to be %d but was %d", tc.expectedPort, svc.Spec.Ports[0].Port)
			}
			if svc.Spec.Ports[0].TargetPort.String() != tc.expectedTargetPort.String() {
				t.Errorf("Expected targetPort to be %q but was %q", tc.expectedTargetPort.String(), svc.Spec.Ports[0].TargetPort.String())
			}
		})
	}
}
