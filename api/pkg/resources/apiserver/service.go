package apiserver

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/nodeportproxy"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ExternalServiceCreator returns the function to reconcile the external API server service
func ExternalServiceCreator(exposeStrategy corev1.ServiceType) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return resources.ApiserverExternalServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			// Always set it to NodePort. Even when using exposeStrategy==LoadBalancer, we create
			// one LoadBalancer for two Services (APIServer and openVPN) and use the nodePortProxy in
			// namespaced mode to redirect the traffic to the right service depending on its port.
			// We use a nodePort Service because that gives us a concurrency-safe allocation mechanism
			// for a unique port
			se.Spec.Type = corev1.ServiceTypeNodePort
			if se.Annotations == nil {
				se.Annotations = map[string]string{}
			}
			if exposeStrategy != corev1.ServiceTypeNodePort && exposeStrategy != corev1.ServiceTypeLoadBalancer {
				return nil, fmt.Errorf("exposeStrategy on the cluster must be one of `NodePort` or `LoadBalancer, got %q`", exposeStrategy)
			}
			if exposeStrategy == corev1.ServiceTypeNodePort {
				se.Annotations["nodeport-proxy.k8s.io/expose"] = "true"
				delete(se.Annotations, nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey)
			} else {
				se.Annotations[nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey] = "true"
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
						TargetPort: intstr.FromInt(resources.ApiServerSecurePort),
					},
				}

				return se, nil
			}

			se.Spec.Ports[0].Name = "secure"
			se.Spec.Ports[0].Protocol = corev1.ProtocolTCP
			se.Spec.Ports[0].TargetPort = intstr.FromInt(resources.ApiServerSecurePort)

			return se, nil
		}
	}
}
