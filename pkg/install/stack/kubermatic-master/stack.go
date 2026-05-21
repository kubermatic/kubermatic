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
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"time"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	operatorcommon "k8c.io/kubermatic/v2/pkg/controller/operator/common"
	gatewayutil "k8c.io/kubermatic/v2/pkg/controller/util/gateway"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/stack/common"
	"k8c.io/kubermatic/v2/pkg/install/util"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/util/crd"

	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	CertManagerChartName   = "cert-manager"
	CertManagerReleaseName = CertManagerChartName
	CertManagerNamespace   = CertManagerChartName

	DexChartName   = "dex"
	DexReleaseName = DexChartName
	DexNamespace   = DexChartName

	KubermaticOperatorChartName      = "kubermatic-operator"
	KubermaticOperatorDeploymentName = "kubermatic-operator" // technically defined in our Helm chart
	KubermaticOperatorReleaseName    = KubermaticOperatorChartName
	KubermaticOperatorNamespace      = "kubermatic"

	TelemetryChartName   = "telemetry"
	TelemetryReleaseName = TelemetryChartName
	TelemetryNamespace   = "telemetry-system"

	NodePortProxyService = "nodeport-proxy"

	ingressReadinessPollInterval           = 3 * time.Second
	ingressReadinessTimeout                = 3 * time.Minute
	defaultGatewayAPIReadinessPollInterval = 3 * time.Second
	defaultGatewayAPIReadinessTimeout      = 10 * time.Minute
)

// errOperatorOwnedExternalGateway is returned when waitForGateway observes that
// the configured external Gateway is operator-managed. It is a configuration
// error rather than a readiness timeout and must not be wrapped with the
// "failed to become ready within X" message that surrounds genuine timeouts.
var errOperatorOwnedExternalGateway = errors.New("external Gateway is operator-managed")

type gatewayAPIReadinessPollConfig struct {
	interval time.Duration
	timeout  time.Duration
}

func defaultGatewayAPIReadinessPollConfig() gatewayAPIReadinessPollConfig {
	return gatewayAPIReadinessPollConfig{
		interval: defaultGatewayAPIReadinessPollInterval,
		timeout:  defaultGatewayAPIReadinessTimeout,
	}
}

func (c gatewayAPIReadinessPollConfig) withDefaults() gatewayAPIReadinessPollConfig {
	if c.interval <= 0 {
		c.interval = defaultGatewayAPIReadinessPollInterval
	}
	if c.timeout <= 0 {
		c.timeout = defaultGatewayAPIReadinessTimeout
	}
	return c
}

type MasterStack struct {
	// showDNSHelp is used by the local command to skip a useless DNS probe.
	showDNSHelp bool
}

func NewStack(enableDNSCheck bool) stack.Stack {
	return &MasterStack{
		showDNSHelp: enableDNSCheck,
	}
}

var _ stack.Stack = &MasterStack{}

func (*MasterStack) Name() string {
	return "KKP master stack"
}

