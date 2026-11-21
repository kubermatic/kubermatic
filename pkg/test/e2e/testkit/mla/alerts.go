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
	"net/http"
	"reflect"
	"strings"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/mla"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/podexec"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/ptr"
)

const (
	alertSinkImage = "docker.io/library/busybox:1.36"
	alertSinkPort  = 8080
	alertSinkLog   = "/tmp/alerts.log"

	metricsAlertName = "E2EMetricsAlwaysFiring"
	logsAlertName    = "E2ELogsAlwaysFiring"
)

// alertSink is a busybox httpd in the MLA namespace that appends every webhook
// request body it receives to a file.
type alertSink struct {
	in   Input
	name string
}

func (s *alertSink) url() string {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d/cgi-bin/hook", s.name, mlaNamespace, alertSinkPort)
}

func (s *alertSink) received(ctx context.Context) (string, error) {
	stdout, _, err := podexec.ExecuteCommand(ctx, s.in.SeedConfig, types.NamespacedName{Namespace: mlaNamespace, Name: s.name}, "sink", "sh", "-c", "cat "+alertSinkLog+" 2>/dev/null || true")

	return stdout, err
}

func deployAlertSink(ctx context.Context, in Input, name string) (*alertSink, func(), error) {
	hook := `#!/bin/sh
echo "Content-Type: text/plain"
echo ""
cat >> ` + alertSinkLog + `
echo >> ` + alertSinkLog + `
echo ok
`
	script := fmt.Sprintf("mkdir -p /www/cgi-bin && cat > /www/cgi-bin/hook <<'EOF'\n%sEOF\nchmod +x /www/cgi-bin/hook && exec httpd -f -p %d -h /www", hook, alertSinkPort)

	labels := map[string]string{"app.kubernetes.io/name": name}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: mlaNamespace, Labels: labels},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:    "sink",
				Image:   alertSinkImage,
				Command: []string{"sh", "-c", script},
				Ports:   []corev1.ContainerPort{{ContainerPort: alertSinkPort}},
				ReadinessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(alertSinkPort)}},
				},
			}},
			TerminationGracePeriodSeconds: ptr.To[int64](0),
		},
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: mlaNamespace},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports:    []corev1.ServicePort{{Port: alertSinkPort, TargetPort: intstr.FromInt32(alertSinkPort)}},
		},
	}

	cleanup := func() {
		deleteIgnoreNotFound(in, pod)
		deleteIgnoreNotFound(in, service)
	}

	if err := in.SeedClient.Create(ctx, service); err != nil {
		return nil, nil, fmt.Errorf("failed to create alert sink Service: %w", err)
	}

	if err := ensurePod(ctx, in.SeedClient, in, pod); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to deploy alert sink: %w", err)
	}

	return &alertSink{in: in, name: name}, cleanup, nil
}

// waitForAlertmanagerConfig waits until Cortex serves config for the cluster's
// tenant and the cluster reports the config as healthy.
func waitForAlertmanagerConfig(ctx context.Context, in Input, config string) error {
	expected := map[string]interface{}{}
	if err := yamlUnmarshal([]byte(config), &expected); err != nil {
		return fmt.Errorf("unable to unmarshal expected config: %w", err)
	}

	proxy, err := newServiceProxy(in.SeedConfig, mlaNamespace, cortexAlertmanagerService, cortexAlertmanagerPort)
	if err != nil {
		return err
	}

	header := http.Header{mla.AlertmanagerTenantHeaderName: []string{in.Cluster.Name}}

	return wait.PollImmediate(ctx, 2*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		code, body, err := proxy.do(ctx, http.MethodGet, mla.AlertmanagerConfigEndpoint, header, nil)
		if err != nil {
			return err, nil
		}
		if code != http.StatusOK {
			return fmt.Errorf("expected HTTP 200, got HTTP %d: %s", code, body), nil
		}

		actual := map[string]interface{}{}
		if err := yamlUnmarshal(body, &actual); err != nil {
			return fmt.Errorf("unable to decode response body: %w", err), nil
		}

		if !reflect.DeepEqual(actual, expected) {
			return errors.New("served config does not match the configured one yet"), nil
		}

		return nil, nil
	})
}

