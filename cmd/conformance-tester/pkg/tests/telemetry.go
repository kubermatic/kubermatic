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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"go.uber.org/zap"

	ctypes "k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticmaster "k8c.io/kubermatic/v2/pkg/install/stack/kubermatic-master"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type telemetryReport struct {
	Version string `json:"version"`
	Records []struct {
		Kind string `json:"kind"`
	} `json:"records"`
}

// TestTelemetry checks if there are telemetry pods available and
// gets the logs from the most recent one, assuming that it output
// a big JSON document with KKP and k8s statistics.
func TestTelemetry(ctx context.Context, log *zap.SugaredLogger, opts *ctypes.Options) error {
	if !opts.Tests.Has(ctypes.TelemetryTests) {
		log.Info("Telemetry tests disabled, skipping.")
		return nil
	}

	log.Info("Testing telemetry availability...")

	pod, err := getLatestTelemetryPod(ctx, opts.SeedClusterClient, kubermaticmaster.TelemetryNamespace)
	if err != nil {
		return fmt.Errorf("failed to determine latest completed telemetry Pod: %w", err)
	}
	if pod == nil {
		return errors.New("no completed telemetry Pod found (either telemetry was not installed or no Pod completed successfully)")
	}

	output, err := getContainerLogs(ctx, opts.SeedGeneratedClient, pod.Namespace, pod.Name, "reporter")
	if err != nil {
		return fmt.Errorf("failed to retrieve Pod logs: %w", err)
	}

	report := telemetryReport{}
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		return fmt.Errorf("Pod did not output valid JSON, but %q: %w", output, err)
	}

	if report.Version == "" {
		return fmt.Errorf("telemetry output does not contain a version: %q", output)
	}

	hasKKP := false
	hasK8s := false

	for _, record := range report.Records {
		switch record.Kind {
		case "kubermatic":
			hasKKP = true
		case "kubernetes":
			hasK8s = true
		}
	}

	if !hasKKP || !hasK8s {
		return fmt.Errorf("telemetry output does not contain both kubermatic and kubernetes records; kubermatic=%v, kubernetes=%v", hasKKP, hasK8s)
	}

	log.Info("Successfully validated telemetry.")
	return nil
}

func getLatestTelemetryPod(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (*corev1.Pod, error) {
	podList := corev1.PodList{}
	if err := client.List(ctx, &podList, ctrlruntimeclient.MatchingLabels{"control-plane": "telemetry"}); err != nil {
		return nil, err
	}

	var (
		latest    time.Time
		latestPod *corev1.Pod
	)

	for i, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodSucceeded && pod.CreationTimestamp.After(latest) {
			latestPod = &podList.Items[i]
			latest = pod.CreationTimestamp.Time
		}
	}

	return latestPod, nil
}

func getContainerLogs(ctx context.Context, client kubernetes.Interface, namespace, name, containerName string) (string, error) {
	readCloser, err := client.
		CoreV1().
		Pods(namespace).
		GetLogs(name, &corev1.PodLogOptions{Container: containerName}).
		Stream(ctx)
	if err != nil {
		return "", err
	}
	defer readCloser.Close()

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, readCloser); err != nil {
		return "", err
	}

	return buf.String(), nil
}
