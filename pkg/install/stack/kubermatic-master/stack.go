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
	// so, verify Gateway/HTTPRoute readiness and then optionally clean up the legacy nginx-ingress release.
	if err := waitForGatewayAndHTTPRoutesReady(ctx, opt.Logger, opt.KubeClient, opt); err != nil {
		return fmt.Errorf("failed to verify Gateway API readiness: %w", err)
	}

	// Legacy Ingress objects are always removed once the matching HTTPRoutes are accepted.
	if err := cleanupLegacyIngresses(ctx, opt.Logger, opt.KubeClient, opt); err != nil {
		return fmt.Errorf("failed to clean up legacy Ingress resources: %w", err)
	}

	// The nginx-ingress-controller release (Deployment + LoadBalancer Service) is removed
	// only when --clean-nginx-lb is set; otherwise a warning is logged so the operator can
	// remove it manually.
	if err := cleanupLegacyNginxIngress(ctx, opt.Logger, opt.KubeClient, opt.HelmClient, opt); err != nil {
		return fmt.Errorf("failed to handle legacy nginx-ingress-controller: %w", err)
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
	hostname, ip := showGatewayDNSSettings(ctx, logger, kubeClient, opt)

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

func showGatewayDNSSettings(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt stack.DeployOptions) (hostname, ip string) {
	gatewayName := gatewayObjectKey(opt.KubermaticConfiguration)
	gatewayLogger := logger.WithField("gateway", gatewayName.String())

	// Readiness is already verified by waitForGatewayAndHTTPRoutesReady earlier in Deploy;
	// this just reads the Gateway to extract its address for the DNS hint.
	gtw := &gatewayapiv1.Gateway{}
	if err := kubeClient.Get(ctx, gatewayName, gtw); err != nil {
		gatewayLogger.Warn("Failed to read the Gateway after the readiness check.")
		gatewayLogger.Warnf("Please run `kubectl -n %s get gateway %s` to inspect the Gateway and confirm its address;", gatewayName.Namespace, gatewayName.Name)
		gatewayLogger.Warn("then configure the DNS records below manually.")
		gatewayLogger.Warn(err.Error())
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

func dexGatewayObjectKey(config *kubermaticv1.KubermaticConfiguration, opt stack.DeployOptions) types.NamespacedName {
	return common.MasterHTTPRouteGatewayReference(config, opt.HelmValues)
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
	reportedMissingGateway := false

	err := wait.PollUntilContextTimeout(ctx, pollConfig.interval, pollConfig.timeout, true, func(ctx context.Context) (bool, error) {
		if err := kubeClient.Get(ctx, gatewayName, &gw); err != nil {
			if apierrors.IsNotFound(err) && !reportedMissingGateway {
				l.Info("Gateway does not exist yet, waiting...")
				reportedMissingGateway = true
			} else {
				l.Debugf("failed to get Gateway, err: %v", err)
			}
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

// waitForGatewayAndHTTPRoutesReady waits unconditionally for the active Gateway to
// be Programmed and for the kubermatic and Dex HTTPRoutes to be Accepted by their
// respective Gateways. The wait is skipped entirely under HeadlessInstallation.
// The Dex stage is skipped when the Dex chart is in opt.SkipCharts. IAP (installed
// by the seed-mla stack into the monitoring namespace) is not the master installer's
// responsibility and is handled by that stack instead.
func waitForGatewayAndHTTPRoutesReady(ctx context.Context, l *logrus.Entry, c ctrlruntimeclient.Client, opt stack.DeployOptions) error {
	return waitForGatewayAndHTTPRoutesReadyWithPollConfig(ctx, l, c, opt, defaultGatewayAPIReadinessPollConfig())
}

func waitForGatewayAndHTTPRoutesReadyWithPollConfig(ctx context.Context, logger *logrus.Entry, c ctrlruntimeclient.Client, opt stack.DeployOptions, pollConfig gatewayAPIReadinessPollConfig) error {
	pollConfig = pollConfig.withDefaults()

	config := opt.KubermaticConfiguration
	if config == nil {
		return errors.New("kubermatic configuration is nil")
	}
	if config.Spec.FeatureGates[features.HeadlessInstallation] {
		logger.Debug("Headless installation requested, skipping Gateway/HTTPRoute readiness checks")
		return nil
	}

	logger.Info("🔁 Verifying Gateway API readiness…")
	l := log.Prefix(logger, "   ")

	gatewayName := gatewayObjectKey(config)
	gatewayLogger := l.WithField("gateway", gatewayName.String())
	isExternalGateway := config.Spec.Ingress.Gateway.UsesExternalGateway()
	if isExternalGateway {
		gatewayLogger.Info("Waiting for external Gateway to become ready...")
	} else {
		gatewayLogger.Info("Waiting for Gateway to become ready...")
	}
	if _, err := waitForGatewayWithPollConfig(ctx, l, c, config, pollConfig); err != nil {
		gatewayLogger.Warn("Timed out waiting for the Gateway to become ready.")
		if isExternalGateway {
			gatewayLogger.Warn("Please check the external Gateway and its controller, and ensure")
			gatewayLogger.Warn("the Gateway listener accepts HTTPRoutes from the kubermatic namespace.")
			gatewayLogger.Warn("Re-run the installer to retry after fixing the configuration.")
		} else {
			gatewayLogger.Warn("Please check the Gateway and EnvoyProxy Service, and if necessary,")
			gatewayLogger.Warn("reconfigure the envoy-gateway-controller Helm chart. Re-run the installer")
			gatewayLogger.Warn("to apply updated configuration afterwards.")
		}
		return fmt.Errorf("failed to wait for Gateway %s to become ready: %w", gatewayName.String(), err)
	}

	kubermaticRouteName := types.NamespacedName{Namespace: config.Namespace, Name: defaulting.DefaultHTTPRouteName}
	if err := waitForHTTPRouteAcceptedByGateway(ctx, l, c, kubermaticRouteName, gatewayName, pollConfig); err != nil {
		l.WithField("httproute", kubermaticRouteName.String()).WithField("gateway", gatewayName.String()).Warn("Kubermatic HTTPRoute did not become accepted by the Gateway.")
		return err
	}

	if !slices.Contains(opt.SkipCharts, DexChartName) {
		dexGatewayName := dexGatewayObjectKey(config, opt)
		if dexGatewayName != gatewayName {
			dexGatewayLogger := l.WithField("gateway", dexGatewayName.String())
			dexGatewayLogger.Info("Waiting for Dex Gateway to become ready...")
			if _, err := waitForGatewayObjectWithPollConfig(ctx, l, c, dexGatewayName, false, false, pollConfig); err != nil {
				dexGatewayLogger.Warn("Timed out waiting for the Dex Gateway to become ready.")
				return fmt.Errorf("failed to wait for Dex Gateway %s to become ready: %w", dexGatewayName.String(), err)
			}
		}
		dexRouteName := types.NamespacedName{Namespace: DexNamespace, Name: DexChartName}
		if err := waitForHTTPRouteAcceptedByGateway(ctx, l, c, dexRouteName, dexGatewayName, pollConfig); err != nil {
			l.WithField("httproute", dexRouteName.String()).WithField("gateway", dexGatewayName.String()).Warn("Dex HTTPRoute did not become accepted by the Gateway.")
			return err
		}
	} else {
		l.Info("Skipping Dex HTTPRoute readiness check because the Dex chart is skipped")
	}

	logger.Info("✅ Gateway and HTTPRoutes ready.")
	return nil
}

// cleanupLegacyIngresses removes Ingress objects left over from the 2.30 nginx-ingress
// installation path. Best-effort: missing objects are ignored. Assumes Gateway/HTTPRoute
// readiness has already been verified by waitForGatewayAndHTTPRoutesReady.
func cleanupLegacyIngresses(ctx context.Context, l *logrus.Entry, c ctrlruntimeclient.Client, opt stack.DeployOptions) error {
	config := opt.KubermaticConfiguration
	if config == nil {
		return errors.New("kubermatic configuration is nil")
	}
	if config.Spec.FeatureGates[features.HeadlessInstallation] {
		l.Debug("Headless installation requested, skipping legacy Ingress cleanup")
		return nil
	}
	if opt.SkipIngressCleanup {
		l.Info("⏭️  --skip-ingress-cleanup is set: leaving legacy Ingress objects in place so nginx-ingress can keep serving traffic. Re-run without the flag once DNS has been flipped to the new Envoy Gateway.")
		return nil
	}
	l.Info("🧹 Removing legacy Ingress resources left over from the nginx-ingress installation path")

	kubermaticIngressName := types.NamespacedName{Namespace: config.Namespace, Name: defaulting.DefaultIngressName}
	if err := deleteIngressByName(ctx, l, c, kubermaticIngressName); err != nil {
		return err
	}

	if !slices.Contains(opt.SkipCharts, DexChartName) {
		dexIngressName := types.NamespacedName{Namespace: DexNamespace, Name: DexChartName}
		if err := deleteIngressByName(ctx, l, c, dexIngressName); err != nil {
			return err
		}
	}

	return nil
}

func deleteIngressByName(ctx context.Context, l *logrus.Entry, c ctrlruntimeclient.Client, name types.NamespacedName) error {
	ing := &networkingv1.Ingress{}
	if err := c.Get(ctx, name, ing); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get Ingress %s: %w", name.String(), err)
	}
	if err := c.Delete(ctx, ing); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete Ingress %s: %w", name.String(), err)
	}
	l.WithField("ingress", name.String()).Info("Deleted legacy Ingress")
	return nil
}

func waitForHTTPRouteAcceptedByGateway(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, routeName, gatewayName types.NamespacedName, pollConfig gatewayAPIReadinessPollConfig) error {
	logger.WithField("httproute", routeName.String()).WithField("gateway", gatewayName.String()).Info("Waiting for HTTPRoute to be accepted by Gateway...")
	if _, err := waitForHTTPRoute(ctx, logger, kubeClient, routeName, gatewayName, pollConfig); err != nil {
		return fmt.Errorf("failed to wait for HTTPRoute %s to be accepted by Gateway %s; verify the Gateway listener hostname and allowedRoutes accept namespace %q: %w", routeName.String(), gatewayName.String(), routeName.Namespace, err)
	}
	return nil
}

func waitForHTTPRoute(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, routeName, gatewayName types.NamespacedName, pollConfig gatewayAPIReadinessPollConfig) (*gatewayapiv1.HTTPRoute, error) {
	pollConfig = pollConfig.withDefaults()
	l := logger.WithField("httproute", routeName.String()).WithField("gateway", gatewayName.String())
	route := gatewayapiv1.HTTPRoute{}
	routeFound := false

	err := wait.PollUntilContextTimeout(ctx, pollConfig.interval, pollConfig.timeout, true, func(ctx context.Context) (bool, error) {
		if err := kubeClient.Get(ctx, routeName, &route); err != nil {
			routeFound = false
			l.Debugf("failed to get HTTPRoute, err: %v", err)
			return false, nil
		}
		routeFound = true

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
		status := "HTTPRoute does not exist"
		if routeFound {
			status = httpRouteGatewayAcceptanceStatus(&route, gatewayName)
		}
		return nil, fmt.Errorf("HTTPRoute %s failed to be accepted by Gateway %s within %s: %s: %w", routeName.String(), gatewayName.String(), pollConfig.timeout, status, err)
	}

	return &route, nil
}

func httpRouteGatewayAcceptanceStatus(route *gatewayapiv1.HTTPRoute, gatewayName types.NamespacedName) string {
	if !gatewayutil.HTTPRouteReferencesGateway(route, gatewayName) {
		return "HTTPRoute does not reference Gateway"
	}

	for _, parentStatus := range route.Status.Parents {
		if !gatewayutil.ParentReferenceMatchesGateway(route.Namespace, parentStatus.ParentRef, gatewayName) {
			continue
		}

		accepted := meta.FindStatusCondition(parentStatus.Conditions, string(gatewayapiv1.RouteConditionAccepted))
		if accepted == nil {
			return "Accepted condition is missing for Gateway parent"
		}

		return fmt.Sprintf("Accepted=%s reason=%q message=%q observedGeneration=%d routeGeneration=%d", accepted.Status, accepted.Reason, accepted.Message, accepted.ObservedGeneration, route.Generation)
	}

	return "HTTPRoute status has no parent entry for Gateway"
}

// cleanupLegacyNginxIngress uninstalls the legacy nginx-ingress-controller Helm release
// and deletes its namespace when --clean-nginx-lb is set. Otherwise, when the namespace
// still exists, it logs a warning so the operator knows to remove it manually.
func cleanupLegacyNginxIngress(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	if opt.KubermaticConfiguration != nil && opt.KubermaticConfiguration.Spec.FeatureGates[features.HeadlessInstallation] {
		logger.Debug("Headless installation requested, skipping legacy nginx-ingress cleanup")
		return nil
	}
	exists, err := common.NginxIngressNamespaceExists(ctx, kubeClient)
	if err != nil {
		return err
	}
	if !exists {
		logger.Debugf("%s namespace not present, nothing to clean up.", common.NginxIngressControllerNamespace)
		return nil
	}

	if !opt.CleanNginxLB {
		logger.Warnf("⚠️  Legacy %s release is still installed.", common.NginxIngressControllerChartName)
		if !opt.SkipIngressCleanup {
			logger.Warn("The legacy Ingress objects have just been removed, so nginx-ingress-controller is currently serving zero rules and will 404 any request that still lands on its LoadBalancer.")
			logger.Warn("Confirm that DNS for the Kubermatic domain resolves to the new Envoy Gateway address printed at the end of this run (`kubectl -n envoy-gateway-controller get svc`) before assuming traffic is healthy.")
		}
		logger.Warnf("Once DNS is on the new Gateway, re-run the installer with --clean-nginx-lb to remove the %s release, or uninstall it manually.", common.NginxIngressControllerChartName)
		return nil
	}

	logger.Info("🧹 Cleaning up legacy nginx-ingress controller…")
	sublogger := log.Prefix(logger, "   ")
	if err := common.UninstallNginxIngressController(ctx, sublogger, kubeClient, helmClient); err != nil {
		return err
	}
	logger.Info("✅ Legacy nginx-ingress controller cleanup complete.")
	return nil
}
