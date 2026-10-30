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
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/mla"
	"k8c.io/kubermatic/v2/pkg/util/s3"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	minioServiceName = "minio"
	minioServicePort = 9000
	minioSecretName  = "minio"

	cortexBlocksBucket = "cortex"
	lokiChunksBucket   = "loki"

	cortexIngesterPort = 8080
	lokiIngesterPort   = 3100
)

// UpgradeState is what the pre-upgrade setup records for the post-upgrade
// verification. It is written to a file between the two runs.
type UpgradeState struct {
	ClusterName string `json:"clusterName"`
	RunID       string `json:"runID"`

	// SampleTime is a moment at which the metrics probe series and the logs
	// probe marker were queryable.
	SampleTime time.Time `json:"sampleTime"`

	GrafanaOrgID  uint   `json:"grafanaOrgID"`
	RuleGroupName string `json:"ruleGroupName"`
	// OriginalAlertmanagerConfig is restored by CleanupUpgrade.
	OriginalAlertmanagerConfig []byte   `json:"originalAlertmanagerConfig"`
	DashboardUID               string   `json:"dashboardUID"`
	AlertmanagerConfig         string   `json:"alertmanagerConfig"`
	CortexObjects              []string `json:"cortexObjects"`
	LokiObjects                []string `json:"lokiObjects"`

	// StatefulSetUIDs and PVCUIDs of the MLA namespace, by name. A changed
	// UID means the object was recreated, which loses or orphans data.
	StatefulSetUIDs map[string]types.UID `json:"statefulSetUIDs"`
	PVCUIDs         map[string]types.UID `json:"pvcUIDs"`
}

// SetupUpgrade runs before the upgrade, on a cluster with MLA enabled: it
// waits for probe metrics and logs, flushes the ingesters so the data lands in
// object storage, and creates a RuleGroup, an Alertmanager config and a
// dashboard that must survive the upgrade.
func SetupUpgrade(ctx context.Context, in Input) (*UpgradeState, error) {
	state := &UpgradeState{
		ClusterName: in.Cluster.Name,
		RunID:       rand.String(8),
	}

	if err := EnsureMetricsProbe(ctx, in); err != nil {
		return nil, err
	}
	if err := EnsureLogsProbe(ctx, in); err != nil {
		return nil, err
	}

	if err := waitForSeries(ctx, in, fmt.Sprintf("%s{run=%q}", metricsProbeMetric, in.Cluster.Name), 8*time.Minute); err != nil {
		return nil, fmt.Errorf("probe metric: %w", err)
	}
	if err := waitForLogMarker(ctx, in, time.Now()); err != nil {
		return nil, fmt.Errorf("probe logs: %w", err)
	}
	state.SampleTime = time.Now().UTC()

	orgID, err := grafanaOrgID(ctx, in)
	if err != nil {
		return nil, err
	}
	state.GrafanaOrgID = orgID

	state.RuleGroupName = "mla-e2e-upgrade-" + state.RunID
	if _, _, err := createRuleGroupObject(ctx, in, upgradeRuleGroup(state.RuleGroupName, state.RunID), kubermaticv1.RuleGroupTypeMetrics); err != nil {
		return nil, err
	}

	if state.OriginalAlertmanagerConfig, err = currentAlertmanagerConfig(ctx, in); err != nil {
		return nil, err
	}

	state.AlertmanagerConfig = upgradeAlertmanagerConfig(state.RunID)
	if _, err := replaceAlertmanagerConfig(ctx, in, []byte(state.AlertmanagerConfig)); err != nil {
		return nil, err
	}
	if err := waitForAlertmanagerConfig(ctx, in, state.AlertmanagerConfig); err != nil {
		return nil, err
	}

	state.DashboardUID = "mla-e2e-upgrade-" + state.RunID
	if err := createUpgradeDashboard(ctx, in, state.DashboardUID); err != nil {
		return nil, err
	}

	if err := flushIngesters(ctx, in); err != nil {
		return nil, err
	}

	if state.CortexObjects, err = waitForTenantObjects(ctx, in, cortexBlocksBucket); err != nil {
		return nil, fmt.Errorf("Cortex blocks: %w", err)
	}
	if state.LokiObjects, err = waitForTenantObjects(ctx, in, lokiChunksBucket); err != nil {
		return nil, fmt.Errorf("Loki chunks: %w", err)
	}

	if state.StatefulSetUIDs, state.PVCUIDs, err = mlaStorageUIDs(ctx, in); err != nil {
		return nil, err
	}

	return state, nil
}

