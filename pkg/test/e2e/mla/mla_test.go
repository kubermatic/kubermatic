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
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/mla"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/test/generator"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
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
	// dashboardUid is the uid for the "Nodes Overview" Grafana dashboard.
	// It is used as a sort of "canary" to check if Grafana dashboards have been
	// created in the Grafana org created for a KKP Project.
	dashboardUid = "13yQpUxiz"
)

var (
	credentials jig.AWSCredentials
	logOptions  = utils.DefaultLogOptions
)

func init() {
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

func TestMLAIntegration(t *testing.T) {
	ctx := context.Background()
	rawLogger := log.NewFromOptions(logOptions)
	logger := rawLogger.Sugar()

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))

	if err := credentials.Parse(); err != nil {
		t.Fatalf("Failed to get credentials: %v", err)
	}

	seedClient, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("failed to get client for seed cluster: %v", err)
	}

	// create test environment
	testJig := jig.NewAWSCluster(seedClient, logger, credentials, 1, nil)
	testJig.ClusterJig.WithTestName("mla")

	project, cluster, err := testJig.Setup(ctx, jig.WaitForReadyPods)
	defer testJig.Cleanup(ctx, t, true)
	if err != nil {
		t.Fatalf("failed to setup test environment: %v", err)
	}

	logger.Info("Enabling MLA...")
	if err := setMLAIntegration(ctx, seedClient, cluster, true); err != nil {
		t.Fatalf("failed to enable MLA: %v", err)
	}

	logger.Info("Waiting for project to get Grafana org annotation...")
	p := &kubermaticv1.Project{}
	orgID := ""
	timeout := 5 * time.Minute
	if !utils.WaitFor(ctx, 1*time.Second, timeout, func() (ok bool) {
		if err := seedClient.Get(ctx, types.NamespacedName{Name: project.Name}, p); err != nil {
			t.Fatalf("failed to get project: %v", err)
		}

		orgID, ok = p.GetAnnotations()[mla.GrafanaOrgAnnotationKey]
		return ok
	}) {
		t.Fatalf("waiting for project annotation %+v", p)
	}

	id, err := strconv.ParseUint(orgID, 10, 32)
	if err != nil {
		t.Fatalf("unable to parse uint from %s", orgID)
	}

	logger.Info("Creating Grafana client for user cluster...")
	grafanaClient, err := getGrafanaClient(ctx, seedClient)
	if err != nil {
		t.Fatalf("unable to initialize Grafana client")
	}

	logger.Info("Fetching Grafana organization...")
	org, err := grafanaClient.GetOrgById(ctx, uint(id))
	if err != nil {
		t.Fatalf("error while getting Grafana org: %v", err)
	}
	grafanaClient.SetOrgIDHeader(org.ID)

	if err := verifyGrafanaDatasource(ctx, logger, grafanaClient, cluster); err != nil {
		t.Errorf("failed to verify grafana datasource: %v", err)
	}

	if err := verifyGrafanaCanaryDashboard(ctx, logger, grafanaClient); err != nil {
		t.Errorf("failed to verify grafana canary dashboard: %v", err)
	}

	if err := verifyGrafanaUser(ctx, logger, grafanaClient, &org); err != nil {
		t.Errorf("failed to verify grafana user: %v", err)
	}

	if err := verifyLogRuleGroup(ctx, logger, seedClient, p, cluster); err != nil {
		t.Errorf("failed to verify logs RuleGroup: %v", err)
	}

	if err := verifyMetricsRuleGroup(ctx, logger, seedClient, p, cluster); err != nil {
		t.Errorf("failed to verify metrics RuleGroup: %v", err)
	}

	if err := verifyAlertmanager(ctx, logger, seedClient, p, cluster); err != nil {
		t.Errorf("failed to verify Alertmanager: %v", err)
	}

	if err := verifyRateLimits(ctx, logger, seedClient, p, cluster); err != nil {
		t.Errorf("failed to verify rate limits: %v", err)
	}

	logger.Info("Disabling MLA...")
	if err := setMLAIntegration(ctx, seedClient, cluster, false); err != nil {
		t.Fatalf("failed to disable MLA: %v", err)
	}

	logger.Info("Waiting for cluster to healthy...")
	if err := testJig.WaitForHealthyControlPlane(ctx, 2*time.Minute); err != nil {
		t.Fatalf("cluster did not get healthy: %v", err)
	}

	logger.Info("Waiting for Grafana org to be gone...")
	if !utils.WaitFor(ctx, 1*time.Second, timeout, func() bool {
		_, err = grafanaClient.GetOrgById(ctx, org.ID)
		return err != nil
	}) {
		t.Fatal("grafana org not cleaned up")
	}

	logger.Info("Waiting for Grafana user to be gone...")
	if !utils.WaitFor(ctx, 1*time.Second, timeout, func() bool {
		_, err = grafanaClient.LookupUser(ctx, "roxy-admin@kubermatic.com")
		return errors.As(err, &grafanasdk.ErrNotFound{})
	}) {
		t.Fatal("grafana user not cleaned up")
	}

	logger.Info("Waiting for project to get rid of grafana org annotation")
	if !utils.WaitFor(ctx, 1*time.Second, timeout, func() bool {
		if err := seedClient.Get(ctx, types.NamespacedName{Name: project.Name}, p); err != nil {
			t.Fatalf("failed to get project: %v", err)
		}

		_, ok := p.GetAnnotations()[mla.GrafanaOrgAnnotationKey]
		return !ok
	}) {
		t.Fatal("project still has the grafana org annotation")
	}
}

