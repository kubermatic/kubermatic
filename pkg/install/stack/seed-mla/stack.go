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
	"k8s.io/apimachinery/pkg/types"
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

	AlloyChartName   = "alloy"
	AlloyReleaseName = AlloyChartName
	AlloyNamespace   = LoggingNamespace
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

	if err := deployAlloy(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Alloy: %w", err)
	}

	if err := removePromtail(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to remove Promtail: %w", err)
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

		err = upgradeNodeExporterDaemonSet(ctx, sublogger, kubeClient, helmClient, opt, chart, release)
		if err != nil {
			sublogger.Warnf("Failed to temporarily remove Node Exporter daemonset: %s", err)
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
			sublogger.Warnf("Failed to temporarily remove Kube State Metrics deployment: %s", err)
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

	v28 := semverlib.MustParse("2.28.0")

	if release != nil && release.Version.LessThan(v28) && !chart.Version.LessThan(v28) {
		sublogger.Warn("Installation process will temporarily remove and then upgrade the deployment set used by blackbox-exporter.")

		err = upgradeBlackboxExporterDeployment(ctx, sublogger, kubeClient, helmClient, opt, chart, release)
		if err != nil {
			sublogger.Warnf("Failed to temporarily remove Blackbox Exporter deployment: %s", err)
		}
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
		sublogger.Warn("Installation process will temporarily remove and then upgrade Alertmanager Statefulset.")

		err = upgradeAlertmanagerStatefulset(ctx, sublogger, kubeClient, helmClient, opt, chart, release)
		if err != nil {
			sublogger.Warnf("Failed to temporarily remove Alertmanager Statefulset: %s", err)
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

func removePromtail(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	if slices.Contains(opt.SkipCharts, PromtailChartName) || opt.MLASkipLogging {
		logger.Info("⭕ Skipping removal of Promtail deployment.")
		return nil
	}

	sublogger := log.Prefix(logger, "   ")
	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, PromtailNamespace, PromtailReleaseName)

	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	if release == nil {
		return nil
	}

	logger.Info("📦 Removing Promtail…")

	if err := helmClient.UninstallRelease(PromtailNamespace, PromtailReleaseName); err != nil {
		return fmt.Errorf("failed to remove Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployAlloy(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	if slices.Contains(opt.SkipCharts, AlloyChartName) || opt.MLASkipLogging {
		logger.Info("⭕ Skipping Grafana Alloy deployment.")
		return nil
	}

	logger.Info("📦 Deploying Grafana Alloy")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, LoggingChartsPrefix, AlloyChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, AlloyNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, AlloyNamespace, AlloyReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, AlloyNamespace, AlloyReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func upgradeAlertmanagerStatefulset(
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

	statefulset := &unstructured.Unstructured{}
	statefulset.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Kind:    "StatefulSet",
		Version: "v1",
	})

	key := types.NamespacedName{Name: AlertManagerReleaseName, Namespace: AlertManagerNamespace}

	err := kubeClient.Get(ctx, key, statefulset)

	if err != nil {
		return fmt.Errorf("failed get statefulset: %w", err)
	}

	// 2: store the statefulset for backup
	backupTS := time.Now().Format("2006-01-02T150405")
	filename := fmt.Sprintf("backup_%s_%s.yaml", AlertManagerReleaseName, backupTS)
	logger.Infof("Attempting to store the statefulset in file: %s", filename)
	if err := util.DumpResources(ctx, filename, []unstructured.Unstructured{*statefulset}); err != nil {
		return fmt.Errorf("failed to back up the statefulsets, it is not removed: %w", err)
	}

	// 3: delete the statefulset
	logger.Info("Deleting the statefulset from the cluster")
	if err := kubeClient.Delete(ctx, statefulset); err != nil {
		return fmt.Errorf("failed to remove the statefulset: %w\n\nuse backup file to check the changes and restore if needed", err)
	}

	return nil
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

	deployment := &unstructured.Unstructured{}
	deployment.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Kind:    "Deployment",
		Version: "v1",
	})
	key := types.NamespacedName{Name: KubeStateMetricsReleaseName, Namespace: KubeStateMetricsNamespace}

	err := kubeClient.Get(ctx, key, deployment)

	if err != nil {
		return fmt.Errorf("failed get deployment: %w", err)
	}

	// 2: store the deployment for backup
	backupTS := time.Now().Format("2006-01-02T150405")
	filename := fmt.Sprintf("backup_%s_%s.yaml", KubeStateMetricsReleaseName, backupTS)
	logger.Infof("Attempting to store the deployments in file: %s", filename)
	if err := util.DumpResources(ctx, filename, []unstructured.Unstructured{*deployment}); err != nil {
		return fmt.Errorf("failed to back up the deployment, it is not removed: %w", err)
	}

	// 3: delete the deployment
	logger.Info("Deleting the deployment from the cluster")
	if err := kubeClient.Delete(ctx, deployment); err != nil {
		return fmt.Errorf("failed to remove the deployment: %w\n\nuse backup file to check the changes and restore if needed", err)
	}

	return nil
}

func upgradeNodeExporterDaemonSet(
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

	daemonset := &unstructured.Unstructured{}
	daemonset.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Kind:    "DaemonSet",
		Version: "v1",
	})

	key := types.NamespacedName{Name: NodeExporterReleaseName, Namespace: NodeExporterNamespace}

	err := kubeClient.Get(ctx, key, daemonset)

	if err != nil {
		return fmt.Errorf("failed get daemonset: %w", err)
	}

	// 2: if deamonset exists, then store the daemonset for backup
	backupTS := time.Now().Format("2006-01-02T150405")
	filename := fmt.Sprintf("backup_%s_%s.yaml", NodeExporterReleaseName, backupTS)
	logger.Infof("Attempting to store the daemonset in file: %s", filename)
	if err := util.DumpResources(ctx, filename, []unstructured.Unstructured{*daemonset}); err != nil {
		return fmt.Errorf("failed to back up the daemonset, it is not removed: %w", err)
	}

	// 3: delete the daemonset
	logger.Info("Deleting the daemonset from the cluster")
	if err := kubeClient.Delete(ctx, daemonset); err != nil {
		return fmt.Errorf("failed to remove the daemonset: %w\n\nuse backup file to check the changes and restore if needed", err)
	}

	return nil
}

func upgradeBlackboxExporterDeployment(
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
	logger.Info("Backing up old blackbox-exporter deployment…")

	deployment := &unstructured.Unstructured{}
	deployment.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Kind:    "Deployment",
		Version: "v1",
	})
	key := types.NamespacedName{Name: BlackboxExporterReleaseName, Namespace: BlackboxExporterNamespace}

	err := kubeClient.Get(ctx, key, deployment)

	if err != nil {
		return fmt.Errorf("failed get deployment: %w", err)
	}

	// 2: store the deployment for backup
	backupTS := time.Now().Format("2006-01-02T150405")
	filename := fmt.Sprintf("backup_%s_%s.yaml", BlackboxExporterReleaseName, backupTS)
	logger.Infof("Attempting to store the deployment in file: %s", filename)
	if err := util.DumpResources(ctx, filename, []unstructured.Unstructured{*deployment}); err != nil {
		return fmt.Errorf("failed to back up the deployment, it is not removed: %w", err)
	}

	// 3: delete the deployment
	logger.Info("Deleting the deployment from the cluster")
	if err := kubeClient.Delete(ctx, deployment); err != nil {
		return fmt.Errorf("failed to remove the deployment: %w\n\nuse backup file to check the changes and restore if needed", err)
	}

	return nil
}