// VerifyUpgrade runs after the upgrade and verifies that the data and the
// configuration SetupUpgrade recorded survived.
func VerifyUpgrade(ctx context.Context, in Input, state *UpgradeState) error {
	var problems []string

	statefulSets, pvcs, err := mlaStorageUIDs(ctx, in)
	if err != nil {
		return err
	}
	problems = append(problems, compareUIDs("StatefulSet", state.StatefulSetUIDs, statefulSets)...)
	problems = append(problems, compareUIDs("PersistentVolumeClaim", state.PVCUIDs, pvcs)...)

	for _, check := range []struct {
		bucket string
		want   []string
	}{{cortexBlocksBucket, state.CortexObjects}, {lokiChunksBucket, state.LokiObjects}} {
		present, err := listTenantObjects(ctx, in, check.bucket)
		if err != nil {
			problems = append(problems, fmt.Sprintf("bucket %s: %v", check.bucket, err))
			continue
		}
		if missing := missingNames(check.want, present); len(missing) > 0 {
			problems = append(problems, fmt.Sprintf("bucket %s lost %d of %d objects, for example %s", check.bucket, len(missing), len(check.want), missing[0]))
		}
	}

	query := fmt.Sprintf("%s{run=%q}", metricsProbeMetric, in.Cluster.Name)
	if samples, err := cortexQueryAt(ctx, in, query, state.SampleTime); err != nil {
		problems = append(problems, fmt.Sprintf("pre-upgrade metrics: %v", err))
	} else if len(samples) == 0 {
		problems = append(problems, fmt.Sprintf("pre-upgrade metrics: %s at %s returns no data", query, state.SampleTime))
	}

	if err := waitForLogMarker(ctx, in, state.SampleTime); err != nil {
		problems = append(problems, fmt.Sprintf("pre-upgrade logs: %v", err))
	}

	if orgID, err := grafanaOrgID(ctx, in); err != nil {
		problems = append(problems, err.Error())
	} else if orgID != state.GrafanaOrgID {
		problems = append(problems, fmt.Sprintf("Grafana org ID changed from %d to %d", state.GrafanaOrgID, orgID))
	}

	ruler, err := newServiceProxy(in.SeedConfig, mlaNamespace, cortexRulerService, cortexRulerPort)
	if err != nil {
		return err
	}
	expectedGroup := map[string]interface{}{}
	if err := yamlUnmarshal(upgradeRuleGroup(state.RuleGroupName, state.RunID), &expectedGroup); err != nil {
		return err
	}
	rulePath := fmt.Sprintf("%s/default/%s", mla.MetricsRuleGroupConfigEndpoint, state.RuleGroupName)
	if err := waitForRuleGroup(ctx, ruler, rulePath, http.Header{mla.RuleGroupTenantHeaderName: []string{in.Cluster.Name}}, expectedGroup); err != nil {
		problems = append(problems, fmt.Sprintf("RuleGroup %s: %v", state.RuleGroupName, err))
	}

	if err := waitForAlertmanagerConfig(ctx, in, state.AlertmanagerConfig); err != nil {
		problems = append(problems, fmt.Sprintf("Alertmanager config: %v", err))
	}

	err = withGrafanaOrg(ctx, in, func(client *grafanasdk.Client, _ grafanasdk.Org) error {
		_, _, err := client.GetDashboardByUID(ctx, state.DashboardUID)
		return err
	})
	if err != nil {
		problems = append(problems, fmt.Sprintf("dashboard %s: %v", state.DashboardUID, err))
	}

	if len(problems) > 0 {
		return fmt.Errorf("upgrade lost data or configuration:\n%s", strings.Join(problems, "\n"))
	}

	return nil
}

// CleanupUpgrade removes what SetupUpgrade created.
func CleanupUpgrade(ctx context.Context, in Input, state *UpgradeState) {
	deleteIgnoreNotFound(in, &kubermaticv1.RuleGroup{ObjectMeta: metav1.ObjectMeta{Name: state.RuleGroupName, Namespace: in.Cluster.Status.NamespaceName}})

	if state.OriginalAlertmanagerConfig != nil {
		if _, err := replaceAlertmanagerConfig(ctx, in, state.OriginalAlertmanagerConfig); err != nil {
			in.Log.Warnw("Failed to restore the Alertmanager config", "error", err)
		}
	}

	deleteIgnoreNotFound(in, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "grafana-dashboards-" + state.DashboardUID, Namespace: mlaNamespace}})
}

func upgradeRuleGroup(name, runID string) []byte {
	return fmt.Appendf(nil, "name: %s\nrules:\n- record: mla_e2e_upgrade_rule\n  expr: vector(1)\n  labels:\n    e2e_run: %s\n", name, runID)
}