func verifyGrafanaDatasource(ctx context.Context, log *zap.SugaredLogger, grafanaClient *grafanasdk.Client, cluster *kubermaticv1.Cluster) (err error) {
	log.Info("Waiting for datasource to be added to Grafana...")

	if !utils.WaitFor(ctx, 1*time.Second, 5*time.Minute, func() bool {
		_, err := grafanaClient.GetDatasourceByUID(ctx, fmt.Sprintf("%s-%s", mla.PrometheusType, cluster.Name))
		return err == nil
	}) {
		return fmt.Errorf("timed out waiting for grafana datasource %s-%s", mla.PrometheusType, cluster.Name)
	}

	log.Info("Grafana datasource successfully verified.")

	return nil
}

func verifyGrafanaCanaryDashboard(ctx context.Context, log *zap.SugaredLogger, grafanaClient *grafanasdk.Client) (err error) {
	log.Info("Waiting for canary dashboard (Nodes Overview) to be present in Grafana...")
	if !utils.WaitFor(ctx, 1*time.Second, 5*time.Minute, func() bool {
		_, _, err := grafanaClient.GetDashboardByUID(ctx, dashboardUid)
		return err == nil
	}) {
		return fmt.Errorf("timed out waiting for grafana dashboard with uid '%s'", dashboardUid)
	}

	log.Info("Grafana canary dashboard found.")

	return nil
}

