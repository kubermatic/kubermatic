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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	seedLabelProbePrefix = "seed-logs-label-probe"

	// seedLabelProbePodLabels is the number of pod labels on the label probe.
	// A user cluster apiserver pod had 24 on KKP v2.31.2 on AWS: app, cluster,
	// app.kubernetes.io/version, pod-template-hash, two topology labels and one
	// revision label per mounted Secret and ConfigMap. The probe adds headroom
	// for new volumes. The seed Alloy adds six labels of its own (namespace,
	// job, instance, container_name, filename, stream), so the stream carries
	// 34 labels.
	seedLabelProbePodLabels = 28

	// controlPlaneLogWindow is how far back the control plane check compares
	// the kubelet logs with Loki.
	controlPlaneLogWindow = 15 * time.Minute
)

// lokiLabelDiscardReasons are the reasons Loki uses when it rejects a stream
// because of its labels. They mean the shipped Alloy and Loki configs do not
// fit each other, so unlike rate limits they never depend on the environment.
var lokiLabelDiscardReasons = []string{
	"duplicate_label_names",
	"invalid_labels",
	"label_name_too_long",
	"label_value_too_long",
	"max_label_names_per_series",
	"missing_labels",
}

// lokiDiscardedSampleLine matches a sample of the counter, with an optional
// trailing timestamp.
var lokiDiscardedSampleLine = regexp.MustCompile(`^loki_discarded_samples_total\{.*\breason="([^"]+)".*\} (\S+)( \S+)?$`)

func seedLoki(in SeedInput) (*serviceProxy, error) {
	return newServiceProxy(in.SeedConfig, loggingNamespace, seedLokiService, seedLokiPort)
}

// lokiLabelDiscards returns the number of log lines Loki rejected per label
// reason since it started.
func lokiLabelDiscards(ctx context.Context, loki *serviceProxy) (map[string]float64, error) {
	code, body, err := loki.do(ctx, http.MethodGet, "/metrics", nil, nil)
	if err != nil {
		return nil, err
	}
	if code != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200 from Loki /metrics, got HTTP %d", code)
	}

	return parseLokiLabelDiscards(string(body))
}

func parseLokiLabelDiscards(metrics string) (map[string]float64, error) {
	discards := map[string]float64{}
	for _, line := range strings.Split(metrics, "\n") {
		match := lokiDiscardedSampleLine.FindStringSubmatch(line)
		if match == nil || !slices.Contains(lokiLabelDiscardReasons, match[1]) {
			continue
		}

		value, err := strconv.ParseFloat(match[2], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %q: %w", line, err)
		}

		// the counter has one series per tenant and retention, sum them up
		discards[match[1]] += value
	}

	return discards, nil
}

func formatDiscards(discards map[string]float64) string {
	var parts []string
	for reason, value := range discards {
		if value > 0 {
			parts = append(parts, fmt.Sprintf("%s=%.0f", reason, value))
		}
	}
	sort.Strings(parts)

	return strings.Join(parts, ", ")
}

// CheckSeedLogDiscards verifies that seed Loki rejected no log line because
// of its labels. Loki answers such pushes with HTTP 400 and Alloy drops the
// batch without retrying, so this counter is the only trace of the lost logs.
func CheckSeedLogDiscards(ctx context.Context, in SeedInput) error {
	loki, err := seedLoki(in)
	if err != nil {
		return err
	}

	discards, err := lokiLabelDiscards(ctx, loki)
	if err != nil {
		return fmt.Errorf("seed Loki: %w", err)
	}

	if summary := formatDiscards(discards); summary != "" {
		return fmt.Errorf("seed Loki rejected log lines because of their labels (loki_discarded_samples_total: %s)", summary)
	}

	return nil
}

