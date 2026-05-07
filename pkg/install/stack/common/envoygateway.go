/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package common

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"time"

	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/util"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/util/crd"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	// GatewayClassName is the name of the GatewayClass resource that is
	// created by the envoy-gateway-controller Helm chart. This must match
	// the value set in the chart's values.yaml under gatewayClass.name.
	GatewayClassName = "kubermatic-envoy-gateway"

	EnvoyGatewayControllerChartName   = "envoy-gateway-controller"
	EnvoyGatewayControllerReleaseName = EnvoyGatewayControllerChartName
	EnvoyGatewayControllerNamespace   = EnvoyGatewayControllerChartName
)

func DeployEnvoyGatewayController(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	if slices.Contains(opt.SkipCharts, EnvoyGatewayControllerChartName) {
		logger.Infof("⭕ Skipping %s deployment.", EnvoyGatewayControllerChartName)
		return nil
	}

	logger.Infof("📦 Deploying %s", EnvoyGatewayControllerChartName)
	sublogger := log.Prefix(logger, "   ")

	if opt.KubermaticConfiguration.Spec.FeatureGates[features.HeadlessInstallation] {
		sublogger.Info("Headless installation requested, skipping.")
		return nil
	}

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, EnvoyGatewayControllerChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := DeployGatewayAPICRDs(ctx, logger, kubeClient, opt); err != nil {
		return err
	}

	err = util.EnsureNamespace(ctx, sublogger, kubeClient, EnvoyGatewayControllerNamespace)
	if err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, EnvoyGatewayControllerNamespace, EnvoyGatewayControllerReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check Helm release: %w", err)
	}

	// do not perform an atomic installation, as this will make Helm wait for the LoadBalancer to
	// get an IP and this can require manual intervention based on the target environment
	sublogger.Info("Deploying Helm chart...")

	err = util.DeployHelmChart(
		ctx,
		sublogger,
		helmClient,
		chart, EnvoyGatewayControllerNamespace, EnvoyGatewayControllerReleaseName, opt.HelmValues,
		false, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release,
	)
	if err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	err = waitForGatewayClass(ctx, sublogger, kubeClient)
	if err != nil {
		return fmt.Errorf("failed to verify that GatewayClass is available: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func DeployGatewayAPICRDs(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt stack.DeployOptions) error {
	if skipGatewayAPICRDs(logger, opt) {
		return nil
	}

	sublogger := log.Prefix(logger, "   ")
	sublogger.Info("Deploying Gateway API Custom Resource Definitions...")

	err := util.DeployCRDs(ctx, kubeClient, sublogger, gatewayAPICRDDirectory(opt), nil, crd.MasterCluster)
	if err != nil {
		return fmt.Errorf("failed to deploy Gateway API CRDs: %w", err)
	}

	return nil
}

// EnsureGatewayAPICRDs creates missing Gateway API CRDs without replacing
// existing CRDs. This is used in BYO Gateway mode where another Gateway
// implementation may own the installed Gateway API CRDs.
func EnsureGatewayAPICRDs(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt stack.DeployOptions) error {
	if skipGatewayAPICRDs(logger, opt) {
		return nil
	}

	sublogger := log.Prefix(logger, "   ")
	sublogger.Info("Ensuring Gateway API Custom Resource Definitions exist...")

	crds, err := crd.LoadFromDirectory(gatewayAPICRDDirectory(opt))
	if err != nil {
		return fmt.Errorf("failed to load Gateway API CRDs: %w", err)
	}

	for _, crdObject := range crds {
		logger := sublogger.WithField("name", crdObject.GetName())
		if crd.SkipCRDOnCluster(crdObject, crd.MasterCluster) {
			logger.Debug("Skipping CRD")
			continue
		}

		existing := crdObject.DeepCopyObject().(ctrlruntimeclient.Object)
		key := ctrlruntimeclient.ObjectKey{Name: crdObject.GetName()}
		if err := kubeClient.Get(ctx, key, existing); err == nil {
			logger.Debug("CRD already exists, leaving it untouched")
			continue
		} else if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to check Gateway API CRD %s: %w", crdObject.GetName(), err)
		}

		logger.Debug("Creating missing CRD…")
		if err := kubeClient.Create(ctx, crdObject); err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create Gateway API CRD %s: %w", crdObject.GetName(), err)
		}
	}

	for _, crdObject := range crds {
		if crd.SkipCRDOnCluster(crdObject, crd.MasterCluster) {
			continue
		}
		if err := util.WaitForReadyCRD(ctx, kubeClient, crdObject.GetName(), 30*time.Second); err != nil {
			return fmt.Errorf("failed to wait for CRD %s to have Established=True condition: %w", crdObject.GetName(), err)
		}
	}

	return nil
}

func skipGatewayAPICRDs(logger *logrus.Entry, opt stack.DeployOptions) bool {
	if slices.Contains(opt.SkipCharts, EnvoyGatewayControllerChartName) {
		logger.Infof("⭕ Skipping Gateway API CRD deployment because %s deployment is skipped.", EnvoyGatewayControllerChartName)
		return true
	}

	if opt.KubermaticConfiguration != nil && opt.KubermaticConfiguration.Spec.FeatureGates[features.HeadlessInstallation] {
		log.Prefix(logger, "   ").Info("Headless installation requested, skipping Gateway API CRD deployment.")
		return true
	}

	return false
}

func gatewayAPICRDDirectory(opt stack.DeployOptions) string {
	return filepath.Join(opt.ChartsDirectory, EnvoyGatewayControllerChartName, "crd")
}

func waitForGatewayClass(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client) error {
	logger.Info("Waiting for GatewayClass to be available...")

	gcName := types.NamespacedName{Name: GatewayClassName}
	gc := gatewayapiv1.GatewayClass{}

	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		if err := kubeClient.Get(ctx, gcName, &gc); err != nil {
			return false, nil
		}

		return meta.IsStatusConditionTrue(
			gc.Status.Conditions,
			string(gatewayapiv1.GatewayClassConditionStatusAccepted),
		), nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for GatewayClass: %w", err)
	}

	logger.Infof("GatewayClass %q is available.", GatewayClassName)

	return nil
}
