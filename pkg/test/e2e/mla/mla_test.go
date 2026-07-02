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
	"net"
	"net/http"
	"net/url"
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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
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

	seedClient, seedConfig, err := utils.GetClients()
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

	if err := verifyGatewayWritePath(ctx, logger, seedClient, seedConfig, testJig, cluster); err != nil {
		t.Errorf("failed to verify MLA Gateway write path: %v", err)
	}

	logger.Info("Disabling MLA...")
	if err := setMLAIntegration(ctx, seedClient, cluster, false); err != nil {
		t.Fatalf("failed to disable MLA: %v", err)
	}

	logger.Info("Waiting for cluster to healthy...")
	if err := testJig.WaitForHealthyControlPlane(ctx, 2*time.Minute); err != nil {
		t.Fatalf("cluster did not get healthy: %v", err)
	}

	// recreate the client without the X-Grafana-Org-Id header set above: the
	// org is deleted during cleanup and once Grafana 13's org cache expires,
	// requests whose org header points to a missing org are rejected with 403,
	// so LookupUser would never return the 404 these checks wait for
	grafanaClient, err = getGrafanaClient(ctx, seedClient)
	if err != nil {
		t.Fatalf("unable to recreate Grafana client: %v", err)
	}

	logger.Info("Waiting for Grafana org to be gone...")
	if !utils.WaitFor(ctx, 1*time.Second, timeout, func() bool {
		// require a real 404; a plain err != nil check would also pass on
		// transport errors, hiding a broken connection or a failed deletion
		_, err = grafanaClient.GetOrgById(ctx, org.ID)
		return errors.As(err, &grafanasdk.ErrNotFound{})
	}) {
		t.Fatalf("grafana org not cleaned up, last error: %v", err)
	}

	// user cleanup is asynchronous: disabling MLA in the Seed makes the KKP
	// operator redeploy the seed-controller-manager without the
	// -enable-user-cluster-mla flag, and only then the one-shot cleanup
	// controller deletes the Grafana user; the timeout covers the redeploy,
	// leader election, and cleanup retries
	logger.Info("Waiting for Grafana user to be gone...")
	if !utils.WaitFor(ctx, 1*time.Second, 15*time.Minute, func() bool {
		_, err = grafanaClient.LookupUser(ctx, "roxy-admin@kubermatic.com")
		return errors.As(err, &grafanasdk.ErrNotFound{})
	}) {
		t.Fatalf("grafana user not cleaned up, last error: %v", err)
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

// gatewayWriteProbeImage is agnhost, which ships curl and is already used by
// other e2e suites for in-cluster HTTP checks.
const (
	gatewayWriteProbeImage = "registry.k8s.io/e2e-test-images/agnhost:2.53"
	gatewayProbePodName    = "mla-gateway-probe"
	gatewayProbeContainer  = "probe"

	// gatewayName matches the unexported constant in
	// pkg/controller/seed-controller-manager/mla/resources.go and the value of
	// the app.kubernetes.io/name label on the gateway pods.
	gatewayName = "mla-gateway"

	// lokiPushJob is the log stream label the probe writes and later queries,
	// so it can be told apart from real agent traffic.
	lokiPushJob = "mla-gateway-e2e-probe"
)

// verifyGatewayWritePath exercises the MLA Gateway data plane directly, which
// the rest of the suite never does (it talks to the Cortex/Loki backends over
// port-forwards, bypassing the gateway). It confirms two NGINX behaviors that
// an image bump could regress: that a real mTLS push round-trips to storage
// (W1) and that a client without a cert is rejected (W2, ssl_verify_client on).
//
// The gateway write port (8080) has no in-cluster Service (mla-gateway ClusterIP
// fronts only the read port 8081), so the probe curls a gateway pod IP on 8080
// directly. This covers the gateway's NGINX behavior across all expose
// strategies; the nodeport-proxy layer in front is out of scope.
func verifyGatewayWritePath(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, seedConfig *rest.Config, testJig *jig.TestJig, cluster *kubermaticv1.Cluster) error {
	clusterNS := cluster.Status.NamespaceName
	if clusterNS == "" {
		return errors.New("cluster has no namespace yet (Status.NamespaceName empty)")
	}

	// the agent mTLS client cert is reconciled into the user cluster and signed
	// by the seed-side mla-gateway-ca, the CA the gateway verifies clients
	// against. The CA itself is not needed here: the probe curls a pod IP, so
	// server cert verification is skipped (-k) and only the client cert matters.
	log.Info("Waiting for MLA Gateway mTLS material...")
	var clientCert, clientKey []byte
	if err := wait.Poll(ctx, 2*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		clusterClient, err := testJig.ClusterJig.ClusterClient(ctx)
		if err != nil {
			return err, nil
		}

		certSecret := &corev1.Secret{}
		if err := clusterClient.Get(ctx, types.NamespacedName{Namespace: resources.UserClusterMLANamespace, Name: resources.MLAMonitoringAgentCertificatesSecretName}, certSecret); err != nil {
			if apierrors.IsNotFound(err) {
				return err, nil
			}

			return nil, fmt.Errorf("failed to get monitoring agent cert secret: %w", err)
		}

		clientCert = certSecret.Data[resources.MLAMonitoringAgentClientCertSecretKey]
		clientKey = certSecret.Data[resources.MLAMonitoringAgentClientKeySecretKey]

		if len(clientCert) == 0 || len(clientKey) == 0 {
			return errors.New("mla material not yet populated"), nil
		}

		return nil, nil
	}); err != nil {
		return fmt.Errorf("failed to gather mTLS material: %w", err)
	}

	// stage the cert/key/CA in the seed cluster-namespace as a Secret the probe
	// pod mounts, so curl can present them without writing them to a command line.
	probeSecretName := gatewayProbePodName + "-tls"
	probeSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      probeSecretName,
			Namespace: clusterNS,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			resources.MLAMonitoringAgentClientCertSecretKey: clientCert,
			resources.MLAMonitoringAgentClientKeySecretKey:  clientKey,
		},
	}

	// best-effort removal of leftovers from a previous aborted run so the
	// Create calls below do not fail with AlreadyExists.
	_ = seedClient.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: probeSecretName, Namespace: clusterNS}})
	_ = seedClient.Delete(ctx, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: gatewayProbePodName, Namespace: clusterNS}})

	err := seedClient.Create(ctx, probeSecret)
	if err != nil {
		return fmt.Errorf("failed to create probe TLS secret: %w", err)
	}
	defer func() {
		if err := seedClient.Delete(context.Background(), probeSecret); err != nil && !apierrors.IsNotFound(err) {
			log.Warnw("Failed to clean up probe TLS secret", zap.Error(err))
		}
	}()

	log.Info("Waiting for MLA Gateway pod to be ready...")

	gatewayPodIP, err := waitGatewayPodReady(ctx, seedClient, clusterNS)
	if err != nil {
		return fmt.Errorf("failed to find ready gateway pod: %w", err)
	}

	log.Infof("Gateway pod IP: %s", gatewayPodIP)

	probe := &utils.TestPodConfig{
		Log:       log,
		Namespace: clusterNS,
		Client:    seedClient,
		Config:    seedConfig,
		CreatePodFunc: func(ns string) *corev1.Pod {
			return newGatewayProbePod(ns, probeSecretName)
		},
	}

	if err := probe.DeployTestPod(ctx, log); err != nil {
		return fmt.Errorf("failed to deploy gateway probe pod: %w", err)
	}

	defer func() {
		if err := probe.CleanUp(context.Background()); err != nil {
			log.Warnw("Failed to clean up gateway probe pod", zap.Error(err))
		}
	}()

	const (
		certPath = "/etc/ssl/mla/client.crt"
		keyPath  = "/etc/ssl/mla/client.key"
	)
	// -k: the server cert SAN does not cover a pod IP, but client-cert
	// verification (ssl_verify_client on) still runs server-side regardless.
	baseCurl := []string{
		"curl", "-s", "-o", "/dev/null", "-w", "%{http_code}",
		"--connect-timeout", "3", "--max-time", "15",
		"--cert", certPath, "--key", keyPath, "-k",
	}

	var problems []string

	// a real mTLS push round-trips to storage. Push a log stream, then read
	// it back through the gateway read path (port 80 -> 8081, no TLS) and assert
	// the stream is visible. Proves proxy_pass on both ports plus the
	// X-Scope-OrgID tenant injection end to end.
	pushTime := time.Now()
	pushBody := fmt.Sprintf(
		`{"streams":[{"stream":{"job":%q},"values":[[%q,"probe"]]}]}`,
		lokiPushJob, strconv.FormatInt(pushTime.UnixNano(), 10),
	)
	pushURLFor := func(podIP string) string {
		return fmt.Sprintf("https://%s/loki/api/v1/push", net.JoinHostPort(podIP, "8080"))
	}

	// pod readiness only proves the 8081 probe endpoint, so the first push can
	// still hit a transient distributor or resolver blip; retry briefly. The
	// pod IP is re-resolved per attempt in case a gateway rollout replaces the
	// pod mid-check.
	if pushErr := wait.Poll(ctx, 2*time.Second, 60*time.Second, func(ctx context.Context) (error, error) {
		podIP, err := readyGatewayPodIP(ctx, seedClient, clusterNS)
		if err != nil {
			return err, nil
		}

		pushCurl := append(append([]string{}, baseCurl...), "-XPOST", "-H", "Content-Type: application/json", "-d", pushBody, pushURLFor(podIP))
		pushCode, _, err := probe.Exec(ctx, gatewayProbeContainer, pushCurl...)
		if err != nil {
			return fmt.Errorf("push failed to execute: %w", err), nil
		}

		if !isSuccessPushCode(pushCode) {
			return fmt.Errorf("push returned HTTP %s", pushCode), nil
		}

		return nil, nil
	}); pushErr != nil {
		problems = append(problems, fmt.Sprintf("W1 push did not succeed: %v", pushErr))
	}

	readURL := fmt.Sprintf(
		"http://%s/loki/api/v1/query_range?query=%s&start=%d&end=%d",
		net.JoinHostPort(fmt.Sprintf("%s.%s.svc.cluster.local", gatewayName, clusterNS), "80"),
		url.QueryEscape(fmt.Sprintf("{job=%q}", lokiPushJob)),
		pushTime.Add(-time.Minute).UnixNano(),
		pushTime.Add(10*time.Minute).UnixNano(),
	)
	// deliberately no X-Scope-OrgID header: the gateway read server overwrites
	// it with the cluster name (proxy_set_header at server scope), so getting
	// the pushed stream back proves the gateway's tenant injection works on
	// both paths, not just that loki honors a header the test set itself.
	readCmd := []string{
		"curl", "-s", "--connect-timeout", "3", "--max-time", "15", readURL,
	}

	// give loki a moment to index the just-pushed stream before reading back
	if readErr := wait.Poll(ctx, 2*time.Second, 90*time.Second, func(ctx context.Context) (error, error) {
		stdout, _, execErr := probe.Exec(ctx, gatewayProbeContainer, readCmd...)
		if execErr != nil {
			return execErr, nil
		}

		// the label alone is not enough, loki error bodies can echo the query;
		// require a successful response that contains the pushed stream.
		if strings.Contains(stdout, `"status":"success"`) && strings.Contains(stdout, lokiPushJob) {
			return nil, nil
		}

		return errors.New("stream not yet visible"), nil
	}); readErr != nil {
		problems = append(problems, fmt.Sprintf("W1 pushed stream not readable via gateway read path: %v", readErr))
	}

	// the write-path checks above can outlive the pod resolved at the start,
	// so pick a live pod again for the no-cert probe.
	if noCertPodIP, err := readyGatewayPodIP(ctx, seedClient, clusterNS); err != nil {
		problems = append(problems, fmt.Sprintf("W2 could not resolve a ready gateway pod: %v", err))
	} else {
		noCertCurl := []string{
			"curl", "-s", "-o", "/dev/null", "-w", "%{http_code}",
			"--connect-timeout", "3", "--max-time", "15", "-k",
			"-XPOST", "-H", "Content-Type: application/json", "-d", pushBody, pushURLFor(noCertPodIP),
		}

		noCertCode, _, noCertErr := probe.Exec(ctx, gatewayProbeContainer, noCertCurl...)
		switch {
		case noCertErr == nil && isSuccessPushCode(noCertCode):
			problems = append(problems, fmt.Sprintf("W2 push without client cert unexpectedly succeeded (HTTP %s); mTLS enforcement broken", noCertCode))
		case noCertErr == nil:
			// handshake completed but nginx refused the request at the HTTP level
			// (400 "No required SSL certificate was sent"); mTLS enforced.
		case isTLSRejection(noCertErr):
			// curl bailed during or right after the handshake; mTLS enforced.
		default:
			// an infrastructure failure (timeout, pod gone) must not pass as
			// proof of mTLS enforcement.
			problems = append(problems, fmt.Sprintf("W2 push without client cert failed for a non-TLS reason, cannot prove mTLS enforcement: %v", noCertErr))
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("gateway write path checks failed:\n%s", strings.Join(problems, "\n"))
	}

	log.Info("MLA Gateway write path verified.")
	return nil
}