// CheckSeedLogLabelLimit runs a pod with as many labels as a user cluster
// apiserver pod and verifies that its log lines reach seed Loki. Alloy turns
// every pod label into a stream label, so this fails when the Loki label
// limit is lower than what KKP control plane pods produce.
func CheckSeedLogLabelLimit(ctx context.Context, in SeedInput) error {
	runID := rand.String(8)
	pod := newSeedLabelProbePod(runID)

	if err := in.SeedClient.Create(ctx, pod); err != nil {
		return fmt.Errorf("failed to create pod %s: %w", pod.Name, err)
	}
	defer func() {
		// the spec context may be done already, cleanup must still happen
		cleanupCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		if err := in.SeedClient.Delete(cleanupCtx, pod); ctrlruntimeclient.IgnoreNotFound(err) != nil {
			in.Log.Warnw("Failed to delete the label probe pod", "pod", pod.Name, "error", err)
		}
	}()

	if !utils.CheckPodsRunningReady(ctx, in.SeedClient, in.Log, pod.Namespace, []string{pod.Name}, probeReadinessTimeout) {
		return fmt.Errorf("timed out waiting for pod %s to become ready", pod.Name)
	}

	loki, err := seedLoki(in)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`{namespace=%q, instance=%q} |= %q`, pod.Namespace, pod.Name, runID)

	err = wait.PollImmediate(ctx, 10*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		found, err := lokiHasLines(ctx, loki, query, time.Now().Add(-10*time.Minute))
		if err != nil {
			return err, nil
		}
		if !found {
			return fmt.Errorf("no lines for %s", query), nil
		}

		return nil, nil
	})
	if err != nil {
		// Loki's discard counter tells a label rejection apart from a slow agent
		if discards, derr := lokiLabelDiscards(ctx, loki); derr == nil {
			if summary := formatDiscards(discards); summary != "" {
				err = fmt.Errorf("%w (loki_discarded_samples_total: %s)", err, summary)
			}
		}

		return fmt.Errorf("pod with %d labels: logs did not reach seed Loki: %w", len(pod.Labels), err)
	}

	return nil
}

func newSeedLabelProbePod(runID string) *corev1.Pod {
	name := seedLabelProbePrefix + "-" + runID

	// app is what the seed Alloy pipeline keys off, the rest only add labels
	labels := map[string]string{"app": seedLabelProbePrefix}
	for i := 1; len(labels) < seedLabelProbePodLabels; i++ {
		labels[fmt.Sprintf("e2e-label-%02d", i)] = runID
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyAlways,
			Containers: []corev1.Container{
				{
					Name:            "probe",
					Image:           logsProbeImage,
					Command:         []string{"sh", "-c", fmt.Sprintf(`while true; do echo "%s %s"; sleep 2; done`, seedLabelProbePrefix, runID)},
					ImagePullPolicy: corev1.PullIfNotPresent,
				},
			},
			TerminationGracePeriodSeconds: ptr.To[int64](0),
		},
	}
}

// CheckControlPlaneLogs verifies that seed Loki has the logs of every user
// cluster control plane container on the seed. A container counts only when
// the kubelet has log lines from it in the last controlPlaneLogWindow, so
// quiet containers do not fail the check.
func CheckControlPlaneLogs(ctx context.Context, in SeedInput) error {
	clusters := &kubermaticv1.ClusterList{}
	if err := in.SeedClient.List(ctx, clusters); err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	var namespaces []string
	for _, cluster := range clusters.Items {
		if cluster.Status.NamespaceName != "" && cluster.DeletionTimestamp == nil {
			namespaces = append(namespaces, cluster.Status.NamespaceName)
		}
	}
	if len(namespaces) == 0 {
		return errors.New("no user cluster on the seed")
	}

	kubeClient, err := kubernetes.NewForConfig(in.SeedConfig)
	if err != nil {
		return fmt.Errorf("failed to create a Kubernetes client: %w", err)
	}

	loki, err := seedLoki(in)
	if err != nil {
		return err
	}

	since := time.Now().Add(-controlPlaneLogWindow)

	var errs []error
	for _, namespace := range namespaces {
		if err := checkNamespaceLogs(ctx, in, kubeClient, loki, namespace, since); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		if discards, derr := lokiLabelDiscards(ctx, loki); derr == nil {
			if summary := formatDiscards(discards); summary != "" {
				errs = append(errs, fmt.Errorf("loki_discarded_samples_total: %s", summary))
			}
		}
	}

	return errors.Join(errs...)
}

