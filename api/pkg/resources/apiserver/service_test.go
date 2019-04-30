package apiserver

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestExternalServiceCreator(t *testing.T) {
	testCases := []struct {
		name                string
		inputServiceType    corev1.ServiceType
		inputService        *corev1.Service
		errExpected         bool
		expectedServiceType corev1.ServiceType
		annotationExpected  bool
	}{
		{
			name:             "Err when servicetype clusterIP",
			inputServiceType: corev1.ServiceTypeClusterIP,
			errExpected:      true,
		},
		{
			name:             "Err when servicetype ExternalName",
			inputServiceType: corev1.ServiceTypeExternalName,
			errExpected:      true,
		},
		{
			name:                "No err when servicetype NodePort",
			inputServiceType:    corev1.ServiceTypeNodePort,
			errExpected:         false,
			expectedServiceType: corev1.ServiceTypeNodePort,
			annotationExpected:  true,
		},
		{
			name:             "No err when servicetype LoadBalancer",
			inputServiceType: corev1.ServiceTypeLoadBalancer,
			inputService: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"nodeport-proxy.k8s.io/expose": "true",
					},
				},
			},
			errExpected:         false,
			expectedServiceType: corev1.ServiceTypeLoadBalancer,
		},
		{
			name:             "Servicetype LoadBalancer doesnt get upated",
			inputServiceType: corev1.ServiceTypeNodePort,
			inputService: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
				},
			},
			errExpected:         false,
			expectedServiceType: corev1.ServiceTypeLoadBalancer,
		},
		{
			name:             "Servicetype NodePort doesnt get upated",
			inputServiceType: corev1.ServiceTypeLoadBalancer,
			inputService: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
				},
			},
			errExpected:         false,
			expectedServiceType: corev1.ServiceTypeNodePort,
			annotationExpected:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.inputService == nil {
				tc.inputService = &corev1.Service{}
			}
			_, creator := ExternalServiceCreator(tc.inputServiceType)()
			service, err := creator(tc.inputService)
			if err != nil {
				if tc.errExpected {
					return
				}
				t.Fatalf("Unexpected error: %v", err)
			}
			if err != nil {
				return
			}

			if service.Spec.Type != tc.expectedServiceType {
				t.Errorf("Expected service type to be %q but was %q", tc.expectedServiceType, service.Spec.Type)
			}

			if (service.Annotations["nodeport-proxy.k8s.io/expose"] != "") != tc.annotationExpected {
				t.Errorf("Expected annotation 'nodeport-proxy.k8s.io/expose' to exist=%t but had value %q",
					tc.annotationExpected, service.Annotations["nodeport-proxy.k8s.io/expose"])
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
					Type: corev1.ServiceTypeLoadBalancer,
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
			name: "Reconfigured LoadBalancer service, port 443",
			inService: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
					Ports: []corev1.ServicePort{
						{
							Name:       "my-fancy-port",
							Port:       int32(8080),
							TargetPort: intstr.FromInt(8080),
							Protocol:   corev1.ProtocolUDP,
						},
					},
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