func (s *MasterStack) Deploy(ctx context.Context, opt stack.DeployOptions) error {
	if opt.KubermaticConfiguration == nil {
		return errors.New("kubermatic configuration is nil")
	}

	if err := deployStorageClass(ctx, opt.Logger, opt.KubeClient, opt); err != nil {
		return fmt.Errorf("failed to deploy StorageClass: %w", err)
	}

	if opt.MigrateToGatewayAPI {
		if opt.KubermaticConfiguration.Spec.Ingress.Gateway.UsesExternalGateway() {
			if err := common.EnsureGatewayAPICRDs(ctx, opt.Logger, opt.KubeClient, opt); err != nil {
				return fmt.Errorf("failed to ensure Gateway API CRDs: %w", err)
			}
			if err := validateExternalGatewayNotOperatorOwned(ctx, opt.KubeClient, opt.KubermaticConfiguration); err != nil {
				return fmt.Errorf("invalid external Gateway configuration: %w", err)
			}
			opt.Logger.Info("⭕ Skipping envoy-gateway-controller deployment because spec.ingress.gateway.externalGateway is configured.")
		} else {
			err := common.DeployEnvoyGatewayController(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt)
			if err != nil {
				return fmt.Errorf("failed to deploy envoy-gateway-controller: %w", err)
			}
		}
	} else {
		err := common.DeployNginxIngressController(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt)
		if err != nil {
			return fmt.Errorf("failed to deploy nginx-ingress-controller: %w", err)
		}
	}

	if err := deployCertManager(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy cert-manager: %w", err)
	}

	if err := deployDex(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Dex: %w", err)
	}

	if err := s.deployKubermaticOperator(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Kubermatic Operator: %w", err)
	}

	if err := applyKubermaticConfiguration(ctx, opt.Logger, opt.KubeClient, opt); err != nil {
		return fmt.Errorf("failed to apply Kubermatic Configuration: %w", err)
	}

	// once Kubermatic Operator is up and running, it will create the managed Gateway object if needed.
	// so, cleanup old resources depending on the mode.
	if err := l7IngressResourceCleanup(ctx, opt); err != nil {
		return fmt.Errorf("L7 ingress resource cleanup failed: %w", err)
	}

	if err := deployTelemetry(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Telemetry: %w", err)
	}

	if err := deployDefaultPolicyTemplateCatalog(ctx, opt.Logger, opt.KubeClient, opt); err != nil {
		return fmt.Errorf("failed to deploy default Policy Template catalog: %w", err)
	}

	if s.showDNSHelp {
		showDNSSettings(ctx, opt.Logger, opt.KubeClient, opt)
	}

	return nil
}

