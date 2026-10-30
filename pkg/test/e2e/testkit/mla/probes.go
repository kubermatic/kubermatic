/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package mla

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// agnhostImage ships curl and the "porter" and "pause" commands and is
	// already used by other e2e suites for in-cluster HTTP checks.
	agnhostImage = "registry.k8s.io/e2e-test-images/agnhost:2.53"

	metricsProbePodName   = "mla-metrics-probe"
	metricsProbeContainer = "porter"
	metricsProbePort      = 9101

	// metricsProbeMetric is the series the probe pod serves and the checks
	// query back out of Cortex.
	metricsProbeMetric = "mla_e2e_probe_total"

	logsProbePodName = "mla-logs-probe"
	logsProbeImage   = "docker.io/library/busybox:1.36"

	// logsProbeMarker is the fixed line the logs probe emits every few seconds;
	// its presence in Loki is the logging-agent round trip.
	logsProbeMarker = "mla-logs-probe-marker"

	// logsProbeJob is the stream label the logging agent's relabel pipeline
	// derives from the pod's namespace and app.kubernetes.io/name labels.
	logsProbeJob = metav1.NamespaceDefault + "/" + logsProbePodName

	// probeReadinessTimeout covers the image pull on a fresh node.
	probeReadinessTimeout = 5 * time.Minute
)

// EnsureMetricsProbe deploys the scrape target the metrics round trip queries
// for. It is idempotent. Deploy it early, so the agent's scrape and remote
// write overlap with other checks.
func EnsureMetricsProbe(ctx context.Context, in Input) error {
	return ensurePod(ctx, in.UserClient, in, newMetricsProbePod(metav1.NamespaceDefault, in.Cluster.Name))
}

// EnsureLogsProbe deploys the log source the logs round trip queries for. It
// is idempotent. Deploy it early, so the agent has shipped lines by the time
// the check runs.
func EnsureLogsProbe(ctx context.Context, in Input) error {
	return ensurePod(ctx, in.UserClient, in, newLogsProbePod())
}

// StartProbes creates the metrics and logs probe pods without waiting for
// them. Call it early, so the agents' scrape and log shipping overlap with
// other checks; the checks that need the probes wait for them. It does not
// block on user clusters without worker nodes.
func StartProbes(ctx context.Context, in Input) error {
	for _, pod := range []*corev1.Pod{newMetricsProbePod(metav1.NamespaceDefault, in.Cluster.Name), newLogsProbePod()} {
		if err := in.UserClient.Create(ctx, pod); err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create pod %s: %w", pod.Name, err)
		}
	}

	return nil
}

// CleanupProbes removes the metrics and logs probe pods.
func CleanupProbes(ctx context.Context, in Input) error {
	var errs []error

	for _, name := range []string{metricsProbePodName, logsProbePodName} {
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: metav1.NamespaceDefault}}
		if err := in.UserClient.Delete(ctx, pod); err != nil && !apierrors.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("failed to delete pod %s: %w", name, err))
		}
	}

	return errors.Join(errs...)
}

// ensurePod creates the pod unless it exists and waits for it to be ready.
// Parallel test processes may race to create the same pod, so an existing pod
// is reused.
func ensurePod(ctx context.Context, client ctrlruntimeclient.Client, in Input, pod *corev1.Pod) error {
	if err := client.Create(ctx, pod); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create pod %s: %w", pod.Name, err)
	}

	if !utils.CheckPodsRunningReady(ctx, client, in.Log, pod.Namespace, []string{pod.Name}, probeReadinessTimeout) {
		return fmt.Errorf("timed out waiting for pod %s to become ready", pod.Name)
	}

	return nil
}

// newMetricsProbePod returns an agnhost "porter" pod, which serves the content of
// SERVE_PORT_<port> on every request path. That makes it a scrape target with no
// ConfigMap and no volume. The prometheus.io annotations are what the monitoring
// agent's kubernetes-pods job keys off.
func newMetricsProbePod(ns, runID string) *corev1.Pod {
	metrics := fmt.Sprintf(`# HELP %s counter served by the MLA e2e metrics probe
# TYPE %s counter
%s{run=%q} 1
`, metricsProbeMetric, metricsProbeMetric, metricsProbeMetric, runID)

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metricsProbePodName,
			Namespace: ns,
			Annotations: map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   strconv.Itoa(metricsProbePort),
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyAlways,
			Containers: []corev1.Container{
				{
					Name:            metricsProbeContainer,
					Image:           agnhostImage,
					Args:            []string{"porter"},
					ImagePullPolicy: corev1.PullIfNotPresent,
					Env: []corev1.EnvVar{
						{
							Name:  fmt.Sprintf("SERVE_PORT_%d", metricsProbePort),
							Value: metrics,
						},
					},
					Ports: []corev1.ContainerPort{
						{ContainerPort: metricsProbePort, Protocol: corev1.ProtocolTCP},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/metrics",
								Port: intstr.FromInt32(metricsProbePort),
							},
						},
					},
				},
			},
			TerminationGracePeriodSeconds: ptr.To[int64](0),
		},
	}
}

// newLogsProbePod returns a busybox pod that emits the marker line every few
// seconds. The app.kubernetes.io/name label is what the logging agent's
// discovery relabeling keys off; without it the pod is invisible to the agent.
func newLogsProbePod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      logsProbePodName,
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				"app.kubernetes.io/name": logsProbePodName,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyAlways,
			Containers: []corev1.Container{
				{
					Name:  logsProbePodName,
					Image: logsProbeImage,
					Command: []string{
						"sh", "-c",
						fmt.Sprintf(`while true; do echo "%s"; sleep 5; done`, logsProbeMarker),
					},
					ImagePullPolicy: corev1.PullIfNotPresent,
				},
			},
		},
	}
}
