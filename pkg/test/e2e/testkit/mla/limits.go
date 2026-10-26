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
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/mla"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// burstRequests is how many requests the enforcement checks send at once
// against a limit of 1 request per second with a burst of 1.
const burstRequests = 20

// CheckRateLimits sets monitoring and logging rate limits through the cluster's
// MLAAdminSetting and verifies that they reach the Cortex runtime config and
// the MLA gateway, and that the gateway rejects requests above them. The
// MLAAdminSetting is deleted afterwards. Changing limits rolls out the
// cluster's MLA gateway, so callers must not run this in parallel with
// data-plane checks.
func CheckRateLimits(ctx context.Context, in Input) error {
	monitoring := kubermaticv1.MonitoringRateLimitSettings{
		IngestionRate:      1,
		IngestionBurstSize: 2,
		MaxSeriesPerMetric: 3,
		MaxSeriesTotal:     4,
		MaxSamplesPerQuery: 5,
		MaxSeriesPerQuery:  6,
		QueryRate:          1,
		QueryBurstSize:     1,
	}
	logging := kubermaticv1.LoggingRateLimitSettings{
		IngestionRate:      1,
		IngestionBurstSize: 1,
		QueryRate:          1,
		QueryBurstSize:     1,
	}

	setting := &kubermaticv1.MLAAdminSetting{
		ObjectMeta: metav1.ObjectMeta{Name: resources.MLAAdminSettingsName, Namespace: in.Cluster.Status.NamespaceName},
		Spec: kubermaticv1.MLAAdminSettingSpec{
			ClusterName:          in.Cluster.Name,
			MonitoringRateLimits: &monitoring,
			LoggingRateLimits:    &logging,
		},
	}

	restore, err := applyMLAAdminSetting(ctx, in, setting)
	if err != nil {
		return fmt.Errorf("unable to set rate limits: %w", err)
	}
	defer func() {
		ctx, cancel := revertContext()
		defer cancel()

		if err := restore(ctx); err != nil {
			in.Log.Warnw("Failed to restore the MLAAdminSetting", "error", err)
			return
		}

		// wait for the test limits to leave the gateway config before
		// waiting for the rollout, or the wait sees the old rollout
		if err := waitForGatewayConfig(ctx, in, false); err != nil {
			in.Log.Warnw("Gateway config still has the test rate limits", "error", err)
			return
		}

		if _, err := waitGatewayPodReady(ctx, in.SeedClient, in.Cluster.Status.NamespaceName); err != nil {
			in.Log.Warnw("MLA gateway did not settle after removing the rate limits", "error", err)
		}
	}()

	if err := waitForCortexOverrides(ctx, in, monitoring); err != nil {
		return err
	}

	if err := waitForGatewayLimits(ctx, in); err != nil {
		return err
	}

	probe, err := ensureGatewayProbe(ctx, in)
	if err != nil {
		return err
	}

	readURL := fmt.Sprintf("http://%s/api/prom/api/v1/query?query=up", gatewayReadAddr(probe.namespace))
	if err := expectThrottled(ctx, probe, "Cortex read", parallelCurl(readURL)); err != nil {
		return err
	}

	podIP, err := readyGatewayPodIP(ctx, in.SeedClient, probe.namespace)
	if err != nil {
		return err
	}

	pushBody := fmt.Sprintf(`{"streams":[{"stream":{"job":%q},"values":[[%q,"limit-probe"]]}]}`, lokiPushJob, strconv.FormatInt(time.Now().UnixNano(), 10))
	pushURL := gatewayPushURL(podIP, lokiPushPath)
	push := []string{
		"curl", "-s", "-o", "/dev/null", "-w", "%{http_code}\n", "--parallel", "--parallel-immediate", "--max-time", "15",
		"--cert", probeCertPath, "--key", probeKeyPath, "-k", "-XPOST", "-H", "Content-Type: application/json", "-d", pushBody,
	}
	for range burstRequests {
		push = append(push, pushURL)
	}

	if err := expectThrottled(ctx, probe, "Loki write", push); err != nil {
		return err
	}

	return nil
}

// applyMLAAdminSetting creates setting, or patches an existing one, and returns
// a function that deletes the created object or restores the patched one.
func applyMLAAdminSetting(ctx context.Context, in Input, setting *kubermaticv1.MLAAdminSetting) (func(context.Context) error, error) {
	err := in.SeedClient.Create(ctx, setting.DeepCopy())
	if err == nil {
		return func(ctx context.Context) error {
			if err := in.SeedClient.Delete(ctx, setting); err != nil && !apierrors.IsNotFound(err) {
				return err
			}
			return nil
		}, nil
	}
	if !apierrors.IsAlreadyExists(err) {
		return nil, err
	}

	existing := &kubermaticv1.MLAAdminSetting{ObjectMeta: metav1.ObjectMeta{Name: setting.Name, Namespace: setting.Namespace}}

	return patchWithRevert(ctx, in.SeedClient, existing, func() {
		existing.Spec = setting.Spec
	})
}

