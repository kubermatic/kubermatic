/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package seedmla

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"time"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/util"
	"k8c.io/kubermatic/v2/pkg/log"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MonitoringNamespace    = "monitoring"
	MonitoringChartsPrefix = "monitoring"

	NodeExporterChartName   = "node-exporter"
	NodeExporterReleaseName = NodeExporterChartName
	NodeExporterNamespace   = MonitoringNamespace

	KubeStateMetricsChartName   = "kube-state-metrics"
	KubeStateMetricsReleaseName = KubeStateMetricsChartName
	KubeStateMetricsNamespace   = MonitoringNamespace

	GrafanaChartName   = "grafana"
	GrafanaReleaseName = GrafanaChartName
	GrafanaNamespace   = MonitoringNamespace

	BlackboxExporterChartName   = "blackbox-exporter"
	BlackboxExporterReleaseName = BlackboxExporterChartName
	BlackboxExporterNamespace   = MonitoringNamespace

	AlertManagerChartName   = "alertmanager"
	AlertManagerReleaseName = AlertManagerChartName
	AlertManagerNamespace   = MonitoringNamespace

	PrometheusChartName   = "prometheus"
	PrometheusReleaseName = PrometheusChartName
	PrometheusNamespace   = MonitoringNamespace

	HelmExporterChartName   = "helm-exporter"
	HelmExporterReleaseName = HelmExporterChartName
	HelmExporterNamespace   = MonitoringNamespace

	KarmaChartName   = "karma"
	KarmaReleaseName = KarmaChartName
	KarmaNamespace   = MonitoringNamespace

	MonitoringIAPChartName = "iap"
	IAPReleaseName         = MonitoringIAPChartName
	IAPNamespace           = MonitoringNamespace

	LoggingNamespace    = "logging"
	LoggingChartsPrefix = "logging"

	LokiChartName   = "loki"
	LokiReleaseName = LokiChartName
	LokiNamespace   = LoggingNamespace

	PromtailChartName   = "promtail"
	PromtailReleaseName = PromtailChartName
	PromtailNamespace   = LoggingNamespace
)

type MonitoringStack struct{}

func NewStack() stack.Stack {
	return &MonitoringStack{}
}

var _ stack.Stack = &MonitoringStack{}

func (*MonitoringStack) Name() string {
	return "KKP Seed MLA Stack"
}

func (s *MonitoringStack) Deploy(ctx context.Context, opt stack.DeployOptions) error {
	if err := deployNodeExporter(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Node Exporter: %w", err)
	}

	if err := deployKubeStateMetrics(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy kube-state-metrics: %w", err)
	}

	if err := deployGrafana(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Grafana: %w", err)
	}

	if err := deployBlackboxExporter(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Blackbox Exporter: %w", err)
	}

	if err := deployAlertManager(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Alertmanager: %w", err)
	}

	if err := deployPrometheus(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Prometheus: %w", err)
	}

	if err := deployHelmExporter(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Helm Exporter: %w", err)
	}

	if err := deployKarma(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Karma: %w", err)
	}

	if opt.MLAIncludeIap {
		if err := deployIap(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
			return fmt.Errorf("failed to deploy IAP: %w", err)
		}
	}

	if err := deployLoki(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Loki: %w", err)
	}

	if err := deployPromtail(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Promtail: %w", err)
	}

	return nil
}