func verifyGrafanaUser(ctx context.Context, log *zap.SugaredLogger, grafanaClient *grafanasdk.Client, org *grafanasdk.Org) error {
	log.Info("Checking that an admin user was added to Grafana...")

	user := grafanasdk.User{}
	err := wait.Poll(ctx, 1*time.Second, 2*time.Minute, func(ctx context.Context) (transient error, terminal error) {
		user, transient = grafanaClient.LookupUser(ctx, "roxy-admin@kubermatic.com")
		if transient != nil {
			return errors.New("user does not yet exist in Grafana"), nil
		}

		if user.IsGrafanaAdmin != true || user.OrgID != org.ID {
			return fmt.Errorf("user expected to be Grafana Admin and have orgID %d", org.ID), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("waiting for grafana user: %w", err)
	}

	log.Info("Verifying Grafana org user's role...")
	orgUser, err := mla.GetGrafanaOrgUser(ctx, grafanaClient, org.ID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get grafana org user: %w", err)
	}

	if orgUser.Role != string(grafanasdk.ROLE_EDITOR) {
		return fmt.Errorf("orgUser %v expected to have Editor role, but has %v", orgUser, orgUser.Role)
	}

	log.Info("Grafana user successfully verified.")

	return nil
}

func verifyLogRuleGroup(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, project *kubermaticv1.Project, cluster *kubermaticv1.Cluster) error {
	log.Info("Creating logs RuleGroup...")

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
	expectedData, err := createRuleGroup(ctx, client, cluster, []byte(lokiRule), kubermaticv1.RuleGroupTypeLogs)
	if err != nil {
		return fmt.Errorf("unable to create rule group: %w", err)
	}

	logRuleGroupURL := fmt.Sprintf("%s%s%s", "http://localhost:3003", mla.LogRuleGroupConfigEndpoint, "/default")
	httpClient := &http.Client{Timeout: 15 * time.Second}

	err = wait.Poll(ctx, 1*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", logRuleGroupURL, "test-rule"), nil)
		if err != nil {
			return fmt.Errorf("unable to create request: %v", err), nil
		}
		req.Header.Add(mla.RuleGroupTenantHeaderName, cluster.Name)

		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("unable to get rule group: %w", err), nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected HTTP 200 OK, got HTTP %s", resp.Status), nil
		}

		config := map[string]interface{}{}
		decoder := yaml.NewDecoder(resp.Body)
		if err := decoder.Decode(&config); err != nil {
			return fmt.Errorf("unable to decode response body: %w", err), nil
		}

		if !reflect.DeepEqual(config, expectedData) {
			return errors.New("response does not match the expected rule group"), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("log rule group not found: %w", err)
	}

	log.Info("RuleGroup successfully verified.")

	return nil
}

func verifyMetricsRuleGroup(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, project *kubermaticv1.Project, cluster *kubermaticv1.Cluster) error {
	log.Info("Creating metrics RuleGroup...")

	testRuleGroup := generator.GenerateTestRuleGroupData("test-metric-rule")
	expectedData, err := createRuleGroup(ctx, client, cluster, testRuleGroup, kubermaticv1.RuleGroupTypeMetrics)
	if err != nil {
		return fmt.Errorf("unable to create rule group: %w", err)
	}

	metricRuleGroupURL := fmt.Sprintf("%s%s%s", "http://localhost:3002", mla.MetricsRuleGroupConfigEndpoint, "/default")
	httpClient := &http.Client{Timeout: 15 * time.Second}

	err = wait.Poll(ctx, 1*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", metricRuleGroupURL, "test-metric-rule"), nil)
		if err != nil {
			return fmt.Errorf("unable to create request: %v", err), nil
		}
		req.Header.Add(mla.RuleGroupTenantHeaderName, cluster.Name)

		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("unable to get rule group: %w", err), nil
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected HTTP 200 OK, got HTTP %s", resp.Status), nil
		}

		defer resp.Body.Close()
		config := map[string]interface{}{}
		decoder := yaml.NewDecoder(resp.Body)
		if err := decoder.Decode(&config); err != nil {
			return fmt.Errorf("unable to decode response body: %w", err), nil
		}

		if !reflect.DeepEqual(config, expectedData) {
			return errors.New("response does not match the expected rule group"), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("metric rule group not found: %w", err)
	}

	log.Info("RuleGroup successfully verified.")

	return nil
}

func verifyAlertmanager(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, project *kubermaticv1.Project, cluster *kubermaticv1.Cluster) error {
	log.Info("Verifying Alertmanager...")
	if err := updateAlertmanager(ctx, client, cluster, []byte(testAlertmanagerConfig)); err != nil {
		return fmt.Errorf("unable to update alertmanager config: %w", err)
	}

	if !utils.WaitFor(ctx, 1*time.Second, 1*time.Minute, func() bool {
		if err := client.Get(ctx, types.NamespacedName{Name: cluster.Name}, cluster); err != nil {
			return false
		}
		return *cluster.Status.ExtendedHealth.AlertmanagerConfig == kubermaticv1.HealthStatusUp
	}) {
		return fmt.Errorf("has alertmanager status: %v", *cluster.Status.ExtendedHealth.AlertmanagerConfig)
	}

	alertmanagerURL := "http://localhost:3001" + mla.AlertmanagerConfigEndpoint
	expectedConfig := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(testAlertmanagerConfig), &expectedConfig); err != nil {
		return fmt.Errorf("unable to unmarshal expected config: %w", err)
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}

	err := wait.Poll(ctx, 1*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		req, err := http.NewRequest(http.MethodGet, alertmanagerURL, nil)
		if err != nil {
			return fmt.Errorf("unable to create request to get alertmanager config: %w", err), nil
		}
		req.Header.Add(mla.AlertmanagerTenantHeaderName, cluster.Name)

		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("unable to get alertmanager config: %w", err), nil
		}
		defer resp.Body.Close()

		// https://cortexmetrics.io/docs/api/#get-alertmanager-configuration
		if resp.StatusCode == http.StatusNotFound {
			return errors.New("alertmanager config not found"), nil
		}
		if resp.StatusCode != http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("unable to read alertmanager config: %w", err), nil
			}
			return fmt.Errorf("status code: %d, response body: %s", resp.StatusCode, string(body)), nil
		}

		config := map[string]interface{}{}
		decoder := yaml.NewDecoder(resp.Body)
		if err := decoder.Decode(&config); err != nil {
			return fmt.Errorf("unable to decode response body: %w", err), nil
		}

		if !reflect.DeepEqual(config, expectedConfig) {
			return errors.New("response does not match the expected rule group"), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for Alertmanager config to be updated: %w", err)
	}

	log.Info("Alertmanager successfully verified.")

	return nil
}

