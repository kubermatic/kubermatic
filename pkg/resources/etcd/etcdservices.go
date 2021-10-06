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

package etcd

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type serviceCreatorData interface {
	Cluster() *kubermaticv1.Cluster
	GetClusterRef() metav1.OwnerReference
}

// ServiceCreator returns the function to reconcile the etcd service
func ServiceCreator(data serviceCreatorData) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return resources.EtcdServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			se.Name = resources.EtcdServiceName
			se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
			se.Spec.ClusterIP = "None"
			se.Spec.PublishNotReadyAddresses = true
			se.Spec.Selector = map[string]string{
				resources.AppLabelKey: name,
				"cluster":             data.Cluster().Name,
			}
			se.Spec.Ports = []corev1.ServicePort{
				{
					Name:       "client",
					Port:       2379,
					TargetPort: intstr.FromInt(2379),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "peer",
					Port:       2380,
					TargetPort: intstr.FromInt(2380),
					Protocol:   corev1.ProtocolTCP,
				},
			}

			return se, nil
		}
	}
}

// GetClientEndpoints returns the slice with the etcd endpoints for client communication
func GetClientEndpoints(namespace string) []string {
	var endpoints []string
	for i := 0; i < 3; i++ {
		// Pod DNS name
		serviceDNSName := resources.GetAbsoluteServiceDNSName(resources.EtcdServiceName, namespace)
		absolutePodDNSName := fmt.Sprintf("https://etcd-%d.%s:2379", i, serviceDNSName)
		endpoints = append(endpoints, absolutePodDNSName)
	}
	return endpoints
}
