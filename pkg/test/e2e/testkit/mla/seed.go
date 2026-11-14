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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/util/wait"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// the seed-mla installer stack deploys charts/monitoring into
	// monitoringNamespace and charts/logging into loggingNamespace.
	monitoringNamespace = "monitoring"
	loggingNamespace    = "logging"

	seedPrometheusService   = "prometheus"
	seedPrometheusPort      = 9090
	seedAlertmanagerService = "alertmanager"
	seedAlertmanagerPort    = 9093
	seedKarmaService        = "karma"
	seedKarmaPort           = 8080
	seedGrafanaService      = "grafana"
	seedGrafanaPort         = 3000
	seedBlackboxService     = "blackbox-exporter"
	seedBlackboxPort        = 9115
	seedLokiService         = "loki"
	seedLokiPort            = 3100

	// seed Grafana runs with basic auth off and auth.proxy on, keyed on this
	// header (charts/monitoring/grafana/config/grafana.ini). The header is not
	// one of the front proxy headers the kube-apiserver strips, so it passes
	// the service proxy.
	seedGrafanaAuthHeader  = "X-Forwarded-Email"
	defaultSeedGrafanaUser = "admin"

	// seedDashboardsConfigMap holds the dashboard JSON files the chart
	// provisions into seed Grafana.
	seedDashboardsConfigMap = "grafana-dashboard-definitions"

	seedTestAlertName = "KKPE2ESeedMLATest"
)

// requiredSeedJobs are scrape jobs of charts/monitoring/prometheus that exist on
// every seed. Each needs at least one target and all of them must be up.
var requiredSeedJobs = []string{"apiserver", "cadvisor", "kube-state-metrics", "kubelet", "node-exporter", "pods"}

// SeedInput is what the seed MLA checks need. They only talk to the seed.
type SeedInput struct {
	SeedClient ctrlruntimeclient.Client
	SeedConfig *rest.Config

	// GrafanaUser is the seed Grafana admin user name (chart value
	// grafana.user). Empty means "admin", the chart default.
	GrafanaUser string

	Log *zap.SugaredLogger
}

// Validate returns an error if a required field is missing.
func (in SeedInput) Validate() error {
	var errs []error

	if in.SeedClient == nil {
		errs = append(errs, errors.New("SeedClient is required"))
	}
	if in.SeedConfig == nil {
		errs = append(errs, errors.New("SeedConfig is required"))
	}
	if in.Log == nil {
		errs = append(errs, errors.New("Log is required"))
	}

	return errors.Join(errs...)
}

// Seed returns the seed part of the input.
func (in Input) Seed() SeedInput {
	return SeedInput{SeedClient: in.SeedClient, SeedConfig: in.SeedConfig, Log: in.Log}
}

func (in SeedInput) grafanaUser() string {
	if in.GrafanaUser == "" {
		return defaultSeedGrafanaUser
	}

	return in.GrafanaUser
}

