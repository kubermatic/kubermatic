package apiserver

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			if (err != nil) != tc.errExpected {
				t.Fatalf("Expected err=%t but got err=%v", tc.errExpected, err)
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
