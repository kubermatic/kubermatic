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
	DefaultExposeStrategy                         = kubermaticv1.ExposeStrategyNodePort
	DefaultVPARecommenderDockerRepository         = "gcr.io/google_containers/vpa-recommender"
	DefaultVPAUpdaterDockerRepository             = "gcr.io/google_containers/vpa-updater"
	DefaultVPAAdmissionControllerDockerRepository = "gcr.io/google_containers/vpa-admission-controller"
	DefaultEnvoyDockerRepository                  = "docker.io/envoyproxy/envoy-alpine"
	DefaultMaximumParallelReconciles              = 10
	DefaultS3Endpoint                             = "s3.amazonaws.com"

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
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("64Mi"),
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
		Default: semver.MustParse("v1.21.5"),
		Versions: []*semver.Version{
			// Kubernetes 1.19
			semver.MustParse("v1.19.15"),
			// Kubernetes 1.20
			semver.MustParse("v1.20.11"),
			// Kubernetes 1.21
			semver.MustParse("v1.21.5"),
			// Kubernetes 1.22
			semver.MustParse("v1.22.2"),
		},
		Updates: []operatorv1alpha1.Update{
			// ======= 1.18 =======
			{
				// Auto-upgrade unsupported clusters
				From:      "1.18.*",
				To:        "1.19.*",
				Automatic: pointer.BoolPtr(true),
			},

			// ======= 1.19 =======
			{
				// Allow to change to any patch version
				From: "1.19.*",
				To:   "1.19.*",
			},
			{
				// Auto-upgrade because of CVE-2021-25741
				From:      ">= 1.19.0, < 1.19.15",
				To:        "1.19.15",
				Automatic: pointer.BoolPtr(true),
			},
			{
				// Allow to next minor release
				From: "1.19.*",
				To:   "1.20.*",
			},

			// ======= 1.20 =======
			{
				// Allow to change to any patch version
				From: "1.20.*",
				To:   "1.20.*",
			},
			{
				// Auto-upgrade because of CVE-2021-25741
				From:      ">= 1.20.0, < 1.20.11",
				To:        "1.20.11",
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
				// Auto-upgrade because of CVE-2021-25741
				From:      ">= 1.21.0, < 1.21.5",
				To:        "1.21.5",
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
		},
		ProviderIncompatibilities: []operatorv1alpha1.Incompatibility{
			{
				Provider:  kubermaticv1.ProviderVSphere,
				Version:   "1.22.*",
				Condition: operatorv1alpha1.AlwaysCondition,
				Operation: operatorv1alpha1.CreateOperation,
			},
			{
				Provider:  kubermaticv1.ProviderVSphere,
				Version:   "1.22.*",
				Condition: operatorv1alpha1.ExternalCloudProviderCondition,
				Operation: operatorv1alpha1.UpdateOperation,
			},
			{
				Provider:  kubermaticv1.ProviderVSphere,
				Version:   "1.22.*",
				Condition: operatorv1alpha1.ExternalCloudProviderCondition,
				Operation: operatorv1alpha1.SupportOperation,
			},
		},
	}
)