// checkNamespaceLogs compares the containers that logged since the given time
// with the streams seed Loki has for the namespace.
func checkNamespaceLogs(ctx context.Context, in SeedInput, kubeClient kubernetes.Interface, loki *serviceProxy, namespace string, since time.Time) error {
	pods := &corev1.PodList{}
	if err := in.SeedClient.List(ctx, pods, ctrlruntimeclient.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list pods in %s: %w", namespace, err)
	}

	var expected []string
	for _, pod := range pods.Items {
		// the seed Alloy only collects pods with a name or app label
		if pod.Status.Phase != corev1.PodRunning || (pod.Labels["app"] == "" && pod.Labels["name"] == "") {
			continue
		}

		for _, container := range pod.Spec.Containers {
			logged, err := containerLoggedSince(ctx, kubeClient, pod, container.Name, since)
			if err != nil {
				return err
			}
			if logged {
				expected = append(expected, pod.Name+"/"+container.Name)
			}
		}
	}

	if len(expected) == 0 {
		return fmt.Errorf("%s: no control plane container logged in the last %s", namespace, controlPlaneLogWindow)
	}

	var missing []string

	err := wait.PollImmediate(ctx, 15*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		streams, err := lokiContainerStreams(ctx, loki, namespace, since)
		if err != nil {
			return err, nil
		}

		missing = missing[:0]
		for _, container := range expected {
			if !streams[container] {
				missing = append(missing, container)
			}
		}
		if len(missing) > 0 {
			return fmt.Errorf("%d of %d containers have no logs in seed Loki", len(missing), len(expected)), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("%s: %w: %s", namespace, err, strings.Join(missing, ", "))
	}

	return nil
}

func containerLoggedSince(ctx context.Context, kubeClient kubernetes.Interface, pod corev1.Pod, container string, since time.Time) (bool, error) {
	sinceTime := metav1.NewTime(since)

	stream, err := kubeClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Container:  container,
		SinceTime:  &sinceTime,
		LimitBytes: ptr.To[int64](1),
	}).Stream(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to read logs of %s/%s/%s: %w", pod.Namespace, pod.Name, container, err)
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		return false, fmt.Errorf("failed to read logs of %s/%s/%s: %w", pod.Namespace, pod.Name, container, err)
	}

	return len(data) > 0, nil
}

// lokiContainerStreams returns the "pod/container" pairs that have streams in
// seed Loki for the namespace since the given time.
func lokiContainerStreams(ctx context.Context, loki *serviceProxy, namespace string, since time.Time) (map[string]bool, error) {
	path := fmt.Sprintf("/loki/api/v1/series?match[]=%s&start=%d&end=%d",
		url.QueryEscape(fmt.Sprintf(`{namespace=%q}`, namespace)),
		since.UnixNano(),
		time.Now().UnixNano(),
	)

	code, body, err := loki.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	if code != http.StatusOK {
		return nil, fmt.Errorf("expected HTTP 200, got HTTP %d: %s", code, body)
	}

	response := struct {
		Status string              `json:"status"`
		Data   []map[string]string `json:"data"`
	}{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("unable to decode response %q: %w", body, err)
	}
	if response.Status != "success" {
		return nil, fmt.Errorf("expected a successful response, got %q", body)
	}

	streams := map[string]bool{}
	for _, series := range response.Data {
		streams[series["instance"]+"/"+series["container_name"]] = true
	}

	return streams, nil
}

func lokiHasLines(ctx context.Context, loki *serviceProxy, query string, since time.Time) (bool, error) {
	path := fmt.Sprintf("/loki/api/v1/query_range?query=%s&start=%d&end=%d&limit=1",
		url.QueryEscape(query),
		since.UnixNano(),
		time.Now().UnixNano(),
	)

	code, body, err := loki.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return false, err
	}
	if code != http.StatusOK {
		return false, fmt.Errorf("expected HTTP 200, got HTTP %d: %s", code, body)
	}

	response, err := parseLokiQueryResponse(string(body))
	if err != nil {
		return false, err
	}

	return len(response.Data.Result) > 0, nil
}