// isTLSRejection reports whether an exec error looks like curl refusing at the
// TLS layer: exit 35 (SSL connect error) when the handshake is rejected
// outright, exit 56 (recv failure) when nginx completes the TLS 1.3 handshake
// and then closes the connection with a certificate-required alert.
func isTLSRejection(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()

	return strings.Contains(msg, "exit code 35") || strings.Contains(msg, "exit code 56")
}

// waitGatewayPodReady returns the pod IP of a ready mla-gateway pod in the
// cluster namespace. It first waits for the gateway Deployment rollout to be
// complete: the rate-limit checks that run just before rewrite the gateway
// nginx ConfigMap, and the checksum annotation forces a rolling update. A pod
// picked mid-rollout can belong to the losing ReplicaSet and be deleted while
// the write-path checks are still curling its IP.
func waitGatewayPodReady(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (string, error) {
	var gatewayPodIP string
	if err := wait.Poll(ctx, 2*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		deployment := &appsv1.Deployment{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: gatewayName}, deployment); err != nil {
			return fmt.Errorf("failed to get gateway deployment: %w", err), nil
		}

		replicas := int32(1)
		if deployment.Spec.Replicas != nil {
			replicas = *deployment.Spec.Replicas
		}

		if deployment.Status.ObservedGeneration < deployment.Generation ||
			deployment.Status.UpdatedReplicas != replicas ||
			deployment.Status.ReadyReplicas != replicas ||
			deployment.Status.Replicas != replicas {
			return errors.New("gateway deployment rollout not complete yet"), nil
		}

		ip, err := readyGatewayPodIP(ctx, client, namespace)
		if err != nil {
			return err, nil
		}

		gatewayPodIP = ip
		return nil, nil
	}); err != nil {
		return "", err
	}

	return gatewayPodIP, nil
}

