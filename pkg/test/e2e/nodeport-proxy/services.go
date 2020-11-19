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

package nodeport_proxy

import (
	"context"
	"fmt"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Based on:
// https://github.com/mgdevstack/kubernetes/blob/9eced040142454a20255ae323279a38dc6b2bc1a/test/e2e/framework/service/jig.go#L60
// ServiceJig is a jig to help service testing.
type ServiceJig struct {
	Log       *zap.SugaredLogger
	Client    ctrlclient.Client
	Namespace string

	Services    []*corev1.Service
	ServicePods map[string][]string
}

// CreateNodePortService deploys a service and the associated pods.
func (n *ServiceJig) CreateNodePortService(name string, nodePort int32, numPods int32, annotations map[string]string) (*corev1.Service, error) {
	// Create the namespace to host the service
	ns := n.newNamespaceTemplate()
	n.Log.Debugw("Creating namespace", "service", ns)
	if err := n.Client.Create(context.TODO(), ns); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, err
		}
	} else {
		// Set back namespace name in case it was generated
		n.Namespace = ns.Name
		n.Log.Debugw("Setting generated namespace to SeriviceJig", "namespace", n.Namespace)
	}

	labels := map[string]string{"apps": name}
	// Finally create service
	svc := n.newServiceTemplate(name, n.Namespace, annotations, labels)
	svc.Spec.Type = corev1.ServiceTypeNodePort
	svc.Spec.Ports[0].NodePort = nodePort
	n.Log.Debugw("Creating nodeport service", "service", svc)
	if err := n.Client.Create(context.TODO(), svc); err != nil {
		errors.Wrap(err, "failed to create service of type nodeport")
	}
	Expect(svc).NotTo(BeNil())
	Expect(ExtractNodePorts(svc)).To(HaveLen(1))

	// Create service pods
	rc := n.newRCTemplate(name, n.Namespace, numPods, labels)
	n.Log.Debugw("Creating replication controller", "rc", rc)
	if err := n.Client.Create(context.TODO(), rc); err != nil {
		return nil, err
	}

	// Wait for the pod to be ready
	pods, err := WaitForPodsCreated(n.Client, int(*rc.Spec.Replicas), rc.Namespace, svc.Spec.Selector)
	if err != nil {
		return svc, errors.Wrap(err, "error occurred while waiting for pods to be ready")
	}
	if !CheckPodsRunningReady(n.Client, n.Namespace, pods, podReadinessTimeout) {
		return svc, fmt.Errorf("timeout waiting for %d pods to be ready", len(pods))
	}
	if n.ServicePods == nil {
		n.ServicePods = map[string][]string{}
	}
	n.ServicePods[svc.Name] = pods
	return svc, err
}

func (n *ServiceJig) newNamespaceTemplate() *corev1.Namespace {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: n.Namespace,
		},
	}
	if n.Namespace == "" {
		ns.ObjectMeta.GenerateName = "np-proxy-test-"
	}
	return ns
}

func (n *ServiceJig) newServiceTemplate(name, namespace string,
	annotations, labels map[string]string) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}
	return svc
}

func (n *ServiceJig) newRCTemplate(name, namespace string, replicas int32, labels map[string]string) *corev1.ReplicationController {
	rc := &corev1.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ReplicationControllerSpec{
			Replicas: &replicas,
			Selector: labels,
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "netexec",
							Image: NetexecImage,
							Args:  []string{"--http-port=8080", "--udp-port=8080"},
							ReadinessProbe: &corev1.Probe{
								PeriodSeconds: 3,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Port: intstr.FromInt(8080),
										Path: "/hostName",
									},
								},
							},
						},
					},
					TerminationGracePeriodSeconds: new(int64),
				},
			},
		},
	}
	return rc
}

// CleanUp removes the resources.
func (n *ServiceJig) CleanUp() error {
	ns := corev1.Namespace{}
	if err := n.Client.Get(context.TODO(), types.NamespacedName{Name: n.Namespace}, &ns); err != nil {
		return err
	}
	n.Log.Infow("deleting test namespace", "namespace", ns)
	return n.Client.Delete(context.TODO(), &ns)
}
