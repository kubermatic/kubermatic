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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestServiceReconcilerRequiresExposeStrategy(t *testing.T) {
	testCases := []struct {
		name            string
		exposeStrategy  kubermaticv1.ExposeStrategy
		internalService string
		errExpected     bool
	}{
		{
			name:           "NodePort is accepted as exposeStrategy",
			exposeStrategy: kubermaticv1.ExposeStrategyNodePort,
		},
		{
			name:           "LoadBalancer is accepted as exposeStrategy",
			exposeStrategy: kubermaticv1.ExposeStrategyNodePort,
		},
		{
			name:        "Empty is not accepted as exposeStrategy",
			errExpected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, creator := ServiceReconciler(tc.exposeStrategy, tc.internalService, nil)()
			_, err := creator(&corev1.Service{})
			if (err != nil) != tc.errExpected {
				t.Errorf("Expected err: %t, but got err %v", tc.errExpected, err)
			}
		})
	}
}

func TestServiceReconcilerSetsPort(t *testing.T) {
	lbServiceType := corev1.ServiceTypeLoadBalancer

	testCases := []struct {
		name                string
		exposeStrategy      kubermaticv1.ExposeStrategy
		internalService     string
		inService           *corev1.Service
		expectedPort        int32
		expectedTargetPort  intstr.IntOrString
		expectedServiceType *corev1.ServiceType
	}{
		{
			name:           "Empty LoadBalancer service, port 443",
			exposeStrategy: kubermaticv1.ExposeStrategyLoadBalancer,
			inService: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
				},
			},
			expectedPort:       int32(443),
			expectedTargetPort: intstr.FromInt(6443),
		},
		{
			name:           "Expose strategy LoadBalancer with service type LoadBalancer",
			exposeStrategy: kubermaticv1.ExposeStrategyLoadBalancer,
			inService: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
				},
			},
			expectedPort:        int32(443),
			expectedTargetPort:  intstr.FromInt(6443),
			expectedServiceType: &lbServiceType,
		},
		{
			name:           "Empty NodePort service, port 443",
			exposeStrategy: kubermaticv1.ExposeStrategyNodePort,
			inService: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
				},
			},
			expectedPort:       int32(443),
			expectedTargetPort: intstr.FromInt(6443),
		},
		{
			name:           "NodePort service with allocation, allocation is used everywhere",
			exposeStrategy: kubermaticv1.ExposeStrategyNodePort,
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
			expectedPort:       int32(443),
			expectedTargetPort: intstr.FromInt(32000),
		},
		{
			name:           "With tunneling strategy KAS uses 6443 as secure port",
			exposeStrategy: kubermaticv1.ExposeStrategyTunneling,
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
			expectedPort:       int32(443),
			expectedTargetPort: intstr.FromInt(6443),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, creator := ServiceReconciler(tc.exposeStrategy, tc.internalService, tc.expectedServiceType)()
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

			if tc.expectedServiceType != nil {
				if svc.Spec.Type != *tc.expectedServiceType {
					t.Errorf("Expected service type to be %q but was %q", *tc.expectedServiceType, svc.Spec.Type)
				}
			}
		})
	}
}