func upgradeAlertmanagerConfig(runID string) string {
	return fmt.Sprintf(`template_files: {}
alertmanager_config: |
  route:
    receiver: e2e-upgrade-%s
  receivers:
    - name: e2e-upgrade-%s
`, runID, runID)
}

func createUpgradeDashboard(ctx context.Context, in Input, uid string) error {
	dashboard, err := json.Marshal(map[string]any{"uid": uid, "title": uid, "panels": []any{}, "schemaVersion": 39})
	if err != nil {
		return err
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "grafana-dashboards-" + uid, Namespace: mlaNamespace},
		Data:       map[string]string{uid + ".json": string(dashboard)},
	}
	if err := in.SeedClient.Create(ctx, configMap); err != nil {
		return fmt.Errorf("failed to create dashboard ConfigMap: %w", err)
	}

	return withGrafanaOrg(ctx, in, func(client *grafanasdk.Client, _ grafanasdk.Org) error {
		return wait.PollImmediate(ctx, 2*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
			_, _, err := client.GetDashboardByUID(ctx, uid)
			return err, nil
		})
	})
}

// waitForLogMarker waits until Loki returns the logs probe marker for the
// cluster's tenant in a window around at.
func waitForLogMarker(ctx context.Context, in Input, at time.Time) error {
	proxy, err := newServiceProxy(in.SeedConfig, mlaNamespace, "loki-distributed-query-frontend", 3100)
	if err != nil {
		return err
	}

	header := http.Header{tenantHeader: []string{in.Cluster.Name}}
	path := fmt.Sprintf("/loki/api/v1/query_range?query=%s&start=%d&end=%d&limit=5",
		url.QueryEscape(fmt.Sprintf("{job=%q}", logsProbeJob)),
		at.Add(-5*time.Minute).UnixNano(),
		at.Add(5*time.Minute).UnixNano(),
	)

	return wait.PollImmediate(ctx, 10*time.Second, 6*time.Minute, func(ctx context.Context) (error, error) {
		code, body, err := proxy.do(ctx, http.MethodGet, path, header, nil)
		if err != nil {
			return err, nil
		}
		if code != http.StatusOK {
			return fmt.Errorf("expected HTTP 200, got HTTP %d: %s", code, body), nil
		}

		if !strings.Contains(string(body), logsProbeMarker) {
			return fmt.Errorf("marker %q not returned yet", logsProbeMarker), nil
		}

		return nil, nil
	})
}

// cortexQueryAt runs an instant query at a given time.
func cortexQueryAt(ctx context.Context, in Input, query string, at time.Time) ([]promVectorSample, error) {
	return cortexQueryWithParams(ctx, in, query, url.Values{"time": []string{fmt.Sprintf("%d", at.Unix())}})
}

// flushIngesters asks every Cortex and Loki ingester to write its in-memory
// data to object storage, so the post-upgrade checks read the stored data
// instead of the ingesters' memory.
func flushIngesters(ctx context.Context, in Input) error {
	for _, target := range []struct {
		selector string
		port     int
		path     string
	}{
		{"app.kubernetes.io/name=cortex,app.kubernetes.io/component=ingester", cortexIngesterPort, "/ingester/flush"},
		{"app.kubernetes.io/name=loki-distributed,app.kubernetes.io/component=ingester", lokiIngesterPort, "/flush"},
	} {
		selector, err := labels.Parse(target.selector)
		if err != nil {
			return err
		}

		pods := &corev1.PodList{}
		if err := in.SeedClient.List(ctx, pods, ctrlruntimeclient.InNamespace(mlaNamespace), ctrlruntimeclient.MatchingLabelsSelector{Selector: selector}); err != nil {
			return err
		}
		if len(pods.Items) == 0 {
			return fmt.Errorf("no ingester pods match %s", target.selector)
		}

		for _, pod := range pods.Items {
			proxy, err := newPodProxy(in.SeedConfig, mlaNamespace, pod.Name, target.port)
			if err != nil {
				return err
			}

			code, body, err := proxy.do(ctx, http.MethodPost, target.path, nil, nil)
			if err != nil {
				return fmt.Errorf("flush %s: %w", pod.Name, err)
			}
			if code >= 300 {
				return fmt.Errorf("flush %s: HTTP %d: %s", pod.Name, code, body)
			}
		}
	}

	return nil
}