// CheckAlertDelivery routes the cluster's alerts to a webhook receiver and
// verifies that an always-firing metrics rule (Cortex ruler) and, when
// withLogs is set, an always-firing logs rule (Loki ruler) are delivered.
// Only one Alertmanager config exists per tenant, so callers must not run this
// in parallel with other checks that write it. The original config is
// restored afterwards.
func checkAlertDelivery(ctx context.Context, in Input, withLogs bool) error {
	suffix := rand.String(6)

	sink, cleanupSink, err := deployAlertSink(ctx, in, "mla-e2e-alert-sink-"+suffix)
	if err != nil {
		return err
	}
	defer cleanupSink()

	config := fmt.Sprintf(`template_files: {}
alertmanager_config: |
  route:
    receiver: e2e-sink
    group_by: [alertname]
    group_wait: 5s
    group_interval: 10s
    repeat_interval: 1m
  receivers:
    - name: e2e-sink
      webhook_configs:
        - url: %s
          send_resolved: false
`, sink.url())

	restore, err := replaceAlertmanagerConfig(ctx, in, []byte(config))
	if err != nil {
		return err
	}
	defer restore()

	if err := waitForAlertmanagerConfig(ctx, in, config); err != nil {
		return fmt.Errorf("Alertmanager config with the e2e receiver not served: %w", err)
	}

	metricsRules := fmt.Sprintf(`name: mla-e2e-metrics-alert-%s
interval: 15s
rules:
- alert: %s
  expr: vector(1)
  labels:
    e2e_run: %s
`, suffix, metricsAlertName, suffix)

	groups := []struct {
		data string
		kind kubermaticv1.RuleGroupType
	}{{metricsRules, kubermaticv1.RuleGroupTypeMetrics}}

	expectedAlerts := []string{metricsAlertName}

	if withLogs {
		logsRules := fmt.Sprintf(`name: mla-e2e-logs-alert-%s
interval: 15s
rules:
- alert: %s
  expr: sum(count_over_time({job=%q}[1m])) > 0
  labels:
    e2e_run: %s
`, suffix, logsAlertName, logsProbeJob, suffix)

		groups = append(groups, struct {
			data string
			kind kubermaticv1.RuleGroupType
		}{logsRules, kubermaticv1.RuleGroupTypeLogs})
		expectedAlerts = append(expectedAlerts, logsAlertName)
	}

	for _, group := range groups {
		ruleGroup, _, err := createRuleGroupObject(ctx, in, []byte(group.data), group.kind)
		if err != nil {
			return err
		}
		defer deleteIgnoreNotFound(in, ruleGroup)
	}

	err = wait.PollImmediate(ctx, 10*time.Second, 6*time.Minute, func(ctx context.Context) (error, error) {
		received, err := sink.received(ctx)
		if err != nil {
			return err, nil
		}

		var missing []string
		for _, alert := range expectedAlerts {
			if !strings.Contains(received, alert) || !strings.Contains(received, suffix) {
				missing = append(missing, alert)
			}
		}

		if len(missing) > 0 {
			return fmt.Errorf("receiver has not received %s yet", strings.Join(missing, ", ")), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("alert delivery: %w", err)
	}

	return nil
}

// CheckMetricsAlertDelivery verifies that a metrics alert reaches a webhook
// receiver configured in the cluster's Alertmanager config.
func CheckMetricsAlertDelivery(ctx context.Context, in Input) error {
	return checkAlertDelivery(ctx, in, false)
}

// CheckLogsAlertDelivery verifies that a metrics alert and a logs alert both
// reach the receiver. The logs rule fires on the logs probe, so it needs the
// logging agent running on worker nodes.
func CheckLogsAlertDelivery(ctx context.Context, in Input) error {
	if err := EnsureLogsProbe(ctx, in); err != nil {
		return err
	}

	return checkAlertDelivery(ctx, in, true)
}

// currentAlertmanagerConfig returns the content of the cluster's Alertmanager
// config Secret.
func currentAlertmanagerConfig(ctx context.Context, in Input) ([]byte, error) {
	alertmanager := &kubermaticv1.Alertmanager{}
	if err := in.SeedClient.Get(ctx, types.NamespacedName{Name: resources.DefaultAlertmanagerConfigSecretName, Namespace: in.Cluster.Status.NamespaceName}, alertmanager); err != nil {
		return nil, fmt.Errorf("failed to get Alertmanager: %w", err)
	}

	secret := &corev1.Secret{}
	if err := in.SeedClient.Get(ctx, types.NamespacedName{Name: alertmanager.Spec.ConfigSecret.Name, Namespace: alertmanager.Namespace}, secret); err != nil {
		return nil, fmt.Errorf("failed to get Alertmanager config Secret: %w", err)
	}

	return secret.Data[resources.AlertmanagerConfigSecretKey], nil
}

// replaceAlertmanagerConfig writes config into the cluster's Alertmanager
// config Secret and returns a function that restores the previous content.
func replaceAlertmanagerConfig(ctx context.Context, in Input, config []byte) (func(), error) {
	alertmanager := &kubermaticv1.Alertmanager{}
	if err := in.SeedClient.Get(ctx, types.NamespacedName{Name: resources.DefaultAlertmanagerConfigSecretName, Namespace: in.Cluster.Status.NamespaceName}, alertmanager); err != nil {
		return nil, fmt.Errorf("failed to get Alertmanager: %w", err)
	}

	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: alertmanager.Spec.ConfigSecret.Name, Namespace: alertmanager.Namespace}}

	revert, err := patchWithRevert(ctx, in.SeedClient, secret, func() {
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		secret.Data[resources.AlertmanagerConfigSecretKey] = config
	})
	if err != nil {
		return nil, err
	}

	return func() {
		ctx, cancel := revertContext()
		defer cancel()

		if err := revert(ctx); err != nil {
			in.Log.Warnw("Failed to restore the Alertmanager config", "error", err)
		}
	}, nil
}

