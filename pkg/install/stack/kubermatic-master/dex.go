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

package kubermaticmaster

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/sirupsen/logrus"

	operatorcommon "k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	stackcommon "k8c.io/kubermatic/v2/pkg/install/stack/common"
	"k8c.io/kubermatic/v2/pkg/install/util"
	"k8c.io/kubermatic/v2/pkg/log"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func deployDex(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	if slices.Contains(opt.SkipCharts, "dex") {
		logger.Info("⭕ Skipping Dex deployment.")
		return nil
	}

	chartName := DexChartName
	releaseName := DexReleaseName
	namespace := DexNamespace

	logger.Info("📦 Deploying Dex…")
	sublogger := log.Prefix(logger, "   ")

	if opt.KubermaticConfiguration.Spec.FeatureGates[features.HeadlessInstallation] {
		sublogger.Info("Headless installation requested, skipping.")
		return nil
	}

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, chartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, namespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// label the `dex` namespace to allow HTTPRoute attachment to the kubermatic Gateway.
	if err := util.EnsureNamespaceLabel(ctx, kubeClient, namespace, operatorcommon.GatewayAccessLabelKey, "true"); err != nil {
		return fmt.Errorf("failed to label namespace for Gateway access: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, namespace, releaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	// ValidateConfiguration normally defaulted this already; keep this defensive
	// for callers that invoke Deploy directly, such as the local installer path.
	stackcommon.DefaultMasterHTTPRouteGatewayValues(opt.KubermaticConfiguration, opt.HelmValues, sublogger)

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, namespace, releaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}
