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

package kubermaticseed

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/stack/common"
	kubermaticmaster "k8c.io/kubermatic/v2/pkg/install/stack/kubermatic-master"
	"k8c.io/kubermatic/v2/pkg/install/util"
	"k8c.io/kubermatic/v2/pkg/log"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MinioChartName   = "minio"
	MinioReleaseName = MinioChartName
	MinioNamespace   = MinioChartName

	S3ExporterChartName   = "s3-exporter"
	S3ExporterReleaseName = S3ExporterChartName
	S3ExporterNamespace   = "kube-system"
)

type SeedStack struct{}

func NewStack() stack.Stack {
	return &SeedStack{}
}

var _ stack.Stack = &SeedStack{}

func (*SeedStack) Name() string {
	return "KKP seed stack"
}

func (s *SeedStack) InstallKubermaticCRDs(ctx context.Context, client ctrlruntimeclient.Client, logger logrus.FieldLogger, opt stack.DeployOptions) error {
	// CRDs on seed clusters are currently identical to the master, even though
	// we do not use all CRs on all clusters
	masterStack := kubermaticmaster.MasterStack{}

	return masterStack.InstallKubermaticCRDs(ctx, client, logger, opt)
}

func (s *SeedStack) Deploy(ctx context.Context, opt stack.DeployOptions) error {
	if err := deployStorageClass(ctx, opt.Logger, opt.KubeClient, opt); err != nil {
		return fmt.Errorf("failed to deploy StorageClass: %w", err)
	}

	if err := deployMinio(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Minio: %w", err)
	}

	if err := deployS3Exporter(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy S3 Exporter: %w", err)
	}

	if err := s.deployKubermatic(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Kubermatic: %w", err)
	}

	if err := migrateUserClustersData(ctx, opt.Logger, opt.KubeClient, opt); err != nil {
		return fmt.Errorf("failed to migrate data in user-clusters: %w", err)
	}

	showDNSSettings(ctx, opt.Logger, opt.KubeClient, opt)

	return nil
}

func (s *SeedStack) deployKubermatic(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying KKP Dependencies…")

	// The KKP Operator will not reconcile the seed cluster if the "kubermatic"
	// namespace doesn't exist yet. This is meant as a "safety mechanism", so we
	// must ensure the namespace exists.
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: kubermaticmaster.KubermaticOperatorNamespace,
		},
	}

	if err := kubeClient.Create(ctx, ns); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create Namespace %s: %w", ns.Name, err)
	}

	if err := s.InstallKubermaticCRDs(ctx, kubeClient, log.Prefix(logger, "   "), opt); err != nil {
		return fmt.Errorf("failed to deploy CRDs: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployStorageClass(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt stack.DeployOptions) error {
	logger.Infof("💾 Deploying %s StorageClass…", common.StorageClassName)
	sublogger := log.Prefix(logger, "   ")

	chosenProvider := opt.StorageClassProvider
	if chosenProvider != "" && !common.SupportedStorageClassProviders().Has(chosenProvider) {
		return fmt.Errorf("invalid provider %q given", chosenProvider)
	}

	cls := storagev1.StorageClass{}
	key := types.NamespacedName{Name: common.StorageClassName}

	err := kubeClient.Get(ctx, key, &cls)

	// storage class exists already
	if err == nil {
		logger.Info("✅ StorageClass exists, nothing to do.")
		return nil
	}

	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check for StorageClass %s: %w", common.StorageClassName, err)
	}

	if opt.StorageClassProvider == "" {
		sublogger.Warnf("The %s StorageClass does not exist yet. Depending on your environment,", common.StorageClassName)
		sublogger.Warn("the installer can auto-create a class for you, see the --storageclass CLI flag.")
		sublogger.Warn("Alternatively, please manually create a StorageClass and then re-run the installer to continue.")

		return fmt.Errorf("no %s StorageClass found", common.StorageClassName)
	}

	factory, err := common.StorageClassCreator(opt.StorageClassProvider)
	if err != nil {
		return fmt.Errorf("invalid StorageClass provider: %w", err)
	}

	sc, err := factory(ctx, sublogger, kubeClient, common.StorageClassName)
	if err != nil {
		return fmt.Errorf("failed to define StorageClass: %w", err)
	}

	if err := kubeClient.Create(ctx, &sc); err != nil {
		return fmt.Errorf("failed to create StorageClass: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployMinio(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying Minio…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, MinioChartName))
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

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, MinioNamespace, MinioReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployS3Exporter(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying S3 Exporter…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, S3ExporterChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, S3ExporterNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, S3ExporterNamespace, S3ExporterReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, S3ExporterNamespace, S3ExporterReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

// showDNSSettings attempts to inform the user about required DNS settings
// to be made. If errors happen, only warnings are printed, but the installation
// can still succeed.
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