func updateAlertmanager(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, newData []byte) error {
	alertmanager := &kubermaticv1.Alertmanager{}
	if err := client.Get(ctx, types.NamespacedName{
		Name:      resources.DefaultAlertmanagerConfigSecretName,
		Namespace: cluster.Status.NamespaceName,
	}, alertmanager); err != nil {
		return fmt.Errorf("failed to get alertmanager: %w", err)
	}

	if alertmanager.Spec.ConfigSecret.Name == "" {
		return errors.New("Alertmanager configuration has no Secret name specified")
	}

	secret := &corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{
		Name:      alertmanager.Spec.ConfigSecret.Name,
		Namespace: alertmanager.Namespace,
	}, secret); err != nil {
		return fmt.Errorf("failed to get config Secret: %w", err)
	}

	secret.Data[resources.AlertmanagerConfigSecretKey] = newData

	return client.Update(ctx, secret)
}

func verifyRateLimits(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, project *kubermaticv1.Project, cluster *kubermaticv1.Cluster) error {
	log.Info("Setting rate limits...")

	rateLimits := kubermaticv1.MonitoringRateLimitSettings{
		IngestionRate:      1,
		IngestionBurstSize: 2,
		MaxSeriesPerMetric: 3,
		MaxSeriesTotal:     4,
		MaxSamplesPerQuery: 5,
		MaxSeriesPerQuery:  6,
	}
	if err := createMonitoringMLARateLimits(ctx, client, cluster, project, rateLimits); err != nil {
		return fmt.Errorf("unable to set monitoring rate limits: %w", err)
	}

	err := wait.Poll(ctx, 1*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		mlaAdminSetting := &kubermaticv1.MLAAdminSetting{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: resources.MLAAdminSettingsName}, mlaAdminSetting); ctrlruntimeclient.IgnoreNotFound(err) != nil {
			return fmt.Errorf("can't get cluster mlaadminsetting: %w", err), nil
		}

		configMap := &corev1.ConfigMap{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: "mla", Name: mla.RuntimeConfigMap}, configMap); err != nil {
			return fmt.Errorf("unable to get configMap: %w", err), nil
		}
		actualOverrides := &mla.Overrides{}
		decoder := yaml.NewDecoder(strings.NewReader(configMap.Data[mla.RuntimeConfigFileName]))
		decoder.KnownFields(true)
		if err := decoder.Decode(actualOverrides); err != nil {
			return fmt.Errorf("unable to unmarshal rate limit config map"), nil
		}

		actualRateLimits, ok := actualOverrides.Overrides[cluster.Name]
		if !ok {
			return errors.New("no data for cluster in actual overrides"), nil
		}

		actualMatches := *actualRateLimits.IngestionRate == rateLimits.IngestionRate &&
			*actualRateLimits.IngestionBurstSize == rateLimits.IngestionBurstSize &&
			*actualRateLimits.MaxSeriesPerMetric == rateLimits.MaxSeriesPerMetric &&
			*actualRateLimits.MaxSeriesTotal == rateLimits.MaxSeriesTotal &&
			*actualRateLimits.MaxSamplesPerQuery == rateLimits.MaxSamplesPerQuery &&
			*actualRateLimits.MaxSeriesPerQuery == rateLimits.MaxSeriesPerQuery

		if !actualMatches {
			return errors.New("actual rate limits do not match configured rate limits"), nil
		}

		configuredMatches := mlaAdminSetting.Spec.MonitoringRateLimits.IngestionRate == rateLimits.IngestionRate &&
			mlaAdminSetting.Spec.MonitoringRateLimits.IngestionBurstSize == rateLimits.IngestionBurstSize &&
			mlaAdminSetting.Spec.MonitoringRateLimits.MaxSeriesPerMetric == rateLimits.MaxSeriesPerMetric &&
			mlaAdminSetting.Spec.MonitoringRateLimits.MaxSeriesTotal == rateLimits.MaxSeriesTotal &&
			mlaAdminSetting.Spec.MonitoringRateLimits.MaxSamplesPerQuery == rateLimits.MaxSamplesPerQuery &&
			mlaAdminSetting.Spec.MonitoringRateLimits.MaxSeriesPerQuery == rateLimits.MaxSeriesPerQuery

		if !configuredMatches {
			return errors.New("configured rate limits do not match intended rate limits"), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("rate limits not equal: %w", err)
	}

	log.Info("Rate limits successfully verified.")

	return nil
}