func waitForCortexOverrides(ctx context.Context, in Input, monitoring kubermaticv1.MonitoringRateLimitSettings) error {
	err := wait.PollImmediate(ctx, time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		configMap := &corev1.ConfigMap{}
		if err := in.SeedClient.Get(ctx, types.NamespacedName{Namespace: mlaNamespace, Name: mla.RuntimeConfigMap}, configMap); err != nil {
			return fmt.Errorf("unable to get the runtime ConfigMap: %w", err), nil
		}

		overrides := &mla.Overrides{}
		decoder := yaml.NewDecoder(strings.NewReader(configMap.Data[mla.RuntimeConfigFileName]))
		decoder.KnownFields(true)
		if err := decoder.Decode(overrides); err != nil {
			return fmt.Errorf("unable to decode the runtime config: %w", err), nil
		}

		actual, ok := overrides.Overrides[in.Cluster.Name]
		if !ok {
			return errors.New("runtime config has no overrides for the cluster yet"), nil
		}

		if !overridesMatch(actual, monitoring) {
			return errors.New("runtime config overrides do not match the configured rate limits"), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("Cortex rate limits not applied: %w", err)
	}

	return nil
}

// waitForGatewayLimits waits until the gateway nginx config carries the query
// and logging limits and the gateway rollout is complete.
func waitForGatewayLimits(ctx context.Context, in Input) error {
	if err := waitForGatewayConfig(ctx, in, true); err != nil {
		return fmt.Errorf("gateway rate limits not applied: %w", err)
	}

	if _, err := waitGatewayPodReady(ctx, in.SeedClient, in.Cluster.Status.NamespaceName); err != nil {
		return fmt.Errorf("gateway did not roll out the rate limits: %w", err)
	}

	return nil
}

// testGatewayLimits are the nginx zones the rate limit check configures.
var testGatewayLimits = []string{
	"zone=cortex_read_limit:1m rate=1r/s",
	"zone=loki_write_limit:1m rate=1r/s",
	"zone=loki_read_limit:1m rate=1r/s",
}

// waitForGatewayConfig waits until the gateway nginx config has all the test
// limits (present) or none of them.
func waitForGatewayConfig(ctx context.Context, in Input, present bool) error {
	return wait.PollImmediate(ctx, 2*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		configMap := &corev1.ConfigMap{}
		if err := in.SeedClient.Get(ctx, types.NamespacedName{Namespace: in.Cluster.Status.NamespaceName, Name: gatewayName}, configMap); err != nil {
			return fmt.Errorf("unable to get the gateway ConfigMap: %w", err), nil
		}

		var config string
		for _, value := range configMap.Data {
			config += value
		}

		var wrong []string
		for _, limit := range testGatewayLimits {
			if strings.Contains(config, limit) != present {
				wrong = append(wrong, limit)
			}
		}
		if len(wrong) > 0 {
			return fmt.Errorf("gateway config: present=%v expected for %s", present, strings.Join(wrong, ", ")), nil
		}

		return nil, nil
	})
}

// parallelCurl sends burstRequests GET requests to url at once and prints one
// HTTP status code per line.
func parallelCurl(url string) []string {
	curl := []string{"curl", "-s", "-o", "/dev/null", "-w", "%{http_code}\n", "--parallel", "--parallel-immediate", "--max-time", "15"}
	for range burstRequests {
		curl = append(curl, url)
	}

	return curl
}

// expectThrottled runs command from the probe pod and expects at least one
// response to be rejected by the gateway's nginx limit_req. nginx answers 503
// by default; 429 is accepted too in case limit_req_status is set.
func expectThrottled(ctx context.Context, probe *gatewayProbe, what string, command []string) error {
	var lastCodes string

	err := wait.PollImmediate(ctx, 5*time.Second, 2*time.Minute, func(ctx context.Context) (error, error) {
		stdout, _, err := probe.exec(ctx, command...)
		if err != nil {
			return fmt.Errorf("burst failed to execute: %w", err), nil
		}

		lastCodes = strings.Join(strings.Fields(stdout), ",")

		var ok, throttled int
		for _, code := range strings.Fields(stdout) {
			switch {
			case code == "503" || code == "429":
				throttled++
			case isSuccessPushCode(code):
				ok++
			}
		}

		if ok == 0 {
			return fmt.Errorf("no request succeeded (codes: %s)", lastCodes), nil
		}
		if throttled == 0 {
			return fmt.Errorf("no request was throttled (codes: %s)", lastCodes), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("%s rate limit not enforced: %w", what, err)
	}

	return nil
}
