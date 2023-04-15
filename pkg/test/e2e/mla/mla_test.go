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
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	mlacontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/mla-controller"
	"k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/mla-controller/cortex"
	"k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/mla-controller/grafana"
	"k8c.io/kubermatic/v3/pkg/log"
	"k8c.io/kubermatic/v3/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/kubermatic/v3/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v3/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v3/pkg/test/generator"
	"k8c.io/kubermatic/v3/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

	grafanaSecret   = "mla/grafana"
	grafanaURL      = "http://localhost:3000"
	alertmanagerURL = "http://localhost:3001"
	rulerURL        = "http://localhost:3002"
	lokiURL         = "http://localhost:3003"
)

var (
	credentials jig.BYOCredentials
	logOptions  = utils.DefaultLogOptions
)

func init() {
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

func TestMLAIntegration(t *testing.T) {
	ctx := context.Background()
	logger := log.NewFromOptions(logOptions).Sugar()

	if err := credentials.Parse(); err != nil {
		t.Fatalf("Failed to get credentials: %v", err)
	}

	seedClient, _, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("failed to get client for seed cluster: %v", err)
	}

	// create test environment
	testJig := jig.NewBYOCluster(seedClient, logger, credentials)
	testJig.ClusterJig.WithTestName("mla")

	cluster, err := testJig.Setup(ctx, jig.WaitForReadyPods)
	defer testJig.Cleanup(ctx, t, true)
	if err != nil {
		t.Fatalf("failed to setup test environment: %v", err)
	}

	logger.Info("Enabling MLA...")
	if err := setMLAIntegration(ctx, seedClient, cluster, true); err != nil {
		t.Fatalf("failed to enable MLA: %v", err)
	}

	logger.Info("Waiting for new seed-controller-manager to acquire leader lease...")
	time.Sleep(30 * time.Second)

	logger.Info("Creating Grafana client for user cluster...")
	httpClient := &http.Client{Timeout: 15 * time.Second}
	cortexClient := cortex.NewClient(httpClient, alertmanagerURL, rulerURL, lokiURL)

	grafanaClientProvider, err := grafana.NewClientProvider(seedClient, httpClient, grafanaSecret, grafanaURL, true)
	if err != nil {
		t.Fatalf("failed to initialize Grafana client: %v", err)
	}

	grafanaClient, err := grafanaClientProvider(ctx)
	if err != nil {
		t.Fatalf("failed to initialize Grafana client: %v", err)
	}

	logger.Info("Waiting for Grafana organization...")
	var org *grafanasdk.Org
	err = wait.PollLog(ctx, logger, 5*time.Second, 3*time.Minute, func() (transient error, terminal error) {
		organization, err := grafanaClient.GetOrgByOrgName(ctx, mlacontroller.GrafanaOrganization)
		if err != nil {
			return err, nil
		}
		org = &organization
		return nil, nil
	})
	if err != nil {
		// this is actually a fatal error, as no other reconcilers can do anything useful without an org
		t.Fatalf("failed to reconcile, Grafana organization never appeared: %v", err)
	}

	grafanaClient.SetOrgIDHeader(org.ID)

	if err := verifyGrafanaDatasource(ctx, logger, grafanaClient, cluster); err != nil {
		t.Errorf("failed to verify Grafana datasource: %v", err)
	}

	kkpUser, err := verifyGrafanaUser(ctx, logger, seedClient, grafanaClient, org)
	if err != nil {
		t.Errorf("failed to verify Grafana user: %v", err)
	}

	if err := verifyLogsRuleGroup(ctx, logger, seedClient, cortexClient, cluster); err != nil {
		t.Errorf("failed to verify logs RuleGroup: %v", err)
	}

	if err := verifyMetricsRuleGroup(ctx, logger, seedClient, cortexClient, cluster); err != nil {
		t.Errorf("failed to verify metrics RuleGroup: %v", err)
	}

	if err := verifyAlertmanager(ctx, logger, seedClient, cortexClient, cluster); err != nil {
		t.Errorf("failed to verify Alertmanager: %v", err)
	}

	if err := verifyRateLimits(ctx, logger, seedClient, cluster); err != nil {
		t.Errorf("failed to verify rate limits: %v", err)
	}

	logger.Info("Disabling MLA...")
	if err := setMLAIntegration(ctx, seedClient, cluster, false); err != nil {
		t.Fatalf("failed to disable MLA: %v", err)
	}

	logger.Info("Waiting for new seed-controller-manager to acquire leader lease...")
	time.Sleep(30 * time.Second)

	logger.Info("Waiting for cluster to healthy...")
	if err := testJig.WaitForHealthyControlPlane(ctx, 2*time.Minute); err != nil {
		t.Fatalf("cluster did not get healthy: %v", err)
	}

	if kkpUser != nil {
		logger.Info("Waiting for Grafana user to be gone...")
		err = wait.PollLog(ctx, logger, 3*time.Second, 5*time.Minute, func() (transient error, terminal error) {
			if _, err := grafanaClient.LookupUser(ctx, kkpUser.Spec.Email); err == nil {
				return errors.New("user still exists"), nil
			}
			return nil, nil
		})
		if err != nil {
			t.Fatal("cleanup did not complete successfully")
		}
	}
}

