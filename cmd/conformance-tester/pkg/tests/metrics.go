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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	ctypes "k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/util"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type metricsData struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}

// testUserClusterMetrics ensures all expected metrics are actually collected
// in Prometheus. Note that this assumes that some time has passed between
// Prometheus' eployment and this test, so it can scrape all targets. This
// includes kubelets, so nodes must have been ready for at least 30 seconds
// before this can succeed.
func TestUserClusterMetrics(ctx context.Context, log *zap.SugaredLogger, opts *ctypes.Options, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client) error {
	if !opts.Tests.Has(ctypes.MetricsTests) {
		log.Info("Metrics tests disabled, skipping.")
		return nil
	}

	log.Info("Testing user cluster metrics availability...")

	res := opts.SeedGeneratedClient.CoreV1().RESTClient().Get().
		Namespace(cluster.Status.NamespaceName).
		Resource("pods").
		Name("prometheus-0:9090").
		SubResource("proxy").
		Suffix("api/v1/label/__name__/values").
		Do(ctx)

	if err := res.Error(); err != nil {
		return fmt.Errorf("request to Prometheus failed: %w", err)
	}

	code := 0
	res.StatusCode(&code)
	if code != http.StatusOK {
		return fmt.Errorf("Prometheus returned HTTP %d", code)
	}

	body, err := res.Raw()
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	data := &metricsData{}
	if err := json.Unmarshal(body, data); err != nil {
		return fmt.Errorf("failed to unmarshal metrics response: %w", err)
	}

	if data.Status != "success" {
		return fmt.Errorf("failed to get prometheus metrics with data status: %s", data.Status)
	}

	expected := sets.New(
		// etcd
		"etcd_disk_backend_defrag_duration_seconds_sum",
		// kube-state-metrics
		"kube_configmap_info",
		// user cluster kubelets
		"kubelet_runtime_operations_duration_seconds_count",
		// machine-controller
		"machine_controller_machines_total",
		// kube controller-manager
		"replicaset_controller_sorting_deletion_age_ratio_bucket",
		// kube apiserver
		"apiserver_request_total",
		// kube scheduler
		"scheduler_schedule_attempts_total",
		// cadvisor
		"machine_cpu_cores",
	)

	fetched := sets.New(data.Data...)
	missing := expected.Difference(fetched)

	if missing.Len() > 0 {
		return fmt.Errorf("failed to get all expected metrics: got: %v, %v are missing", sets.List(fetched), sets.List(missing))
	}

	log.Info("Successfully validated user cluster metrics.")
	return nil
}

func TestUserClusterPodAndNodeMetrics(ctx context.Context, log *zap.SugaredLogger, opts *ctypes.Options, cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client) error {
	if !opts.Tests.Has(ctypes.MetricsTests) {
		log.Info("Metrics tests disabled, skipping.")
		return nil
	}

	log.Info("Testing user cluster pod and node metrics availability...")

	// check node metrics
	err := wait.PollLog(ctx, log, opts.UserClusterPollInterval, opts.CustomTestTimeout, func(ctx context.Context) (transient error, terminal error) {
		allNodeMetricsList := &v1beta1.NodeMetricsList{}
		if err := userClusterClient.List(ctx, allNodeMetricsList); err != nil {
			return fmt.Errorf("error getting node metrics list: %w", err), nil
		}
		if len(allNodeMetricsList.Items) == 0 {
			return fmt.Errorf("node metrics list is empty"), nil
		}

		for _, nodeMetric := range allNodeMetricsList.Items {
			// check a metric to see if it works
			if nodeMetric.Usage.Memory().IsZero() {
				return fmt.Errorf("node %q memory usage metric is 0", nodeMetric.Name), nil
			}
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to check if test node metrics: %w", err)
	}

	// create pod to check metrics
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "check-metrics",
			Namespace: "default",
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

	if err := userClusterClient.Create(ctx, pod); ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
		return fmt.Errorf("failed to create Pod: %w", err)
	}

	err = wait.PollLog(ctx, log, opts.UserClusterPollInterval, opts.CustomTestTimeout, func(ctx context.Context) (transient error, terminal error) {
		metricPod := &corev1.Pod{}
		if err := userClusterClient.Get(ctx, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}, metricPod); err != nil {
			return fmt.Errorf("failed to get test metric pod: %w", err), nil
		}

		if !util.PodIsReady(metricPod) {
			return errors.New("Pod is not ready"), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to check if test metrics pod is ready: %w", err)
	}

	// check pod metrics
	err = wait.PollLog(ctx, log, opts.UserClusterPollInterval, opts.CustomTestTimeout, func(ctx context.Context) (transient error, terminal error) {
		podMetrics := &v1beta1.PodMetrics{}
		if err := userClusterClient.Get(ctx, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}, podMetrics); err != nil {
			return fmt.Errorf("failed to get test metric pod metrics: %w", err), nil
		}

		for _, cont := range podMetrics.Containers {
			if cont.Usage.Memory().IsZero() {
				return errors.New("metrics test pod memory usage is 0"), nil
			}
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to get metric test pod metrics: %w", err)
	}

	log.Info("Successfully validated user cluster pod and node metrics.")
	return nil
}