// readyGatewayPodIP returns the pod IP of a ready, non-terminating mla-gateway
// pod. Callers that curl the gateway over a longer period re-resolve the IP
// per attempt so a rollout between attempts does not leave them talking to a
// deleted pod.
func readyGatewayPodIP(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (string, error) {
	pods := &corev1.PodList{}
	if err := client.List(ctx, pods,
		ctrlruntimeclient.InNamespace(namespace),
		ctrlruntimeclient.MatchingLabels{"app.kubernetes.io/name": gatewayName}); err != nil {
		return "", fmt.Errorf("failed to list gateway pods: %w", err)
	}

	for i := range pods.Items {
		p := &pods.Items[i]
		if p.DeletionTimestamp == nil && p.Status.PodIP != "" && isPodReady(p) {
			return p.Status.PodIP, nil
		}
	}

	return "", errors.New("no ready gateway pod")
}

func isPodReady(p *corev1.Pod) bool {
	for _, c := range p.Status.ContainerStatuses {
		if c.Name != "nginx" {
			continue
		}

		if c.Ready {
			return true
		}
	}

	return false
}

// isSuccessPushCode treats any 2xx as a successful write (Loki push normally
// returns 204). An empty code means curl bailed before any HTTP response
// (TLS handshake failure), which is a failure for positive assertions and a
// pass for the mTLS-negative one.
func isSuccessPushCode(code string) bool {
	return len(code) == 3 && code[0] == '2'
}

// newGatewayProbePod returns an agnhost pod that stays alive (pause) and mounts
// the staged TLS material so curl inside it can present the client cert.
func newGatewayProbePod(ns, tlsSecretName string) *corev1.Pod {
	const mountPath = "/etc/ssl/mla"
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gatewayProbePodName,
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:            gatewayProbeContainer,
					Image:           gatewayWriteProbeImage,
					Args:            []string{"pause"},
					ImagePullPolicy: corev1.PullIfNotPresent,
					VolumeMounts: []corev1.VolumeMount{
						{Name: "tls", MountPath: mountPath, ReadOnly: true},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "tls",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  tlsSecretName,
							DefaultMode: ptr.To[int32](0400),
						},
					},
				},
			},
			TerminationGracePeriodSeconds: ptr.To[int64](0),
		},
	}
}
