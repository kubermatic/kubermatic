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
	"github.com/distribution/reference"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

// All Default* constants live here, except for those used by other reconciling
// code, for which those constants live in pkg/resources.

const (
	DefaultPProfEndpoint                          = ":6600"
	DefaultEtcdVolumeSize                         = "5Gi"
	DefaultAuthClientID                           = "kubermatic"
	DefaultIngressClass                           = "nginx"
	DefaultCABundleConfigMapName                  = "ca-bundle"
	DefaultAPIReplicas                            = 2
	DefaultUIReplicas                             = 2
	DefaultSeedControllerMgrReplicas              = 1
	DefaultMasterControllerMgrReplicas            = 1
	DefaultAPIServerReplicas                      = 2
	DefaultWebhookReplicas                        = 1
	DefaultControllerManagerReplicas              = 1
	DefaultSchedulerReplicas                      = 1
	DefaultExposeStrategy                         = kubermaticv1.ExposeStrategyNodePort
	DefaultVPARecommenderDockerRepository         = "registry.k8s.io/autoscaling/vpa-recommender"
	DefaultVPAUpdaterDockerRepository             = "registry.k8s.io/autoscaling/vpa-updater"
	DefaultVPAAdmissionControllerDockerRepository = "registry.k8s.io/autoscaling/vpa-admission-controller"
	DefaultEnvoyDockerRepository                  = "docker.io/envoyproxy/envoy-distroless"
	DefaultUserClusterScrapeAnnotationPrefix      = "monitoring.kubermatic.io"
	DefaultMaximumParallelReconciles              = 10
	DefaultS3Endpoint                             = "s3.amazonaws.com"

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
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("150Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
	}

	DefaultMasterControllerMgrResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("400Mi"),
		},
	}

	DefaultSeedControllerMgrResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
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

	DefaultVPARecommenderResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("3Gi"),
		},
	}

	DefaultVPAUpdaterResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
	}

	DefaultVPAAdmissionControllerResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
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
		Default: semver.NewSemverOrDie("v1.31.8"),
		// NB: We keep all patch releases that we supported, even if there's
		// an auto-upgrade rule in place. That's because removing a patch
		// release from this slice can break reconciliation loop for clusters
		// running that version, and it might take some time to upgrade all
		// the clusters in large KKP installations.
		// Dashboard hides version that are not supported any longer from the
		// cluster creation/upgrade page.
		Versions: []semver.Semver{
			// Kubernetes 1.30
			newSemver("v1.30.5"),
			newSemver("v1.30.9"),
			newSemver("v1.30.11"),
			newSemver("v1.30.12"),
			// Kubernetes 1.31
			newSemver("v1.31.1"),
			newSemver("v1.31.5"),
			newSemver("v1.31.7"),
			newSemver("v1.31.8"),
			// Kubernetes 1.32
			newSemver("v1.32.1"),
			newSemver("v1.32.3"),
			newSemver("v1.32.4"),
			// Kubernetes 1.33
			newSemver("v1.33.0"),
		},
		Updates: []kubermaticv1.Update{
			{
				// Allow to next minor release
				From: "1.29.*",
				To:   "1.30.*",
			},
			// ======= 1.30 =======
			{
				// Allow to change to any patch version
				From: "1.30.*",
				To:   "1.30.*",
			},
			{
				// Allow to next minor release
				From: "1.30.*",
				To:   "1.31.*",
			},
			// ======= 1.31 =======
			{
				// Allow to change to any patch version
				From: "1.31.*",
				To:   "1.31.*",
			},
			{
				// Allow to next minor release
				From: "1.31.*",
				To:   "1.32.*",
			},
			// ======= 1.32 =======
			{
				// Allow to change to any patch version
				From: "1.32.*",
				To:   "1.32.*",
			},
			{
				// Allow to next minor release
				From: "1.32.*",
				To:   "1.33.*",
			},
			// ======= 1.33 =======
			{
				// Allow to change to any patch version
				From: "1.33.*",
				To:   "1.33.*",
			},
		},
		ProviderIncompatibilities: []kubermaticv1.Incompatibility{
			// In-tree cloud providers have been fully removed in Kubernetes 1.30.
			// Thus, no in-tree provider is available anymore, and no cluster with in-tree CCM
			// can be upgraded to 1.29.
			{
				Provider:  "",
				Version:   ">= 1.29.0",
				Condition: kubermaticv1.InTreeCloudProviderCondition,
				Operation: kubermaticv1.UpdateOperation,
			},
		},
	}

	eksProviderVersioningConfiguration = kubermaticv1.ExternalClusterProviderVersioningConfiguration{
		// List of Supported versions
		// https://docs.aws.amazon.com/eks/latest/userguide/kubernetes-versions.html
		Default: semver.NewSemverOrDie("v1.31"),
		Versions: []semver.Semver{
			newSemver("v1.31"),
			newSemver("v1.30"),
			newSemver("v1.29"),
			newSemver("v1.28"),
		},
	}

	aksProviderVersioningConfiguration = kubermaticv1.ExternalClusterProviderVersioningConfiguration{
		// List of Supported versions
		// https://docs.microsoft.com/en-us/azure/aks/supported-kubernetes-versions
		Default: semver.NewSemverOrDie("v1.31"),
		Versions: []semver.Semver{
			newSemver("v1.31"),
			newSemver("v1.30"),
			newSemver("v1.29"),
			newSemver("v1.28"),
		},
	}

	ExternalClusterDefaultKubernetesVersioning = map[kubermaticv1.ExternalClusterProviderType]kubermaticv1.ExternalClusterProviderVersioningConfiguration{
		kubermaticv1.EKSProviderType: eksProviderVersioningConfiguration,
		kubermaticv1.AKSProviderType: aksProviderVersioningConfiguration,
	}
)