func DefaultConfiguration(config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) (*operatorv1alpha1.KubermaticConfiguration, error) {
	logger.Debug("Applying defaults to Kubermatic configuration")

	copy := config.DeepCopy()

	if copy.Spec.ExposeStrategy == "" {
		copy.Spec.ExposeStrategy = DefaultExposeStrategy
		logger.Debugw("Defaulting field", "field", "exposeStrategy", "value", copy.Spec.ExposeStrategy)
	}

	if copy.Spec.CABundle.Name == "" {
		copy.Spec.CABundle.Name = DefaultCABundleConfigMapName
		logger.Debugw("Defaulting field", "field", "caBundle.name", "value", copy.Spec.CABundle.Name)
	}

	if copy.Spec.SeedController.MaximumParallelReconciles == 0 {
		copy.Spec.SeedController.MaximumParallelReconciles = DefaultMaximumParallelReconciles
		logger.Debugw("Defaulting field", "field", "seedController.maximumParallelReconciles", "value", copy.Spec.SeedController.MaximumParallelReconciles)
	}

	if copy.Spec.SeedController.BackupStoreContainer == "" {
		if copy.Spec.SeedController.BackupRestore.Enabled {
			copy.Spec.SeedController.BackupStoreContainer = strings.TrimSpace(DefaultNewBackupStoreContainer)
		} else {
			copy.Spec.SeedController.BackupStoreContainer = strings.TrimSpace(DefaultBackupStoreContainer)
		}
		logger.Debugw("Defaulting field", "field", "seedController.backupStoreContainer")
	}

	if copy.Spec.SeedController.BackupCleanupContainer == "" && !copy.Spec.SeedController.BackupRestore.Enabled {
		copy.Spec.SeedController.BackupCleanupContainer = strings.TrimSpace(DefaultBackupCleanupContainer)
		logger.Debugw("Defaulting field", "field", "seedController.backupCleanupContainer")
	}

	if copy.Spec.SeedController.BackupRestore.Enabled {
		if copy.Spec.SeedController.BackupRestore.S3Endpoint == "" {
			copy.Spec.SeedController.BackupRestore.S3Endpoint = DefaultS3Endpoint
		}
		if copy.Spec.SeedController.BackupRestore.S3BucketName == "" {
			return nil, fmt.Errorf("backupRestore.enabled is set, but s3BucketName is unset")
		}
		if copy.Spec.SeedController.BackupDeleteContainer == "" {
			copy.Spec.SeedController.BackupDeleteContainer = strings.TrimSpace(DefaultNewBackupDeleteContainer)
			logger.Debugw("Defaulting field", "field", "seedController.backupDeleteContainer")
		}
	}

	if copy.Spec.SeedController.Replicas == nil {
		copy.Spec.SeedController.Replicas = pointer.Int32Ptr(DefaultSeedControllerMgrReplicas)
		logger.Debugw("Defaulting field", "field", "seedController.replicas", "value", *copy.Spec.SeedController.Replicas)
	}

	if copy.Spec.API.PProfEndpoint == nil {
		copy.Spec.API.PProfEndpoint = pointer.StringPtr(DefaultPProfEndpoint)
		logger.Debugw("Defaulting field", "field", "api.pprofEndpoint", "value", *copy.Spec.API.PProfEndpoint)
	}

	if copy.Spec.SeedController.PProfEndpoint == nil {
		copy.Spec.SeedController.PProfEndpoint = pointer.StringPtr(DefaultPProfEndpoint)
		logger.Debugw("Defaulting field", "field", "seedController.pprofEndpoint", "value", *copy.Spec.SeedController.PProfEndpoint)
	}

	if copy.Spec.MasterController.PProfEndpoint == nil {
		copy.Spec.MasterController.PProfEndpoint = pointer.StringPtr(DefaultPProfEndpoint)
		logger.Debugw("Defaulting field", "field", "masterController.pprofEndpoint", "value", *copy.Spec.MasterController.PProfEndpoint)
	}

	if copy.Spec.MasterController.Replicas == nil {
		copy.Spec.MasterController.Replicas = pointer.Int32Ptr(DefaultMasterControllerMgrReplicas)
		logger.Debugw("Defaulting field", "field", "masterController.replicas", "value", *copy.Spec.MasterController.Replicas)
	}

	if len(copy.Spec.UserCluster.Addons.Kubernetes.Default) == 0 && copy.Spec.UserCluster.Addons.Kubernetes.DefaultManifests == "" {
		copy.Spec.UserCluster.Addons.Kubernetes.DefaultManifests = strings.TrimSpace(DefaultKubernetesAddons)
		logger.Debugw("Defaulting field", "field", "userCluster.addons.kubernetes.defaultManifests")
	}

	if copy.Spec.UserCluster.APIServerReplicas == nil {
		copy.Spec.UserCluster.APIServerReplicas = pointer.Int32Ptr(DefaultAPIServerReplicas)
		logger.Debugw("Defaulting field", "field", "userCluster.apiserverReplicas", "value", *copy.Spec.UserCluster.APIServerReplicas)
	}

	if len(copy.Spec.API.AccessibleAddons) == 0 {
		copy.Spec.API.AccessibleAddons = DefaultAccessibleAddons
		logger.Debugw("Defaulting field", "field", "api.accessibleAddons", "value", copy.Spec.API.AccessibleAddons)
	}

	if copy.Spec.API.Replicas == nil {
		copy.Spec.API.Replicas = pointer.Int32Ptr(DefaultAPIReplicas)
		logger.Debugw("Defaulting field", "field", "api.replicas", "value", *copy.Spec.API.Replicas)
	}

	if copy.Spec.UserCluster.NodePortRange == "" {
		copy.Spec.UserCluster.NodePortRange = DefaultNodePortRange
		logger.Debugw("Defaulting field", "field", "userCluster.nodePortRange", "value", copy.Spec.UserCluster.NodePortRange)
	}

	if copy.Spec.UserCluster.EtcdVolumeSize == "" {
		copy.Spec.UserCluster.EtcdVolumeSize = DefaultEtcdVolumeSize
		logger.Debugw("Defaulting field", "field", "userCluster.etcdVolumeSize", "value", copy.Spec.UserCluster.EtcdVolumeSize)
	}

	if copy.Spec.Ingress.ClassName == "" {
		copy.Spec.Ingress.ClassName = DefaultIngressClass
		logger.Debugw("Defaulting field", "field", "ingress.className", "value", copy.Spec.Ingress.ClassName)
	}

	// cert-manager's default is Issuer, but since we do not create an Issuer,
	// it does not make sense to force to change the configuration for the
	// default case
	if copy.Spec.Ingress.CertificateIssuer.Kind == "" {
		copy.Spec.Ingress.CertificateIssuer.Kind = certmanagerv1.ClusterIssuerKind
		logger.Debugw("Defaulting field", "field", "ingress.certificateIssuer.kind", "value", copy.Spec.Ingress.CertificateIssuer.Kind)
	}

	if copy.Spec.UI.Config == "" {
		copy.Spec.UI.Config = strings.TrimSpace(DefaultUIConfig)
		logger.Debugw("Defaulting field", "field", "ui.config", "value", copy.Spec.UI.Config)
	}

	if copy.Spec.UI.Replicas == nil {
		copy.Spec.UI.Replicas = pointer.Int32Ptr(DefaultUIReplicas)
		logger.Debugw("Defaulting field", "field", "ui.replicas", "value", *copy.Spec.UI.Replicas)
	}

	if err := defaultVersioning(&copy.Spec.Versions.Kubernetes, DefaultKubernetesVersioning, "versions.kubernetes", logger); err != nil {
		return copy, err
	}

	auth := copy.Spec.Auth

	if auth.ClientID == "" {
		auth.ClientID = DefaultAuthClientID
		logger.Debugw("Defaulting field", "field", "auth.clientID", "value", auth.ClientID)
	}

	if auth.IssuerClientID == "" {
		auth.IssuerClientID = fmt.Sprintf("%sIssuer", auth.ClientID)
		logger.Debugw("Defaulting field", "field", "auth.issuerClientID", "value", auth.IssuerClientID)
	}

	if auth.TokenIssuer == "" && copy.Spec.Ingress.Domain != "" {
		auth.TokenIssuer = fmt.Sprintf("https://%s/dex", copy.Spec.Ingress.Domain)
		logger.Debugw("Defaulting field", "field", "auth.tokenIssuer", "value", auth.TokenIssuer)
	}

	if auth.IssuerRedirectURL == "" && copy.Spec.Ingress.Domain != "" {
		auth.IssuerRedirectURL = fmt.Sprintf("https://%s/api/v1/kubeconfig", copy.Spec.Ingress.Domain)
		logger.Debugw("Defaulting field", "field", "auth.issuerRedirectURL", "value", auth.IssuerRedirectURL)
	}

	copy.Spec.Auth = auth

	if err := defaultDockerRepo(&copy.Spec.API.DockerRepository, DefaultKubermaticImage, "api.dockerRepository", logger); err != nil {
		return copy, err
	}

	if err := defaultDockerRepo(&copy.Spec.UI.DockerRepository, DefaultDashboardImage, "ui.dockerRepository", logger); err != nil {
		return copy, err
	}

	if err := defaultDockerRepo(&copy.Spec.MasterController.DockerRepository, DefaultKubermaticImage, "masterController.dockerRepository", logger); err != nil {
		return copy, err
	}

	if err := defaultDockerRepo(&copy.Spec.SeedController.DockerRepository, DefaultKubermaticImage, "seedController.dockerRepository", logger); err != nil {
		return copy, err
	}

	if err := defaultDockerRepo(&copy.Spec.UserCluster.KubermaticDockerRepository, DefaultKubermaticImage, "userCluster.kubermaticDockerRepository", logger); err != nil {
		return copy, err
	}

	if err := defaultDockerRepo(&copy.Spec.UserCluster.DNATControllerDockerRepository, DefaultDNATControllerImage, "userCluster.dnatControllerDockerRepository", logger); err != nil {
		return copy, err
	}

	if err := defaultDockerRepo(&copy.Spec.UserCluster.EtcdLauncherDockerRepository, DefaultEtcdLauncherImage, "userCluster.etcdLauncher.DockerRepository", logger); err != nil {
		return copy, err
	}

	if err := defaultDockerRepo(&copy.Spec.UserCluster.Addons.Kubernetes.DockerRepository, DefaultKubernetesAddonImage, "userCluster.addons.kubernetes.dockerRepository", logger); err != nil {
		return copy, err
	}

	if err := defaultDockerRepo(&copy.Spec.VerticalPodAutoscaler.Recommender.DockerRepository, DefaultVPARecommenderDockerRepository, "verticalPodAutoscaler.recommender.dockerRepository", logger); err != nil {
		return copy, err
	}

	if err := defaultDockerRepo(&copy.Spec.VerticalPodAutoscaler.Updater.DockerRepository, DefaultVPAUpdaterDockerRepository, "verticalPodAutoscaler.updater.dockerRepository", logger); err != nil {
		return copy, err
	}

	if err := defaultDockerRepo(&copy.Spec.VerticalPodAutoscaler.AdmissionController.DockerRepository, DefaultVPAAdmissionControllerDockerRepository, "verticalPodAutoscaler.admissionController.dockerRepository", logger); err != nil {
		return copy, err
	}

	if err := defaultResources(&copy.Spec.UI.Resources, DefaultUIResources, "ui.resources", logger); err != nil {
		return copy, err
	}

	if err := defaultResources(&copy.Spec.API.Resources, DefaultAPIResources, "api.resources", logger); err != nil {
		return copy, err
	}

	if err := defaultResources(&copy.Spec.SeedController.Resources, DefaultSeedControllerMgrResources, "seedController.resources", logger); err != nil {
		return copy, err
	}

	if err := defaultResources(&copy.Spec.MasterController.Resources, DefaultMasterControllerMgrResources, "masterController.resources", logger); err != nil {
		return copy, err
	}

	if err := defaultResources(&copy.Spec.VerticalPodAutoscaler.Recommender.Resources, DefaultVPARecommenderResources, "verticalPodAutoscaler.recommender.resources", logger); err != nil {
		return copy, err
	}

	if err := defaultResources(&copy.Spec.VerticalPodAutoscaler.Updater.Resources, DefaultVPAUpdaterResources, "verticalPodAutoscaler.updater.resources", logger); err != nil {
		return copy, err
	}

	if err := defaultResources(&copy.Spec.VerticalPodAutoscaler.AdmissionController.Resources, DefaultVPAAdmissionControllerResources, "verticalPodAutoscaler.admissionController.resources", logger); err != nil {
		return copy, err
	}

	return copy, nil
}