func verifyGrafanaDatasource(ctx context.Context, log *zap.SugaredLogger, grafanaClient grafana.Client, cluster *kubermaticv1.Cluster) error {
	datasource := mlacontroller.DatasourceUIDForCluster(grafana.DatasourceTypePrometheus, cluster)

	log = log.With("datasource", datasource)
	log.Info("Waiting for datasource to be added to Grafana...")

	err := wait.PollImmediateLog(ctx, log, 3*time.Second, 3*time.Minute, func() (transient error, terminal error) {
		_, err := grafanaClient.GetDatasourceByUID(ctx, datasource)
		return err, nil
	})
	if err != nil {
		return errors.New("timed out waiting for datasource")
	}

	log.Info("Grafana datasource successfully verified.")

	return nil
}

func verifyGrafanaUser(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, grafanaClient grafana.Client, org *grafanasdk.Org) (*kubermaticv1.User, error) {
	log.Info("Checking that an admin user was added to Grafana...")

	users := kubermaticv1.UserList{}
	if err := client.List(ctx, &users); err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	if len(users.Items) != 1 {
		return nil, fmt.Errorf("expected to find exactly 1 KKP User, but found %d", len(users.Items))
	}

	kkpUser := users.Items[0]
	log = log.With("email", kkpUser.Spec.Email)

	user := grafanasdk.User{}
	err := wait.PollLog(ctx, log, 3*time.Second, 1*time.Minute, func() (transient error, terminal error) {
		user, transient = grafanaClient.LookupUser(ctx, kkpUser.Spec.Email)
		return transient, nil
	})
	if err != nil {
		return nil, fmt.Errorf("waiting for grafana user: %w", err)
	}

	if user.IsGrafanaAdmin != true || user.OrgID != org.ID {
		return nil, fmt.Errorf("user expected to be Grafana Admin and have orgID %d, but is %+v", org.ID, user)
	}

	log.Info("Verifying Grafana org user's role...")
	orgUser, err := grafanaClient.GetOrgUser(ctx, org.ID, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get grafana org user: %w", err)
	}

	if orgUser.Role != string(grafanasdk.ROLE_ADMIN) {
		return nil, fmt.Errorf("orgUser %v expected to have Admin role, but has %v", orgUser, orgUser.Role)
	}

	log.Info("Grafana user successfully verified.")

	return &kkpUser, nil
}

