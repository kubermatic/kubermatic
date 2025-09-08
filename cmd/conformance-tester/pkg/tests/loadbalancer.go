/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package tests

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	ctypes "k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func supportsLoadBalancer(cluster *kubermaticv1.Cluster) bool {
	return cluster.Spec.Cloud.Azure != nil ||
		cluster.Spec.Cloud.AWS != nil ||
		cluster.Spec.Cloud.GCP != nil ||
		cluster.Spec.Cloud.Hetzner != nil ||
		cluster.Spec.Cloud.Kubevirt != nil ||
		cluster.Spec.Cloud.Openstack != nil
}

func TestLoadBalancer(ctx context.Context, log *zap.SugaredLogger, opts *ctypes.Options, cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client, attempt int) error {
	if !opts.Tests.Has(ctypes.LoadbalancerTests) {
		log.Info("LoadBalancers tests disabled, skipping.")
		return nil
	}

	if !supportsLoadBalancer(cluster) {
		log.Info("Provider does not support LoadBalancers, skipping.")
		return nil
	}

	log.Info("Testing support for LoadBalancers...")

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("lb-test-%d", attempt),
		},
	}
	if err := userClusterClient.Create(ctx, ns); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	log.Info("Creating a Service of type LoadBalancer...")
	labels := map[string]string{"app": "hello"}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns.Name,
			Annotations: map[string]string{
				"load-balancer.hetzner.cloud/location": "nbg1",
			},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeLoadBalancer,
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}
	if err := userClusterClient.Create(ctx, service); err != nil {
		return fmt.Errorf("failed to create Service: %w", err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hello-kubernetes",
			Namespace: ns.Name,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "hello-kubernetes",
					Image: "us-docker.pkg.dev/google-samples/containers/gke/hello-app:2.0",
					Ports: []corev1.ContainerPort{
						{
							Name:          "http",
							ContainerPort: 8080,
							Protocol:      corev1.ProtocolTCP,
						},
					},
				},
			},
		},
	}

	if err := userClusterClient.Create(ctx, pod); err != nil {
		return fmt.Errorf("failed to create Pod: %w", err)
	}

	var host string
	log.Debug("Waiting until the Service has a public IP/Name...")
	err := wait.Poll(ctx, 3*time.Second, opts.CustomTestTimeout, func(ctx context.Context) (transient error, terminal error) {
		currentService := &corev1.Service{}
		if err := userClusterClient.Get(ctx, types.NamespacedName{Namespace: ns.Name, Name: service.Name}, currentService); err != nil {
			return fmt.Errorf("failed to fetch Service %s/%s: %w", ns.Name, service.Name, err), nil
		}

		if len(currentService.Status.LoadBalancer.Ingress) > 0 {
			host = currentService.Status.LoadBalancer.Ingress[0].IP
			if host == "" {
				host = currentService.Status.LoadBalancer.Ingress[0].Hostname

				// We wait until we can actually resolve the name
				var r net.Resolver
				ips, err := r.LookupHost(ctx, host)
				if err != nil || len(ips) == 0 {
					return fmt.Errorf("failed to resolve %q: %w", host, err), nil
				}
			}

			return nil, nil
		}

		return errors.New("LoadBalancer has no Ingress status"), nil
	})
	if err != nil {
		return fmt.Errorf("failed to check if Service is ready: %w", err)
	}
	log.Debug("The Service has an external IP/Name")

	hostURL := fmt.Sprintf("http://%s", net.JoinHostPort(host, "80"))
	log.Debug("Waiting until the pod is available via the LoadBalancer...")
	err = wait.Poll(ctx, 3*time.Second, opts.CustomTestTimeout, func(ctx context.Context) (transient error, terminal error) {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, hostURL, nil)
		if err != nil {
			return nil, err
		}

		resp, err := http.DefaultClient.Do(request)
		if err != nil {
			return fmt.Errorf("failed to call Pod via LoadBalancer (%s): %w", hostURL, err), nil
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Warnf("Failed to close response body from Hello-Kubernetes Pod (%s): %v", hostURL, err)
			}
		}()

		contents, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body from Hello-Kubernetes Pod (%s): %w", hostURL, err), nil
		}

		needle := "Hello, world!"
		if strings.Contains(string(contents), needle) {
			return nil, nil
		}

		return fmt.Errorf("response did not contain %q: %q", needle, string(contents)), nil
	})
	if err != nil {
		return fmt.Errorf("failed to check if Pod is available via LoadBalancer: %w", err)
	}

	log.Info("Successfully validated LoadBalancer support")
	return nil
}
