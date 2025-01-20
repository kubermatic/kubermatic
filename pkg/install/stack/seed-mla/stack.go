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

	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/util"
	"k8c.io/kubermatic/v2/pkg/log"

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

type Monitoring struct{}

func NewStack() stack.Stack {
	return &Monitoring{}
}

var _ stack.Stack = &Monitoring{}

func (*Monitoring) Name() string {
	return "KKP Seed MLA Stack"
}

func (s *Monitoring) Deploy(ctx context.Context, opt stack.DeployOptions) error {
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
