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

package defaulting

import (
	"errors"
	"fmt"
	"strings"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/distribution/distribution/v3/reference"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/api/v3/pkg/semver"
	"k8c.io/kubermatic/v3/pkg/features"
	"k8c.io/kubermatic/v3/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"
)

// All Default* constants live here, except for those used by other reconciling
// code, for which those constants live in pkg/resources.

const (
	DefaultPProfEndpoint                       = ":6600"
	DefaultEtcdVolumeSize                      = "5Gi"
	DefaultAuthClientID                        = "kubermatic"
	DefaultIngressClass                        = "nginx"
	DefaultCABundleConfigMapName               = "ca-bundle"
	DefaultAPIReplicas                         = 2
	DefaultUIReplicas                          = 2
	DefaultSeedControllerManagerReplicas       = 1
	DefaultWebhookReplicas                     = 1
	DefaultKubernetesApiserverReplicas         = 2
	DefaultKubernetesControllerManagerReplicas = 1
	DefaultKubernetesSchedulerReplicas         = 1
	DefaultExposeStrategy                      = kubermaticv1.ExposeStrategyNodePort
	DefaultEnvoyDockerRepository               = "docker.io/envoyproxy/envoy-alpine"
	DefaultUserClusterScrapeAnnotationPrefix   = "monitoring.kubermatic.io"
	DefaultMaximumParallelReconciles           = 10
	DefaultS3Endpoint                          = "s3.amazonaws.com"
	DefaultSSHPort                             = 22
	DefaultKubeletPort                         = 10250

	// DefaultCloudProviderReconciliationInterval is the time in between deep cloud provider reconciliations
	// in case the user did not configure a special interval for the given datacenter.
	DefaultCloudProviderReconciliationInterval = 6 * time.Hour

	// DefaultNoProxy is a set of domains/networks that should never be
	// routed through a proxy. All user-supplied values are appended to
	// this constant.
	DefaultNoProxy = "127.0.0.1/8,localhost,.local,.local.,kubernetes,.default,.svc"
)

func newSemver(s string) semver.Semver {
	sv := semver.NewSemverOrDie(s)
	return *sv
}

