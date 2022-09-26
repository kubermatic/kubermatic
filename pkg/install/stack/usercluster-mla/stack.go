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
	"time"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	kubermaticmaster "k8c.io/kubermatic/v2/pkg/install/stack/kubermatic-master"
	"k8c.io/kubermatic/v2/pkg/install/util"
	"k8c.io/kubermatic/v2/pkg/log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
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

	if err := deployCortex(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Cortex: %w", err)
	}

	if err := deployGrafana(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Grafana: %w", err)
	}

	if err := deployLoki(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Loki: %w", err)
	}

	if err := deployMinio(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Minio: %w", err)
	}

	if err := deployMinioLifecycleMgr(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Minio Bucket Lifecycle Manager: %w", err)
	}

	showDNSSettings(ctx, opt.Logger, opt.KubeClient, opt)

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

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, MlaSecretsNamespace, MlaSecretsReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
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

// showDNSSettings attempts to inform the user about required DNS settings
// to be made. If errors happen, only warnings are printed, but the installation
// can still succeed.

// TODO: rewrite to usercluster MLA instead of kubermatic master
func showDNSSettings(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Reader, opt stack.DeployOptions) {
	logger.Info("📡 Determining DNS settings…")

	logger = log.Prefix(logger, "   ")
	ns := kubermaticmaster.KubermaticOperatorNamespace

	// step 1: to ensure that a Seed has been created on the master, we check
	// if the Seed's copy has already been created in this cluster (by the KKP
	// seed-sync controller). Besides checking its existence, we also need to
	// know the Seed's name to construct the FQDN for the DNS settings.
	seedList := kubermaticv1.SeedList{}
	err := kubeClient.List(ctx, &seedList, &ctrlruntimeclient.ListOptions{Namespace: ns})
	if err != nil || len(seedList.Items) == 0 {
		logger.Warnf("No Seed resource was found in the %s namespace.", ns)
		logger.Warn("Make sure to create the Seed resource on the *master* cluster, from where KKP")
		logger.Warn("will automatically synchronize it to the seed cluster. Once this is done, re-run")
		logger.Warn("the installer to determine the DNS settings automatically.")
		logger.Warn("If you already created the Seed resource and its copy is not present on the")
		logger.Warn("seed cluster, check the KKP Master Controller's logs.")
		return
	}

	// step 2: find the nodeport-proxy Service
	svcName := types.NamespacedName{
		Namespace: ns,
		Name:      kubermaticmaster.NodePortProxyService,
	}

	logger.Debugf("Waiting for %q to be ready…", svcName)

	var ingresses []corev1.LoadBalancerIngress
	err = wait.PollImmediate(5*time.Second, 3*time.Minute, func() (bool, error) {
		svc := corev1.Service{}
		if err := kubeClient.Get(ctx, svcName, &svc); err != nil {
			return false, err
		}

		ingresses = svc.Status.LoadBalancer.Ingress

		return len(ingresses) > 0, nil
	})
	if err != nil {
		logger.Warnf("Timed out waiting for the LoadBalancer service %q to become ready.", svcName)
		logger.Warn("Note that the LoadBalancer is created by the KKP Operator after the Seed")
		logger.Warn("resource is created on the *master* cluster. If the Seed is installed and")
		logger.Warn("no LoadBalancer is provisioned, check the KKP Operator's logs.")
		return
	}

	logger.Info("The main LoadBalancer is ready.")
	logger.Info("")
	logger.Infof("  Service             : %s / %s", svcName.Namespace, svcName.Name)

	seed := seedList.Items[0]
	domain := opt.KubermaticConfiguration.Spec.Ingress.Domain

	if hostname := ingresses[0].Hostname; hostname != "" {
		logger.Infof("  Ingress via hostname: %s", hostname)
		logger.Info("")
		logger.Infof("Please ensure your DNS settings for %q includes the following record:", domain)
		logger.Info("")
		logger.Infof("   *.%s.%s.  IN  CNAME  %s.", seed.Name, domain, hostname)
	} else if ip := ingresses[0].IP; ip != "" {
		logger.Infof("  Ingress via IP      : %s", ip)
		logger.Info("")
		logger.Infof("Please ensure your DNS settings for %q includes the following record:", domain)
		logger.Info("")
		logger.Infof("   *.%s.%s.  IN  A  %s", seed.Name, domain, ip)
	}

	logger.Info("")
}
