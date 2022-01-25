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

package defaults

import (
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/docker/distribution/reference"
	certmanagerv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"
)

const (
	DefaultPProfEndpoint                          = ":6600"
	DefaultNodePortRange                          = "30000-32767"
	DefaultEtcdVolumeSize                         = "5Gi"
	DefaultAuthClientID                           = "kubermatic"
	DefaultIngressClass                           = "nginx"
	DefaultCABundleConfigMapName                  = "ca-bundle"
	DefaultAPIReplicas                            = 2
	DefaultUIReplicas                             = 2
	DefaultSeedControllerMgrReplicas              = 1
	DefaultMasterControllerMgrReplicas            = 1
	DefaultAPIServerReplicas                      = 2
	DefaultControllerManagerReplicas              = 1
	DefaultSchedulerReplicas                      = 1
	DefaultExposeStrategy                         = kubermaticv1.ExposeStrategyNodePort
	DefaultVPARecommenderDockerRepository         = "gcr.io/google_containers/vpa-recommender"
	DefaultVPAUpdaterDockerRepository             = "gcr.io/google_containers/vpa-updater"
	DefaultVPAAdmissionControllerDockerRepository = "gcr.io/google_containers/vpa-admission-controller"
	DefaultEnvoyDockerRepository                  = "docker.io/envoyproxy/envoy-alpine"
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

var (
	DefaultAccessibleAddons = []string{
		"cluster-autoscaler",
		"node-exporter",
		"kube-state-metrics",
		"multus",
		"hubble",
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
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
	}

	DefaultMasterControllerMgrResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
	}

	DefaultSeedControllerMgrResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
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
			corev1.ResourceMemory: resource.MustParse("32Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
	}

	DefaultVPAAdmissionControllerResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("32Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
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

	DefaultKubernetesVersioning = operatorv1alpha1.KubermaticVersioningConfiguration{
		Default: semver.MustParse("v1.21.8"),
		Versions: []*semver.Version{
			// Kubernetes 1.20
			semver.MustParse("v1.20.13"),
			semver.MustParse("v1.20.14"),
			// Kubernetes 1.21
			semver.MustParse("v1.21.8"),
			// Kubernetes 1.22
			semver.MustParse("v1.22.5"),
		},
		Updates: []operatorv1alpha1.Update{
			// ======= 1.19 =======
			{
				// Auto-upgrade unsupported clusters
				From:      "1.19.*",
				To:        "1.20.*",
				Automatic: pointer.BoolPtr(true),
			},

			// ======= 1.20 =======
			{
				// Allow to change to any patch version
				From: "1.20.*",
				To:   "1.20.*",
			},
			{
				// Auto-upgrade because of CVEs:
				// - CVE-2021-25741 (fixed >= 1.20.11)
				// - CVE-2021-3711 (fixed >= 1.20.13)
				// - CVE-2021-3712 (fixed >= 1.20.13)
				// - CVE-2021-33910 (fixed >= 1.20.13)
				From:      ">= 1.20.0, < 1.20.13",
				To:        "1.20.13",
				Automatic: pointer.BoolPtr(true),
			},
			{
				// Allow to next minor release
				From: "1.20.*",
				To:   "1.21.*",
			},

			// ======= 1.21 =======
			{
				// Allow to change to any patch version
				From: "1.21.*",
				To:   "1.21.*",
			},
			{
				// Auto-upgrade because of CVEs:
				// - CVE-2021-25741 (fixed >= 1.21.5)
				// - CVE-2021-3711 (fixed >= 1.21.7)
				// - CVE-2021-3712 (fixed >= 1.21.7)
				// - CVE-2021-33910 (fixed >= 1.21.7)
				// - CVE-2021-44716 (fixed >= 1.21.8)
				// - CVE-2021-44717 (fixed >= 1.21.8)
				From:      ">= 1.21.0, < 1.21.8",
				To:        "1.21.8",
				Automatic: pointer.BoolPtr(true),
			},
			{
				// Allow to next minor release
				From: "1.21.*",
				To:   "1.22.*",
			},

			// ======= 1.22 =======
			{
				// Allow to change to any patch version
				From: "1.22.*",
				To:   "1.22.*",
			},
			{
				// Auto-upgrade because of CVEs:
				// - CVE-2021-3711 (fixed >= 1.22.4)
				// - CVE-2021-3712 (fixed >= 1.22.4)
				// - CVE-2021-33910 (fixed >= 1.22.4)
				// - CVE-2021-44716 (fixed >= 1.22.5)
				// - CVE-2021-44717 (fixed >= 1.22.5)
				From:      ">= 1.22.0, < 1.22.5",
				To:        "1.22.5",
				Automatic: pointer.BoolPtr(true),
			},
		},
		ProviderIncompatibilities: []operatorv1alpha1.Incompatibility{
			{
				Provider:  kubermaticv1.VSphereCloudProvider,
				Version:   "1.23.*",
				Condition: operatorv1alpha1.AlwaysCondition,
				Operation: operatorv1alpha1.CreateOperation,
			},
			{
				Provider:  kubermaticv1.VSphereCloudProvider,
				Version:   "1.23.*",
				Condition: operatorv1alpha1.ExternalCloudProviderCondition,
				Operation: operatorv1alpha1.UpdateOperation,
			},
			{
				Provider:  kubermaticv1.VSphereCloudProvider,
				Version:   "1.23.*",
				Condition: operatorv1alpha1.ExternalCloudProviderCondition,
				Operation: operatorv1alpha1.SupportOperation,
			},
		},
	}
)

func DefaultConfiguration(config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) (*operatorv1alpha1.KubermaticConfiguration, error) {
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

	if configCopy.Spec.SeedController.BackupStoreContainer == "" {
		if configCopy.Spec.SeedController.BackupRestore.Enabled {
			configCopy.Spec.SeedController.BackupStoreContainer = strings.TrimSpace(DefaultNewBackupStoreContainer)
		} else {
			configCopy.Spec.SeedController.BackupStoreContainer = strings.TrimSpace(DefaultBackupStoreContainer)
		}
		logger.Debugw("Defaulting field", "field", "seedController.backupStoreContainer")
	}

	if configCopy.Spec.SeedController.BackupCleanupContainer == "" && !configCopy.Spec.SeedController.BackupRestore.Enabled {
		configCopy.Spec.SeedController.BackupCleanupContainer = strings.TrimSpace(DefaultBackupCleanupContainer)
		logger.Debugw("Defaulting field", "field", "seedController.backupCleanupContainer")
	}

	if configCopy.Spec.SeedController.BackupRestore.Enabled {
		if configCopy.Spec.SeedController.BackupRestore.S3Endpoint == "" {
			configCopy.Spec.SeedController.BackupRestore.S3Endpoint = DefaultS3Endpoint
		}
		if configCopy.Spec.SeedController.BackupRestore.S3BucketName == "" {
			return nil, fmt.Errorf("backupRestore.enabled is set, but s3BucketName is unset")
		}
		if configCopy.Spec.SeedController.BackupDeleteContainer == "" {
			configCopy.Spec.SeedController.BackupDeleteContainer = strings.TrimSpace(DefaultNewBackupDeleteContainer)
			logger.Debugw("Defaulting field", "field", "seedController.backupDeleteContainer")
		}
	}

	if configCopy.Spec.SeedController.Replicas == nil {
		configCopy.Spec.SeedController.Replicas = pointer.Int32Ptr(DefaultSeedControllerMgrReplicas)
		logger.Debugw("Defaulting field", "field", "seedController.replicas", "value", *configCopy.Spec.SeedController.Replicas)
	}

	if configCopy.Spec.API.PProfEndpoint == nil {
		configCopy.Spec.API.PProfEndpoint = pointer.StringPtr(DefaultPProfEndpoint)
		logger.Debugw("Defaulting field", "field", "api.pprofEndpoint", "value", *configCopy.Spec.API.PProfEndpoint)
	}

	if configCopy.Spec.SeedController.PProfEndpoint == nil {
		configCopy.Spec.SeedController.PProfEndpoint = pointer.StringPtr(DefaultPProfEndpoint)
		logger.Debugw("Defaulting field", "field", "seedController.pprofEndpoint", "value", *configCopy.Spec.SeedController.PProfEndpoint)
	}

	if configCopy.Spec.MasterController.PProfEndpoint == nil {
		configCopy.Spec.MasterController.PProfEndpoint = pointer.StringPtr(DefaultPProfEndpoint)
		logger.Debugw("Defaulting field", "field", "masterController.pprofEndpoint", "value", *configCopy.Spec.MasterController.PProfEndpoint)
	}

	if configCopy.Spec.MasterController.Replicas == nil {
		configCopy.Spec.MasterController.Replicas = pointer.Int32Ptr(DefaultMasterControllerMgrReplicas)
		logger.Debugw("Defaulting field", "field", "masterController.replicas", "value", *configCopy.Spec.MasterController.Replicas)
	}

	if len(configCopy.Spec.UserCluster.Addons.Kubernetes.Default) == 0 && configCopy.Spec.UserCluster.Addons.Kubernetes.DefaultManifests == "" {
		configCopy.Spec.UserCluster.Addons.Kubernetes.DefaultManifests = strings.TrimSpace(DefaultKubernetesAddons)
		logger.Debugw("Defaulting field", "field", "userCluster.addons.kubernetes.defaultManifests")
	}

	if configCopy.Spec.UserCluster.APIServerReplicas == nil {
		configCopy.Spec.UserCluster.APIServerReplicas = pointer.Int32Ptr(DefaultAPIServerReplicas)
		logger.Debugw("Defaulting field", "field", "userCluster.apiserverReplicas", "value", *configCopy.Spec.UserCluster.APIServerReplicas)
	}

	if len(configCopy.Spec.API.AccessibleAddons) == 0 {
		configCopy.Spec.API.AccessibleAddons = DefaultAccessibleAddons
		logger.Debugw("Defaulting field", "field", "api.accessibleAddons", "value", configCopy.Spec.API.AccessibleAddons)
	}

	if configCopy.Spec.API.Replicas == nil {
		configCopy.Spec.API.Replicas = pointer.Int32Ptr(DefaultAPIReplicas)
		logger.Debugw("Defaulting field", "field", "api.replicas", "value", *configCopy.Spec.API.Replicas)
	}

	if configCopy.Spec.UserCluster.NodePortRange == "" {
		configCopy.Spec.UserCluster.NodePortRange = DefaultNodePortRange
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

	if configCopy.Spec.UI.Config == "" {
		configCopy.Spec.UI.Config = strings.TrimSpace(DefaultUIConfig)
		logger.Debugw("Defaulting field", "field", "ui.config", "value", configCopy.Spec.UI.Config)
	}

	if configCopy.Spec.UI.Replicas == nil {
		configCopy.Spec.UI.Replicas = pointer.Int32Ptr(DefaultUIReplicas)
		logger.Debugw("Defaulting field", "field", "ui.replicas", "value", *configCopy.Spec.UI.Replicas)
	}

	if err := defaultVersioning(&configCopy.Spec.Versions.Kubernetes, DefaultKubernetesVersioning, "versions.kubernetes", logger); err != nil {
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

	if err := defaultDockerRepo(&configCopy.Spec.API.DockerRepository, DefaultKubermaticImage, "api.dockerRepository", logger); err != nil {
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

	if err := defaultDockerRepo(&configCopy.Spec.UserCluster.KubermaticDockerRepository, DefaultKubermaticImage, "userCluster.kubermaticDockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultDockerRepo(&configCopy.Spec.UserCluster.DNATControllerDockerRepository, DefaultDNATControllerImage, "userCluster.dnatControllerDockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultDockerRepo(&configCopy.Spec.UserCluster.EtcdLauncherDockerRepository, DefaultEtcdLauncherImage, "userCluster.etcdLauncher.DockerRepository", logger); err != nil {
		return configCopy, err
	}

	if err := defaultDockerRepo(&configCopy.Spec.UserCluster.Addons.Kubernetes.DockerRepository, DefaultKubernetesAddonImage, "userCluster.addons.kubernetes.dockerRepository", logger); err != nil {
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

// DefaultSeed fills in missing values in the Seed's spec by copying them from the global
// defaults in the KubermaticConfiguration (in which some fields might already be deprecated,
// as we move configuration down into the Seeds). This function assumes that the config has
// already been defaulted.
func DefaultSeed(seed *kubermaticv1.Seed, config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) (*kubermaticv1.Seed, error) {
	logger = logger.With("seed", seed.Name)
	logger.Debug("Applying defaults to Seed")

	seedCopy := seed.DeepCopy()

	if seedCopy.Spec.ExposeStrategy == "" {
		seedCopy.Spec.ExposeStrategy = config.Spec.ExposeStrategy
	}

	if err := defaultDockerRepo(&seedCopy.Spec.NodeportProxy.Envoy.DockerRepository, DefaultEnvoyDockerRepository, "nodeportProxy.envoy.dockerRepository", logger); err != nil {
		return seedCopy, err
	}

	if err := defaultDockerRepo(&seedCopy.Spec.NodeportProxy.EnvoyManager.DockerRepository, DefaultNodeportProxyDockerRepository, "nodeportProxy.envoyManager.dockerRepository", logger); err != nil {
		return seedCopy, err
	}

	if err := defaultDockerRepo(&seedCopy.Spec.NodeportProxy.Updater.DockerRepository, DefaultNodeportProxyDockerRepository, "nodeportProxy.updater.dockerRepository", logger); err != nil {
		return seedCopy, err
	}

	if err := defaultResources(&seedCopy.Spec.NodeportProxy.Envoy.Resources, DefaultNodeportProxyEnvoyResources, "nodeportProxy.envoy.resources", logger); err != nil {
		return seedCopy, err
	}

	if err := defaultResources(&seedCopy.Spec.NodeportProxy.EnvoyManager.Resources, DefaultNodeportProxyEnvoyManagerResources, "nodeportProxy.envoyManager.resources", logger); err != nil {
		return seedCopy, err
	}

	if err := defaultResources(&seedCopy.Spec.NodeportProxy.Updater.Resources, DefaultNodeportProxyUpdaterResources, "nodeportProxy.updater.resources", logger); err != nil {
		return seedCopy, err
	}

	if len(seedCopy.Spec.NodeportProxy.Annotations) == 0 {
		seedCopy.Spec.NodeportProxy.Annotations = DefaultNodeportProxyServiceAnnotations
		logger.Debugw("Defaulting field", "field", "nodeportProxy.annotations", "value", seedCopy.Spec.NodeportProxy.Annotations)
	}

	// apply settings from the KubermaticConfiguration to the Seed, in case they are not set there;
	// over time, we move pretty much all of this into the Seed, but this code copies the still existing,
	// deprecated fields over.
	settings := &seedCopy.Spec.DefaultComponentSettings

	if settings.Apiserver.Replicas == nil {
		settings.Apiserver.Replicas = config.Spec.UserCluster.APIServerReplicas
	}

	if settings.Apiserver.NodePortRange == "" {
		settings.Apiserver.NodePortRange = config.Spec.UserCluster.NodePortRange
	}

	if settings.Apiserver.EndpointReconcilingDisabled == nil && config.Spec.UserCluster.DisableAPIServerEndpointReconciling {
		settings.Apiserver.EndpointReconcilingDisabled = &config.Spec.UserCluster.DisableAPIServerEndpointReconciling
	}

	if settings.ControllerManager.Replicas == nil {
		settings.ControllerManager.Replicas = pointer.Int32Ptr(DefaultControllerManagerReplicas)
	}

	if settings.Scheduler.Replicas == nil {
		settings.Scheduler.Replicas = pointer.Int32Ptr(DefaultSchedulerReplicas)
	}

	if settings.Etcd.DiskSize == nil {
		etcdDiskSize, err := resource.ParseQuantity(config.Spec.UserCluster.EtcdVolumeSize)
		if err != nil {
			return seedCopy, fmt.Errorf("failed to parse spec.userCluster.etcdVolumeSize %q in KubermaticConfiguration: %w", config.Spec.UserCluster.EtcdVolumeSize, err)
		}
		settings.Etcd.DiskSize = &etcdDiskSize
	}

	if settings.Etcd.ClusterSize == nil {
		settings.Etcd.ClusterSize = pointer.Int32(kubermaticv1.DefaultEtcdClusterSize)
	}

	return seedCopy, nil
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

func defaultVersioning(settings *operatorv1alpha1.KubermaticVersioningConfiguration, defaults operatorv1alpha1.KubermaticVersioningConfiguration, key string, logger *zap.SugaredLogger) error {
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

const DefaultBackupStoreContainer = `
name: store-container
image: quay.io/kubermatic/s3-storer:v0.1.6
command:
- /bin/sh
- -c
- |
  set -euo pipefail

  endpoint=minio.minio.svc.cluster.local:9000
  bucket=kubermatic-etcd-backups

  s3-storeuploader store \
    --ca-bundle=/etc/ca-bundle/ca-bundle.pem \
    --file /backup/snapshot.db \
    --endpoint "$endpoint" \
    --bucket "$bucket" \
    --create-bucket \
    --prefix $CLUSTER

  s3-storeuploader delete-old-revisions \
    --ca-bundle=/etc/ca-bundle/ca-bundle.pem \
    --max-revisions 20 \
    --endpoint "$endpoint" \
    --bucket "$bucket" \
    --prefix $CLUSTER
env:
- name: ACCESS_KEY_ID
  valueFrom:
    secretKeyRef:
      name: s3-credentials
      key: ACCESS_KEY_ID
- name: SECRET_ACCESS_KEY
  valueFrom:
    secretKeyRef:
      name: s3-credentials
      key: SECRET_ACCESS_KEY
volumeMounts:
- name: etcd-backup
  mountPath: /backup
`

const DefaultNewBackupStoreContainer = `
name: store-container
image: d3fk/s3cmd@sha256:2061883abbf0ebcf0ea3d5d218558c9c229f212e9c08af4acdaa3758980eb67a
command:
- /bin/sh
- -c
- |
  set -e
  s3cmd \
    --ca-certs=/etc/ca-bundle/ca-bundle.pem \
    --access_key=$ACCESS_KEY_ID \
    --secret_key=$SECRET_ACCESS_KEY \
    --host=$ENDPOINT \
    --host-bucket='%(bucket).'$ENDPOINT \
    put /backup/snapshot.db s3://$BUCKET_NAME/$CLUSTER-$BACKUP_TO_CREATE
env:
- name: ACCESS_KEY_ID
  valueFrom:
    secretKeyRef:
      name: backup-s3
      key: ACCESS_KEY_ID
- name: SECRET_ACCESS_KEY
  valueFrom:
    secretKeyRef:
      name: backup-s3
      key: SECRET_ACCESS_KEY
- name: BUCKET_NAME
  valueFrom:
    configMapKeyRef:
      name: s3-settings
      key: BUCKET_NAME
- name: ENDPOINT
  valueFrom:
    configMapKeyRef:
      name: s3-settings
      key: ENDPOINT
volumeMounts:
- name: etcd-backup
  mountPath: /backup
`

const DefaultNewBackupDeleteContainer = `
name: delete-container
image: d3fk/s3cmd@sha256:2061883abbf0ebcf0ea3d5d218558c9c229f212e9c08af4acdaa3758980eb67a
command:
- /bin/sh
- -c
- |
  s3cmd \
    --ca-certs=/etc/ca-bundle/ca-bundle.pem \
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
env:
- name: ACCESS_KEY_ID
  valueFrom:
    secretKeyRef:
      name: backup-s3
      key: ACCESS_KEY_ID
- name: SECRET_ACCESS_KEY
  valueFrom:
    secretKeyRef:
      name: backup-s3
      key: SECRET_ACCESS_KEY
- name: BUCKET_NAME
  valueFrom:
    configMapKeyRef:
      name: s3-settings
      key: BUCKET_NAME
- name: ENDPOINT
  valueFrom:
    configMapKeyRef:
      name: s3-settings
      key: ENDPOINT
`

const DefaultBackupCleanupContainer = `
name: cleanup-container
image: quay.io/kubermatic/s3-storer:v0.1.6
command:
- /bin/sh
- -c
- |
  set -euo pipefail

  endpoint=minio.minio.svc.cluster.local:9000
  bucket=kubermatic-etcd-backups

  # by default, we keep the most recent backup for every user cluster
  s3-storeuploader delete-old-revisions \
    --ca-bundle=/etc/ca-bundle/ca-bundle.pem \
    --max-revisions 1 \
    --endpoint "$endpoint" \
    --bucket "$bucket" \
    --prefix $CLUSTER

  # alternatively, delete all backups for this cluster
  #s3-storeuploader delete-all \
  # --ca-bundle=/etc/ca-bundle/ca-bundle.pem \
  # --endpoint "$endpoint" \
  # --bucket "$bucket" \
  # --prefix $CLUSTER
env:
- name: ACCESS_KEY_ID
  valueFrom:
    secretKeyRef:
      name: s3-credentials
      key: ACCESS_KEY_ID
- name: SECRET_ACCESS_KEY
  valueFrom:
    secretKeyRef:
      name: s3-credentials
      key: SECRET_ACCESS_KEY
`

const DefaultUIConfig = `
{
  "share_kubeconfig": false
}`

const DefaultKubernetesAddons = `
apiVersion: v1
kind: List
items:
- apiVersion: kubermatic.k8s.io/v1
  kind: Addon
  metadata:
    name: canal
    labels:
      addons.kubermatic.io/ensure: true
- apiVersion: kubermatic.k8s.io/v1
  kind: Addon
  metadata:
    name: cilium
    labels:
      addons.kubermatic.io/ensure: true
- apiVersion: kubermatic.k8s.io/v1
  kind: Addon
  metadata:
    name: csi
    labels:
      addons.kubermatic.io/ensure: true
- apiVersion: kubermatic.k8s.io/v1
  kind: Addon
  metadata:
    name: kube-proxy
    labels:
      addons.kubermatic.io/ensure: true
- apiVersion: kubermatic.k8s.io/v1
  kind: Addon
  metadata:
    name: openvpn
    labels:
      addons.kubermatic.io/ensure: true
- apiVersion: kubermatic.k8s.io/v1
  kind: Addon
  metadata:
    name: rbac
    labels:
      addons.kubermatic.io/ensure: true
- apiVersion: kubermatic.k8s.io/v1
  kind: Addon
  metadata:
    name: kubeadm-configmap
    labels:
      addons.kubermatic.io/ensure: true
- apiVersion: kubermatic.k8s.io/v1
  kind: Addon
  metadata:
    name: kubelet-configmap
- apiVersion: kubermatic.k8s.io/v1
  kind: Addon
  metadata:
    name: default-storage-class
- apiVersion: kubermatic.k8s.io/v1
  kind: Addon
  metadata:
    name: pod-security-policy
    labels:
      addons.kubermatic.io/ensure: true
- apiVersion: kubermatic.k8s.io/v1
  kind: Addon
  metadata:
    name: aws-node-termination-handler
    labels:
      addons.kubermatic.io/ensure: true
`