var (
	DefaultAccessibleAddons = []string{
		"cluster-autoscaler",
		"node-exporter",
		"kube-state-metrics",
		"multus",
		"hubble",
		"metallb",
	}

	DefaultUIResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("64Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
	}

	DefaultAPIResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("150Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
	}

	DefaultSeedControllerManagerResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
	}

	DefaultWebhookResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("64Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
	}

	DefaultNodeportProxyEnvoyResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("32Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
	}

	DefaultNodeportProxyEnvoyManagerResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("32Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("150m"),
			corev1.ResourceMemory: resource.MustParse("48Mi"),
		},
	}

	DefaultNodeportProxyUpdaterResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("32Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("150m"),
			corev1.ResourceMemory: resource.MustParse("32Mi"),
		},
	}

	DefaultNodeportProxyServiceAnnotations = map[string]string{
		// If we're running on AWS, use an NLB. It has a fixed IP & we can use VPC endpoints
		// https://docs.aws.amazon.com/de_de/eks/latest/userguide/load-balancing.html
		"service.beta.kubernetes.io/aws-load-balancer-type": "nlb",
		// On AWS default timeout is 60s, which means: kubectl logs -f will receive EOF after 60s.
		"service.beta.kubernetes.io/aws-load-balancer-connection-idle-timeout": "3600",
	}

	DefaultKubernetesVersioning = kubermaticv1.KubermaticVersioningConfiguration{
		Default: semver.NewSemverOrDie("v1.25.9"),
		// NB: We keep all patch releases that we supported, even if there's
		// an auto-upgrade rule in place. That's because removing a patch
		// release from this slice can break reconciliation loop for clusters
		// running that version, and it might take some time to upgrade all
		// the clusters in large KKP installations.
		// Dashboard hides version that are not supported any longer from the
		// cluster creation/upgrade page.
		Versions: []semver.Semver{
			// Kubernetes 1.24
			newSemver("v1.24.3"),
			newSemver("v1.24.6"),
			newSemver("v1.24.8"),
			newSemver("v1.24.9"),
			newSemver("v1.24.10"),
			newSemver("v1.24.13"),
			// Kubernetes 1.25
			newSemver("v1.25.2"),
			newSemver("v1.25.4"),
			newSemver("v1.25.5"),
			newSemver("v1.25.6"),
			newSemver("v1.25.9"),
			// Kubernetes 1.26
			newSemver("v1.26.1"),
			newSemver("v1.26.4"),
		},
		Updates: []kubermaticv1.Update{
			// ======= 1.23 =======
			{
				// Auto-upgrade unsupported clusters.
				From:      "1.23.*",
				To:        "1.24.10",
				Automatic: pointer.Bool(true),
			},
			// ======= 1.24 =======
			{
				// Allow to change to any patch version
				From: "1.24.*",
				To:   "1.24.*",
			},
			{
				// Auto-upgrade because of CVEs:
				// - CVE-2022-3172 (fixed >= 1.24.5)
				// - CVE-2021-25749 (fixed >= 1.24.5)
				// - CVE-2022-3162 (fixed >= 1.24.8)
				// - CVE-2022-3294 (fixed >= 1.24.8)
				From:      ">= 1.24.0, < 1.24.8",
				To:        "1.24.8",
				Automatic: pointer.Bool(true),
			},
			{
				// Allow to next minor release
				From: "1.24.*",
				To:   "1.25.*",
			},
			// ======= 1.25 =======
			{
				// Allow to change to any patch version
				From: "1.25.*",
				To:   "1.25.*",
			},
			{
				// Auto-upgrade because of CVEs:
				// - CVE-2022-3162 (fixed >= 1.25.4)
				// - CVE-2022-3294 (fixed >= 1.25.4)
				From:      ">= 1.25.0, < 1.25.4",
				To:        "1.25.4",
				Automatic: pointer.Bool(true),
			},
			{
				// Allow to next minor release
				From: "1.25.*",
				To:   "1.26.*",
			},
			// ======= 1.26 =======
			{
				// Allow to change to any patch version
				From: "1.26.*",
				To:   "1.26.*",
			},
		},
		ProviderIncompatibilities: []kubermaticv1.Incompatibility{
			// External CCM on AWS requires Kubernetes 1.24+
			// this can be removed for 2.23 - while we don't support < 1.24 anymore,
			// we are still going to have 1.23 clusters temporarily during an upgrade,
			// so let's keep this just to make sure.
			{
				Provider:  kubermaticv1.CloudProviderAWS,
				Version:   "< 1.24.0",
				Condition: kubermaticv1.ConditionExternalCloudProvider,
				Operation: kubermaticv1.OperationSupport,
			},
			{
				Provider:  kubermaticv1.CloudProviderAWS,
				Version:   "< 1.24.0",
				Condition: kubermaticv1.ConditionExternalCloudProvider,
				Operation: kubermaticv1.OperationCreate,
			},
			{
				Provider:  kubermaticv1.CloudProviderAWS,
				Version:   "< 1.24.0",
				Condition: kubermaticv1.ConditionExternalCloudProvider,
				Operation: kubermaticv1.OperationUpdate,
			},
			// In-tree cloud provider for OpenStack is not supported starting with Kubernetes 1.26.
			// This can be removed once we drop support for Kubernetes 1.26 (note: not for 1.25, because
			// at that point we still might have clusters that needs to be upgraded from 1.25 to 1.26).
			{
				Provider:  kubermaticv1.CloudProviderOpenStack,
				Version:   ">= 1.26.0",
				Condition: kubermaticv1.ConditionInTreeCloudProvider,
				Operation: kubermaticv1.OperationCreate,
			},
			{
				Provider:  kubermaticv1.CloudProviderOpenStack,
				Version:   ">= 1.26.0",
				Condition: kubermaticv1.ConditionInTreeCloudProvider,
				Operation: kubermaticv1.OperationUpdate,
			},
			// In-tree cloud provider for vSphere is not supported by KKP 2.22.0 since CSI
			// migration is on by default for Kubernetes 1.25. We want to make sure that
			// migrations happen before upgrading to that version, so we are enforcing it.
			// This can be removed once we drop support for Kubernetes 1.25.
			{
				Provider:  kubermaticv1.CloudProviderVSphere,
				Version:   ">= 1.25.0",
				Condition: kubermaticv1.ConditionInTreeCloudProvider,
				Operation: kubermaticv1.OperationCreate,
			},
			{
				Provider:  kubermaticv1.CloudProviderVSphere,
				Version:   ">= 1.25.0",
				Condition: kubermaticv1.ConditionInTreeCloudProvider,
				Operation: kubermaticv1.OperationUpdate,
			},
		},
	}

	eksProviderVersioningConfiguration = kubermaticv1.ExternalClusterProviderVersioningConfiguration{
		// List of Supported versions
		// https://docs.aws.amazon.com/eks/latest/userguide/kubernetes-versions.html
		Default: semver.NewSemverOrDie("v1.24"),
		Versions: []semver.Semver{
			newSemver("v1.24"),
			newSemver("v1.23"),
			newSemver("v1.22"),
			newSemver("v1.21"),
		},
	}

	aksProviderVersioningConfiguration = kubermaticv1.ExternalClusterProviderVersioningConfiguration{
		// List of Supported versions
		// https://docs.microsoft.com/en-us/azure/aks/supported-kubernetes-versions
		Default: semver.NewSemverOrDie("v1.24"),
		Versions: []semver.Semver{
			newSemver("v1.25"),
			newSemver("v1.24"),
			newSemver("v1.23"),
		},
	}

	ExternalClusterDefaultKubernetesVersioning = map[kubermaticv1.ExternalClusterProvider]kubermaticv1.ExternalClusterProviderVersioningConfiguration{
		kubermaticv1.ExternalClusterProviderEKS: eksProviderVersioningConfiguration,
		kubermaticv1.ExternalClusterProviderAKS: aksProviderVersioningConfiguration,
	}
)