// CheckSeedWorkloads verifies that every Deployment, StatefulSet and DaemonSet
// of the seed MLA stack is fully ready. The logging namespace is optional,
// because the installer can skip logging with --mla-skip-logging.
func CheckSeedWorkloads(ctx context.Context, in SeedInput) error {
	err := wait.PollImmediate(ctx, 5*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		var problems []string

		for _, namespace := range []string{monitoringNamespace, loggingNamespace} {
			ns := &corev1.Namespace{}
			if err := in.SeedClient.Get(ctx, types.NamespacedName{Name: namespace}, ns); err != nil {
				if apierrors.IsNotFound(err) && namespace == loggingNamespace {
					continue
				}

				return nil, fmt.Errorf("failed to get namespace %s: %w", namespace, err)
			}

			notReady, total, err := notReadyWorkloads(ctx, in.SeedClient, namespace)
			if err != nil {
				return err, nil
			}

			if total == 0 {
				problems = append(problems, fmt.Sprintf("namespace %s has no workloads", namespace))
			}

			problems = append(problems, notReady...)
		}

		if len(problems) > 0 {
			return fmt.Errorf("not ready: %s", strings.Join(problems, ", ")), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("seed MLA workloads are not ready: %w", err)
	}

	return nil
}

func notReadyWorkloads(ctx context.Context, client ctrlruntimeclient.Client, namespace string) ([]string, int, error) {
	var notReady []string

	deployments := &appsv1.DeploymentList{}
	if err := client.List(ctx, deployments, ctrlruntimeclient.InNamespace(namespace)); err != nil {
		return nil, 0, fmt.Errorf("failed to list Deployments: %w", err)
	}
	for _, d := range deployments.Items {
		want := int32(1)
		if d.Spec.Replicas != nil {
			want = *d.Spec.Replicas
		}
		if d.Status.ReadyReplicas != want || d.Status.UpdatedReplicas != want {
			notReady = append(notReady, fmt.Sprintf("Deployment %s/%s (%d/%d)", namespace, d.Name, d.Status.ReadyReplicas, want))
		}
	}

	statefulSets := &appsv1.StatefulSetList{}
	if err := client.List(ctx, statefulSets, ctrlruntimeclient.InNamespace(namespace)); err != nil {
		return nil, 0, fmt.Errorf("failed to list StatefulSets: %w", err)
	}
	for _, s := range statefulSets.Items {
		want := int32(1)
		if s.Spec.Replicas != nil {
			want = *s.Spec.Replicas
		}
		if s.Status.ReadyReplicas != want {
			notReady = append(notReady, fmt.Sprintf("StatefulSet %s/%s (%d/%d)", namespace, s.Name, s.Status.ReadyReplicas, want))
		}
	}

	daemonSets := &appsv1.DaemonSetList{}
	if err := client.List(ctx, daemonSets, ctrlruntimeclient.InNamespace(namespace)); err != nil {
		return nil, 0, fmt.Errorf("failed to list DaemonSets: %w", err)
	}
	for _, d := range daemonSets.Items {
		if d.Status.DesiredNumberScheduled == 0 || d.Status.NumberReady != d.Status.DesiredNumberScheduled {
			notReady = append(notReady, fmt.Sprintf("DaemonSet %s/%s (%d/%d)", namespace, d.Name, d.Status.NumberReady, d.Status.DesiredNumberScheduled))
		}
	}

	total := len(deployments.Items) + len(statefulSets.Items) + len(daemonSets.Items)

	return notReady, total, nil
}

// promAPIResponse is the envelope of every Prometheus HTTP API response.
type promAPIResponse struct {
	Status string          `json:"status"`
	Error  string          `json:"error"`
	Data   json.RawMessage `json:"data"`
}

func promGet(ctx context.Context, proxy *serviceProxy, path string, out any) error {
	code, body, err := proxy.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return err
	}

	if code != http.StatusOK {
		return fmt.Errorf("expected HTTP 200, got HTTP %d: %s", code, body)
	}

	response := &promAPIResponse{}
	if err := json.Unmarshal(body, response); err != nil {
		return fmt.Errorf("unable to decode response: %w", err)
	}

	if response.Status != "success" {
		return fmt.Errorf("query failed: %s", response.Error)
	}

	return json.Unmarshal(response.Data, out)
}

type promVectorSample struct {
	Metric map[string]string `json:"metric"`
	Value  []any             `json:"value"`
}

type promVector struct {
	ResultType string             `json:"resultType"`
	Result     []promVectorSample `json:"result"`
}

func promQuery(ctx context.Context, proxy *serviceProxy, query string) ([]promVectorSample, error) {
	vector := &promVector{}
	if err := promGet(ctx, proxy, "/api/v1/query?query="+url.QueryEscape(query), vector); err != nil {
		return nil, fmt.Errorf("query %s: %w", query, err)
	}

	return vector.Result, nil
}

func sampleValue(sample promVectorSample) string {
	if len(sample.Value) != 2 {
		return ""
	}

	value, _ := sample.Value[1].(string)

	return value
}

func seedPrometheus(in SeedInput) (*serviceProxy, error) {
	return newServiceProxy(in.SeedConfig, monitoringNamespace, seedPrometheusService, seedPrometheusPort)
}

