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
	"os"
	"sort"
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

const (
	// tagPrefix namespaces the traceability annotations on the LB test Service.
	tagPrefix = "kkp-test/"

	// awsAdditionalTagsKey is honored by the AWS CCM (provider-aws) to apply the
	// listed key=value pairs as tags on the ELB/NLB created for the Service.
	awsAdditionalTagsKey = "service.beta.kubernetes.io/aws-load-balancer-additional-resource-tags"
)

// buildTraceabilityAnnotations returns Service annotations identifying the CI
// run that requested this LoadBalancer. Values come from Prow-injected env
// (JOB_NAME, BUILD_ID, PULL_NUMBER); they are empty outside Prow.
func buildTraceabilityAnnotations(now time.Time) map[string]string {
	ann := map[string]string{
		tagPrefix + "triggered-at": now.UTC().Format(time.RFC3339),
	}

	if v := os.Getenv("JOB_NAME"); v != "" {
		ann[tagPrefix+"prowjob"] = v
	}
	if v := os.Getenv("BUILD_ID"); v != "" {
		ann[tagPrefix+"build-id"] = v
	}
	if v := os.Getenv("PULL_NUMBER"); v != "" {
		ann[tagPrefix+"pr"] = v
	}

	return ann
}

// awsAdditionalResourceTags builds the comma-separated key=value string the AWS
// CCM expects for the aws-load-balancer-additional-resource-tags annotation.
// Keys are sorted so the value is deterministic.
func awsAdditionalResourceTags(tags map[string]string) string {
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, tags[k]))
	}
	return strings.Join(parts, ",")
}

// mergeServiceAnnotations combines the traceability annotations with the
// provider-specific ones the test already uses. For AWS clusters it also mirrors
// the traceability values into the AWS additional-resource-tags annotation so
// they become ELB tags visible in the AWS console.
func mergeServiceAnnotations(traceability map[string]string, cluster *kubermaticv1.Cluster) map[string]string {
	ann := map[string]string{
		// preserved existing behavior; no-op on non-Hetzner providers
		"load-balancer.hetzner.cloud/location": "nbg1",
	}
	for k, v := range traceability {
		ann[k] = v
	}

	if cluster.Spec.Cloud.AWS != nil {
		ann[awsAdditionalTagsKey] = awsAdditionalResourceTags(traceability)
	}

	return ann
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

	// clean up the namespace on every exit (pass or fail) so a failed attempt does not
	// leave an orphaned Service of type LoadBalancer behind, which keeps a cloud LB alive
	// against the provider's per-region quota. use a detached context so deletion still
	// runs if ctx was cancelled by the failing test.
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		err := userClusterClient.Delete(cleanupCtx, ns)
		if ctrlruntimeclient.IgnoreNotFound(err) != nil {
			log.Warnf("Failed to delete LoadBalancer test namespace %q: %v", ns.Name, err)
		}
	}()

	log.Info("Creating a Service of type LoadBalancer...")
	labels := map[string]string{"app": "hello"}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test",
			Namespace:   ns.Name,
			Annotations: mergeServiceAnnotations(buildTraceabilityAnnotations(time.Now()), cluster),
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
