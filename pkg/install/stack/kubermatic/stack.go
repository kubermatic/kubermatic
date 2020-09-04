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

package kubermatic

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/util"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/util/yamled"

	storagev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NginxIngressControllerChartName   = "nginx-ingress-controller"
	NginxIngressControllerReleaseName = NginxIngressControllerChartName
	NginxIngressControllerNamespace   = NginxIngressControllerChartName

	CertManagerChartName   = "cert-manager"
	CertManagerReleaseName = CertManagerChartName
	CertManagerNamespace   = CertManagerChartName

	DexChartName   = "oauth"
	DexReleaseName = DexChartName
	DexNamespace   = DexChartName

	KubermaticOperatorChartName   = "kubermatic-operator"
	KubermaticOperatorReleaseName = KubermaticOperatorChartName
	KubermaticOperatorNamespace   = "kubermatic"

	StorageClassName = "kubermatic-fast"
)

type Options struct {
	HelmValues                 *yamled.Document
	KubermaticConfiguration    *operatorv1alpha1.KubermaticConfiguration
	RawKubermaticConfiguration *unstructured.Unstructured
	ForceHelmReleaseUpgrade    bool
}

func Deploy(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt Options) error {
	if err := deployStorageClass(ctx, logger, kubeClient); err != nil {
		return fmt.Errorf("failed to deploy StorageClass: %v", err)
	}

	if err := deployNginxIngressController(ctx, logger, kubeClient, helmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy nginx-ingress-controller: %v", err)
	}

	if err := deployCertManager(ctx, logger, kubeClient, helmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy cert-manager: %v", err)
	}

	if err := deployDex(ctx, logger, kubeClient, helmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Dex: %v", err)
	}

	if err := deployKubermaticOperator(ctx, logger, kubeClient, helmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Kubermatic Operator: %v", err)
	}

	if err := applyKubermaticConfiguration(ctx, logger, kubeClient, opt); err != nil {
		return fmt.Errorf("failed to apply Kubermatic Configuration: %v", err)
	}

	return nil
}

func deployStorageClass(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client) error {
	logger.Infof("💾 Deploying %s StorageClass…", StorageClassName)

	cls := storagev1.StorageClass{}
	key := types.NamespacedName{Name: StorageClassName}

	err := kubeClient.Get(ctx, key, &cls)

	// storage class exists already
	if err == nil {
		logger.Info("✅ StorageClass exists, nothing to do.")
		return nil
	}

	if !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to check for StorageClass %s: %v", StorageClassName, err)
	}

	sc := storageClassForProvider(StorageClassName, "gke")
	if sc == nil {
		return fmt.Errorf("cannot automatically create StorageClass %s for this cloud provider, please create it manually", StorageClassName)
	}

	if err := kubeClient.Create(ctx, sc); err != nil {
		return fmt.Errorf("failed to create StorageClass %s: %v", StorageClassName, err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployNginxIngressController(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt Options) error {
	logger.Info("📦 Deploying nginx-ingress-controller…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart("charts/nginx-ingress-controller")
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %v", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, NginxIngressControllerNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %v", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, NginxIngressControllerNamespace, NginxIngressControllerReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %v", err)
	}

	// do not perform an atomic installation, as this will make Helm wait for the LoadBalancer to
	// get an IP and this can require manual intervention based on the target environment
	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, NginxIngressControllerNamespace, NginxIngressControllerReleaseName, opt.HelmValues, false, opt.ForceHelmReleaseUpgrade, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %v", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployCertManager(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt Options) error {
	logger.Info("📦 Deploying cert-manager…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart("charts/cert-manager")
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %v", err)
	}

	if err := util.DeployCRDs(ctx, kubeClient, sublogger, "charts/cert-manager/crd"); err != nil {
		return fmt.Errorf("failed to deploy CRDs: %v", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, CertManagerNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %v", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, CertManagerNamespace, CertManagerReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %v", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, CertManagerNamespace, CertManagerReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %v", err)
	}

	if err := waitForCertManagerWebhook(ctx, sublogger, kubeClient); err != nil {
		return fmt.Errorf("failed to verify that the webhook is functioning: %v", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployDex(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt Options) error {
	logger.Info("📦 Deploying Dex…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart("charts/oauth")
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %v", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, DexNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %v", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, DexNamespace, DexReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %v", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, DexNamespace, DexReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %v", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployKubermaticOperator(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt Options) error {
	logger.Info("📦 Deploying Kubermatic Operator…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart("charts/kubermatic-operator")
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %v", err)
	}

	if err := util.DeployCRDs(ctx, kubeClient, sublogger, "charts/kubermatic/crd"); err != nil {
		return fmt.Errorf("failed to deploy CRDs: %v", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, KubermaticOperatorNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %v", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, KubermaticOperatorNamespace, KubermaticOperatorReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %v", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, KubermaticOperatorNamespace, KubermaticOperatorReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %v", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func applyKubermaticConfiguration(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt Options) error {
	logger.Info("📝 Applying Kubermatic Configuration…")

	existingConfig := &operatorv1alpha1.KubermaticConfiguration{}
	name := types.NamespacedName{
		Name:      opt.KubermaticConfiguration.Name,
		Namespace: opt.KubermaticConfiguration.Namespace,
	}

	err := kubeClient.Get(ctx, name, existingConfig)
	if err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to check for existing KubermaticConfiguration: %v", err)
	}

	if err == nil {
		// found existing config, need to patch it
		opt.RawKubermaticConfiguration.SetResourceVersion(existingConfig.ResourceVersion)
		opt.RawKubermaticConfiguration.SetAnnotations(existingConfig.Annotations)
		opt.RawKubermaticConfiguration.SetLabels(existingConfig.Labels)
		opt.RawKubermaticConfiguration.SetFinalizers(existingConfig.Finalizers)
		opt.RawKubermaticConfiguration.SetOwnerReferences(existingConfig.OwnerReferences)

		err = kubeClient.Patch(ctx, opt.RawKubermaticConfiguration, ctrlruntimeclient.MergeFrom(existingConfig))
	} else {
		// no config exists yet
		err = kubeClient.Create(ctx, opt.RawKubermaticConfiguration)
	}

	logger.Info("✅ Success.")

	return err
}
