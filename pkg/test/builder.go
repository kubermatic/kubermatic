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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
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

func (b *ObjectBuilder) WithGeneration(gen int64) *ObjectBuilder {
	b.Generation = gen
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

// EndpointsBuilder is a builder providing a fluent API for v1.Endpoints
// creation.
type EndpointsBuilder struct {
	ObjectBuilder

	epsSubsets []corev1.EndpointSubset
}

// NewEndpointsBuilder returns a ServiceBuilder to be used to build a
// v1.Endpoints with name and namespace given in input.
func NewEndpointsBuilder(nn NamespacedName) *EndpointsBuilder {
	return &EndpointsBuilder{
		ObjectBuilder: ObjectBuilder{
			Name:      nn.Name,
			Namespace: nn.Namespace,
		},
	}
}

func (b *EndpointsBuilder) WithResourceVersion(rs string) *EndpointsBuilder {
	_ = b.ObjectBuilder.WithResourceVersion(rs)
	return b
}

// EndpointsSubsetBuilder starts the creation of an Endpoints Subset, the creation
// must me terminated with a call to BuildEndpointSubset, after ports and
// addresses are added.
// nolint:golint
func (b *EndpointsBuilder) EndpointsSubsetBuilder() *epsSubsetBuilder {
	return &epsSubsetBuilder{eb: b}
}

func (b *EndpointsBuilder) Build() *corev1.Endpoints {
	return &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta(b.ObjectBuilder),
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Endpoints",
		},
		Subsets: b.epsSubsets,
	}
}

type epsSubsetBuilder struct {
	// Used to come back to main builder
	eb *EndpointsBuilder

	epsAddresses         []corev1.EndpointAddress
	epsNotReadyAddresses []corev1.EndpointAddress
	epsPorts             []corev1.EndpointPort
}

func (b *epsSubsetBuilder) WithReadyAddressIP(ip string) *epsSubsetBuilder {
	return b.WithReadyAddresses(corev1.EndpointAddress{IP: ip})
}

func (b *epsSubsetBuilder) WithNotReadyAddressIP(ip string) *epsSubsetBuilder {
	return b.WithNotReadyAddresses(corev1.EndpointAddress{IP: ip})
}

func (b *epsSubsetBuilder) WithReadyAddresses(eas ...corev1.EndpointAddress) *epsSubsetBuilder {
	b.epsAddresses = append(b.epsAddresses, eas...)
	return b
}

func (b *epsSubsetBuilder) WithNotReadyAddresses(eas ...corev1.EndpointAddress) *epsSubsetBuilder {
	b.epsNotReadyAddresses = append(b.epsNotReadyAddresses, eas...)
	return b
}

func (b *epsSubsetBuilder) WithEndpointPorts(eps ...corev1.EndpointPort) *epsSubsetBuilder {
	b.epsPorts = append(b.epsPorts, eps...)
	return b
}

func (b *epsSubsetBuilder) WithEndpointPort(
	name string,
	port int32,
	protocol corev1.Protocol) *epsSubsetBuilder {
	return b.WithEndpointPorts(corev1.EndpointPort{
		Name:     name,
		Port:     port,
		Protocol: protocol,
	})
}

// BuildEndpointSubset concludes the creation of the Subset and returns the
// EndpointsBuilder to start the creation of a new Subset or create the
// Endpoints with Build method.
func (b *epsSubsetBuilder) BuildEndpointSubset(eps ...corev1.EndpointPort) *EndpointsBuilder {
	b.epsPorts = append(b.epsPorts, eps...)
	b.eb.epsSubsets = append(b.eb.epsSubsets, corev1.EndpointSubset{
		Addresses:         b.epsAddresses,
		NotReadyAddresses: b.epsNotReadyAddresses,
		Ports:             b.epsPorts,
	})
	return b.eb
}

// DeploymentBuilder is a builder providing a fluent API for v1.Deployment
// creation.
type DeploymentBuilder struct {
	ObjectBuilder

	replicas int32
	spec     corev1.PodSpec
	status   appsv1.DeploymentStatus
}