func deployNodeExporter(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	if slices.Contains(opt.SkipCharts, NodeExporterChartName) {
		logger.Info("⭕ Skipping Node Exporter deployment.")
		return nil
	}
	logger.Info("📦 Deploying Node Exporter ...")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, MonitoringChartsPrefix, NodeExporterChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, NodeExporterNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, NodeExporterNamespace, NodeExporterReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	v28 := semverlib.MustParse("2.28.0")

	if release != nil && release.Version.LessThan(v28) && !chart.Version.LessThan(v28) {
		sublogger.Warn("Installation process will temporarily remove and then upgrade DaemonSet used by Node Exporter.")

		err = upgradeNodeExporterDaemonSets(ctx, sublogger, kubeClient, helmClient, opt, chart, release)
		if err != nil {
			return fmt.Errorf("failed to prepare Node Exporter for upgrade: %w", err)
		}
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, NodeExporterNamespace, NodeExporterReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployKubeStateMetrics(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	if slices.Contains(opt.SkipCharts, KubeStateMetricsChartName) {
		logger.Info("⭕ Skipping kube-state-metrics deployment.")
		return nil
	}
	logger.Info("📦 Deploying kube-state-metrics…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, MonitoringChartsPrefix, KubeStateMetricsChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, KubeStateMetricsNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, KubeStateMetricsNamespace, KubeStateMetricsReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	v28 := semverlib.MustParse("2.28.0")

	if release != nil && release.Version.LessThan(v28) && !chart.Version.LessThan(v28) {
		sublogger.Warn("Installation process will temporarily remove and then upgrade the deployment set used by kube-state-metrics.")

		err = upgradeKubeStateMetricsDeployment(ctx, sublogger, kubeClient, helmClient, opt, chart, release)
		if err != nil {
			return fmt.Errorf("failed to prepare kube-state-metrics for upgrade: %w", err)
		}
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, KubeStateMetricsNamespace, KubeStateMetricsReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployGrafana(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	if slices.Contains(opt.SkipCharts, GrafanaChartName) {
		logger.Info("⭕ Skipping Grafana deployment.")
		return nil
	}
	logger.Info("📦 Deploying Grafana…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, MonitoringChartsPrefix, GrafanaChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, GrafanaNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, GrafanaNamespace, GrafanaReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, GrafanaNamespace, GrafanaReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployBlackboxExporter(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	if slices.Contains(opt.SkipCharts, BlackboxExporterChartName) {
		logger.Info("⭕ Skipping Blackbox Exporter deployment.")
		return nil
	}
	logger.Info("📦 Deploying Blackbox Exporter…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, MonitoringChartsPrefix, BlackboxExporterChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, BlackboxExporterNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, BlackboxExporterNamespace, BlackboxExporterReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, BlackboxExporterNamespace, BlackboxExporterReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployAlertManager(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	if slices.Contains(opt.SkipCharts, AlertManagerChartName) {
		logger.Info("⭕ Skipping Alertmanager deployment.")
		return nil
	}
	logger.Info("📦 Deploying Alertmanager…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, MonitoringChartsPrefix, AlertManagerChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, AlertManagerNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, AlertManagerNamespace, AlertManagerReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	v28 := semverlib.MustParse("2.28.0")

	if release != nil && release.Version.LessThan(v28) && !chart.Version.LessThan(v28) {
		sublogger.Warn("Installation process will temporarily remove and then upgrade Alertmanager Statefulsets.")

		err = upgradeAlertmanagerStatefulsets(ctx, sublogger, kubeClient, helmClient, opt, chart, release)
		if err != nil {
			return fmt.Errorf("failed to prepare Alertmanager for upgrade: %w", err)
		}
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, AlertManagerNamespace, AlertManagerReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployPrometheus(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	if slices.Contains(opt.SkipCharts, PrometheusChartName) {
		logger.Info("⭕ Skipping Prometheus deployment.")
		return nil
	}

	logger.Info("📦 Deploying Prometheus…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, MonitoringChartsPrefix, PrometheusChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, PrometheusNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, PrometheusNamespace, PrometheusReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, PrometheusNamespace, PrometheusReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployHelmExporter(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	if slices.Contains(opt.SkipCharts, HelmExporterChartName) {
		logger.Info("⭕ Skipping Helm Exporter deployment.")
		return nil
	}

	logger.Info("📦 Deploying Helm Exporter…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, MonitoringChartsPrefix, HelmExporterChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, HelmExporterNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, HelmExporterNamespace, HelmExporterReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, HelmExporterNamespace, HelmExporterReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployKarma(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	if slices.Contains(opt.SkipCharts, KarmaChartName) {
		logger.Info("⭕ Skipping Karma deployment.")
		return nil
	}

	logger.Info("📦 Deploying Karma…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, MonitoringChartsPrefix, KarmaChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, KarmaNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, KarmaNamespace, KarmaReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, KarmaNamespace, KarmaReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployIap(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying IAP…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, MonitoringIAPChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, IAPNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, IAPNamespace, IAPReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, IAPNamespace, IAPReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployLoki(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	if slices.Contains(opt.SkipCharts, LokiChartName) || opt.MLASkipLogging {
		logger.Info("⭕ Skipping Loki deployment.")
		return nil
	}

	logger.Info("📦 Deploying Loki…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, LoggingChartsPrefix, LokiChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, LokiNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, LokiNamespace, LokiReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, LokiNamespace, LokiReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployPromtail(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	if slices.Contains(opt.SkipCharts, PromtailChartName) || opt.MLASkipLogging {
		logger.Info("⭕ Skipping Promtail deployment.")
		return nil
	}

	logger.Info("📦 Deploying Promtail…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, LoggingChartsPrefix, PromtailChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, PromtailNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, PromtailNamespace, PromtailReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, PromtailNamespace, PromtailReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func upgradeAlertmanagerStatefulsets(
	ctx context.Context,
	logger *logrus.Entry,
	kubeClient ctrlruntimeclient.Client,
	helmClient helm.Client,
	opt stack.DeployOptions,
	chart *helm.Chart,
	release *helm.Release,
) error {
	logger.Infof("%s: %s detected, performing upgrade to %s…", release.Name, release.Version.String(), chart.Version.String())
	// 1: find the old deployment
	logger.Info("Backing up old alertmanager statefulset…")

	statefulsetsList := &unstructured.UnstructuredList{}
	statefulsetsList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Kind:    "StatefulSetList",
		Version: "v1",
	})

	alertmanagerSts, err := getAlertmanagerStatefulsets(ctx, kubeClient, release, "alertmanager")
	if err != nil {
		return err
	}

	statefulsetsList.Items = append(statefulsetsList.Items, alertmanagerSts.Items...)

	// 2: store the statefulset for backup
	backupTS := time.Now().Format("2006-01-02T150405")
	filename := fmt.Sprintf("backup_%s_%s.yaml", AlertManagerReleaseName, backupTS)
	logger.Infof("Attempting to store the statefulset in file: %s", filename)
	if err := util.DumpResources(ctx, filename, statefulsetsList.Items); err != nil {
		return fmt.Errorf("failed to back up the statefulsets, it is not removed: %w", err)
	}

	// 3: delete the statefulset
	logger.Info("Deleting the statefulset from the cluster")
	for _, obj := range statefulsetsList.Items {
		if err := kubeClient.Delete(ctx, &obj); err != nil {
			return fmt.Errorf("failed to remove the statefulset: %w\n\nuse backup file to check the changes and restore if needed", err)
		}
	}
	return nil
}

func getAlertmanagerStatefulsets(
	ctx context.Context,
	kubeClient ctrlruntimeclient.Client,
	release *helm.Release,
	appName string,
) (*unstructured.UnstructuredList, error) {
	statefulsetsList := &unstructured.UnstructuredList{}
	statefulsetsList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Kind:    "StatefulSetList",
		Version: "v1",
	})

	alertmanagerMatcher := ctrlruntimeclient.MatchingLabels{
		"app": appName,
	}
	if err := kubeClient.List(ctx, statefulsetsList, ctrlruntimeclient.InNamespace(AlertManagerNamespace), alertmanagerMatcher); err != nil {
		return nil, fmt.Errorf("Error querying API for the existing Deployment object, aborting upgrade process.")
	}
	return statefulsetsList, nil
}

func upgradeKubeStateMetricsDeployment(
	ctx context.Context,
	logger *logrus.Entry,
	kubeClient ctrlruntimeclient.Client,
	helmClient helm.Client,
	opt stack.DeployOptions,
	chart *helm.Chart,
	release *helm.Release,
) error {
	logger.Infof("%s: %s detected, performing upgrade to %s…", release.Name, release.Version.String(), chart.Version.String())
	// 1: find the old deployment
	logger.Info("Backing up old kube-state-metrics deployment…")

	deploymentsList := &unstructured.UnstructuredList{}
	deploymentsList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Kind:    "DeploymentList",
		Version: "v1",
	})

	ksbMatcher := ctrlruntimeclient.MatchingLabels{
		"app":                          KubeStateMetricsReleaseName,
		"app.kubernetes.io/managed-by": "Helm",
		"release":                      release.Name,
	}
	if err := kubeClient.List(ctx, deploymentsList, ctrlruntimeclient.InNamespace(KubeStateMetricsNamespace), ksbMatcher); err != nil {
		return fmt.Errorf("Error querying API for the existing Deployment object, aborting upgrade process.")
	}

	// 2: store the deployment for backup
	backupTS := time.Now().Format("2006-01-02T150405")
	filename := fmt.Sprintf("backup_%s_%s.yaml", KubeStateMetricsReleaseName, backupTS)
	logger.Infof("Attempting to store the deployments in file: %s", filename)
	if err := util.DumpResources(ctx, filename, deploymentsList.Items); err != nil {
		return fmt.Errorf("failed to back up the deployments, it is not removed: %w", err)
	}

	// 3: delete the deployment
	logger.Info("Deleting the deployments from the cluster")
	for _, obj := range deploymentsList.Items {
		if err := kubeClient.Delete(ctx, &obj); err != nil {
			return fmt.Errorf("failed to remove the deployment: %w\n\nuse backup file to check the changes and restore if needed", err)
		}
	}
	return nil
}

func upgradeNodeExporterDaemonSets(
	ctx context.Context,
	logger *logrus.Entry,
	kubeClient ctrlruntimeclient.Client,
	helmClient helm.Client,
	opt stack.DeployOptions,
	chart *helm.Chart,
	release *helm.Release,
) error {
	logger.Infof("%s: %s detected, performing upgrade to %s…", release.Name, release.Version.String(), chart.Version.String())
	// 1: find the old daemonset
	logger.Info("Backing up old Node Exporter daemonset…")

	daemonsetsList := &unstructured.UnstructuredList{}
	daemonsetsList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Kind:    "DaemonSetList",
		Version: "v1",
	})

	nodeExporterMatcher := ctrlruntimeclient.MatchingLabels{
		"app.kubernetes.io/name": NodeExporterReleaseName,
	}
	if err := kubeClient.List(ctx, daemonsetsList, ctrlruntimeclient.InNamespace(NodeExporterNamespace), nodeExporterMatcher); err != nil {
		return fmt.Errorf("error querying API for the existing DaemonSet object, aborting upgrade process.")
	}

	// 2: store the daemonset for backup
	backupTS := time.Now().Format("2006-01-02T150405")
	filename := fmt.Sprintf("backup_%s_%s.yaml", NodeExporterReleaseName, backupTS)
	logger.Infof("Attempting to store the daemonsets in file: %s", filename)
	if err := util.DumpResources(ctx, filename, daemonsetsList.Items); err != nil {
		return fmt.Errorf("failed to back up the daemonsets, it is not removed: %w", err)
	}

	// 3: delete the daemonset
	logger.Info("Deleting the daemonsets from the cluster")
	for _, obj := range daemonsetsList.Items {
		if err := kubeClient.Delete(ctx, &obj); err != nil {
			return fmt.Errorf("failed to remove the daemonset: %w\n\nuse backup file to check the changes and restore if needed", err)
		}
	}
	return nil
}