func deployTelemetry(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying Telemetry…")
	sublogger := log.Prefix(logger, "   ")

	if opt.DisableTelemetry {
		sublogger.Info("Telemetry creation has been disabled in the KubermaticConfiguration, skipping.")
		return nil
	}

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, "telemetry"))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, TelemetryNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, TelemetryNamespace, TelemetryReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, TelemetryNamespace, TelemetryReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployStorageClass(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt stack.DeployOptions) error {
	logger.Infof("💾 Deploying %s StorageClass…", common.StorageClassName)
	sublogger := log.Prefix(logger, "   ")

	// Check if the StorageClass exists already.
	cls := storagev1.StorageClass{}
	err := kubeClient.Get(ctx, types.NamespacedName{Name: common.StorageClassName}, &cls)
	if err == nil {
		logger.Info("✅ StorageClass exists, nothing to do.")
		return nil
	}

	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check for StorageClass %s: %w", common.StorageClassName, err)
	}

	// Class does not yet exist. We can automatically create it based on CSIDrivers if the
	// cluster is already using out-of-tree CSI drivers.
	csiDriverName, cloudProvider, err := common.GetPreferredCSIDriver(ctx, kubeClient)
	if err != nil {
		return fmt.Errorf("failed to determine existing CSIDrivers: %w", err)
	}

	// If no suitable CSIDriver was found, we have to rely on the user to tell us about their provider
	// and then we assume an in-tree (legacy) provider should be used.
	if csiDriverName == "" {
		if opt.StorageClassProvider == "" {
			sublogger.Warnf("The %s StorageClass does not exist yet and no suitable CSIDriver was detected.", common.StorageClassName)
			sublogger.Warn("Depending on your environment, the installer can auto-create a class for you,")
			sublogger.Warn("see the --storageclass CLI flag (should only be used when in-tree CSI driver is still used).")
			sublogger.Warn("Alternatively, please manually create a StorageClass and then re-run the installer to continue.")

			return errors.New("no --storageclass flag given")
		}

		chosenProvider := opt.StorageClassProvider
		if !common.SupportedStorageClassProviders().Has(chosenProvider) {
			return fmt.Errorf("invalid --storageclass flag %q given", chosenProvider)
		}

		cloudProvider = kubermaticv1.ProviderType(chosenProvider)
	} else if opt.StorageClassProvider == string(common.CopyDefaultCloudProvider) {
		// Even if a CSI Driver was found, the user might not want us to blindly create our
		// own StorageClass, but instead copy the default. So if --storageclass=copy-default,
		// this has precedence over the detected cloud provider.
		cloudProvider = common.CopyDefaultCloudProvider
	}

	factory, err := common.StorageClassCreator(cloudProvider)
	if err != nil {
		return fmt.Errorf("invalid StorageClass provider: %w", err)
	}

	storageClass := storagev1.StorageClass{
		Parameters: map[string]string{},
	}
	storageClass.Name = common.StorageClassName

	if err := factory(ctx, sublogger, kubeClient, &storageClass, csiDriverName); err != nil {
		return fmt.Errorf("failed to define StorageClass: %w", err)
	}

	if err := kubeClient.Create(ctx, &storageClass); err != nil {
		return fmt.Errorf("failed to create StorageClass: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func (s *MasterStack) deployKubermaticOperator(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying Kubermatic Operator…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, "kubermatic-operator"))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	sublogger.Info("Deploying Custom Resource Definitions…")
	if err := s.InstallKubermaticCRDs(ctx, kubeClient, sublogger, opt); err != nil {
		return fmt.Errorf("failed to deploy CRDs: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, KubermaticOperatorNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	sublogger.Info("Deploying Helm chart…")
	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, KubermaticOperatorNamespace, KubermaticOperatorReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, KubermaticOperatorNamespace, KubermaticOperatorReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func (*MasterStack) InstallKubermaticCRDs(ctx context.Context, client ctrlruntimeclient.Client, logger logrus.FieldLogger, opt stack.DeployOptions) error {
	crdDirectory := filepath.Join(opt.ChartsDirectory, "kubermatic-operator", "crd")

	// install KKP CRDs
	if err := util.DeployCRDs(ctx, client, logger, filepath.Join(crdDirectory, "k8c.io"), &opt.Versions, crd.MasterCluster); err != nil {
		return err
	}

	// install VPA CRDs
	if err := util.DeployCRDs(ctx, client, logger, filepath.Join(crdDirectory, "k8s.io"), nil, crd.MasterCluster); err != nil {
		return err
	}

	return nil
}

func applyKubermaticConfiguration(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt stack.DeployOptions) error {
	// if no --config was given, no opt.RawKubermaticConfiguration is set and we
	// auto-detected the configuration; in this case we do not want to update
	// the config in the cluster (which would be bad because an auto-detected
	// KubermaticConfiguration is also defaulted and we do not want to persist
	// the defaulted values).
	if opt.RawKubermaticConfiguration == nil {
		return nil
	}

	logger.Info("📝 Applying Kubermatic Configuration…")

	existingConfig := &kubermaticv1.KubermaticConfiguration{}
	name := types.NamespacedName{
		Name:      opt.KubermaticConfiguration.Name,
		Namespace: opt.KubermaticConfiguration.Namespace,
	}

	err := kubeClient.Get(ctx, name, existingConfig)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check for existing KubermaticConfiguration: %w", err)
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

// showDNSSettings attempts to inform the user about required DNS settings
// to be made. If errors happen, only warnings are printed, but the installation
// can still succeed.
func showDNSSettings(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt stack.DeployOptions) {
	logger.Info("📡 Determining DNS settings…")
	sublogger := log.Prefix(logger, "   ")

	if opt.KubermaticConfiguration.Spec.FeatureGates[features.HeadlessInstallation] {
		sublogger.Info("Headless installation requested, skipping.")
		return
	}

	domain := opt.KubermaticConfiguration.Spec.Ingress.Domain
	var hostname, ip string

	if opt.MigrateToGatewayAPI {
		hostname, ip = showGatewayDNSSettings(ctx, logger, kubeClient, opt)
	} else {
		if opt.KubermaticConfiguration.Spec.Ingress.Disable {
			sublogger.Info("Ingress creation has been disabled in the KubermaticConfiguration, skipping.")
			return
		}
		hostname, ip = showIngressDNSSettings(ctx, logger, kubeClient, opt)
	}

	if hostname != "" {
		logger.Infof("  Address via hostname: %s", hostname)
		logger.Info("")
		logger.Infof("Please ensure your DNS settings for %q include the following records:", domain)
		logger.Info("")
		logger.Infof("   %s.    IN  CNAME  %s.", domain, hostname)
		logger.Infof("   *.%s.  IN  CNAME  %s.", domain, hostname)
	} else if ip != "" {
		logger.Infof("  Address via IP      : %s", ip)
		logger.Info("")
		logger.Infof("Please ensure your DNS settings for %q include the following records:", domain)
		logger.Info("")
		logger.Infof("   %s.    IN  A  %s", domain, ip)
		logger.Infof("   *.%s.  IN  A  %s", domain, ip)
	}

	logger.Info("")
}

func showIngressDNSSettings(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt stack.DeployOptions) (hostname, ip string) {
	ingressName := types.NamespacedName{
		Namespace: opt.KubermaticConfiguration.Namespace,
		Name:      operatorcommon.IngressName,
	}

	logger.WithField("ingress", ingressName).Debug("Waiting for Ingress to be ready…")

	var ingresses []networkingv1.IngressLoadBalancerIngress
	err := wait.PollUntilContextTimeout(ctx, ingressReadinessPollInterval, ingressReadinessTimeout, true, func(ctx context.Context) (bool, error) {
		ingress := networkingv1.Ingress{}
		if err := kubeClient.Get(ctx, ingressName, &ingress); err != nil {
			return false, err
		}

		ingresses = ingress.Status.LoadBalancer.Ingress

		return len(ingresses) > 0, nil
	})
	if err != nil {
		logger.Warn("Timed out waiting for the Ingress to become ready.")
		logger.Warn("Please check the Service and, if necessary, reconfigure the")
		logger.Warn("nginx-ingress-controller Helm chart. Re-run the installer to apply")
		logger.Warn("updated configuration afterwards.")
		return "", ""
	}

	logger.Info("The main Ingress is ready.")
	logger.Info("")
	logger.Infof("  Ingress             : %s / %s", ingressName.Namespace, ingressName.Name)

	for _, ingress := range ingresses {
		if ingress.Hostname != "" {
			hostname = ingress.Hostname
			break
		}
		if ingress.IP != "" {
			if ip == "" {
				ip = ingress.IP
			}
			if isPublicIP(ingress.IP) {
				ip = ingress.IP
			}
		}
	}

	return hostname, ip
}

func showGatewayDNSSettings(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt stack.DeployOptions) (hostname, ip string) {
	gatewayName := gatewayObjectKey(opt.KubermaticConfiguration)

	logger.WithField("gateway", gatewayName).Debug("Waiting for Gateway to be ready")

	gtw, err := waitForGateway(ctx, logger, kubeClient, opt.KubermaticConfiguration)
	if err != nil {
		logger.Warn("Timed out waiting for the Gateway to become ready.")
		logger.Warn("Please check the Gateway and EnvoyProxy Service, and if necessary,")
		logger.Warn("reconfigure the envoy-gateway-controller Helm chart. Re-run the installer")
		logger.Warn("to apply updated configuration afterwards.")
		logger.Warn(err.Error())
		return "", ""
	}

	logger.Info("The main Gateway is ready.")
	logger.Info("")
	logger.Infof("  Gateway             : %s / %s", gatewayName.Namespace, gatewayName.Name)

	addresses := gtw.Status.Addresses
	for _, addr := range addresses {
		if addr.Type != nil && *addr.Type == gatewayapiv1.HostnameAddressType {
			hostname = addr.Value
			break
		}
		if addr.Type != nil && *addr.Type == gatewayapiv1.IPAddressType {
			if ip == "" {
				ip = addr.Value
			}
			if isPublicIP(addr.Value) {
				ip = addr.Value
			}
		}
	}

	return hostname, ip
}

func gatewayObjectKey(config *kubermaticv1.KubermaticConfiguration) types.NamespacedName {
	key := types.NamespacedName{
		Namespace: config.Namespace,
		Name:      operatorcommon.GatewayName,
	}

	gatewayConfig := config.Spec.Ingress.Gateway
	if gatewayConfig.UsesExternalGateway() {
		key.Name = gatewayConfig.ExternalGateway.Name
		key.Namespace = gatewayConfig.ExternalGatewayNamespace(config.Namespace)
	}

	return key
}

// cleanupGatewayAPIResources removes the Gateway and HTTPRoute when switching from Gateway API to Ingress.
func cleanupGatewayAPIResources(ctx context.Context, l *logrus.Entry, c ctrlruntimeclient.Client, config *kubermaticv1.KubermaticConfiguration) error {
	l.Info("Removing existing Gateway API resources (if any) since Ingress is enabled for Kubermatic")
	if config == nil {
		return errors.New("kubermatic configuration is nil")
	}

	gtw := &gatewayapiv1.Gateway{}

	err := c.Get(ctx, types.NamespacedName{Namespace: config.Namespace, Name: defaulting.DefaultGatewayName}, gtw)
	if err == nil {
		if isGatewayOwnedByKubermaticConfiguration(gtw) {
			err = c.Delete(ctx, gtw)
			if err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to delete Gateway: %w", err)
			}
		} else {
			l.WithField("gateway", types.NamespacedName{Namespace: config.Namespace, Name: defaulting.DefaultGatewayName}.String()).Debug("Leaving non-operator-owned Gateway untouched during cleanup")
		}
	} else if !apierrors.IsNotFound(err) && !meta.IsNoMatchError(err) {
		return fmt.Errorf("failed to get Gateway: %w", err)
	}

	hr := &gatewayapiv1.HTTPRoute{}

	httpRouteKey := types.NamespacedName{Namespace: config.Namespace, Name: defaulting.DefaultHTTPRouteName}
	err = c.Get(ctx, httpRouteKey, hr)
	if err == nil {
		if isHTTPRouteOwnedByKubermaticConfiguration(hr) {
			err = c.Delete(ctx, hr)
			if err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to delete HTTPRoute: %w", err)
			}
		} else {
			l.WithField("httproute", httpRouteKey.String()).Debug("Leaving non-operator-owned HTTPRoute untouched during cleanup")
		}
	} else if !apierrors.IsNotFound(err) && !meta.IsNoMatchError(err) {
		return fmt.Errorf("failed to get HTTPRoute: %w", err)
	}

	return nil
}

// isGatewayOwnedByKubermaticConfiguration matches Gateways carrying a controller
// owner reference from any KubermaticConfiguration, including stale references
// left over after a KubermaticConfiguration was deleted and recreated. This
// mirrors the operator runtime behavior so the installer does not leave behind
// stale resources for the operator to clean up after the fact.
// KubermaticConfiguration is a cluster-wide singleton, so the usual concern of
// adopting unrelated controllers' children does not apply.
func isGatewayOwnedByKubermaticConfiguration(gw *gatewayapiv1.Gateway) bool {
	return operatorcommon.HasAnyKubermaticConfigurationControllerOwnerReference(gw.OwnerReferences)
}

func isHTTPRouteOwnedByKubermaticConfiguration(route *gatewayapiv1.HTTPRoute) bool {
	return operatorcommon.HasAnyKubermaticConfigurationControllerOwnerReference(route.OwnerReferences)
}

func validateExternalGatewayNotOperatorOwned(ctx context.Context, c ctrlruntimeclient.Client, config *kubermaticv1.KubermaticConfiguration) error {
	if config == nil || config.Spec.Ingress.Gateway == nil || !config.Spec.Ingress.Gateway.UsesExternalGateway() {
		return nil
	}

	gatewayName := gatewayObjectKey(config)
	gw := &gatewayapiv1.Gateway{}
	if err := c.Get(ctx, gatewayName, gw); err != nil {
		if apierrors.IsNotFound(err) || meta.IsNoMatchError(err) {
			return nil
		}

		return fmt.Errorf("failed to get external Gateway %s: %w", gatewayName.String(), err)
	}

	if gw.DeletionTimestamp != nil {
		return nil
	}

	return rejectOperatorOwnedExternalGatewayReference(gatewayName, gw)
}

func rejectOperatorOwnedExternalGatewayReference(gatewayName types.NamespacedName, gw *gatewayapiv1.Gateway) error {
	if !isGatewayOwnedByKubermaticConfiguration(gw) {
		return nil
	}

	return fmt.Errorf("%w: %s cannot be used as spec.ingress.gateway.externalGateway; remove KubermaticConfiguration controller ownerReferences before reusing it as an external Gateway", errOperatorOwnedExternalGateway, gatewayName.String())
}

// cleanupIngress removes the Ingress resource when switching from Ingress mode to Gateway API mode.
func cleanupIngress(ctx context.Context, l *logrus.Entry, c ctrlruntimeclient.Client, opt stack.DeployOptions) error {
	return cleanupIngressWithPollConfig(ctx, l, c, opt, defaultGatewayAPIReadinessPollConfig())
}

func cleanupIngressWithPollConfig(ctx context.Context, l *logrus.Entry, c ctrlruntimeclient.Client, opt stack.DeployOptions, pollConfig gatewayAPIReadinessPollConfig) error {
	pollConfig = pollConfig.withDefaults()

	l.Info("Removing existing Ingress resources (if any) since Gateway API is enabled for Kubermatic")
	config := opt.KubermaticConfiguration
	if config == nil {
		return errors.New("kubermatic configuration is nil")
	}

	kubermaticIngressName := types.NamespacedName{Namespace: config.Namespace, Name: defaulting.DefaultIngressName}
	dexIngressName := types.NamespacedName{Namespace: DexNamespace, Name: DexChartName}

	kubermaticIngressExists, err := ingressExists(ctx, c, kubermaticIngressName)
	if err != nil {
		return err
	}

	gatewayName := gatewayObjectKey(config)
	if kubermaticIngressExists {
		l.Info("Waiting for Gateway to be ready...")
		_, err := waitForGatewayWithPollConfig(ctx, l, c, config, pollConfig)
		if err != nil {
			l.Errorf("failed to wait for Gateway to be ready, err: %v", err)
			return err
		}
	} else {
		l.Debug("No legacy Kubermatic Ingress found, skipping Gateway readiness wait")
	}

	if err := deleteIngressAfterHTTPRouteReadyWithPollConfig(ctx, l, c,
		kubermaticIngressName,
		types.NamespacedName{Namespace: config.Namespace, Name: defaulting.DefaultHTTPRouteName},
		gatewayName,
		pollConfig,
	); err != nil {
		return err
	}

	if slices.Contains(opt.SkipCharts, DexChartName) {
		l.Info("Skipping Dex Ingress cleanup because the Dex chart is skipped")
		return nil
	}

	dexIngressExists, err := ingressExists(ctx, c, dexIngressName)
	if err != nil {
		return err
	}

	// Dex is installed as a separate Helm chart, so its HTTPRoute Gateway target can
	// be overridden via Helm values instead of following only the KubermaticConfiguration.
	dexGatewayName := dexGatewayObjectKey(config, opt)
	if dexIngressExists {
		if dexGatewayName == gatewayName {
			if !kubermaticIngressExists {
				l.Info("Waiting for Dex Gateway to be ready...")
				_, err := waitForGatewayWithPollConfig(ctx, l, c, config, pollConfig)
				if err != nil {
					l.Errorf("failed to wait for Dex Gateway to be ready, err: %v", err)
					return err
				}
			}
		} else {
			l.Info("Waiting for Dex Gateway to be ready...")
			if _, err := waitForGatewayObjectWithPollConfig(ctx, l, c, dexGatewayName, false, false, pollConfig); err != nil {
				l.Errorf("failed to wait for Dex Gateway to be ready, err: %v", err)
				return err
			}
		}
	} else {
		l.Debug("No legacy Dex Ingress found, skipping Dex Gateway readiness wait")
	}

	if err := deleteIngressAfterHTTPRouteReadyWithPollConfig(ctx, l, c,
		dexIngressName,
		types.NamespacedName{Namespace: DexNamespace, Name: DexChartName},
		dexGatewayName,
		pollConfig,
	); err != nil {
		return err
	}

	return nil
}

func dexGatewayObjectKey(config *kubermaticv1.KubermaticConfiguration, opt stack.DeployOptions) types.NamespacedName {
	return common.MasterHTTPRouteGatewayReference(config, opt.HelmValues)
}

func ingressExists(ctx context.Context, kubeClient ctrlruntimeclient.Client, ingressName types.NamespacedName) (bool, error) {
	ingress := &networkingv1.Ingress{}
	if err := kubeClient.Get(ctx, ingressName, ingress); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get Ingress %s: %w", ingressName.String(), err)
	}

	return true, nil
}

func deleteIngressAfterHTTPRouteReady(
	ctx context.Context,
	logger *logrus.Entry,
	kubeClient ctrlruntimeclient.Client,
	ingressName types.NamespacedName,
	routeName types.NamespacedName,
	gatewayName types.NamespacedName,
) error {
	return deleteIngressAfterHTTPRouteReadyWithPollConfig(ctx, logger, kubeClient, ingressName, routeName, gatewayName, defaultGatewayAPIReadinessPollConfig())
}

func deleteIngressAfterHTTPRouteReadyWithPollConfig(
	ctx context.Context,
	logger *logrus.Entry,
	kubeClient ctrlruntimeclient.Client,
	ingressName types.NamespacedName,
	routeName types.NamespacedName,
	gatewayName types.NamespacedName,
	pollConfig gatewayAPIReadinessPollConfig,
) error {
	ingress := &networkingv1.Ingress{}
	if err := kubeClient.Get(ctx, ingressName, ingress); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get Ingress %s: %w", ingressName.String(), err)
	}

	logger.WithField("httproute", routeName.String()).Info("Waiting HTTPRoute to be accepted by Gateway...")
	if _, err := waitForHTTPRoute(ctx, logger, kubeClient, routeName, gatewayName, pollConfig); err != nil {
		return fmt.Errorf("failed to wait for HTTPRoute %s to be accepted by Gateway %s; verify the Gateway listener allowedRoutes accepts namespace %q: %w", routeName.String(), gatewayName.String(), routeName.Namespace, err)
	}

	if err := kubeClient.Delete(ctx, ingress); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete Ingress %s: %w", ingressName.String(), err)
	}

	return nil
}