// CheckSeedTargets verifies that every required scrape job of the seed
// Prometheus has targets and that all of them are up.
func CheckSeedTargets(ctx context.Context, in SeedInput) error {
	proxy, err := seedPrometheus(in)
	if err != nil {
		return err
	}

	type targets struct {
		ActiveTargets []struct {
			Labels    map[string]string `json:"labels"`
			Health    string            `json:"health"`
			LastError string            `json:"lastError"`
			ScrapeURL string            `json:"scrapeUrl"`
		} `json:"activeTargets"`
	}

	err = wait.PollImmediate(ctx, 10*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		result := &targets{}
		if err := promGet(ctx, proxy, "/api/v1/targets?state=active", result); err != nil {
			return err, nil
		}

		total := map[string]int{}
		var down []string

		for _, target := range result.ActiveTargets {
			job := target.Labels["job"]
			total[job]++

			if target.Health != "up" {
				down = append(down, fmt.Sprintf("%s %s (%s)", job, target.ScrapeURL, target.LastError))
			}
		}

		var problems []string
		for _, job := range requiredSeedJobs {
			if total[job] == 0 {
				problems = append(problems, fmt.Sprintf("job %s has no targets", job))
			}
		}

		for _, d := range down {
			for _, job := range requiredSeedJobs {
				if strings.HasPrefix(d, job+" ") {
					problems = append(problems, "target down: "+d)
				}
			}
		}

		if len(problems) > 0 {
			return errors.New(strings.Join(problems, "; ")), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("seed Prometheus targets: %w", err)
	}

	return nil
}

// CheckSeedRules verifies that the seed Prometheus loaded rules and that none
// of them fails to evaluate.
func CheckSeedRules(ctx context.Context, in SeedInput) error {
	proxy, err := seedPrometheus(in)
	if err != nil {
		return err
	}

	type rules struct {
		Groups []struct {
			Name  string `json:"name"`
			Rules []struct {
				Name      string `json:"name"`
				Health    string `json:"health"`
				LastError string `json:"lastError"`
			} `json:"rules"`
		} `json:"groups"`
	}

	// "unknown" means not evaluated yet, which settles after one interval
	err = wait.PollImmediate(ctx, 10*time.Second, 3*time.Minute, func(ctx context.Context) (error, error) {
		result := &rules{}
		if err := promGet(ctx, proxy, "/api/v1/rules", result); err != nil {
			return err, nil
		}

		if len(result.Groups) == 0 {
			return errors.New("no rule groups loaded"), nil
		}

		var failing, pending []string
		for _, group := range result.Groups {
			for _, rule := range group.Rules {
				switch rule.Health {
				case "ok":
				case "unknown":
					pending = append(pending, group.Name+"/"+rule.Name)
				default:
					failing = append(failing, fmt.Sprintf("%s/%s: %s", group.Name, rule.Name, rule.LastError))
				}
			}
		}

		if len(failing) > 0 {
			return nil, fmt.Errorf("%d rules fail to evaluate: %s", len(failing), strings.Join(failing, "; "))
		}

		if len(pending) > 0 {
			return fmt.Errorf("%d rules not evaluated yet", len(pending)), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("seed Prometheus rules: %w", err)
	}

	return nil
}

// CheckSeedLabels verifies the label contract the seed rules rely on: the
// node-exporter series carry app_kubernetes_io_name and not the old app label,
// and the node recording rules produce data.
func CheckSeedLabels(ctx context.Context, in SeedInput) error {
	proxy, err := seedPrometheus(in)
	if err != nil {
		return err
	}

	err = wait.PollImmediate(ctx, 15*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		var problems []string

		up, err := promQuery(ctx, proxy, `up{job="node-exporter"}`)
		if err != nil {
			return err, nil
		}
		if len(up) == 0 {
			problems = append(problems, "no node-exporter targets")
		}
		for _, sample := range up {
			if sampleValue(sample) != "1" {
				problems = append(problems, fmt.Sprintf("node-exporter %s is down", sample.Metric["instance"]))
			}
		}

		// count() returns no sample when the selector matches nothing
		dead, err := promQuery(ctx, proxy, `count({app="node-exporter"})`)
		if err != nil {
			return err, nil
		}
		if len(dead) > 0 {
			problems = append(problems, `series still carry the old label app="node-exporter"`)
		}

		for _, query := range []string{`count({app_kubernetes_io_name="node-exporter"})`, `node:node_num_cpu:sum`, `node:node_cpu_utilisation:avg1m`} {
			result, err := promQuery(ctx, proxy, query)
			if err != nil {
				return err, nil
			}
			if len(result) == 0 {
				problems = append(problems, query+" returns no data")
			}
		}

		if len(problems) > 0 {
			return errors.New(strings.Join(problems, "; ")), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("seed label contract: %w", err)
	}

	return nil
}

// CheckSeedAlertRouting posts a short-lived test alert to the seed
// Alertmanager and verifies that Alertmanager and karma show it. The alert
// goes through the seed's routing tree, so receivers configured there are
// notified.
func CheckSeedAlertRouting(ctx context.Context, in SeedInput) error {
	alertmanager, err := newServiceProxy(in.SeedConfig, monitoringNamespace, seedAlertmanagerService, seedAlertmanagerPort)
	if err != nil {
		return err
	}

	karma, err := newServiceProxy(in.SeedConfig, monitoringNamespace, seedKarmaService, seedKarmaPort)
	if err != nil {
		return err
	}

	runID := rand.String(8)
	now := time.Now().UTC()

	alert := []map[string]any{{
		"labels": map[string]string{
			"alertname": seedTestAlertName,
			"e2e_run":   runID,
			"severity":  "none",
		},
		"annotations": map[string]string{
			"summary": "test alert posted by the KKP seed MLA e2e suite",
		},
		"startsAt": now.Format(time.RFC3339),
		"endsAt":   now.Add(5 * time.Minute).Format(time.RFC3339),
	}}

	payload, err := json.Marshal(alert)
	if err != nil {
		return err
	}

	header := http.Header{"Content-Type": []string{"application/json"}}

	code, body, err := alertmanager.do(ctx, http.MethodPost, "/api/v2/alerts", header, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to post alert: %w", err)
	}
	if code != http.StatusOK {
		return fmt.Errorf("posting alert: expected HTTP 200, got HTTP %d: %s", code, body)
	}

	filter := url.QueryEscape(fmt.Sprintf(`e2e_run=%q`, runID))

	err = wait.PollImmediate(ctx, 5*time.Second, 2*time.Minute, func(ctx context.Context) (error, error) {
		code, body, err := alertmanager.do(ctx, http.MethodGet, "/api/v2/alerts?filter="+filter, nil, nil)
		if err != nil {
			return err, nil
		}
		if code != http.StatusOK {
			return fmt.Errorf("expected HTTP 200, got HTTP %d: %s", code, body), nil
		}

		var alerts []json.RawMessage
		if err := json.Unmarshal(body, &alerts); err != nil {
			return fmt.Errorf("unable to decode alerts: %w", err), nil
		}
		if len(alerts) == 0 {
			return errors.New("test alert not in Alertmanager yet"), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("Alertmanager: %w", err)
	}

	karmaQuery, err := json.Marshal(map[string]any{
		"filters":           []string{"alertname=" + seedTestAlertName, "e2e_run=" + runID},
		"gridLabel":         "",
		"gridLimits":        map[string]int{},
		"sortOrder":         "",
		"sortLabel":         "",
		"sortReverse":       false,
		"defaultGroupLimit": 5,
		"groupLimits":       map[string]int{},
	})
	if err != nil {
		return err
	}

	// karma polls Alertmanager on its own interval, one minute by default
	err = wait.PollImmediate(ctx, 10*time.Second, 3*time.Minute, func(ctx context.Context) (error, error) {
		code, body, err := karma.do(ctx, http.MethodPost, "/alerts.json", header, bytes.NewReader(karmaQuery))
		if err != nil {
			return err, nil
		}
		if code != http.StatusOK {
			return fmt.Errorf("expected HTTP 200, got HTTP %d: %s", code, body), nil
		}

		response := struct {
			TotalAlerts int `json:"totalAlerts"`
		}{}
		if err := json.Unmarshal(body, &response); err != nil {
			return fmt.Errorf("unable to decode karma response: %w", err), nil
		}
		if response.TotalAlerts == 0 {
			return errors.New("test alert not in karma yet"), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("karma: %w", err)
	}

	return nil
}

// CheckSeedGrafana verifies that every seed Grafana datasource is healthy and
// that the dashboards the chart ships are provisioned.
func CheckSeedGrafana(ctx context.Context, in SeedInput) error {
	grafana, err := newServiceProxy(in.SeedConfig, monitoringNamespace, seedGrafanaService, seedGrafanaPort)
	if err != nil {
		return err
	}

	header := http.Header{seedGrafanaAuthHeader: []string{in.grafanaUser()}}

	get := func(ctx context.Context, path string, out any) error {
		code, body, err := grafana.do(ctx, http.MethodGet, path, header, nil)
		if err != nil {
			return err
		}
		if code != http.StatusOK {
			return fmt.Errorf("%s: expected HTTP 200, got HTTP %d: %s", path, code, body)
		}

		return json.Unmarshal(body, out)
	}

	expectedDashboards := 0
	configMap := &corev1.ConfigMap{}
	if err := in.SeedClient.Get(ctx, types.NamespacedName{Namespace: monitoringNamespace, Name: seedDashboardsConfigMap}, configMap); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get ConfigMap %s: %w", seedDashboardsConfigMap, err)
		}
	} else {
		for key := range configMap.Data {
			if strings.HasSuffix(key, ".json") {
				expectedDashboards++
			}
		}
	}

	err = wait.PollImmediate(ctx, 10*time.Second, 3*time.Minute, func(ctx context.Context) (error, error) {
		var datasources []struct {
			Name string `json:"name"`
			UID  string `json:"uid"`
		}
		if err := get(ctx, "/api/datasources", &datasources); err != nil {
			return err, nil
		}
		if len(datasources) == 0 {
			return errors.New("seed Grafana has no datasources"), nil
		}

		var problems []string
		for _, ds := range datasources {
			health := struct {
				Status  string `json:"status"`
				Message string `json:"message"`
			}{}
			if err := get(ctx, "/api/datasources/uid/"+ds.UID+"/health", &health); err != nil {
				problems = append(problems, fmt.Sprintf("datasource %s: %v", ds.Name, err))
				continue
			}
			if health.Status != "OK" {
				problems = append(problems, fmt.Sprintf("datasource %s: %s %s", ds.Name, health.Status, health.Message))
			}
		}

		var dashboards []json.RawMessage
		if err := get(ctx, "/api/search?type=dash-db&limit=5000", &dashboards); err != nil {
			return err, nil
		}
		if len(dashboards) == 0 || len(dashboards) < expectedDashboards {
			problems = append(problems, fmt.Sprintf("%d dashboards provisioned, ConfigMap %s ships %d", len(dashboards), seedDashboardsConfigMap, expectedDashboards))
		}

		if len(problems) > 0 {
			return errors.New(strings.Join(problems, "; ")), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("seed Grafana: %w", err)
	}

	return nil
}

// CheckSeedLogs verifies that seed pod logs reach the seed Loki.
func CheckSeedLogs(ctx context.Context, in SeedInput) error {
	loki, err := seedLoki(in)
	if err != nil {
		return err
	}

	err = wait.PollImmediate(ctx, 10*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		found, err := lokiHasLines(ctx, loki, `{namespace="kubermatic"}`, time.Now().Add(-10*time.Minute))
		if err != nil {
			return err, nil
		}
		if !found {
			return errors.New("no log lines from namespace kubermatic in the last 10 minutes"), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("seed Loki: %w", err)
	}

	return nil
}

// CheckBlackboxExporter runs an HTTP probe through the seed blackbox exporter
// against the seed Prometheus, and verifies that every probe the seed
// Prometheus already scrapes succeeds.
func CheckBlackboxExporter(ctx context.Context, in SeedInput) error {
	blackbox, err := newServiceProxy(in.SeedConfig, monitoringNamespace, seedBlackboxService, seedBlackboxPort)
	if err != nil {
		return err
	}

	target := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d/-/healthy", seedPrometheusService, monitoringNamespace, seedPrometheusPort)
	path := "/probe?module=http_2xx&target=" + url.QueryEscape(target)

	err = wait.PollImmediate(ctx, 5*time.Second, 2*time.Minute, func(ctx context.Context) (error, error) {
		code, body, err := blackbox.do(ctx, http.MethodGet, path, nil, nil)
		if err != nil {
			return err, nil
		}
		if code != http.StatusOK {
			return fmt.Errorf("expected HTTP 200, got HTTP %d: %s", code, body), nil
		}

		for _, line := range strings.Split(string(body), "\n") {
			if line == "probe_success 1" {
				return nil, nil
			}
		}

		return fmt.Errorf("probe of %s did not succeed", target), nil
	})
	if err != nil {
		return fmt.Errorf("blackbox exporter: %w", err)
	}

	prometheus, err := seedPrometheus(in)
	if err != nil {
		return err
	}

	probes, err := promQuery(ctx, prometheus, "probe_success")
	if err != nil {
		return err
	}

	var failing []string
	for _, sample := range probes {
		if sampleValue(sample) != "1" {
			failing = append(failing, sample.Metric["instance"])
		}
	}
	sort.Strings(failing)

	if len(failing) > 0 {
		return fmt.Errorf("probes configured in the seed Prometheus fail: %s", strings.Join(failing, ", "))
	}

	return nil
}