func verifyLogsRuleGroup(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, cortexClient cortex.Client, cluster *kubermaticv1.Cluster) error {
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
		return fmt.Errorf("failed to create rule group: %w", err)
	}

	err = wait.PollLog(ctx, log, 3*time.Second, 3*time.Minute, func() (error, error) {
		ruleGroup, err := cortexClient.GetRuleGroupConfiguration(ctx, cluster.Name, kubermaticv1.RuleGroupTypeLogs, "test-rule")
		if err != nil {
			return fmt.Errorf("failed to get rule group: %w", err), nil
		}

		config := map[string]interface{}{}
		decoder := yaml.NewDecoder(bytes.NewReader(ruleGroup))
		if err := decoder.Decode(&config); err != nil {
			return fmt.Errorf("failed to decode rule group: %w", err), nil
		}

		if !reflect.DeepEqual(config, expectedData) {
			return errors.New("Cortex rule group does not match the expected rule group"), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("log rule group not found: %w", err)
	}

	log.Info("RuleGroup successfully verified.")

	return nil
}

func verifyMetricsRuleGroup(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, cortexClient cortex.Client, cluster *kubermaticv1.Cluster) error {
	log.Info("Creating metrics RuleGroup...")

	testRuleGroup := generator.GenerateTestRuleGroupData("test-metric-rule")
	expectedData, err := createRuleGroup(ctx, client, cluster, testRuleGroup, kubermaticv1.RuleGroupTypeMetrics)
	if err != nil {
		return fmt.Errorf("failed to create rule group: %w", err)
	}

	err = wait.Poll(ctx, 3*time.Second, 3*time.Minute, func() (error, error) {
		ruleGroup, err := cortexClient.GetRuleGroupConfiguration(ctx, cluster.Name, kubermaticv1.RuleGroupTypeMetrics, "test-metric-rule")
		if err != nil {
			return fmt.Errorf("failed to get rule group: %w", err), nil
		}

		config := map[string]interface{}{}
		decoder := yaml.NewDecoder(bytes.NewReader(ruleGroup))
		if err := decoder.Decode(&config); err != nil {
			return fmt.Errorf("failed to decode rule group: %w", err), nil
		}

		if !reflect.DeepEqual(config, expectedData) {
			return errors.New("Cortex rule group does not match the expected rule group"), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("metric rule group not found: %w", err)
	}

	log.Info("RuleGroup successfully verified.")

	return nil
}

func verifyAlertmanager(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, cortexClient cortex.Client, cluster *kubermaticv1.Cluster) error {
	log.Info("Verifying Alertmanager...")

	if err := updateAlertmanager(ctx, client, cluster, []byte(testAlertmanagerConfig)); err != nil {
		return fmt.Errorf("failed to update alertmanager config: %w", err)
	}

	err := wait.PollLog(ctx, log, 3*time.Second, 3*time.Minute, func() (transient error, terminal error) {
		if err := client.Get(ctx, types.NamespacedName{Name: cluster.Name}, cluster); err != nil {
			return err, nil
		}

		if *cluster.Status.ExtendedHealth.AlertmanagerConfig != kubermaticv1.HealthStatusUp {
			return fmt.Errorf("alertmanager config is %v", *cluster.Status.ExtendedHealth.AlertmanagerConfig), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for alertmanager: %w", err)
	}

	expectedConfig := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(testAlertmanagerConfig), &expectedConfig); err != nil {
		return fmt.Errorf("failed to unmarshal expected config: %w", err)
	}

	err = wait.PollLog(ctx, log, 3*time.Second, 3*time.Minute, func() (error, error) {
		ruleGroup, err := cortexClient.GetAlertmanagerConfiguration(ctx, cluster.Name)
		if err != nil {
			return fmt.Errorf("failed to get rule group: %w", err), nil
		}

		config := map[string]interface{}{}
		decoder := yaml.NewDecoder(bytes.NewReader(ruleGroup))
		if err := decoder.Decode(&config); err != nil {
			return fmt.Errorf("failed to decode rule group: %w", err), nil
		}

		if !reflect.DeepEqual(config, expectedConfig) {
			return errors.New("Alertmanager rule group does not match the expected rule group"), nil
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

func verifyRateLimits(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	log.Info("Setting rate limits...")

	rateLimits := kubermaticv1.MonitoringRateLimitSettings{
		IngestionRate:      1,
		IngestionBurstSize: 2,
		MaxSeriesPerMetric: 3,
		MaxSeriesTotal:     4,
		MaxSamplesPerQuery: 5,
		MaxSeriesPerQuery:  6,
	}
	if err := createMonitoringMLARateLimits(ctx, client, cluster, rateLimits); err != nil {
		return fmt.Errorf("failed to set monitoring rate limits: %w", err)
	}

	err := wait.PollLog(ctx, log, 3*time.Second, 3*time.Minute, func() (error, error) {
		mlaAdminSetting := &kubermaticv1.MLAAdminSetting{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: resources.MLAAdminSettingsName}, mlaAdminSetting); ctrlruntimeclient.IgnoreNotFound(err) != nil {
			return fmt.Errorf("failed to get cluster mlaadminsetting: %w", err), nil
		}

		configMap := &corev1.ConfigMap{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: "mla", Name: mlacontroller.RuntimeConfigMap}, configMap); err != nil {
			return fmt.Errorf("failed to get configMap: %w", err), nil
		}

		actualOverrides := &mlacontroller.Overrides{}
		decoder := yaml.NewDecoder(strings.NewReader(configMap.Data[mlacontroller.RuntimeConfigFileName]))
		decoder.KnownFields(true)
		if err := decoder.Decode(actualOverrides); err != nil {
			return fmt.Errorf("failed to unmarshal rate limit config map"), nil
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

func createMonitoringMLARateLimits(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, rateLimits kubermaticv1.MonitoringRateLimitSettings) error {
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
	if err := toggleMLAInConfiguration(ctx, client, enabled); err != nil {
		return fmt.Errorf("failed to toggle MLA integration to %v: %w", enabled, err)
	}

	oldCluster := cluster.DeepCopy()
	cluster.Spec.MLA = &kubermaticv1.MLASettings{
		MonitoringEnabled: enabled,
		LoggingEnabled:    enabled,
	}

	return client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}

func toggleMLAInConfiguration(ctx context.Context, client ctrlruntimeclient.Client, enable bool) error {
	config, err := kubernetes.GetRawKubermaticConfiguration(ctx, client, jig.KubermaticNamespace())
	if err != nil {
		return fmt.Errorf("failed to get KubermaticConfiguration: %w", err)
	}

	if config.Spec.UserCluster == nil {
		config.Spec.UserCluster = &kubermaticv1.KubermaticUserClusterConfiguration{}
	}

	config.Spec.UserCluster.MLA = &kubermaticv1.KubermaticUserClusterMLAConfiguration{
		Enabled: enable,
	}

	if err := client.Update(ctx, config); err != nil {
		return fmt.Errorf("failed to update KubermaticConfiguration: %w", err)
	}

	return nil
}

func createRuleGroup(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, data []byte, kind kubermaticv1.RuleGroupType) (map[string]interface{}, error) {
	expected := map[string]interface{}{}
	if err := yaml.Unmarshal(data, &expected); err != nil {
		return nil, fmt.Errorf("failed to unmarshal expected rule group: %w", err)
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