func DefaultConfiguration(config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) (*kubermaticv1.KubermaticConfiguration, error) {
	if config == nil {
		return nil, errors.New("config must not be nil")
	}

	logger.Debug("Applying defaults to Kubermatic configuration")

	configCopy := config.DeepCopy()

	if configCopy.Spec.NodeportProxy == nil {
		configCopy.Spec.NodeportProxy = &kubermaticv1.NodeportProxyConfig{}
	}

	if configCopy.Spec.NodeportProxy.Envoy == nil {
		configCopy.Spec.NodeportProxy.Envoy = &kubermaticv1.NodePortProxyComponentEnvoy{}
	}

	if configCopy.Spec.NodeportProxy.Envoy.LoadBalancerService == nil {
		configCopy.Spec.NodeportProxy.Envoy.LoadBalancerService = &kubermaticv1.EnvoyLoadBalancerService{}
	}

	if configCopy.Spec.NodeportProxy.EnvoyManager == nil {
		configCopy.Spec.NodeportProxy.EnvoyManager = &kubermaticv1.NodeportProxyComponent{}
	}

	if configCopy.Spec.NodeportProxy.Updater == nil {
		configCopy.Spec.NodeportProxy.Updater = &kubermaticv1.NodeportProxyComponent{}
	}

	if configCopy.Spec.ExposeStrategy == "" {
		configCopy.Spec.ExposeStrategy = DefaultExposeStrategy
		logger.Debugw("Defaulting field", "field", "exposeStrategy", "value", configCopy.Spec.ExposeStrategy)
	}

	if configCopy.Spec.CABundle.Name == "" {
		configCopy.Spec.CABundle.Name = DefaultCABundleConfigMapName
		logger.Debugw("Defaulting field", "field", "caBundle.name", "value", configCopy.Spec.CABundle.Name)
	}

	if configCopy.Spec.ControllerManager == nil {
		configCopy.Spec.ControllerManager = &kubermaticv1.KubermaticControllerManagerConfiguration{}
	}

	if configCopy.Spec.ControllerManager.MaximumParallelReconciles == 0 {
		configCopy.Spec.ControllerManager.MaximumParallelReconciles = DefaultMaximumParallelReconciles
		logger.Debugw("Defaulting field", "field", "controllerManager.maximumParallelReconciles", "value", configCopy.Spec.ControllerManager.MaximumParallelReconciles)
	}

	if configCopy.Spec.ControllerManager.Replicas == nil {
		configCopy.Spec.ControllerManager.Replicas = pointer.Int32(DefaultSeedControllerManagerReplicas)
		logger.Debugw("Defaulting field", "field", "controllerManager.replicas", "value", *configCopy.Spec.ControllerManager.Replicas)
	}

	if configCopy.Spec.ControllerManager.PProfEndpoint == nil {
		configCopy.Spec.ControllerManager.PProfEndpoint = pointer.String(DefaultPProfEndpoint)
		logger.Debugw("Defaulting field", "field", "controllerManager.pprofEndpoint", "value", *configCopy.Spec.ControllerManager.PProfEndpoint)
	}

	if configCopy.Spec.Webhook == nil {
		configCopy.Spec.Webhook = &kubermaticv1.KubermaticWebhookConfiguration{}
	}

	if configCopy.Spec.Webhook.Replicas == nil {
		configCopy.Spec.Webhook.Replicas = pointer.Int32(DefaultWebhookReplicas)
		logger.Debugw("Defaulting field", "field", "webhook.replicas", "value", *configCopy.Spec.Webhook.Replicas)
	}

	if configCopy.Spec.Webhook.PProfEndpoint == nil {
		configCopy.Spec.Webhook.PProfEndpoint = pointer.String(DefaultPProfEndpoint)
		logger.Debugw("Defaulting field", "field", "webhook.pprofEndpoint", "value", *configCopy.Spec.Webhook.PProfEndpoint)
	}

	if configCopy.Spec.API == nil {
		configCopy.Spec.API = &kubermaticv1.KubermaticAPIConfiguration{}
	}

	if configCopy.Spec.API.PProfEndpoint == nil {
		configCopy.Spec.API.PProfEndpoint = pointer.String(DefaultPProfEndpoint)
		logger.Debugw("Defaulting field", "field", "api.pprofEndpoint", "value", *configCopy.Spec.API.PProfEndpoint)
	}

	if configCopy.Spec.UserCluster == nil {
		configCopy.Spec.UserCluster = &kubermaticv1.KubermaticUserClusterConfiguration{}
	}

	if configCopy.Spec.UserCluster.Addons == nil {
		configCopy.Spec.UserCluster.Addons = &kubermaticv1.KubermaticAddonsConfiguration{}
	}

	if len(configCopy.Spec.UserCluster.Addons.Default) == 0 && configCopy.Spec.UserCluster.Addons.DefaultManifests == "" {
		configCopy.Spec.UserCluster.Addons.DefaultManifests = strings.TrimSpace(DefaultKubernetesAddons)
		logger.Debugw("Defaulting field", "field", "userCluster.addons.defaultManifests")
	}

	if configCopy.Spec.UserCluster.APIServerReplicas == nil {
		configCopy.Spec.UserCluster.APIServerReplicas = pointer.Int32(DefaultKubernetesApiserverReplicas)
		logger.Debugw("Defaulting field", "field", "userCluster.apiserverReplicas", "value", *configCopy.Spec.UserCluster.APIServerReplicas)
	}

	// only default the accessible addons if they are not configured at all (nil)
	if configCopy.Spec.API.AccessibleAddons == nil {
		configCopy.Spec.API.AccessibleAddons = DefaultAccessibleAddons
		logger.Debugw("Defaulting field", "field", "api.accessibleAddons", "value", configCopy.Spec.API.AccessibleAddons)
	}

	if configCopy.Spec.API.Replicas == nil {
		configCopy.Spec.API.Replicas = pointer.Int32(DefaultAPIReplicas)
		logger.Debugw("Defaulting field", "field", "api.replicas", "value", *configCopy.Spec.API.Replicas)
	}

	if configCopy.Spec.UserCluster.NodePortRange == "" {
		configCopy.Spec.UserCluster.NodePortRange = resources.DefaultNodePortRange
		logger.Debugw("Defaulting field", "field", "userCluster.nodePortRange", "value", configCopy.Spec.UserCluster.NodePortRange)
	}

	if configCopy.Spec.UserCluster.EtcdVolumeSize == "" {
		configCopy.Spec.UserCluster.EtcdVolumeSize = DefaultEtcdVolumeSize
		logger.Debugw("Defaulting field", "field", "userCluster.etcdVolumeSize", "value", configCopy.Spec.UserCluster.EtcdVolumeSize)
	}

	if configCopy.Spec.Ingress.ClassName == "" {
		configCopy.Spec.Ingress.ClassName = DefaultIngressClass
		logger.Debugw("Defaulting field", "field", "ingress.className", "value", configCopy.Spec.Ingress.ClassName)
	}

	if configCopy.Spec.UserCluster.Monitoring == nil {
		configCopy.Spec.UserCluster.Monitoring = &kubermaticv1.KubermaticUserClusterMonitoringConfiguration{}
	}

	if configCopy.Spec.UserCluster.Monitoring.ScrapeAnnotationPrefix == "" {
		configCopy.Spec.UserCluster.Monitoring.ScrapeAnnotationPrefix = DefaultUserClusterScrapeAnnotationPrefix
		logger.Debugw("Defaulting field", "field", "userCluster.monitoring.scrapeAnnotationPrefix", "value", configCopy.Spec.UserCluster.Monitoring.ScrapeAnnotationPrefix)
	}

	if configCopy.Spec.Ingress.CertificateIssuer == nil {
		configCopy.Spec.Ingress.CertificateIssuer = &corev1.TypedLocalObjectReference{}
	}

	// cert-manager's default is Issuer, but since we do not create an Issuer,
	// it does not make sense to force to change the configuration for the
	// default case
	if configCopy.Spec.Ingress.CertificateIssuer.Kind == "" {
		configCopy.Spec.Ingress.CertificateIssuer.Kind = certmanagerv1.ClusterIssuerKind
		logger.Debugw("Defaulting field", "field", "ingress.certificateIssuer.kind", "value", configCopy.Spec.Ingress.CertificateIssuer.Kind)
	}

	if configCopy.Spec.UI == nil {
		configCopy.Spec.UI = &kubermaticv1.KubermaticUIConfiguration{}
	}

	if configCopy.Spec.UI.Replicas == nil {
		configCopy.Spec.UI.Replicas = pointer.Int32(DefaultUIReplicas)
		logger.Debugw("Defaulting field", "field", "ui.replicas", "value", *configCopy.Spec.UI.Replicas)
	}

	if err := defaultVersioning(&configCopy.Spec.Versions, DefaultKubernetesVersioning); err != nil {
		return configCopy, err
	}

	auth := configCopy.Spec.Auth
	if auth == nil {
		auth = &kubermaticv1.KubermaticAuthConfiguration{}
	}

	if auth.ClientID == "" {
		auth.ClientID = DefaultAuthClientID
		logger.Debugw("Defaulting field", "field", "auth.clientID", "value", auth.ClientID)
	}

	if auth.IssuerClientID == "" {
		auth.IssuerClientID = fmt.Sprintf("%sIssuer", auth.ClientID)
		logger.Debugw("Defaulting field", "field", "auth.issuerClientID", "value", auth.IssuerClientID)
	}

	if auth.TokenIssuer == "" && configCopy.Spec.Ingress.Domain != "" {
		auth.TokenIssuer = fmt.Sprintf("https://%s/dex", configCopy.Spec.Ingress.Domain)
		logger.Debugw("Defaulting field", "field", "auth.tokenIssuer", "value", auth.TokenIssuer)
	}

	if auth.IssuerRedirectURL == "" && configCopy.Spec.Ingress.Domain != "" {
		auth.IssuerRedirectURL = fmt.Sprintf("https://%s/api/v1/kubeconfig", configCopy.Spec.Ingress.Domain)
		logger.Debugw("Defaulting field", "field", "auth.issuerRedirectURL", "value", auth.IssuerRedirectURL)
	}

	configCopy.Spec.Auth = auth

	// default etcdLauncher feature flag if it is not set
	if _, etcdLauncherFeatureGateSet := configCopy.Spec.FeatureGates[features.EtcdLauncher]; !etcdLauncherFeatureGateSet {
		if configCopy.Spec.FeatureGates == nil {
			configCopy.Spec.FeatureGates = make(map[string]bool)
		}

		configCopy.Spec.FeatureGates[features.EtcdLauncher] = true
	}

	if len(configCopy.Spec.NodeportProxy.Envoy.LoadBalancerService.Annotations) == 0 {
		configCopy.Spec.NodeportProxy.Envoy.LoadBalancerService.Annotations = DefaultNodeportProxyServiceAnnotations
		logger.Debugw("Defaulting field", "field", "nodeportProxy.envoy.loadBalancerService.annotations", "value", configCopy.Spec.NodeportProxy.Annotations)
	}

	if err := defaultDockerRepo(&configCopy.Spec.API.DockerRepository, DefaultDashboardImage, "api.dockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultDockerRepo(&configCopy.Spec.UI.DockerRepository, DefaultDashboardImage, "ui.dockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultDockerRepo(&configCopy.Spec.ControllerManager.DockerRepository, DefaultKubermaticImage, "controllerManager.dockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultDockerRepo(&configCopy.Spec.Webhook.DockerRepository, DefaultKubermaticImage, "webhook.dockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultDockerRepo(&configCopy.Spec.UserCluster.KubermaticDockerRepository, DefaultKubermaticImage, "userCluster.kubermaticDockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultDockerRepo(&configCopy.Spec.UserCluster.DNATControllerDockerRepository, DefaultDNATControllerImage, "userCluster.dnatControllerDockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultDockerRepo(&configCopy.Spec.UserCluster.EtcdLauncherDockerRepository, DefaultEtcdLauncherImage, "userCluster.etcdLauncher.DockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultDockerRepo(&configCopy.Spec.UserCluster.Addons.DockerRepository, DefaultKubernetesAddonImage, "userCluster.addons.dockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultDockerRepo(&configCopy.Spec.NodeportProxy.Envoy.DockerRepository, DefaultEnvoyDockerRepository, "nodeportProxy.envoy.dockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultDockerRepo(&configCopy.Spec.NodeportProxy.EnvoyManager.DockerRepository, DefaultNodeportProxyDockerRepository, "nodeportProxy.envoyManager.dockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultDockerRepo(&configCopy.Spec.NodeportProxy.Updater.DockerRepository, DefaultNodeportProxyDockerRepository, "nodeportProxy.updater.dockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultResources(&configCopy.Spec.NodeportProxy.Envoy.Resources, DefaultNodeportProxyEnvoyResources, "nodeportProxy.envoy.resources", logger); err != nil {
		return configCopy, err
	}

	if err := defaultResources(&configCopy.Spec.NodeportProxy.EnvoyManager.Resources, DefaultNodeportProxyEnvoyManagerResources, "nodeportProxy.envoyManager.resources", logger); err != nil {
		return configCopy, err
	}

	if err := defaultResources(&configCopy.Spec.NodeportProxy.Updater.Resources, DefaultNodeportProxyUpdaterResources, "nodeportProxy.updater.resources", logger); err != nil {
		return configCopy, err
	}

	if configCopy.Spec.UserCluster.SystemApplications == nil {
		configCopy.Spec.UserCluster.SystemApplications = &kubermaticv1.SystemApplicationsConfiguration{}
	}

	if err := defaultDockerRepo(&configCopy.Spec.UserCluster.SystemApplications.HelmRepository, DefaultSystemApplicationsHelmRepository, "userCluster.systemApplications.helmRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultResources(&configCopy.Spec.UI.Resources, DefaultUIResources, "ui.resources", logger); err != nil {
		return configCopy, err
	}

	if err := defaultResources(&configCopy.Spec.API.Resources, DefaultAPIResources, "api.resources", logger); err != nil {
		return configCopy, err
	}

	if err := defaultResources(&configCopy.Spec.ControllerManager.Resources, DefaultSeedControllerManagerResources, "controllerManager.resources", logger); err != nil {
		return configCopy, err
	}

	if err := defaultResources(&configCopy.Spec.Webhook.Resources, DefaultWebhookResources, "webhook.resources", logger); err != nil {
		return configCopy, err
	}

	return configCopy, nil
}

func defaultDockerRepo(repo *string, defaultRepo string, key string, logger *zap.SugaredLogger) error {
	if *repo == "" {
		*repo = defaultRepo
		logger.Debugw("Defaulting Docker repository", "field", key, "value", defaultRepo)

		return nil
	}

	ref, err := reference.Parse(*repo)
	if err != nil {
		return fmt.Errorf("invalid docker repository '%s' configured for %s: %w", *repo, key, err)
	}

	if _, ok := ref.(reference.Tagged); ok {
		return fmt.Errorf("it is not allowed to specify an image tag for the %s repository", key)
	}

	return nil
}

func defaultResources(settings **corev1.ResourceRequirements, defaults corev1.ResourceRequirements, key string, logger *zap.SugaredLogger) error {
	if *settings == nil {
		*settings = &corev1.ResourceRequirements{}
	}

	if err := defaultResourceList(&(*settings).Requests, defaults.Requests, key+".requests", logger); err != nil {
		return fmt.Errorf("failed to default requests: %w", err)
	}

	if err := defaultResourceList(&(*settings).Limits, defaults.Limits, key+".limits", logger); err != nil {
		return fmt.Errorf("failed to default limits: %w", err)
	}

	return nil
}

func defaultResourceList(list *corev1.ResourceList, defaults corev1.ResourceList, key string, logger *zap.SugaredLogger) error {
	if list == nil || *list == nil {
		*list = defaults
		logger.Debugw("Defaulting resource constraints", "field", key, "memory", defaults.Memory(), "cpu", defaults.Cpu())
		return nil
	}

	for _, name := range []corev1.ResourceName{corev1.ResourceMemory, corev1.ResourceCPU} {
		quantity := (*list)[name]
		if !quantity.IsZero() {
			continue
		}

		(*list)[name] = defaults[name]
		logger.Debugw("Defaulting resource constraint", "field", key+"."+name.String(), "value", (*list)[name])
	}

	return nil
}

func defaultVersioning(settings *kubermaticv1.KubermaticVersioningConfiguration, defaults kubermaticv1.KubermaticVersioningConfiguration) error {
	// this should never happen as the resources are not pointers in a KubermaticConfiguration
	if settings == nil {
		return nil
	}

	if len(settings.Versions) == 0 {
		settings.Versions = defaults.Versions
	}

	if settings.Default == nil {
		settings.Default = defaults.Default
	}

	if len(settings.Updates) == 0 {
		settings.Updates = defaults.Updates
	}

	if len(settings.ProviderIncompatibilities) == 0 {
		settings.ProviderIncompatibilities = defaults.ProviderIncompatibilities
	}

	return nil
}

const DefaultKubernetesAddons = `
apiVersion: v1
kind: List
items:
- apiVersion: kubermatic.k8c.io/v1
  kind: Addon
  metadata:
    name: canal
    labels:
      addons.kubermatic.io/ensure: true
- apiVersion: kubermatic.k8c.io/v1
  kind: Addon
  metadata:
    name: cilium
    labels:
      addons.kubermatic.io/ensure: true
- apiVersion: kubermatic.k8c.io/v1
  kind: Addon
  metadata:
    name: csi
    labels:
      addons.kubermatic.io/ensure: true
- apiVersion: kubermatic.k8c.io/v1
  kind: Addon
  metadata:
    name: kube-proxy
    labels:
      addons.kubermatic.io/ensure: true
- apiVersion: kubermatic.k8c.io/v1
  kind: Addon
  metadata:
    name: openvpn
    labels:
      addons.kubermatic.io/ensure: true
- apiVersion: kubermatic.k8c.io/v1
  kind: Addon
  metadata:
    name: rbac
    labels:
      addons.kubermatic.io/ensure: true
- apiVersion: kubermatic.k8c.io/v1
  kind: Addon
  metadata:
    name: kubeadm-configmap
    labels:
      addons.kubermatic.io/ensure: true
- apiVersion: kubermatic.k8c.io/v1
  kind: Addon
  metadata:
    name: kubelet-configmap
    labels:
      addons.kubermatic.io/ensure: true
- apiVersion: kubermatic.k8c.io/v1
  kind: Addon
  metadata:
    name: default-storage-class
- apiVersion: kubermatic.k8c.io/v1
  kind: Addon
  metadata:
    name: pod-security-policy
    labels:
      addons.kubermatic.io/ensure: true
- apiVersion: kubermatic.k8c.io/v1
  kind: Addon
  metadata:
    name: aws-node-termination-handler
    labels:
      addons.kubermatic.io/ensure: true
- apiVersion: kubermatic.k8c.io/v1
  kind: Addon
  metadata:
    name: azure-cloud-node-manager
    labels:
      addons.kubermatic.io/ensure: true
`
