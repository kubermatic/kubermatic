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

	ctypes "k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/util"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	k8csemverv1 "k8c.io/kubermatic/v2/pkg/semver/v1"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	expected := sets.NewString(
		"etcd_disk_backend_defrag_duration_seconds_sum",
		"kube_daemonset_labels",
		"kubelet_runtime_operations_duration_seconds_count",
		"machine_controller_machines_total",
		"replicaset_controller_rate_limiter_use",
		"apiserver_request_total",
		"workqueue_retries_total",
		"machine_cpu_cores",
	)

	if cluster.Spec.Version.LessThan(k8csemverv1.NewSemverOrDie("v1.23.0")) {
		expected.Insert("scheduler_e2e_scheduling_duration_seconds_count")
	} else {
		// this metric is only available in 1.23 and replaces scheduler_e2e_scheduling_duration_seconds_counts
		expected.Insert("scheduler_scheduling_attempt_duration_seconds_count")
	}

	fetched := sets.NewString(data.Data...)
	missing := expected.Difference(fetched)

	if missing.Len() > 0 {
		return fmt.Errorf("failed to get all expected metrics: got: %v, %v are missing", fetched.List(), missing.List())
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
	err := wait.PollLog(ctx, log, opts.UserClusterPollInterval, opts.CustomTestTimeout, func() (transient error, terminal error) {
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
					Image: "gcr.io/google-samples/node-hello:1.0",
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

	if err := userClusterClient.Create(ctx, pod); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create Pod: %w", err)
	}

	err = wait.PollLog(ctx, log, opts.UserClusterPollInterval, opts.CustomTestTimeout, func() (transient error, terminal error) {
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
	err = wait.PollLog(ctx, log, opts.UserClusterPollInterval, opts.CustomTestTimeout, func() (transient error, terminal error) {
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
