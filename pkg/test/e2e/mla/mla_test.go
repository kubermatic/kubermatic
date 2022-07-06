//go:build mla

/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/mla"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testAlertmanagerConfig = `
template_files: {}
alertmanager_config: |
  global:
    smtp_smarthost: 'localhost:25'
    smtp_from: 'test@example.org'
  route:
    receiver: "test"
  receivers:
    - name: "test"
      email_configs:
      - to: 'test@example.org'

`
)

var (
	datacenter = "kubermatic"
	location   = "hetzner-hel1"
	version    = utils.KubernetesVersion()
	credential = "e2e-hetzner"
)

func TestMLAIntegration(t *testing.T) {
	ctx := context.Background()

	seedClient, _, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("failed to get client for seed cluster: %v", err)
	}

	// login
	masterToken, err := utils.RetrieveMasterToken(ctx)
	if err != nil {
		t.Fatalf("failed to get master token: %v", err)
	}
	masterClient := utils.NewTestClient(masterToken, t)

	masterAdminToken, err := utils.RetrieveAdminMasterToken(ctx)
	if err != nil {
		t.Fatalf("failed to get master admin token: %v", err)
	}
	masterAdminClient := utils.NewTestClient(masterAdminToken, t)

	// create dummy project
	t.Log("creating project...")
	project, err := masterClient.CreateProject(rand.String(10))
	if err != nil {
		t.Fatalf("failed to create project: %v", getErrorResponse(err))
	}
	defer masterClient.CleanupProject(t, project.ID)

	t.Log("creating cluster...")
	apiCluster, err := masterClient.CreateHetznerCluster(project.ID, datacenter, rand.String(10), credential, version, location, 1)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", getErrorResponse(err))
	}

	// wait for the cluster to become healthy
	if err := masterClient.WaitForClusterHealthy(project.ID, datacenter, apiCluster.ID); err != nil {
		t.Fatalf("cluster did not become healthy: %v", err)
	}

	if err := masterClient.WaitForClusterNodeDeploymentsToByReady(project.ID, datacenter, apiCluster.ID, 1); err != nil {
		t.Fatalf("cluster nodes not ready: %v", err)
	}

	// get the cluster object (the CRD, not the API's representation)
	cluster := &kubermaticv1.Cluster{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: apiCluster.ID}, cluster); err != nil {
		t.Fatalf("failed to get cluster: %v", err)
	}

	seed := &kubermaticv1.Seed{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: "kubermatic", Namespace: "kubermatic"}, seed); err != nil {
		t.Fatalf("failed to get seed: %v", err)
	}
	seed.Spec.MLA = &kubermaticv1.SeedMLASettings{
		UserClusterMLAEnabled: true,
	}
	if err := seedClient.Update(ctx, seed); err != nil {
		t.Fatalf("failed to update seed: %v", err)
	}

	// enable MLA
	t.Log("enabling MLA...")
	if err := setMLAIntegration(ctx, seedClient, cluster, true); err != nil {
		t.Fatalf("failed to set MLA integration to true: %v", err)
	}

	t.Log("waiting for project to get grafana org annotation")
	p := &kubermaticv1.Project{}
	timeout := 300 * time.Second
	if !utils.WaitFor(1*time.Second, timeout, func() bool {
		if err := seedClient.Get(ctx, types.NamespacedName{Name: project.ID}, p); err != nil {
			t.Fatalf("failed to get project: %v", err)
		}

		_, ok := p.GetAnnotations()[mla.GrafanaOrgAnnotationKey]
		return ok
	}) {
		t.Fatalf("waiting for project annotation %+v", p)
	}

	t.Log("creating client for user cluster...")
	grafanaSecret := "mla/grafana"

	split := strings.Split(grafanaSecret, "/")
	if n := len(split); n != 2 {
		t.Fatalf("splitting value of %q didn't yield two but %d results",
			grafanaSecret, n)
	}

	secret := corev1.Secret{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: split[1], Namespace: split[0]}, &secret); err != nil {
		t.Fatalf("failed to get Grafana Secret: %v", err)
	}
	adminName, ok := secret.Data[mla.GrafanaUserKey]
	if !ok {
		t.Fatalf("Grafana Secret %q does not contain %s key", grafanaSecret, mla.GrafanaUserKey)
	}
	adminPass, ok := secret.Data[mla.GrafanaPasswordKey]
	if !ok {
		t.Fatalf("Grafana Secret %q does not contain %s key", grafanaSecret, mla.GrafanaPasswordKey)
	}
	grafanaAuth := fmt.Sprintf("%s:%s", adminName, adminPass)
	httpClient := &http.Client{Timeout: 15 * time.Second}

	grafanaURL := "http://localhost:3000"
	grafanaClient, err := grafanasdk.NewClient(grafanaURL, grafanaAuth, httpClient)
	if err != nil {
		t.Fatalf("unable to initialize grafana client")
	}
	orgID, ok := p.GetAnnotations()[mla.GrafanaOrgAnnotationKey]
	if !ok {
		t.Fatal("project should have grafana org annotation set")
	}
	id, err := strconv.ParseUint(orgID, 10, 32)
	if err != nil {
		t.Fatalf("unable to parse uint from %s", orgID)
	}
	org, err := grafanaClient.GetOrgById(ctx, uint(id))
	if err != nil {
		t.Fatalf("error while getting grafana org:  %s", err)
	}
	t.Log("org added to Grafana")

	if err := seedClient.Get(ctx, types.NamespacedName{Name: apiCluster.ID}, cluster); err != nil {
		t.Fatalf("failed to get cluster: %v", err)
	}

	grafanaClient.SetOrgIDHeader(org.ID)
	t.Log("waiting for datasource added to grafana")
	if !utils.WaitFor(1*time.Second, timeout, func() bool {
		_, err := grafanaClient.GetDatasourceByUID(ctx, fmt.Sprintf("%s-%s", mla.PrometheusType, cluster.Name))
		return err == nil
	}) {
		t.Fatalf("waiting for grafana datasource %s-%s", mla.PrometheusType, cluster.Name)
	}

	user := grafanasdk.User{}
	if !utils.WaitFor(1*time.Second, timeout, func() bool {
		user, err = grafanaClient.LookupUser(ctx, "roxy-admin@kubermatic.com")
		return err == nil
	}) {
		t.Fatalf("waiting for grafana user: %v", err)
	}
	t.Log("user added to Grafana")
	if !utils.WaitFor(1*time.Second, timeout, func() bool {
		user, _ = grafanaClient.LookupUser(ctx, "roxy-admin@kubermatic.com")
		return (user.IsGrafanaAdmin == true) && (user.OrgID == org.ID)
	}) {
		t.Fatalf("user[%+v] expected to be Grafana Admin and have orgID=%d", user, org.ID)
	}

	orgUser, err := mla.GetGrafanaOrgUser(ctx, grafanaClient, org.ID, user.ID)
	if err != nil {
		t.Fatalf("failed to get grafana org user: %v", err)
	}

	if orgUser.Role != string(grafanasdk.ROLE_EDITOR) {
		t.Fatalf("orgUser[%v] expected to be had Editor role", orgUser)
	}

	// Logs RuleGroup
	lokiRule := `
