package apiserver

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/nodeportproxy"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// InternalServiceCreator returns the function to reconcile the internal API server service
func InternalServiceCreator(apiserverPort int32) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return resources.ApiserverInternalServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			// Validation checks if port > 1 and < 65535 and because we do not have a DAG yet to properly
			// model our dependencies, this service gets created at a point in time where we don't know
			// the port yet
			if apiserverPort == 0 {
				apiserverPort = 1
			}
			se.Name = resources.ApiserverInternalServiceName
			se.Labels = resources.BaseAppLabel(name, nil)
			if se.Annotations == nil {
				se.Annotations = map[string]string{}
			}
			// We always set this because we don't know the expose strategy and don't need to
			// This has no effect when we expose via NodePorts
			se.Annotations[nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey] = "true"

			if se.Spec.Type == "" {
				se.Spec.Type = corev1.ServiceTypeNodePort
			}
			se.Spec.Selector = map[string]string{
				resources.AppLabelKey: name,
			}
			se.Spec.Ports = []corev1.ServicePort{
				{
					Name:       "kube-apiserver",
					Port:       apiserverPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(int(apiserverPort)),
				},
			}
			return se, nil
		}
	}
}

// ExternalServiceCreator returns the function to reconcile the external API server service
func ExternalServiceCreator(serviceType corev1.ServiceType) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return resources.ApiserverExternalServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			if serviceType != corev1.ServiceTypeNodePort && serviceType != corev1.ServiceTypeLoadBalancer {
				return nil, fmt.Errorf("service.spec.type must be either NodePort or LoadBalanancer, was %q", serviceType)
			}
			se.Name = resources.ApiserverExternalServiceName
			if se.Spec.Type == "" {
				se.Spec.Type = serviceType
			}
			if se.Annotations == nil {
				se.Annotations = map[string]string{}
			}
			if se.Spec.Type == corev1.ServiceTypeNodePort {
				se.Annotations["nodeport-proxy.k8s.io/expose"] = "true"
			} else {
				delete(se.Annotations, "nodeport-proxy.k8s.io/expose")
			}

			se.Spec.Selector = map[string]string{
				resources.AppLabelKey: name,
			}

			if len(se.Spec.Ports) == 0 {
				se.Spec.Ports = []corev1.ServicePort{
					{
						Name:       "secure",
						Port:       443,
						Protocol:   corev1.ProtocolTCP,
						TargetPort: intstr.FromInt(443),
					},
				}

				return se, nil
			}

			se.Spec.Ports[0].Name = "secure"
			se.Spec.Ports[0].Protocol = corev1.ProtocolTCP

			if se.Spec.Type == corev1.ServiceTypeLoadBalancer {
				se.Spec.Ports[0].Port = int32(443)
				se.Spec.Ports[0].TargetPort = intstr.FromInt(443)
			} else {
				se.Spec.Ports[0].Port = se.Spec.Ports[0].NodePort
				se.Spec.Ports[0].TargetPort = intstr.FromInt(int(se.Spec.Ports[0].NodePort))
			}

			return se, nil
		}
	}
}