// NewEndpointsBuilder returns a ServiceBuilder to be used to build a
// v1.Deployment with name and namespace given in input.
func NewDeploymentBuilder(nn NamespacedName) *DeploymentBuilder {
	return &DeploymentBuilder{
		ObjectBuilder: ObjectBuilder{
			Name:      nn.Name,
			Namespace: nn.Namespace,
		},
	}
}

func (db *DeploymentBuilder) WithGeneration(gen int64) *DeploymentBuilder {
	_ = db.ObjectBuilder.WithGeneration(gen)
	return db
}

func (db *DeploymentBuilder) WithReplicas(replicas int32) *DeploymentBuilder {
	db.replicas = replicas
	return db
}

func (db *DeploymentBuilder) WithStatus(s appsv1.DeploymentStatus) *DeploymentBuilder {
	db.status = s
	return db
}

func (db *DeploymentBuilder) WithRolloutInProgress() *DeploymentBuilder {
	db.status = appsv1.DeploymentStatus{ObservedGeneration: 2, Replicas: 2, AvailableReplicas: 2, UpdatedReplicas: 1}
	db.ObjectBuilder.Generation = 2
	db.replicas = 2
	return db
}

func (db *DeploymentBuilder) WithRolloutComplete() *DeploymentBuilder {
	db.status = appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 1, AvailableReplicas: 1, UpdatedReplicas: 1}
	db.ObjectBuilder.Generation = 1
	db.replicas = 1
	return db
}

// ContainerBuilder starts the creation of a new Container, the creation
// must me terminated with a call to BuildContainer.
// nolint:golint
func (db *DeploymentBuilder) ContainerBuilder() *deploymentContainerBuilder {
	return &deploymentContainerBuilder{
		db: db,
		containerBuilder: containerBuilder{
			psb: &podSpecBuilder{},
		},
	}
}

func (db *DeploymentBuilder) Build() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta(db.ObjectBuilder),
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &db.replicas,
			Template: corev1.PodTemplateSpec{
				Spec: db.spec,
			},
		},
		Status: db.status,
	}
}

type deploymentContainerBuilder struct {
	containerBuilder
	db *DeploymentBuilder
}

func (dcb *deploymentContainerBuilder) WithName(name string) *deploymentContainerBuilder {
	_ = dcb.containerBuilder.WithName(name)
	return dcb
}

func (dcb *deploymentContainerBuilder) WithImage(image string) *deploymentContainerBuilder {
	_ = dcb.containerBuilder.WithImage(image)
	return dcb
}

func (dcb *deploymentContainerBuilder) WithCommand(command ...string) *deploymentContainerBuilder {
	_ = dcb.containerBuilder.WithCommand(command...)
	return dcb
}

func (dcb *deploymentContainerBuilder) WithArgs(command ...string) *deploymentContainerBuilder {
	_ = dcb.containerBuilder.WithArgs(command...)
	return dcb
}

func (dcb *deploymentContainerBuilder) AddContainer() *DeploymentBuilder {
	ps := dcb.containerBuilder.AddContainer().Build()
	dcb.db.spec.Containers = append(dcb.db.spec.Containers, ps.Containers...)
	return dcb.db
}

type podSpecBuilder struct {
	containers []corev1.Container
}

func (psb *podSpecBuilder) WithContainer(c corev1.Container) *podSpecBuilder {
	psb.containers = append(psb.containers, c)
	return psb
}

type containerBuilder struct {
	psb *podSpecBuilder

	name    string
	image   string
	command []string
	args    []string
}

func (cb *containerBuilder) WithName(name string) *containerBuilder {
	cb.name = name
	return cb
}

func (cb *containerBuilder) WithImage(image string) *containerBuilder {
	cb.image = image
	return cb
}

func (cb *containerBuilder) WithCommand(command ...string) *containerBuilder {
	cb.command = command
	return cb
}

func (cb *containerBuilder) WithArgs(args ...string) *containerBuilder {
	cb.args = args
	return cb
}

func (cb *containerBuilder) AddContainer() *podSpecBuilder {
	_ = cb.psb.WithContainer(corev1.Container{
		Name:    cb.name,
		Image:   cb.image,
		Args:    cb.args,
		Command: cb.command,
	})
	return cb.psb
}

func (psb *podSpecBuilder) Build() *corev1.PodSpec {
	return &corev1.PodSpec{
		Containers: psb.containers,
	}
}