name: test-rule
rules:
- alert: HighThroughputLogStreams
  expr: sum by(container)(rate({job=~"kube-system/.*"}[1m])) >= 50
  for: 2s
  labels:
    severity: critical
  annotations:
    summary: "log stream is a bit high"
    description: "log stream is a bit high"
`
	expectedLogRuleGroup := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(lokiRule), &expectedLogRuleGroup); err != nil {
		t.Fatalf("unable to unmarshal expected rule group: %v", err)
	}

	_, err = masterClient.CreateRuleGroup(cluster.Name, project.ID, kubermaticv1.RuleGroupTypeLogs, []byte(lokiRule))
	if err != nil {
		t.Fatalf("unable to create logs rule group: %v", err)
	}
	logRuleGroupURL := fmt.Sprintf("%s%s%s", "http://localhost:3003", mla.LogRuleGroupConfigEndpoint, "/default")

	if !utils.WaitFor(1*time.Second, timeout, func() bool {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", logRuleGroupURL, "test-rule"), nil)
		if err != nil {
			t.Fatalf("unable to create rule group request: %v", err)
		}
		req.Header.Add(mla.RuleGroupTenantHeaderName, cluster.Name)

		resp, err := httpClient.Do(req)
		if err != nil {
			t.Fatalf("unable to get rule group: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			return false
		}
		defer resp.Body.Close()
		config := map[string]interface{}{}
		decoder := yaml.NewDecoder(resp.Body)
		if err := decoder.Decode(&config); err != nil {
			t.Fatalf("unable to decode response body: %v", err)
		}
		return reflect.DeepEqual(config, expectedLogRuleGroup)
	}) {
		t.Fatal("log rule group not found")
	}
	t.Log("log rule group added")

	// Metric RuleGroup
	testRuleGroup := test.GenerateTestRuleGroupData("test-metric-rule")
	expectedRuleGroup := map[string]interface{}{}
	if err := yaml.Unmarshal(testRuleGroup, &expectedRuleGroup); err != nil {
		t.Fatalf("unable to unmarshal expected rule group: %v", err)
	}

	_, err = masterClient.CreateRuleGroup(cluster.Name, project.ID, kubermaticv1.RuleGroupTypeMetrics, testRuleGroup)
	if err != nil {
		t.Fatalf("unable to create metric rule group: %v", err)
	}
	metricRuleGroupURL := fmt.Sprintf("%s%s%s", "http://localhost:3002", mla.MetricsRuleGroupConfigEndpoint, "/default")

	if !utils.WaitFor(1*time.Second, timeout, func() bool {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", metricRuleGroupURL, "test-metric-rule"), nil)
		if err != nil {
			t.Fatalf("unable to create rule group request: %v", err)
		}
		req.Header.Add(mla.RuleGroupTenantHeaderName, cluster.Name)

		resp, err := httpClient.Do(req)
		if err != nil {
			t.Fatalf("unable to get rule group: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			return false
		}
		defer resp.Body.Close()
		config := map[string]interface{}{}
		decoder := yaml.NewDecoder(resp.Body)
		if err := decoder.Decode(&config); err != nil {
			t.Fatalf("unable to decode response body: %v", err)
		}
		return reflect.DeepEqual(config, expectedRuleGroup)

	}) {
		t.Fatal("metric rule group not found")
	}
	t.Log("metric rule group added")

	// AlertManager
	_, err = masterClient.UpdateAlertmanager(cluster.Name, project.ID, testAlertmanagerConfig)
	if err != nil {
		t.Fatalf("unable to update alertmanager config: %v", err)
	}

	if !utils.WaitFor(1*time.Second, timeout, func() bool {
		if err := seedClient.Get(ctx, types.NamespacedName{Name: apiCluster.ID}, cluster); err != nil {
			t.Fatalf("failed to get cluster: %v", err)
		}
		return *cluster.Status.ExtendedHealth.AlertmanagerConfig == kubermaticv1.HealthStatusUp
	}) {
		t.Fatalf("has alertmanager status: %v", *cluster.Status.ExtendedHealth.AlertmanagerConfig)
	}

	alertmanagerURL := "http://localhost:3001" + mla.AlertmanagerConfigEndpoint
	expectedConfig := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(testAlertmanagerConfig), &expectedConfig); err != nil {
		t.Fatalf("unable to unmarshal expected config: %v", err)
	}
	if !utils.WaitFor(1*time.Second, timeout, func() bool {
		req, err := http.NewRequest(http.MethodGet, alertmanagerURL, nil)
		if err != nil {
			t.Fatalf("unable to create request to get alertmanager config: %v", err)
		}
		req.Header.Add(mla.AlertmanagerTenantHeaderName, cluster.Name)
		resp, err := httpClient.Do(req)
		if err != nil {
			t.Fatalf("unable to get alertmanager config: %v", err)
		}
		defer resp.Body.Close()
		// https://cortexmetrics.io/docs/api/#get-alertmanager-configuration
		if resp.StatusCode == http.StatusNotFound {
			t.Fatal("alertmanager config not found")
		}
		if resp.StatusCode != http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("unable to read alertmanager config: %s", err.Error())
			}
			t.Fatalf("status code: %d, response body: %s", resp.StatusCode, string(body))
		}
		config := map[string]interface{}{}
		decoder := yaml.NewDecoder(resp.Body)
		if err := decoder.Decode(&config); err != nil {
			t.Fatalf("unable to decode response body: %v", err)
		}
		return reflect.DeepEqual(config, expectedConfig)
	}) {
		t.Fatalf("config not equal")
	}
	t.Log("alertmanager config added")

	// Rate limits
	rateLimits := kubermaticv1.MonitoringRateLimitSettings{
		IngestionRate:      1,
		IngestionBurstSize: 2,
		MaxSeriesPerMetric: 3,
		MaxSeriesTotal:     4,
		MaxSamplesPerQuery: 5,
		MaxSeriesPerQuery:  6,
	}
	_, err = masterAdminClient.SetMonitoringMLARateLimits(cluster.Name, project.ID, rateLimits)
	if err != nil {
		t.Fatalf("unable to set monitoring rate limits: %s", err.Error())
	}

	if !utils.WaitFor(1*time.Second, timeout, func() bool {
		mlaAdminSetting := &kubermaticv1.MLAAdminSetting{}
		if err := seedClient.Get(ctx, types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: resources.MLAAdminSettingsName}, mlaAdminSetting); ctrlruntimeclient.IgnoreNotFound(err) != nil {
			t.Fatalf("can't get cluster mlaadminsetting: %v", err)
		}

		configMap := &corev1.ConfigMap{}
		if err := seedClient.Get(ctx, types.NamespacedName{Namespace: "mla", Name: mla.RuntimeConfigMap}, configMap); err != nil {
			t.Fatalf("unable to get configMap: %v", err)
		}
		actualOverrides := &mla.Overrides{}
		decoder := yaml.NewDecoder(strings.NewReader(configMap.Data[mla.RuntimeConfigFileName]))
		decoder.KnownFields(true)
		if err := decoder.Decode(actualOverrides); err != nil {
			t.Fatalf("unable to unmarshal rate limit config map")
		}
		actualRateLimits, ok := actualOverrides.Overrides[cluster.Name]
		if !ok {
			return false
		}
		return (*actualRateLimits.IngestionRate == rateLimits.IngestionRate &&
			*actualRateLimits.IngestionBurstSize == rateLimits.IngestionBurstSize &&
			*actualRateLimits.MaxSeriesPerMetric == rateLimits.MaxSeriesPerMetric &&
			*actualRateLimits.MaxSeriesTotal == rateLimits.MaxSeriesTotal &&
			*actualRateLimits.MaxSamplesPerQuery == rateLimits.MaxSamplesPerQuery &&
			*actualRateLimits.MaxSeriesPerQuery == rateLimits.MaxSeriesPerQuery) && (mlaAdminSetting.Spec.MonitoringRateLimits.IngestionRate == rateLimits.IngestionRate &&
			mlaAdminSetting.Spec.MonitoringRateLimits.IngestionBurstSize == rateLimits.IngestionBurstSize &&
			mlaAdminSetting.Spec.MonitoringRateLimits.MaxSeriesPerMetric == rateLimits.MaxSeriesPerMetric &&
			mlaAdminSetting.Spec.MonitoringRateLimits.MaxSeriesTotal == rateLimits.MaxSeriesTotal &&
			mlaAdminSetting.Spec.MonitoringRateLimits.MaxSamplesPerQuery == rateLimits.MaxSamplesPerQuery &&
			mlaAdminSetting.Spec.MonitoringRateLimits.MaxSeriesPerQuery == rateLimits.MaxSeriesPerQuery)
	}) {
		t.Fatal("monitoring rate limits not equal")
	}

	// Disable MLA Integration
	t.Log("disabling MLA...")
	if err := setMLAIntegration(ctx, seedClient, cluster, false); err != nil {
		t.Fatalf("failed to set MLA integration to false: %v", err)
	}

	seed.Spec.MLA = &kubermaticv1.SeedMLASettings{
		UserClusterMLAEnabled: false,
	}
	if err := seedClient.Update(ctx, seed); err != nil {
		t.Fatalf("failed to update seed: %v", err)
	}

	// Check that cluster is healthy
	t.Log("waiting for cluster to healthy after disabling MLA...")
	if err := masterClient.WaitForClusterHealthy(project.ID, datacenter, apiCluster.ID); err != nil {
		t.Fatalf("cluster not healthy: %v", err)
	}
	if !utils.WaitFor(1*time.Second, timeout, func() bool {
		_, err = grafanaClient.GetOrgById(ctx, org.ID)
		return err != nil
	}) {
		t.Fatal("grafana org not cleaned up")
	}

	if !utils.WaitFor(1*time.Second, timeout, func() bool {
		_, err = grafanaClient.LookupUser(ctx, "roxy-admin@kubermatic.com")
		return errors.As(err, &grafanasdk.ErrNotFound{})
	}) {

		t.Fatal("grafana user not cleaned up")
	}

	t.Log("waiting for project to get rid of grafana org annotation")
	if !utils.WaitFor(1*time.Second, timeout, func() bool {
		if err := seedClient.Get(ctx, types.NamespacedName{Name: project.ID}, p); err != nil {
			t.Fatalf("failed to get project: %v", err)
		}

		_, ok := p.GetAnnotations()[mla.GrafanaOrgAnnotationKey]
		return !ok
	}) {
		t.Fatalf("waiting for project annotation removed %+v", p)
	}

	// Test that cluster deletes cleanly
	masterClient.CleanupCluster(t, project.ID, datacenter, apiCluster.ID)
}

// getErrorResponse converts the client error response to string
func getErrorResponse(err error) string {
	rawData, newErr := json.Marshal(err)
	if newErr != nil {
		return err.Error()
	}
	return string(rawData)
}

func setMLAIntegration(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, enabled bool) error {
	oldCluster := cluster.DeepCopy()
	cluster.Spec.MLA = &kubermaticv1.MLASettings{
		MonitoringEnabled: enabled,
		LoggingEnabled:    enabled,
	}

	return client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}