func createMonitoringMLARateLimits(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, project *kubermaticv1.Project, rateLimits kubermaticv1.MonitoringRateLimitSettings) error {
	mlaAdminSetting := &kubermaticv1.MLAAdminSetting{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.MLAAdminSettingsName,
			Namespace: cluster.Status.NamespaceName,
		},
		Spec: kubermaticv1.MLAAdminSettingSpec{
			ClusterName:          cluster.Name,
			MonitoringRateLimits: &rateLimits,
		},
	}

	return client.Create(ctx, mlaAdminSetting)
}

func setMLAIntegration(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, enabled bool) error {
	if err := toggleMLAInSeed(ctx, client, enabled); err != nil {
		return fmt.Errorf("failed to update seed: %w", err)
	}

	oldCluster := cluster.DeepCopy()
	cluster.Spec.MLA = &kubermaticv1.MLASettings{
		MonitoringEnabled: enabled,
		LoggingEnabled:    enabled,
	}

	return client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}

func getGrafanaClient(ctx context.Context, client ctrlruntimeclient.Client) (*grafanasdk.Client, error) {
	grafanaSecret := "mla/grafana"

	split := strings.Split(grafanaSecret, "/")
	if n := len(split); n != 2 {
		return nil, fmt.Errorf("splitting value of %q didn't yield two but %d results", grafanaSecret, n)
	}

	secret := corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{Name: split[1], Namespace: split[0]}, &secret); err != nil {
		return nil, fmt.Errorf("failed to get Grafana Secret: %w", err)
	}

	adminName, ok := secret.Data[mla.GrafanaUserKey]
	if !ok {
		return nil, fmt.Errorf("Grafana Secret %q does not contain %s key", grafanaSecret, mla.GrafanaUserKey)
	}
	adminPass, ok := secret.Data[mla.GrafanaPasswordKey]
	if !ok {
		return nil, fmt.Errorf("Grafana Secret %q does not contain %s key", grafanaSecret, mla.GrafanaPasswordKey)
	}

	grafanaAuth := fmt.Sprintf("%s:%s", adminName, adminPass)
	httpClient := &http.Client{Timeout: 15 * time.Second}
	grafanaURL := "http://localhost:3000"

	return grafanasdk.NewClient(grafanaURL, grafanaAuth, httpClient)
}

func toggleMLAInSeed(ctx context.Context, client ctrlruntimeclient.Client, enable bool) error {
	seed, _, err := jig.Seed(ctx, client, credentials.KKPDatacenter)
	if err != nil {
		return fmt.Errorf("failed to get seed: %w", err)
	}

	seed.Spec.MLA = &kubermaticv1.SeedMLASettings{
		UserClusterMLAEnabled: enable,
	}

	if err := client.Update(ctx, seed); err != nil {
		return fmt.Errorf("failed to update seed: %w", err)
	}

	return nil
}

func createRuleGroup(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, data []byte, kind kubermaticv1.RuleGroupType) (map[string]interface{}, error) {
	expected := map[string]interface{}{}
	if err := yaml.Unmarshal(data, &expected); err != nil {
		return nil, fmt.Errorf("unable to unmarshal expected rule group: %w", err)
	}

	ruleGroup := &kubermaticv1.RuleGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      expected["name"].(string),
			Namespace: cluster.Status.NamespaceName,
		},
		Spec: kubermaticv1.RuleGroupSpec{
			RuleGroupType: kind,
			Cluster: corev1.ObjectReference{
				Name: cluster.Name,
			},
			Data: data,
		},
	}

	if err := client.Create(ctx, ruleGroup); err != nil {
		return nil, err
	}

	return expected, nil
}