func DefaultConfiguration(config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) (*kubermaticv1.KubermaticConfiguration, error) {
	if config == nil {
		return nil, errors.New("config must not be nil")
	}

	logger.Debug("Applying defaults to Kubermatic configuration")

	configCopy := config.DeepCopy()

	if configCopy.Spec.ExposeStrategy == "" {
		configCopy.Spec.ExposeStrategy = DefaultExposeStrategy
		logger.Debugw("Defaulting field", "field", "exposeStrategy", "value", configCopy.Spec.ExposeStrategy)
	}

	if configCopy.Spec.CABundle.Name == "" {
		configCopy.Spec.CABundle.Name = DefaultCABundleConfigMapName
		logger.Debugw("Defaulting field", "field", "caBundle.name", "value", configCopy.Spec.CABundle.Name)
	}

	if configCopy.Spec.SeedController.MaximumParallelReconciles == 0 {
		configCopy.Spec.SeedController.MaximumParallelReconciles = DefaultMaximumParallelReconciles
		logger.Debugw("Defaulting field", "field", "seedController.maximumParallelReconciles", "value", configCopy.Spec.SeedController.MaximumParallelReconciles)
	}

	if configCopy.Spec.SeedController.Replicas == nil {
		configCopy.Spec.SeedController.Replicas = ptr.To[int32](DefaultSeedControllerMgrReplicas)
		logger.Debugw("Defaulting field", "field", "seedController.replicas", "value", *configCopy.Spec.SeedController.Replicas)
	}

	if configCopy.Spec.Webhook.Replicas == nil {
		configCopy.Spec.Webhook.Replicas = ptr.To[int32](DefaultWebhookReplicas)
		logger.Debugw("Defaulting field", "field", "webhook.replicas", "value", *configCopy.Spec.Webhook.Replicas)
	}

	if configCopy.Spec.Webhook.PProfEndpoint == nil {
		configCopy.Spec.Webhook.PProfEndpoint = ptr.To(DefaultPProfEndpoint)
		logger.Debugw("Defaulting field", "field", "webhook.pprofEndpoint", "value", *configCopy.Spec.Webhook.PProfEndpoint)
	}

	if configCopy.Spec.API.PProfEndpoint == nil {
		configCopy.Spec.API.PProfEndpoint = ptr.To(DefaultPProfEndpoint)
		logger.Debugw("Defaulting field", "field", "api.pprofEndpoint", "value", *configCopy.Spec.API.PProfEndpoint)
	}

	if configCopy.Spec.SeedController.PProfEndpoint == nil {
		configCopy.Spec.SeedController.PProfEndpoint = ptr.To(DefaultPProfEndpoint)
		logger.Debugw("Defaulting field", "field", "seedController.pprofEndpoint", "value", *configCopy.Spec.SeedController.PProfEndpoint)
	}

	if configCopy.Spec.MasterController.PProfEndpoint == nil {
		configCopy.Spec.MasterController.PProfEndpoint = ptr.To(DefaultPProfEndpoint)
		logger.Debugw("Defaulting field", "field", "masterController.pprofEndpoint", "value", *configCopy.Spec.MasterController.PProfEndpoint)
	}

	if configCopy.Spec.MasterController.Replicas == nil {
		configCopy.Spec.MasterController.Replicas = ptr.To[int32](DefaultMasterControllerMgrReplicas)
		logger.Debugw("Defaulting field", "field", "masterController.replicas", "value", *configCopy.Spec.MasterController.Replicas)
	}

	if len(configCopy.Spec.UserCluster.Addons.Default) == 0 && configCopy.Spec.UserCluster.Addons.DefaultManifests == "" {
		configCopy.Spec.UserCluster.Addons.DefaultManifests = strings.TrimSpace(DefaultKubernetesAddons)
		logger.Debugw("Defaulting field", "field", "userCluster.addons.defaultManifests")
	}

	if configCopy.Spec.UserCluster.APIServerReplicas == nil {
		configCopy.Spec.UserCluster.APIServerReplicas = ptr.To[int32](DefaultAPIServerReplicas)
		logger.Debugw("Defaulting field", "field", "userCluster.apiserverReplicas", "value", *configCopy.Spec.UserCluster.APIServerReplicas)
	}

	// only default the accessible addons if they are not configured at all (nil)
	if configCopy.Spec.API.AccessibleAddons == nil {
		configCopy.Spec.API.AccessibleAddons = DefaultAccessibleAddons
		logger.Debugw("Defaulting field", "field", "api.accessibleAddons", "value", configCopy.Spec.API.AccessibleAddons)
	}

	if configCopy.Spec.API.Replicas == nil {
		configCopy.Spec.API.Replicas = ptr.To[int32](DefaultAPIReplicas)
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

	if configCopy.Spec.UserCluster.Monitoring.ScrapeAnnotationPrefix == "" {
		configCopy.Spec.UserCluster.Monitoring.ScrapeAnnotationPrefix = DefaultUserClusterScrapeAnnotationPrefix
		logger.Debugw("Defaulting field", "field", "userCluster.monitoring.scrapeAnnotationPrefix", "value", configCopy.Spec.UserCluster.Monitoring.ScrapeAnnotationPrefix)
	}

	// cert-manager's default is Issuer, but since we do not create an Issuer,
	// it does not make sense to force to change the configuration for the
	// default case
	if configCopy.Spec.Ingress.CertificateIssuer.Kind == "" {
		configCopy.Spec.Ingress.CertificateIssuer.Kind = certmanagerv1.ClusterIssuerKind
		logger.Debugw("Defaulting field", "field", "ingress.certificateIssuer.kind", "value", configCopy.Spec.Ingress.CertificateIssuer.Kind)
	}

	if configCopy.Spec.UI.Replicas == nil {
		configCopy.Spec.UI.Replicas = ptr.To[int32](DefaultUIReplicas)
		logger.Debugw("Defaulting field", "field", "ui.replicas", "value", *configCopy.Spec.UI.Replicas)
	}

	if err := defaultVersioning(&configCopy.Spec.Versions, DefaultKubernetesVersioning); err != nil {
		return configCopy, err
	}

	if err := defaultExternalClusterVersioning(&configCopy.Spec.Versions, ExternalClusterDefaultKubernetesVersioning); err != nil {
		return configCopy, err
	}

	auth := configCopy.Spec.Auth

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

	if err := defaultDockerRepo(&configCopy.Spec.API.DockerRepository, DefaultDashboardImage, "api.dockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultDockerRepo(&configCopy.Spec.UI.DockerRepository, DefaultDashboardImage, "ui.dockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultDockerRepo(&configCopy.Spec.MasterController.DockerRepository, DefaultKubermaticImage, "masterController.dockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultDockerRepo(&configCopy.Spec.SeedController.DockerRepository, DefaultKubermaticImage, "seedController.dockerRepository", logger); err != nil {
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

	if err := defaultDockerRepo(&configCopy.Spec.VerticalPodAutoscaler.Recommender.DockerRepository, DefaultVPARecommenderDockerRepository, "verticalPodAutoscaler.recommender.dockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultDockerRepo(&configCopy.Spec.VerticalPodAutoscaler.Updater.DockerRepository, DefaultVPAUpdaterDockerRepository, "verticalPodAutoscaler.updater.dockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultDockerRepo(&configCopy.Spec.VerticalPodAutoscaler.AdmissionController.DockerRepository, DefaultVPAAdmissionControllerDockerRepository, "verticalPodAutoscaler.admissionController.dockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultResources(&configCopy.Spec.UI.Resources, DefaultUIResources, "ui.resources", logger); err != nil {
		return configCopy, err
	}

	if err := defaultResources(&configCopy.Spec.API.Resources, DefaultAPIResources, "api.resources", logger); err != nil {
		return configCopy, err
	}

	if err := defaultResources(&configCopy.Spec.SeedController.Resources, DefaultSeedControllerMgrResources, "seedController.resources", logger); err != nil {
		return configCopy, err
	}

	if err := defaultResources(&configCopy.Spec.MasterController.Resources, DefaultMasterControllerMgrResources, "masterController.resources", logger); err != nil {
		return configCopy, err
	}

	if err := defaultResources(&configCopy.Spec.Webhook.Resources, DefaultWebhookResources, "webhook.resources", logger); err != nil {
		return configCopy, err
	}

	if err := defaultResources(&configCopy.Spec.VerticalPodAutoscaler.Recommender.Resources, DefaultVPARecommenderResources, "verticalPodAutoscaler.recommender.resources", logger); err != nil {
		return configCopy, err
	}

	if err := defaultResources(&configCopy.Spec.VerticalPodAutoscaler.Updater.Resources, DefaultVPAUpdaterResources, "verticalPodAutoscaler.updater.resources", logger); err != nil {
		return configCopy, err
	}

	if err := defaultResources(&configCopy.Spec.VerticalPodAutoscaler.AdmissionController.Resources, DefaultVPAAdmissionControllerResources, "verticalPodAutoscaler.admissionController.resources", logger); err != nil {
		return configCopy, err
	}

	return configCopy, nil
}

func defaultHelmRepo(repo *string, defaultRepo string, key string, logger *zap.SugaredLogger) error {
	if *repo != "" && strings.HasPrefix(*repo, "oci://") {
		normalizedRepo := strings.TrimPrefix(*repo, "oci://")
		return defaultDockerRepo(&normalizedRepo, defaultRepo, key, logger)
	}

	return defaultDockerRepo(repo, defaultRepo, key, logger)
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

func defaultResources(settings *corev1.ResourceRequirements, defaults corev1.ResourceRequirements, key string, logger *zap.SugaredLogger) error {
	// this should never happen as the resources are not pointers in a KubermaticConfiguration
	if settings == nil {
		return nil
	}

	if err := defaultResourceList(&settings.Requests, defaults.Requests, key+".requests", logger); err != nil {
		return fmt.Errorf("failed to default requests: %w", err)
	}

	if err := defaultResourceList(&settings.Limits, defaults.Limits, key+".limits", logger); err != nil {
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

func defaultExternalClusterVersioning(settings *kubermaticv1.KubermaticVersioningConfiguration, defaults map[kubermaticv1.ExternalClusterProviderType]kubermaticv1.ExternalClusterProviderVersioningConfiguration) error {
	// this should never happen as the resources are not pointers in a KubermaticConfiguration
	if settings == nil {
		return nil
	}

	for provider, providerVersions := range defaults {
		curSettings := settings.ExternalClusters[provider]

		if curSettings.Default == nil {
			curSettings.Default = providerVersions.Default
		}

		if len(curSettings.Versions) == 0 {
			curSettings.Versions = providerVersions.Versions
		}

		if len(curSettings.Updates) == 0 {
			curSettings.Updates = providerVersions.Updates
		}

		if settings.ExternalClusters == nil {
			settings.ExternalClusters = map[kubermaticv1.ExternalClusterProviderType]kubermaticv1.ExternalClusterProviderVersioningConfiguration{}
		}

		settings.ExternalClusters[provider] = curSettings
	}

	return nil
}

const DefaultBackupStoreContainer = `
name: store-container
image: d3fk/s3cmd@sha256:fb4c4dcf3b842c3d0ead58bda26d05d045b77546e11ac2143d90abca02cbe823
command:
- /bin/sh
- -c
- |
  set -e

  SSL_FLAGS="--ca-certs=/etc/ca-bundle/ca-bundle.pem"
  if [ "${INSECURE:-false}" == "true" ]; then
    SSL_FLAGS="--no-ssl"
  fi

  s3cmd $SSL_FLAGS \
    --access_key=$ACCESS_KEY_ID \
    --secret_key=$SECRET_ACCESS_KEY \
    --host=$ENDPOINT \
    --host-bucket='%(bucket).'$ENDPOINT \
    put /backup/snapshot.db.gz s3://$BUCKET_NAME/$CLUSTER-$BACKUP_TO_CREATE
volumeMounts:
- name: etcd-backup
  mountPath: /backup
`

const DefaultBackupDeleteContainer = `
name: delete-container
image: d3fk/s3cmd@sha256:fb4c4dcf3b842c3d0ead58bda26d05d045b77546e11ac2143d90abca02cbe823
command:
- /bin/sh
- -c
- |
  SSL_FLAGS="--ca-certs=/etc/ca-bundle/ca-bundle.pem"
  if [ "${INSECURE:-false}" == "true" ]; then
    SSL_FLAGS="--no-ssl"
  fi

  s3cmd $SSL_FLAGS \
    --access_key=$ACCESS_KEY_ID \
    --secret_key=$SECRET_ACCESS_KEY \
    --host=$ENDPOINT \
    --host-bucket='%(bucket).'$ENDPOINT \
    del s3://$BUCKET_NAME/$CLUSTER-$BACKUP_TO_DELETE

  case $? in
  12)
    # backup no longer exists, which is fine
    exit 0
    ;;
  0)
    exit 0
    ;;
  *)
    exit $?
    ;;
  esac
`

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
