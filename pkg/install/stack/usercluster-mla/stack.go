/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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
	"slices"
	"time"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/mla"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/util"
	"k8c.io/kubermatic/v2/pkg/log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	UserClusterMLANamespace    = "mla"
	UserClusterMLAChartsPrefix = "mla"

	MLASecretsChartName   = "mla-secrets"
	MLASecretsReleaseName = MLASecretsChartName
	MLASecretsNamespace   = UserClusterMLANamespace

	AlertmanagerProxyChartName   = "alertmanager-proxy"
	AlertmanagerProxyReleaseName = AlertmanagerProxyChartName
	AlertmanagerProxyNamespace   = UserClusterMLANamespace

	ConsulChartName   = "consul"
	ConsulReleaseName = ConsulChartName
	ConsulNamespace   = UserClusterMLANamespace

	CortexChartName   = "cortex"
	CortexReleaseName = CortexChartName
	CortexNamespace   = UserClusterMLANamespace

	GrafanaChartName   = "grafana"
	GrafanaReleaseName = GrafanaChartName
	GrafanaNamespace   = UserClusterMLANamespace

	LokiChartName   = "loki-distributed"
	LokiReleaseName = LokiChartName
	LokiNamespace   = UserClusterMLANamespace

	MinioChartName   = "minio"
	MinioReleaseName = MinioChartName
	MinioNamespace   = UserClusterMLANamespace

	MinioLifecycleMgrChartName   = "minio-lifecycle-mgr"
	MinioLifecycleMgrReleaseName = MinioLifecycleMgrChartName
	MinioLifecycleMgrNamespace   = UserClusterMLANamespace

	MLAIAPChartName   = "iap"
	MLAIAPReleaseName = MLAIAPChartName
	MLAIAPNamespace   = UserClusterMLANamespace
)

type UserClusterMLAStack struct{}

func NewStack() stack.Stack {
	return &UserClusterMLAStack{}
}

var _ stack.Stack = &UserClusterMLAStack{}

func (*UserClusterMLAStack) Name() string {
	return "KKP User Cluster MLA"
}