func DefaultSeed(seed *kubermaticv1.Seed, logger *zap.SugaredLogger) (*kubermaticv1.Seed, error) {
	logger = logger.With("seed", seed.Name)
	logger.Debug("Applying defaults to Seed")

	copy := seed.DeepCopy()

	if err := defaultDockerRepo(&copy.Spec.NodeportProxy.Envoy.DockerRepository, DefaultEnvoyDockerRepository, "nodeportProxy.envoy.dockerRepository", logger); err != nil {
		return copy, err
	}

	if err := defaultDockerRepo(&copy.Spec.NodeportProxy.EnvoyManager.DockerRepository, DefaultNodeportProxyDockerRepository, "nodeportProxy.envoyManager.dockerRepository", logger); err != nil {
		return copy, err
	}

	if err := defaultDockerRepo(&copy.Spec.NodeportProxy.Updater.DockerRepository, DefaultNodeportProxyDockerRepository, "nodeportProxy.updater.dockerRepository", logger); err != nil {
		return copy, err
	}

	if err := defaultResources(&copy.Spec.NodeportProxy.Envoy.Resources, DefaultNodeportProxyEnvoyResources, "nodeportProxy.envoy.resources", logger); err != nil {
		return copy, err
	}

	if err := defaultResources(&copy.Spec.NodeportProxy.EnvoyManager.Resources, DefaultNodeportProxyEnvoyManagerResources, "nodeportProxy.envoyManager.resources", logger); err != nil {
		return copy, err
	}

	if err := defaultResources(&copy.Spec.NodeportProxy.Updater.Resources, DefaultNodeportProxyUpdaterResources, "nodeportProxy.updater.resources", logger); err != nil {
		return copy, err
	}

	if len(copy.Spec.NodeportProxy.Annotations) == 0 {
		copy.Spec.NodeportProxy.Annotations = DefaultNodeportProxyServiceAnnotations
		logger.Debugw("Defaulting field", "field", "nodeportProxy.annotations", "value", copy.Spec.NodeportProxy.Annotations)
	}

	return copy, nil
}

func defaultDockerRepo(repo *string, defaultRepo string, key string, logger *zap.SugaredLogger) error {
	if *repo == "" {
		*repo = defaultRepo
		logger.Debugw("Defaulting Docker repository", "field", key, "value", defaultRepo)

		return nil
	}

	ref, err := reference.Parse(*repo)
	if err != nil {
		return fmt.Errorf("invalid docker repository '%s' configured for %s: %v", *repo, key, err)
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
		return fmt.Errorf("failed to default requests: %v", err)
	}

	if err := defaultResourceList(&settings.Limits, defaults.Limits, key+".limits", logger); err != nil {
		return fmt.Errorf("failed to default limits: %v", err)
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
