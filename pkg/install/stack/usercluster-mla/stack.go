/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package userclustermla

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/util"
	"k8c.io/kubermatic/v2/pkg/log"

	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	UserClusterMlaNamespace    = "mla"
	UserClusterMlaChartsPrefix = "mla"

	MlaSecretsChartName   = "mla-secrets"
	MlaSecretsReleaseName = MlaSecretsChartName
	MlaSecretsNamespace   = UserClusterMlaNamespace

	AlertmanagerProxyChartName   = "alertmanager-proxy"
	AlertmanagerProxyReleaseName = AlertmanagerProxyChartName
	AlertmanagerProxyNamespace   = UserClusterMlaNamespace

	ConsulChartName   = "consul"
	ConsulReleaseName = ConsulChartName
	ConsulNamespace   = UserClusterMlaNamespace

	CortexChartName   = "cortex"
	CortexReleaseName = CortexChartName
	CortexNamespace   = UserClusterMlaNamespace

	GrafanaChartName   = "grafana"
	GrafanaReleaseName = GrafanaChartName
	GrafanaNamespace   = UserClusterMlaNamespace

	LokiChartName   = "loki-distributed"
	LokiReleaseName = LokiChartName
	LokiNamespace   = UserClusterMlaNamespace

	MinioChartName   = "minio"
	MinioReleaseName = MinioChartName
	MinioNamespace   = UserClusterMlaNamespace

	MinioLifecycleMgrChartName   = "minio-lifecycle-mgr"
	MinioLifecycleMgrReleaseName = MinioLifecycleMgrChartName
	MinioLifecycleMgrNamespace   = UserClusterMlaNamespace

	MlaIapChartName   = "iap"
	MlaIapReleaseName = MlaIapChartName
	MlaIapNamespace   = UserClusterMlaNamespace
)

type UserClusterMLA struct{}

func NewStack() stack.Stack {
	return &UserClusterMLA{}
}

var _ stack.Stack = &UserClusterMLA{}

func (*UserClusterMLA) Name() string {
	return "KKP User Cluster MLA"
}

func (s *UserClusterMLA) Deploy(ctx context.Context, opt stack.DeployOptions) error {
	if err := deployMlaSecrets(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy MLA Secrets: %w", err)
	}

	if err := deployAlertmanagerProxy(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy AlertManager Proxy: %w", err)
	}

	if err := deployConsul(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Consul: %w", err)
	}

	if err := deployMinio(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Minio: %w", err)
	}

	if err := deployCortex(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Cortex: %w", err)
	}

	if err := deployGrafana(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Grafana: %w", err)
	}

	if err := deployLoki(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Loki: %w", err)
	}

	if err := deployMinioLifecycleMgr(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Minio Bucket Lifecycle Manager: %w", err)
	}

	if opt.MlaIncludeIap {
		if err := deployMlaIap(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
			return fmt.Errorf("failed to deploy IAP: %w", err)
		}
	}

	return nil
}

func deployMlaSecrets(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying MLA Secrets…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, UserClusterMlaChartsPrefix, MlaSecretsChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, MlaSecretsNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, MlaSecretsNamespace, MlaSecretsReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	// If secrets upgrade wasn't forced and there's no newer version, don't upgrade the secrets
	if !opt.MlaForceSecrets && (release != nil && !release.Version.LessThan(chart.Version)) {
		logger.Info("⏭️  Skipped.")
		return nil
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, MlaSecretsNamespace, MlaSecretsReleaseName, opt.HelmValues, true, (opt.MlaForceSecrets || opt.ForceHelmReleaseUpgrade), opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployAlertmanagerProxy(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying Alertmanager Proxy…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, UserClusterMlaChartsPrefix, AlertmanagerProxyChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, AlertmanagerProxyNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, AlertmanagerProxyNamespace, AlertmanagerProxyReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, AlertmanagerProxyNamespace, AlertmanagerProxyReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployConsul(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying Consul…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, UserClusterMlaChartsPrefix, ConsulChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, ConsulNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, ConsulNamespace, ConsulReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, ConsulNamespace, ConsulReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployCortex(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying Cortex…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, UserClusterMlaChartsPrefix, CortexChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, CortexNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, CortexNamespace, CortexReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	runtimeConfigMap := &corev1.ConfigMap{
		Data: map[string]string{"runtime-config.yaml": ""},
	}
	runtimeConfigMap.Name = "cortex-runtime-config"
	runtimeConfigMap.Namespace = CortexNamespace

	if err := ctrlruntimeclient.IgnoreAlreadyExists(kubeClient.Create(ctx, runtimeConfigMap)); err != nil {
		return fmt.Errorf("failed to create runtime-config ConfigMap: %w", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, CortexNamespace, CortexReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployGrafana(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying Grafana…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, UserClusterMlaChartsPrefix, GrafanaChartName))
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

func deployLoki(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying Loki…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, UserClusterMlaChartsPrefix, LokiChartName))
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

func deployMinio(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying Minio…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, UserClusterMlaChartsPrefix, MinioChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, MinioNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, MinioNamespace, MinioReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, MinioNamespace, MinioReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployMinioLifecycleMgr(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying Minio Bucket Lifecycle Manager…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, UserClusterMlaChartsPrefix, MinioLifecycleMgrChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, MinioLifecycleMgrNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, MinioLifecycleMgrNamespace, MinioLifecycleMgrReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, MinioLifecycleMgrNamespace, MinioLifecycleMgrReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployMlaIap(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying IAP…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, MlaIapChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, MlaIapNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, MlaIapNamespace, MlaIapReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, MlaIapNamespace, MlaIapReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}