// waitForHTTPRoute waits for an HTTPRoute to reference and be accepted by the active Gateway.
func waitForHTTPRoute(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, routeName, gatewayName types.NamespacedName, pollConfig gatewayAPIReadinessPollConfig) (*gatewayapiv1.HTTPRoute, error) {
	pollConfig = pollConfig.withDefaults()
	l := logger.WithField("httproute", routeName.String()).WithField("gateway", gatewayName.String())
	route := gatewayapiv1.HTTPRoute{}

	err := wait.PollUntilContextTimeout(ctx, pollConfig.interval, pollConfig.timeout, true, func(ctx context.Context) (bool, error) {
		if err := kubeClient.Get(ctx, routeName, &route); err != nil {
			l.Debugf("failed to get HTTPRoute, err: %v", err)
			return false, nil
		}

		if !gatewayutil.HTTPRouteReferencesGateway(&route, gatewayName) {
			l.Debug("HTTPRoute does not reference the active Gateway yet")
			return false, nil
		}

		if !gatewayutil.HTTPRouteAcceptedByGateway(&route, gatewayName) {
			l.Debug("HTTPRoute has not been accepted by the active Gateway yet")
			return false, nil
		}

		l.Info("HTTPRoute is accepted by the active Gateway")
		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("HTTPRoute %s failed to be accepted by Gateway %s within %s: %w", routeName.String(), gatewayName.String(), pollConfig.timeout, err)
	}

	return &route, nil
}

// waitForGateway waits for the Gateway to be Programmed. Operator-managed
// Gateways must also have an address; external Gateways may omit addresses, so
// BYO mode relies on Programmed plus HTTPRoute acceptance before cleanup.
// Per-listener conditions are not checked because HTTPS listeners depend on
// TLS certificates that require DNS to be configured first, which only happens
// after the installer finishes.
func waitForGateway(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, config *kubermaticv1.KubermaticConfiguration) (*gatewayapiv1.Gateway, error) {
	return waitForGatewayWithPollConfig(ctx, logger, kubeClient, config, defaultGatewayAPIReadinessPollConfig())
}

func waitForGatewayWithPollConfig(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, config *kubermaticv1.KubermaticConfiguration, pollConfig gatewayAPIReadinessPollConfig) (*gatewayapiv1.Gateway, error) {
	if config == nil {
		return nil, fmt.Errorf("Invalid KubermaticConfiguration provided")
	}

	gatewayName := gatewayObjectKey(config)
	requireAddress := !config.Spec.Ingress.Gateway.UsesExternalGateway()
	rejectOperatorOwnedExternalGateway := config.Spec.Ingress.Gateway.UsesExternalGateway()

	return waitForGatewayObjectWithPollConfig(ctx, logger, kubeClient, gatewayName, requireAddress, rejectOperatorOwnedExternalGateway, pollConfig)
}

func waitForGatewayObjectWithPollConfig(
	ctx context.Context,
	logger *logrus.Entry,
	kubeClient ctrlruntimeclient.Client,
	gatewayName types.NamespacedName,
	requireAddress bool,
	rejectOperatorOwnedExternalGateway bool,
	pollConfig gatewayAPIReadinessPollConfig,
) (*gatewayapiv1.Gateway, error) {
	pollConfig = pollConfig.withDefaults()
	l := logger.WithField("gateway", gatewayName.String())
	gw := gatewayapiv1.Gateway{}

	err := wait.PollUntilContextTimeout(ctx, pollConfig.interval, pollConfig.timeout, true, func(ctx context.Context) (bool, error) {
		if err := kubeClient.Get(ctx, gatewayName, &gw); err != nil {
			l.Debugf("failed to get Gateway, err: %v", err)
			return false, nil
		}

		if gw.DeletionTimestamp != nil {
			l.Debug("Gateway is being deleted")
			return false, nil
		}

		if rejectOperatorOwnedExternalGateway {
			if err := rejectOperatorOwnedExternalGatewayReference(gatewayName, &gw); err != nil {
				return false, err
			}
		}

		if requireAddress && len(gw.Status.Addresses) == 0 {
			l.Debug("Gateway does not have addresses assigned yet")
			return false, nil
		}

		programmed := meta.IsStatusConditionTrue(
			gw.Status.Conditions,
			string(gatewayapiv1.GatewayConditionProgrammed),
		)
		if !programmed {
			condition := meta.FindStatusCondition(gw.Status.Conditions, string(gatewayapiv1.GatewayConditionProgrammed))
			reason := "unknown"
			message := "no condition"
			if condition != nil {
				reason = condition.Reason
				message = condition.Message
			}

			l.Debugf("Gateway not yet programmed: %s - %s", reason, message)
			return false, nil
		}

		l.Infof("Gateway is ready with %d address(es) and %d listener(s)",
			len(gw.Status.Addresses),
			len(gw.Status.Listeners),
		)

		return true, nil
	})
	if err != nil {
		if errors.Is(err, errOperatorOwnedExternalGateway) {
			return nil, err
		}
		return nil, fmt.Errorf("Gateway %s failed to become ready within %s: %w", gatewayName.String(), pollConfig.timeout, err)
	}

	return &gw, nil
}

func l7IngressResourceCleanup(ctx context.Context, opt stack.DeployOptions) error {
	if opt.SkipIngressCleanup {
		opt.Logger.Info("Skipping L7 ingress resource cleanup as requested")
		return nil
	}

	if opt.MigrateToGatewayAPI {
		err := cleanupIngress(ctx, opt.Logger, opt.KubeClient, opt)
		if err != nil {
			return fmt.Errorf("cleanup Ingress resources failed: %w", err)
		}

		opt.Logger.Info("Successfully cleaned up Ingress resources")

		return nil
	}

	err := cleanupGatewayAPIResources(ctx, opt.Logger, opt.KubeClient, opt.KubermaticConfiguration)
	if err != nil {
		return fmt.Errorf("cleanup Gateway API resources failed: %w", err)
	}

	opt.Logger.Info("Successfully cleaned up Gateway API resources")

	return nil
}