func (s *UserClusterMLAStack) Deploy(ctx context.Context, opt stack.DeployOptions) error {
	if err := deployMLASecrets(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy MLA Secrets: %w", err)
	}

	if err := deployAlertmanagerProxy(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy AlertManager Proxy: %w", err)
	}

	if err := deployConsul(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Consul: %w", err)
	}

	if !opt.MLASkipMinio {
		if err := deployMinio(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
			return fmt.Errorf("failed to deploy Minio: %w", err)
		}
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

	if !opt.MLASkipMinioLifecycleMgr {
		if err := deployMinioLifecycleMgr(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
			return fmt.Errorf("failed to deploy Minio Bucket Lifecycle Manager: %w", err)
		}
	}

	if opt.MLAIncludeIap {
		if err := deployMLAIap(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
			return fmt.Errorf("failed to deploy IAP: %w", err)
		}
	}

	return nil
}

func deployMLASecrets(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying MLA Secrets…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, UserClusterMLAChartsPrefix, MLASecretsChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, MLASecretsNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, MLASecretsNamespace, MLASecretsReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	// If secrets upgrade wasn't forced and there's no newer version, don't upgrade the secrets
	if !opt.MLAForceSecrets && (release != nil && !release.Version.LessThan(chart.Version)) {
		logger.Info("⏭️  Skipped.")
		return nil
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, MLASecretsNamespace, MLASecretsReleaseName, opt.HelmValues, true, (opt.MLAForceSecrets || opt.ForceHelmReleaseUpgrade), opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployAlertmanagerProxy(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	if slices.Contains(opt.SkipCharts, AlertmanagerProxyChartName) {
		logger.Info("⭕ Skipping Alertmanager Proxy deployment.")
		return nil
	}
	logger.Info("📦 Deploying Alertmanager Proxy…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, UserClusterMLAChartsPrefix, AlertmanagerProxyChartName))
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
	if slices.Contains(opt.SkipCharts, ConsulChartName) {
		logger.Info("⭕ Skipping Consul deployment.")
		return nil
	}

	logger.Info("📦 Deploying Consul…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, UserClusterMLAChartsPrefix, ConsulChartName))
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
	v22 := semverlib.MustParse("2.22.0")

	if release != nil && release.Version.LessThan(v22) && !chart.Version.LessThan(v22) {
		sublogger.Warn("Installation process will temporarily remove and then upgrade Statefulset used by Consul.")

		err = upgradeConsulStatefulsets(ctx, sublogger, kubeClient, helmClient, opt, chart, release)
		if err != nil {
			return fmt.Errorf("failed to prepare Consul for upgrade: %w", err)
		}
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, ConsulNamespace, ConsulReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployCortex(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	if slices.Contains(opt.SkipCharts, CortexChartName) {
		logger.Info("⭕ Skipping Cortex deployment.")
		return nil
	}

	logger.Info("📦 Deploying Cortex…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, UserClusterMLAChartsPrefix, CortexChartName))
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

	v22 := semverlib.MustParse("2.22.0")
	v30 := semverlib.MustParse("2.30.0")

	if release != nil && release.Version.LessThan(v22) && !chart.Version.LessThan(v22) {
		sublogger.Warn("Installation process will temporarily remove and then upgrade memcached instances used by Cortex.")

		err = upgradeCortexStatefulsets(ctx, sublogger, kubeClient, helmClient, opt, chart, release)
		if err != nil {
			return fmt.Errorf("failed to prepare Cortex for upgrade: %w", err)
		}
	}

	if release != nil && release.Version.LessThan(v30) && !chart.Version.LessThan(v30) {
		sublogger.Warn("Installation process will delete memcached services for v2.30.0+ upgrade (immutable clusterIP change).")

		err = upgradeCortexMemcachedServices(ctx, sublogger, kubeClient, chart, release)
		if err != nil {
			return fmt.Errorf("failed to prepare memcached services for upgrade: %w", err)
		}
	}

	runtimeConfigMap := &corev1.ConfigMap{
		Data: map[string]string{mla.RuntimeConfigFileName: "overrides:\n"},
	}
	runtimeConfigMap.Name = mla.RuntimeConfigMap
	runtimeConfigMap.Namespace = CortexNamespace

	if err := ctrlruntimeclient.IgnoreAlreadyExists(kubeClient.Create(ctx, runtimeConfigMap)); err != nil {
		return fmt.Errorf("failed to create runtime-config ConfigMap: %w", err)
	}

	// Upgrade runtime config key from runtime-config.yaml to runtime_config.yaml after ensuring ConfigMap exists
	if release != nil && release.Version.LessThan(v30) && !chart.Version.LessThan(v30) {
		err = upgradeCortexRuntimeConfigMap(ctx, sublogger, kubeClient, runtimeConfigMap)
		if err != nil {
			return fmt.Errorf("failed to upgrade runtime config ConfigMap: %w", err)
		}
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, CortexNamespace, CortexReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
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

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, UserClusterMLAChartsPrefix, GrafanaChartName))
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
	if opt.MLASkipLogging || slices.Contains(opt.SkipCharts, LokiChartName) {
		logger.Info("⭕ Skipping Loki deployment.")
		return nil
	}

	logger.Info("📦 Deploying Loki…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, UserClusterMLAChartsPrefix, LokiChartName))
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
	if slices.Contains(opt.SkipCharts, MinioChartName) {
		logger.Info("⭕ Skipping Minio deployment.")
		return nil
	}

	logger.Info("📦 Deploying Minio…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, UserClusterMLAChartsPrefix, MinioChartName))
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
	if slices.Contains(opt.SkipCharts, MinioLifecycleMgrChartName) {
		logger.Info("⭕ Skipping Minio Lifecycle Manager deployment.")
		return nil
	}

	logger.Info("📦 Deploying Minio Bucket Lifecycle Manager…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, UserClusterMLAChartsPrefix, MinioLifecycleMgrChartName))
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

func deployMLAIap(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying IAP…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, MLAIAPChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, MLAIAPNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, MLAIAPNamespace, MLAIAPReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, MLAIAPNamespace, MLAIAPReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func upgradeCortexStatefulsets(
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
	logger.Info("Backing up old memcached statefulsets…")

	statefulsetsList := &unstructured.UnstructuredList{}
	statefulsetsList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Kind:    "StatefulSetList",
		Version: "v1",
	})

	memcachedBlocksSts, err := getMemcachedStatefulsets(ctx, kubeClient, release, "memcached-blocks")
	if err != nil {
		return err
	}
	memcachedBlocksIndexSts, err := getMemcachedStatefulsets(ctx, kubeClient, release, "memcached-blocks-index")
	if err != nil {
		return err
	}
	memcachedBlocksMetadataSts, err := getMemcachedStatefulsets(ctx, kubeClient, release, "memcached-blocks-metadata")
	if err != nil {
		return err
	}

	statefulsetsList.Items = append(statefulsetsList.Items, memcachedBlocksSts.Items...)
	statefulsetsList.Items = append(statefulsetsList.Items, memcachedBlocksIndexSts.Items...)
	statefulsetsList.Items = append(statefulsetsList.Items, memcachedBlocksMetadataSts.Items...)

	// 2: store the deployment for backup
	backupTS := time.Now().Format("2006-01-02T150405")
	filename := fmt.Sprintf("backup_%s_%s.yaml", CortexReleaseName, backupTS)
	logger.Infof("Attempting to store the statefulsets in file: %s", filename)
	if err := util.DumpResources(ctx, filename, statefulsetsList.Items); err != nil {
		return fmt.Errorf("failed to back up the statefulsets, it is not removed: %w", err)
	}

	// 3: delete the deployment
	logger.Info("Deleting the statefulsets from the cluster")
	for _, obj := range statefulsetsList.Items {
		if err := kubeClient.Delete(ctx, &obj); err != nil {
			return fmt.Errorf("failed to remove the statefulset: %w\n\nuse backup file to check the changes and restore if needed", err)
		}
	}
	return nil
}

func getMemcachedStatefulsets(
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

	memcachedMatcher := ctrlruntimeclient.MatchingLabels{
		"app.kubernetes.io/name":       appName,
		"app.kubernetes.io/managed-by": "Helm",
		"app.kubernetes.io/instance":   release.Name,
	}
	if err := kubeClient.List(ctx, statefulsetsList, ctrlruntimeclient.InNamespace(CortexNamespace), memcachedMatcher); err != nil {
		return nil, fmt.Errorf("Error querying API for the existing Deployment object, aborting upgrade process.")
	}
	return statefulsetsList, nil
}

func upgradeConsulStatefulsets(
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
	logger.Info("Backing up old consul statefulset…")

	statefulsetsList := &unstructured.UnstructuredList{}
	statefulsetsList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Kind:    "StatefulSetList",
		Version: "v1",
	})

	consulMatcher := ctrlruntimeclient.MatchingLabels{
		"app":                          ConsulReleaseName,
		"app.kubernetes.io/managed-by": "Helm",
		"release":                      release.Name,
	}
	if err := kubeClient.List(ctx, statefulsetsList, ctrlruntimeclient.InNamespace(ConsulNamespace), consulMatcher); err != nil {
		return fmt.Errorf("Error querying API for the existing StatefulSet object, aborting upgrade process.")
	}

	// 2: store the deployment for backup
	backupTS := time.Now().Format("2006-01-02T150405")
	filename := fmt.Sprintf("backup_%s_%s.yaml", ConsulReleaseName, backupTS)
	logger.Infof("Attempting to store the statefulsets in file: %s", filename)
	if err := util.DumpResources(ctx, filename, statefulsetsList.Items); err != nil {
		return fmt.Errorf("failed to back up the statefulsets, it is not removed: %w", err)
	}

	// 3: delete the deployment
	logger.Info("Deleting the statefulsets from the cluster")
	for _, obj := range statefulsetsList.Items {
		if err := kubeClient.Delete(ctx, &obj); err != nil {
			return fmt.Errorf("failed to remove the statefulset: %w\n\nuse backup file to check the changes and restore if needed", err)
		}
	}
	return nil
}

func upgradeCortexMemcachedServices(
	ctx context.Context,
	logger *logrus.Entry,
	kubeClient ctrlruntimeclient.Client,
	chart *helm.Chart,
	release *helm.Release,
) error {
	logger.Infof("%s: %s detected, performing upgrade to %s", release.Name, release.Version.String(), chart.Version.String())
	logger.Info("Removing memcached services")

	// List of memcached service names to delete
	memcachedServices := []string{
		release.Name + "-memcached-blocks",
		release.Name + "-memcached-blocks-index",
		release.Name + "-memcached-blocks-metadata",
	}

	for _, serviceName := range memcachedServices {
		service := &corev1.Service{}
		service.SetName(serviceName)
		service.SetNamespace(CortexNamespace)

		logger.Infof("Deleting service: %s", serviceName)
		if err := kubeClient.Delete(ctx, service); ctrlruntimeclient.IgnoreNotFound(err) != nil {
			return fmt.Errorf("failed to delete service %s: %w", serviceName, err)
		}
	}

	logger.Info("Memcached services deleted successfully, Helm will recreate them with correct configuration")
	return nil
}

func upgradeCortexRuntimeConfigMap(
	ctx context.Context,
	logger *logrus.Entry,
	kubeClient ctrlruntimeclient.Client,
	configMap *corev1.ConfigMap,
) error {
	logger.Info("Upgrading runtime config ConfigMap key from runtime-config.yaml to runtime_config.yaml")

	// Fetch the actual ConfigMap from the cluster
	existingConfigMap := &corev1.ConfigMap{}
	configMapKey := ctrlruntimeclient.ObjectKey{
		Name:      configMap.Name,
		Namespace: configMap.Namespace,
	}

	if err := kubeClient.Get(ctx, configMapKey, existingConfigMap); err != nil {
		if ctrlruntimeclient.IgnoreNotFound(err) == nil {
			logger.Info("Runtime config ConfigMap does not exist yet, no migration needed")
			return nil
		}
		return fmt.Errorf("failed to get runtime config ConfigMap: %w", err)
	}

	// Check if old key exists in the actual cluster ConfigMap
	oldKey := "runtime-config.yaml"
	newKey := mla.RuntimeConfigFileName // This should be "runtime_config.yaml"

	if oldData, hasOldKey := existingConfigMap.Data[oldKey]; hasOldKey {
		logger.Infof("Found old key '%s', migrating to '%s'", oldKey, newKey)

		// Preserve the old data
		if existingConfigMap.Data == nil {
			existingConfigMap.Data = make(map[string]string)
		}

		// Copy data from old key to new key
		existingConfigMap.Data[newKey] = oldData

		// Remove old key
		delete(existingConfigMap.Data, oldKey)

		// Update the ConfigMap
		if err := kubeClient.Update(ctx, existingConfigMap); err != nil {
			return fmt.Errorf("failed to update runtime config ConfigMap: %w", err)
		}

		logger.Infof("Successfully migrated ConfigMap key from '%s' to '%s'", oldKey, newKey)
	} else if _, hasNewKey := existingConfigMap.Data[newKey]; hasNewKey {
		logger.Infof("ConfigMap already has correct key '%s', no migration needed", newKey)
	} else {
		// Neither key exists, add the new key with default value
		logger.Infof("ConfigMap exists but has no runtime config key, adding '%s'", newKey)
		if existingConfigMap.Data == nil {
			existingConfigMap.Data = make(map[string]string)
		}
		existingConfigMap.Data[newKey] = "overrides:\n"

		if err := kubeClient.Update(ctx, existingConfigMap); err != nil {
			return fmt.Errorf("failed to update runtime config ConfigMap: %w", err)
		}
	}

	return nil
}