// CheckDefaultRuleGroup creates an admin default RuleGroup in the MLA
// namespace and verifies it is synced into the cluster and its ruler, and
// removed from both when deleted. Every MLA cluster on the seed gets the rule
// group while it exists.
func CheckDefaultRuleGroup(ctx context.Context, in Input) error {
	suffix := rand.String(6)
	name := "mla-e2e-default-" + suffix
	data := fmt.Sprintf("name: %s\nrules:\n- record: mla_e2e_default_rule_%s\n  expr: vector(1)\n", name, suffix)

	ruleGroup := &kubermaticv1.RuleGroup{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: mlaNamespace},
		Spec: kubermaticv1.RuleGroupSpec{
			IsDefault:     true,
			RuleGroupType: kubermaticv1.RuleGroupTypeMetrics,
			Data:          []byte(data),
		},
	}
	if err := in.SeedClient.Create(ctx, ruleGroup); err != nil {
		return fmt.Errorf("failed to create default RuleGroup: %w", err)
	}
	defer deleteIgnoreNotFound(in, ruleGroup)

	proxy, err := newServiceProxy(in.SeedConfig, mlaNamespace, cortexRulerService, cortexRulerPort)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s/default/%s", mla.MetricsRuleGroupConfigEndpoint, name)
	header := http.Header{mla.RuleGroupTenantHeaderName: []string{in.Cluster.Name}}

	err = wait.PollImmediate(ctx, 2*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		synced := &kubermaticv1.RuleGroup{}
		if err := in.SeedClient.Get(ctx, types.NamespacedName{Namespace: in.Cluster.Status.NamespaceName, Name: name}, synced); err != nil {
			return fmt.Errorf("RuleGroup not synced into the cluster namespace yet: %w", err), nil
		}

		code, body, err := proxy.do(ctx, http.MethodGet, path, header, nil)
		if err != nil {
			return err, nil
		}
		if code != http.StatusOK {
			return fmt.Errorf("ruler: expected HTTP 200, got HTTP %d: %s", code, body), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("default rule group not synced: %w", err)
	}

	if err := in.SeedClient.Delete(ctx, ruleGroup); err != nil {
		return fmt.Errorf("failed to delete default RuleGroup: %w", err)
	}

	err = wait.PollImmediate(ctx, 2*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		code, body, err := proxy.do(ctx, http.MethodGet, path, header, nil)
		if err != nil {
			return err, nil
		}
		if code != http.StatusNotFound {
			return fmt.Errorf("ruler still serves the group (HTTP %d: %s)", code, body), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("default rule group not removed: %w", err)
	}

	return nil
}
