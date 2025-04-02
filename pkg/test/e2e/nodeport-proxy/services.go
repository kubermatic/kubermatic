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
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Based on:
// https://github.com/mgdevstack/kubernetes/blob/9eced040142454a20255ae323279a38dc6b2bc1a/test/e2e/framework/service/jig.go#L60
// ServiceJig is a jig to help service testing.
type ServiceJig struct {
	Log       *zap.SugaredLogger
	Client    ctrlruntimeclient.Client
	Namespace string
}

// CreateServiceWithPods deploys a service and the associated pods, returning the
// created service and the names of all pods that act as endpoints for the service.
func (j *ServiceJig) CreateServiceWithPods(ctx context.Context, svc *corev1.Service, numPods int32, https bool) (*corev1.Service, []string, error) {
	if len(svc.Spec.Ports) == 0 {
		return nil, nil, errors.New("failed to create service: at least one port is required")
	}
	if len(svc.Spec.Selector) == 0 {
		return nil, nil, errors.New("failed to create service: selector is required")
	}

	// Create the namespace to host the service
	ns := j.newNamespaceTemplate()
	j.Log.Infow("Creating namespace…")
	if err := j.Client.Create(ctx, ns); ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
		return nil, nil, err
	}

	// Set back namespace name in case it was generated
	j.Namespace = ns.Name

	// Create the service
	svc.Namespace = j.Namespace
	j.Log.Infow("Creating nodeport service", "service", svc)
	if err := j.Client.Create(ctx, svc); err != nil {
		return nil, nil, fmt.Errorf("failed to create service: %w", err)
	}

	// Create service pods
	deployment := j.newDeployment(svc, https, numPods)
	j.Log.Infow("Creating deployment…", "deployment", deployment, "replicas", deployment.Spec.Replicas)
	if err := j.Client.Create(ctx, deployment); err != nil {
		return nil, nil, fmt.Errorf("failed to create deployment backing the service: %w", err)
	}

	// Wait for the pod to be ready
	pods, err := e2eutils.WaitForPodsCreated(ctx, j.Client, j.Log, int(*deployment.Spec.Replicas), deployment.Namespace, svc.Spec.Selector)
	if err != nil {
		return svc, nil, fmt.Errorf("error occurred while waiting for pods to be ready: %w", err)
	}
	if !e2eutils.CheckPodsRunningReady(ctx, j.Client, j.Log, j.Namespace, pods, 2*time.Minute) {
		return svc, nil, errors.New("timeout waiting for all pods to be ready")
	}

	return svc, pods, err
}

// Cleanup removes the resources.
func (j *ServiceJig) Cleanup(ctx context.Context) error {
	ns := corev1.Namespace{}
	if err := j.Client.Get(ctx, types.NamespacedName{Name: j.Namespace}, &ns); err != nil {
		return err
	}
	j.Log.Infow("Deleting test namespace…", "namespace", j.Namespace)
	return j.Client.Delete(ctx, &ns)
}

func (j *ServiceJig) newNamespaceTemplate() *corev1.Namespace {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: j.Namespace,
		},
	}
	if j.Namespace == "" {
		ns.GenerateName = "np-proxy-test-"
	}
	return ns
}

func (j *ServiceJig) newDeployment(svc *corev1.Service, https bool, replicas int32) *appsv1.Deployment {
	port := svc.Spec.Ports[0].TargetPort
	args := []string{"netexec", fmt.Sprintf("--http-port=%d", port.IntValue())}
	scheme := corev1.URISchemeHTTP

	if https {
		args = append(args, "--tls-cert-file=/localhost.crt", "--tls-private-key-file=/localhost.key")
		scheme = corev1.URISchemeHTTPS
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Labels:    svc.Spec.Selector,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: svc.Spec.Selector,
			},
			Template: corev1.PodTemplateSpec{
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
								ProbeHandler: corev1.ProbeHandler{
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
}