func minioClient(ctx context.Context, in Input) (*minio.Client, func(), error) {
	secret := &corev1.Secret{}
	if err := in.SeedClient.Get(ctx, types.NamespacedName{Namespace: mlaNamespace, Name: minioSecretName}, secret); err != nil {
		return nil, nil, fmt.Errorf("failed to get MinIO Secret: %w", err)
	}

	localPort, stop, err := forwardServicePort(ctx, in, mlaNamespace, minioServiceName, minioServicePort)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to port-forward to MinIO: %w", err)
	}

	client, err := s3.NewClient(fmt.Sprintf("http://127.0.0.1:%d", localPort), string(secret.Data["rootUser"]), string(secret.Data["rootPassword"]), "")
	if err != nil {
		stop()
		return nil, nil, err
	}

	return client, stop, nil
}

// listTenantObjects lists the object keys of the cluster's tenant in bucket.
func listTenantObjects(ctx context.Context, in Input, bucket string) ([]string, error) {
	client, stop, err := minioClient(ctx, in)
	if err != nil {
		return nil, err
	}
	defer stop()

	var keys []string
	for object := range client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Prefix: in.Cluster.Name + "/", Recursive: true}) {
		if object.Err != nil {
			return nil, object.Err
		}
		keys = append(keys, object.Key)
	}
	sort.Strings(keys)

	return keys, nil
}

func waitForTenantObjects(ctx context.Context, in Input, bucket string) ([]string, error) {
	var keys []string

	err := wait.PollImmediate(ctx, 10*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		var err error
		keys, err = listTenantObjects(ctx, in, bucket)
		if err != nil {
			return err, nil
		}
		if len(keys) == 0 {
			return fmt.Errorf("no objects for tenant %s in bucket %s yet", in.Cluster.Name, bucket), nil
		}

		return nil, nil
	})

	return keys, err
}

func mlaStorageUIDs(ctx context.Context, in Input) (map[string]types.UID, map[string]types.UID, error) {
	statefulSets := &appsv1.StatefulSetList{}
	if err := in.SeedClient.List(ctx, statefulSets, ctrlruntimeclient.InNamespace(mlaNamespace)); err != nil {
		return nil, nil, err
	}

	pvcs := &corev1.PersistentVolumeClaimList{}
	if err := in.SeedClient.List(ctx, pvcs, ctrlruntimeclient.InNamespace(mlaNamespace)); err != nil {
		return nil, nil, err
	}

	stsUIDs := map[string]types.UID{}
	for _, s := range statefulSets.Items {
		stsUIDs[s.Name] = s.UID
	}

	pvcUIDs := map[string]types.UID{}
	for _, p := range pvcs.Items {
		pvcUIDs[p.Name] = p.UID
	}

	return stsUIDs, pvcUIDs, nil
}

func compareUIDs(kind string, before, after map[string]types.UID) []string {
	var problems []string

	for name, uid := range before {
		switch now, ok := after[name]; {
		case !ok:
			problems = append(problems, fmt.Sprintf("%s %s was removed", kind, name))
		case now != uid:
			problems = append(problems, fmt.Sprintf("%s %s was recreated", kind, name))
		}
	}
	sort.Strings(problems)

	return problems
}

// VerifySeedRetention verifies that the seed Prometheus and the seed Loki
// still return data from before the upgrade.
func VerifySeedRetention(ctx context.Context, in SeedInput, at time.Time) error {
	prometheus, err := seedPrometheus(in)
	if err != nil {
		return err
	}

	vector := &promVector{}
	path := fmt.Sprintf("/api/v1/query?query=%s&time=%d", url.QueryEscape(`up{job="node-exporter"}`), at.Unix())
	if err := promGet(ctx, prometheus, path, vector); err != nil {
		return fmt.Errorf("seed Prometheus: %w", err)
	}
	if len(vector.Result) == 0 {
		return fmt.Errorf("seed Prometheus has no node-exporter data at %s", at)
	}

	loki, err := newServiceProxy(in.SeedConfig, loggingNamespace, seedLokiService, seedLokiPort)
	if err != nil {
		return err
	}

	lokiPath := fmt.Sprintf("/loki/api/v1/query_range?query=%s&start=%d&end=%d&limit=5",
		url.QueryEscape(`{namespace="kubermatic"}`), at.Add(-5*time.Minute).UnixNano(), at.Add(5*time.Minute).UnixNano())

	code, body, err := loki.do(ctx, http.MethodGet, lokiPath, nil, nil)
	if err != nil {
		return fmt.Errorf("seed Loki: %w", err)
	}
	if code != http.StatusOK {
		return fmt.Errorf("seed Loki: HTTP %d: %s", code, body)
	}

	response, err := parseLokiQueryResponse(string(body))
	if err != nil {
		return fmt.Errorf("seed Loki: %w", err)
	}
	if len(response.Data.Result) == 0 {
		return fmt.Errorf("seed Loki has no kubermatic logs around %s", at)
	}

	return nil
}
