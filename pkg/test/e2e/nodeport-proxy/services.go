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

package nodeportproxy

import (
	"context"
	"fmt"

	"github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"
)

// Based on:
// https://github.com/mgdevstack/kubernetes/blob/9eced040142454a20255ae323279a38dc6b2bc1a/test/e2e/framework/service/jig.go#L60
// ServiceJig is a jig to help service testing.
type ServiceJig struct {
	Log       *zap.SugaredLogger
	Client    ctrlruntimeclient.Client
	Namespace string

	Services    []*corev1.Service
	ServicePods map[string][]string
}

// CreateServiceWithPods deploys a service and the associated pods.
func (n *ServiceJig) CreateServiceWithPods(svc *corev1.Service, numPods int32, https bool) (*corev1.Service, error) {
	if len(svc.Spec.Ports) == 0 {
		return nil, errors.New("failed to create service: at least one port is required")
	}
	if len(svc.Spec.Selector) == 0 {
		return nil, errors.New("failed to create service: selector is required")
	}

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
		n.Log.Debugw("Setting generated namespace to ServiceJig", "namespace", n.Namespace)
	}

	svc.Namespace = n.Namespace
	n.Log.Debugw("Creating nodeport service", "service", svc)
	if err := n.Client.Create(context.TODO(), svc); err != nil {
		return nil, errors.Wrap(err, "failed to create service of type nodeport")
	}
	gomega.Expect(svc).NotTo(gomega.BeNil())

	// Create service pods
	rc := n.newRCFromService(svc, https, numPods)
	n.Log.Debugw("Creating replication controller", "rc", rc)
	if err := n.Client.Create(context.TODO(), rc); err != nil {
		return nil, err
	}

	// Wait for the pod to be ready
	pods, err := e2eutils.WaitForPodsCreated(n.Client, int(*rc.Spec.Replicas), rc.Namespace, svc.Spec.Selector)
	if err != nil {
		return svc, errors.Wrap(err, "error occurred while waiting for pods to be ready")
	}
	if !e2eutils.CheckPodsRunningReady(n.Client, n.Namespace, pods, podReadinessTimeout) {
		return svc, fmt.Errorf("timeout waiting for %d pods to be ready", len(pods))
	}
	if n.ServicePods == nil {
		n.ServicePods = map[string][]string{}
	}
	n.Services = append(n.Services, svc)
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

func (n *ServiceJig) newRCFromService(svc *corev1.Service, https bool, replicas int32) *corev1.ReplicationController {
	var args []string
	var port intstr.IntOrString
	var scheme corev1.URIScheme
	if https {
		port = svc.Spec.Ports[0].TargetPort
		args = []string{"netexec", fmt.Sprintf("--http-port=%d", port.IntValue()), "--tls-cert-file=/localhost.crt", "--tls-private-key-file=/localhost.key"}
		scheme = corev1.URISchemeHTTPS
	} else {
		port = svc.Spec.Ports[0].TargetPort
		args = []string{"netexec", fmt.Sprintf("--http-port=%d", port.IntValue())}
		scheme = corev1.URISchemeHTTP
	}

	rc := &corev1.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Labels:    svc.Spec.Selector,
		},
		Spec: corev1.ReplicationControllerSpec{
			Replicas: &replicas,
			Selector: svc.Spec.Selector,
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: svc.Spec.Selector,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "agnhost",
							Image: AgnhostImage,
							Args:  args,
							ReadinessProbe: &corev1.Probe{
								PeriodSeconds: 3,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Port:   port,
										Path:   "/",
										Scheme: scheme,
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
