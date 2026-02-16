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

package test

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

type NamespacedName types.NamespacedName

type ObjectBuilder metav1.ObjectMeta

func (b *ObjectBuilder) WithResourceVersion(rv string) *ObjectBuilder {
	b.ResourceVersion = rv
	return b
}

func (b *ObjectBuilder) WithLabel(key, value string) *ObjectBuilder {
	if b.Labels == nil {
		b.Labels = map[string]string{}
	}
	b.Labels[key] = value
	return b
}

func (b *ObjectBuilder) WithAnnotation(key, value string) *ObjectBuilder {
	if b.Annotations == nil {
		b.Annotations = map[string]string{}
	}
	b.Annotations[key] = value
	return b
}

func (b *ObjectBuilder) WithCreationTimestamp(time time.Time) *ObjectBuilder {
	b.CreationTimestamp = metav1.NewTime(time)
	return b
}

// ServiceBuilder is a builder providing a fluent API for v1.Service creation.
type ServiceBuilder struct {
	ObjectBuilder

	serviceType  corev1.ServiceType
	servicePorts []corev1.ServicePort
	selector     map[string]string
}

// NewServiceBuilder returns a ServiceBuilder to be used to build a v1.Service
// with name and namespace given in input.
func NewServiceBuilder(nn NamespacedName) *ServiceBuilder {
	return &ServiceBuilder{
		ObjectBuilder: ObjectBuilder{
			Name:      nn.Name,
			Namespace: nn.Namespace,
		},
		serviceType: corev1.ServiceTypeClusterIP,
	}
}

func (b *ServiceBuilder) WithLabel(key, value string) *ServiceBuilder {
	_ = b.ObjectBuilder.WithLabel(key, value)
	return b
}

func (b *ServiceBuilder) WithAnnotation(key, value string) *ServiceBuilder {
	_ = b.ObjectBuilder.WithAnnotation(key, value)
	return b
}

func (b *ServiceBuilder) WithCreationTimestamp(time time.Time) *ServiceBuilder {
	_ = b.ObjectBuilder.WithCreationTimestamp(time)
	return b
}

func (b *ServiceBuilder) WithServiceType(serviceType corev1.ServiceType) *ServiceBuilder {
	b.serviceType = serviceType
	return b
}

func (b *ServiceBuilder) WithSelector(selector map[string]string) *ServiceBuilder {
	b.selector = selector
	return b
}

func (b *ServiceBuilder) WithServicePorts(sp ...corev1.ServicePort) *ServiceBuilder {
	b.servicePorts = append(b.servicePorts, sp...)
	return b
}

func (b *ServiceBuilder) WithServicePort(
	name string,
	port int32,
	nodePort int32,
	targetPort intstr.IntOrString,
	protocol corev1.Protocol) *ServiceBuilder {
	return b.WithServicePorts(corev1.ServicePort{
		Name:       name,
		NodePort:   nodePort,
		Port:       port,
		TargetPort: targetPort,
		Protocol:   protocol,
	})
}

func (b ServiceBuilder) Build() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta(b.ObjectBuilder),
		Spec: corev1.ServiceSpec{
			Type:     b.serviceType,
			Ports:    b.servicePorts,
			Selector: b.selector,
		},
	}
}

// EndpointSliceBuilder is a builder providing a fluent API for discoveryv1.EndpointSlice creation.
type EndpointSliceBuilder struct {
	ObjectBuilder

	addressType discoveryv1.AddressType
	endpoints   []discoveryv1.Endpoint
	ports       []discoveryv1.EndpointPort
	serviceName string
}

// NewEndpointSliceBuilder returns an EndpointSliceBuilder to be used to build a
// discoveryv1.EndpointSlice with name, namespace, and service name given in input.
func NewEndpointSliceBuilder(nn NamespacedName, serviceName string) *EndpointSliceBuilder {
	return &EndpointSliceBuilder{
		ObjectBuilder: ObjectBuilder{
			Name:      nn.Name,
			Namespace: nn.Namespace,
		},
		addressType: discoveryv1.AddressTypeIPv4,
		serviceName: serviceName,
	}
}

func (b *EndpointSliceBuilder) WithResourceVersion(rv string) *EndpointSliceBuilder {
	_ = b.ObjectBuilder.WithResourceVersion(rv)
	return b
}

func (b *EndpointSliceBuilder) WithAddressType(addressType discoveryv1.AddressType) *EndpointSliceBuilder {
	b.addressType = addressType
	return b
}

// WithEndpoint adds an endpoint with the given ready status and addresses.
func (b *EndpointSliceBuilder) WithEndpoint(ready bool, addresses ...string) *EndpointSliceBuilder {
	b.endpoints = append(b.endpoints, discoveryv1.Endpoint{
		Addresses: addresses,
		Conditions: discoveryv1.EndpointConditions{
			Ready: ptr.To(ready),
		},
	})
	return b
}

// WithNotReadyEndpoint adds an endpoint that is not ready with the given addresses.
func (b *EndpointSliceBuilder) WithNotReadyEndpoint(addresses ...string) *EndpointSliceBuilder {
	return b.WithEndpoint(false, addresses...)
}

// WithPort adds a port to the EndpointSlice.
func (b *EndpointSliceBuilder) WithPort(name string, port int32, protocol corev1.Protocol) *EndpointSliceBuilder {
	b.ports = append(b.ports, discoveryv1.EndpointPort{
		Name:     ptr.To(name),
		Port:     ptr.To(port),
		Protocol: ptr.To(protocol),
	})
	return b
}

func (b EndpointSliceBuilder) Build() *discoveryv1.EndpointSlice {
	return &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.Name,
			Namespace: b.Namespace,
			Labels: map[string]string{
				discoveryv1.LabelServiceName: b.serviceName,
			},
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "discovery.k8s.io/v1",
			Kind:       "EndpointSlice",
		},
		AddressType: b.addressType,
		Endpoints:   b.endpoints,
		Ports:       b.ports,
	}
}
