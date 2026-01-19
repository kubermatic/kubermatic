/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	addonutil "k8c.io/kubermatic/v2/pkg/addon"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/images"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/validation"
	"k8c.io/kubermatic/v2/pkg/version"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/apimachinery/pkg/util/sets"
)

type MirrorImagesOptions struct {
	Options

	Registry                  string
	Config                    string
	Versions                  kubermaticversion.Versions
	VersionFilter             string
	RegistryPrefix            string
	IgnoreRepositoryOverrides bool
	Archive                   bool
	ArchivePath               string
	LoadFrom                  string
	DryRun                    bool
	Insecure                  bool

	AddonsPath  string
	AddonsImage string

	HelmValuesFile string
	HelmTimeout    time.Duration
	HelmBinary     string
}

func MirrorImagesCommand(logger *logrus.Logger, versions kubermaticversion.Versions) *cobra.Command {
	opt := MirrorImagesOptions{
		HelmTimeout: 5 * time.Minute,
		HelmBinary:  "helm",
	}

	cmd := &cobra.Command{
		Use:   "mirror-images [registry]",
		Short: "Mirror images used by KKP to a private image registry or local archive",
		Long:  "Downloads all container images used by KKP, then either archives them into a tar-ball, or re-tags them and pushes them to a user-defined registry",
		PreRun: func(cmd *cobra.Command, args []string) {
			options.CopyInto(&opt.Options)

			if opt.Config == "" {
				opt.Config = os.Getenv("CONFIG_YAML")
			}

			if opt.HelmValuesFile == "" {
				opt.HelmValuesFile = os.Getenv("HELM_VALUES")
			}

			if opt.HelmBinary == "" {
				opt.HelmBinary = os.Getenv("HELM_BINARY")
			}

			if len(args) >= 1 {
				opt.Registry = args[0]
			}

			if strings.HasPrefix(opt.Registry, "local://") {
				opt.Archive = true
				opt.ArchivePath = strings.TrimPrefix(opt.Registry, "local://")
				opt.Registry = ""
			}

			opt.Versions = versions
		},

		RunE:         MirrorImagesFunc(logger, versions, &opt),
		SilenceUsage: true,
	}

	cmd.PersistentFlags().StringVar(&opt.Config, "config", "", "Path to the KubermaticConfiguration YAML file")
	cmd.PersistentFlags().StringVar(&opt.VersionFilter, "version-filter", "", "Version constraint which can be used to filter for specific versions")
	cmd.PersistentFlags().StringVar(&opt.RegistryPrefix, "registry-prefix", "", "Check source registries against this prefix and only include images that match it")
	cmd.PersistentFlags().StringVar(&opt.LoadFrom, "load-from", "", "Path to an image-archive to (up)load to the provided registry")
	cmd.PersistentFlags().BoolVar(&opt.DryRun, "dry-run", false, "Only print the names of source and destination images")
	cmd.PersistentFlags().BoolVar(&opt.Insecure, "insecure", false, "Insecure option to bypass HTTPS/TLS certificate verification")

	cmd.PersistentFlags().BoolVar(&opt.IgnoreRepositoryOverrides, "ignore-repository-overrides", true, "Ignore any configured registry overrides in the referenced KubermaticConfiguration to reuse a configuration that already specifies overrides (note that custom tags will still be observed and that this does not affect Helm charts configured via values.yaml; defaults to true)")

	cmd.PersistentFlags().StringVar(&opt.AddonsPath, "addons-path", "", "Path to a local directory containing KKP addons. Takes precedence over --addons-image")
	cmd.PersistentFlags().StringVar(&opt.AddonsImage, "addons-image", "", "Docker image containing KKP addons, if not given, falls back to the Docker image configured in the KubermaticConfiguration")

	cmd.PersistentFlags().DurationVar(&opt.HelmTimeout, "helm-timeout", opt.HelmTimeout, "time to wait for Helm operations to finish")
	cmd.PersistentFlags().StringVar(&opt.HelmValuesFile, "helm-values", "", "Use this values.yaml when rendering Helm charts")
	cmd.PersistentFlags().StringVar(&opt.HelmBinary, "helm-binary", opt.HelmBinary, "Helm 3.x binary to use for rendering charts")

	return cmd
}

func getKubermaticConfiguration(options *MirrorImagesOptions) (*kubermaticv1.KubermaticConfiguration, error) {
	if !options.Archive && options.Registry == "" {
		return nil, errors.New("no target registry was passed")
	}

	if options.AddonsImage != "" && options.AddonsPath != "" {
		return nil, errors.New("--addons-image and --addons-path must not be set at the same time")
	}

	config, _, err := loadKubermaticConfiguration(options.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to load KubermaticConfiguration: %w", err)
	}

	if config == nil {
		return nil, errors.New("please specify your KubermaticConfiguration via --config")
	}

	// Validate the MirrorImages field in the KubermaticConfiguration to ensure all images are properly formatted.
	// Each image must follow the format "repository:tag". Validation errors will prevent further processing.
	if len(config.Spec.MirrorImages) > 0 {
		err := validation.ValidateMirrorImages(config.Spec.MirrorImages)
		if err != nil {
			return nil, fmt.Errorf("invalid mirrorImages configuration in KubermaticConfiguration: %w", err)
		}
	}

	// if we pass the option to ignore repository overrides in the KubermaticConfiguration,
	// we make sure we omit any repository configured in the loaded config so they get
	// properly defaulted.
	if options.IgnoreRepositoryOverrides {
		config.Spec.API.DockerRepository = ""
		config.Spec.UI.DockerRepository = ""
		config.Spec.MasterController.DockerRepository = ""
		config.Spec.SeedController.DockerRepository = ""
		config.Spec.Webhook.DockerRepository = ""
		config.Spec.UserCluster.KubermaticDockerRepository = ""
		config.Spec.UserCluster.EtcdLauncherDockerRepository = ""
		config.Spec.UserCluster.Addons.DockerRepository = ""
		config.Spec.VerticalPodAutoscaler.Recommender.DockerRepository = ""
		config.Spec.VerticalPodAutoscaler.Updater.DockerRepository = ""
		config.Spec.VerticalPodAutoscaler.AdmissionController.DockerRepository = ""
	}

	kubermaticConfig, err := defaulting.DefaultConfiguration(config, zap.NewNop().Sugar())
	if err != nil {
		return nil, fmt.Errorf("failed to default KubermaticConfiguration: %w", err)
	}

	return kubermaticConfig, nil
}

func getAddonsPath(ctx context.Context, logger *logrus.Logger, options *MirrorImagesOptions, kubermaticConfig *kubermaticv1.KubermaticConfiguration) (string, error) {
	// if no local addons path is given, use the configured addons
	// Docker image and extract the addons from there
	addonsImage := options.AddonsImage
	if addonsImage == "" {
		suffix := kubermaticConfig.Spec.UserCluster.Addons.DockerTagSuffix

		tag := options.Versions.KubermaticContainerTag
		if suffix != "" {
			tag = fmt.Sprintf("%s-%s", tag, suffix)
		}

		addonsImage = kubermaticConfig.Spec.UserCluster.Addons.DockerRepository + ":" + tag
	}

	tempDir, err := images.ExtractAddons(ctx, logger, addonsImage)
	if err != nil {
		return "", fmt.Errorf("failed to create local addons path: %w", err)
	}

	return tempDir, nil
}

// CollectImageMatrix aggregates images for all cluster versions, cloud providers, and CNI plugins,
// including both Konnectivity and non-Konnectivity configurations.
func CollectImageMatrix(
	logger logrus.FieldLogger,
	clusterVersions []*version.Version,
	kubermaticConfig *kubermaticv1.KubermaticConfiguration,
	allAddons map[string]*addonutil.Addon,
	versions kubermaticversion.Versions,
	caBundle resources.CABundle,
	registryPrefix string,
) ([]string, error) {
	var imageList []string
	for _, clusterVersion := range clusterVersions {
		for _, cloudSpec := range images.GetCloudSpecs() {
			for _, cniPlugin := range images.GetCNIPlugins() {
				versionLogger := logger.WithFields(logrus.Fields{
					"version":     clusterVersion.Version.String(),
					"provider":    cloudSpec.ProviderName,
					"cni-plugin":  string(cniPlugin.Type),
					"cni-version": cniPlugin.Version,
				})

				versionLogger.Debug("Collecting images…")
				imagesWithKonnectivity, err := images.GetImagesForVersion(
					versionLogger,
					clusterVersion,
					cloudSpec,
					cniPlugin,
					true,
					kubermaticConfig,
					allAddons,
					versions,
					caBundle,
					registryPrefix,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to get images: %w", err)
				}
				imageList = append(imageList, imagesWithKonnectivity...)
			}
		}
	}
	return imageList, nil
}

func MirrorImagesFunc(logger *logrus.Logger, versions kubermaticversion.Versions, options *MirrorImagesOptions) cobraFuncE {
	return handleErrors(logger, func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		userAgent := fmt.Sprintf("kubermatic-installer/%s", versions.GitVersion)

		if options.LoadFrom == "" {
			if err := mirrorImages(ctx, logger, versions, options, userAgent); err != nil {
				return err
			}
		} else {
			if err := loadImages(ctx, logger, options, userAgent); err != nil {
				return err
			}
		}

		return nil
	})
}

func mirrorImages(ctx context.Context, logger *logrus.Logger, versions kubermaticversion.Versions, options *MirrorImagesOptions, userAgent string) error {
	kubermaticConfig, err := getKubermaticConfiguration(options)
	if err != nil {
		return fmt.Errorf("failed to get KubermaticConfiguration: %w", err)
	}

	clusterVersions, err := images.GetVersions(logger, kubermaticConfig, options.VersionFilter)
	if err != nil {
		return fmt.Errorf("failed to load versions: %w", err)
	}

	caBundle, err := certificates.NewCABundleFromFile(filepath.Join(options.ChartsDirectory, "kubermatic-operator/static/ca-bundle.pem"))
	if err != nil {
		return fmt.Errorf("failed to load CA bundle: %w", err)
	}

	if options.AddonsPath == "" {
		options.AddonsPath, err = getAddonsPath(ctx, logger, options, kubermaticConfig)
		if err != nil {
			return fmt.Errorf("failed to get addons path: %w", err)
		}
		defer os.RemoveAll(options.AddonsPath)
	}

	allAddons, err := addonutil.LoadAddonsFromDirectory(options.AddonsPath)
	if err != nil {
		return fmt.Errorf("failed to load addons: %w", err)
	}

	logger.Info("🚀 Collecting images…")

	// Using a set here for deduplication
	imageSet := sets.New[string]()

	imageList, err := CollectImageMatrix(logger, clusterVersions, kubermaticConfig, allAddons, versions, caBundle, options.RegistryPrefix)
	if err != nil {
		return err
	}
	imageSet.Insert(imageList...)

	// Populate the imageSet with images specified in the KubermaticConfiguration's MirrorImages field.
	// This ensures that all required images for mirroring are included in the set for further processing.
	if len(kubermaticConfig.Spec.MirrorImages) > 0 {
		imageSet.Insert(kubermaticConfig.Spec.MirrorImages...)
	}

	// if we have a charts directory, we try to render the charts and add the images to our list
	helmChartImages, err := collectHelmChartImages(ctx, logger, kubermaticConfig, clusterVersions, options)
	if err != nil {
		return err
	}
	imageSet.Insert(sets.List(helmChartImages)...)

	// get images from system and default applications
	applicationImages, err := collectApplicationImages(logger, kubermaticConfig, options)
	if err != nil {
		return err
	}
	imageSet.Insert(sets.List(applicationImages)...)

	// finally, add some static images that are not covered by any of the above
	imageSet.Insert(staticImages()...)
	return archiveOrCopyImages(ctx, logger, imageSet, options, userAgent)
}

func collectHelmChartImages(ctx context.Context, logger *logrus.Logger, kubermaticConfig *kubermaticv1.KubermaticConfiguration, clusterVersions []*version.Version, options *MirrorImagesOptions) (sets.Set[string], error) {
	imageSet := sets.New[string]()

	if options.ChartsDirectory == "" {
		return imageSet, nil
	}

	// error out early if there is no useful Helm binary
	helmClient, err := helm.NewCLI(options.HelmBinary, "", "", options.HelmTimeout, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Helm client: %w", err)
	}

	helmVersion, err := helmClient.Version()
	if err != nil {
		return nil, fmt.Errorf("failed to check Helm version: %w", err)
	}

	if helmVersion.LessThan(MinHelmVersion) {
		return nil, fmt.Errorf(
			"the installer requires Helm >= %s, but detected %q as %s (use --helm-binary or $HELM_BINARY to override)",
			MinHelmVersion,
			options.HelmBinary,
			helmVersion,
		)
	}

	chartsLogger := logger.WithField("charts-directory", options.ChartsDirectory)
	chartsLogger.Info("🚀 Rendering Helm charts…")

	// Because charts can specify a desired kubeVersion and the helm render default is hardcoded to 1.20, we need to set a custom kubeVersion.
	// Otherwise some charts would fail to render (e.g. consul).
	// Since we are just rendering from the client-side, it makes sense to use the latest kubeVersion we support.
	latestClusterVersion := clusterVersions[len(clusterVersions)-1]
	images, err := images.GetImagesForHelmCharts(ctx, chartsLogger, kubermaticConfig, helmClient, options.ChartsDirectory, options.HelmValuesFile, options.RegistryPrefix, latestClusterVersion.Version.Original())
	if err != nil {
		return nil, fmt.Errorf("failed to get images from helm charts: %w", err)
	}
	imageSet.Insert(images...)

	return imageSet, nil
}

func collectApplicationImages(logger *logrus.Logger, kubermaticConfig *kubermaticv1.KubermaticConfiguration, options *MirrorImagesOptions) (sets.Set[string], error) {
	imageSet := sets.New[string]()

	helmClient, err := helm.NewCLI(options.HelmBinary, "", "", options.HelmTimeout, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Helm client: %w", err)
	}

	copyKubermaticConfig := kubermaticConfig.DeepCopy()

	if _, ok := copyKubermaticConfig.Spec.FeatureGates[features.ExternalApplicationCatalogManager]; ok {
		logger.Info("🚀 Getting images for configured application catalog and its manager…")
		imageSet.Insert(fmt.Sprintf("%s:%s", strings.Replace(copyKubermaticConfig.Spec.Applications.CatalogManager.RegistrySettings.RegistryURL, "oci://", "", 1), copyKubermaticConfig.Spec.Applications.CatalogManager.RegistrySettings.Tag))
		imageSet.Insert(fmt.Sprintf("%s:%s", copyKubermaticConfig.Spec.Applications.CatalogManager.Image.Repository, copyKubermaticConfig.Spec.Applications.CatalogManager.Image.Tag))
	}

	logger.Info("🚀 Getting images from system Applications Helm charts…")
	for sysChart, err := range images.SystemAppsHelmCharts(copyKubermaticConfig, logger, helmClient, options.HelmTimeout, options.RegistryPrefix) {
		if err != nil {
			return nil, err
		}
		chartImage := fmt.Sprintf("%s/%s:%s", sysChart.Template.Source.Helm.URL, sysChart.Template.Source.Helm.ChartName, sysChart.Template.Source.Helm.ChartVersion)
		// Check if the chartImage starts with "oci://"
		if strings.HasPrefix(chartImage, "oci://") {
			// remove oci:// prefix and insert the chartImage into imageSet.
			imageSet.Insert(chartImage[len("oci://"):])
		}
		imageSet.Insert(sysChart.WorkloadImages...)
	}

	logger.Info("🚀 Getting images from default Applications Helm charts…")
	for defaultChart, err := range images.DefaultAppsHelmCharts(copyKubermaticConfig, logger, helmClient, options.HelmTimeout, options.RegistryPrefix) {
		if err != nil {
			return nil, err
		}
		chartImage := fmt.Sprintf("%s/%s:%s",
			defaultChart.Template.Source.Helm.URL,
			defaultChart.Template.Source.Helm.ChartName,
			defaultChart.Template.Source.Helm.ChartVersion)
		// Check if the chartImage starts with "oci://"
		if strings.HasPrefix(chartImage, "oci://") {
			// remove oci:// prefix and insert the chartImage into imageSet.
			imageSet.Insert(chartImage[len("oci://"):])
		}
		imageSet.Insert(defaultChart.WorkloadImages...)
	}

	return imageSet, nil
}

func archiveOrCopyImages(ctx context.Context, logger *logrus.Logger, imageSet sets.Set[string], options *MirrorImagesOptions, userAgent string) error {
	if options.Archive && options.ArchivePath == "" {
		currentPath, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		options.ArchivePath = fmt.Sprintf("%s/kubermatic-v%s-images.tar.gz", currentPath, options.Versions.GitVersion)
	}

	var verb string
	var count, fullCount int
	var err error

	if options.Archive {
		logger.WithField("archive-path", options.ArchivePath).Info("🚀 Archiving images…")
		count, fullCount, err = images.ArchiveImages(ctx, logger, options.ArchivePath, options.DryRun, sets.List(imageSet))
		if err != nil {
			return fmt.Errorf("failed to export images: %w", err)
		}
		verb = "archiving"
		if options.DryRun {
			verb = "archiving (dry-run)"
		}
	} else {
		logger.WithField("registry", options.Registry).Info("🚀 Mirroring images…")
		count, fullCount, err = images.CopyImages(ctx, logger, options.DryRun, options.Insecure, sets.List(imageSet), options.Registry, userAgent)
		if err != nil {
			return fmt.Errorf("failed to mirror all images (successfully copied %d/%d): %w", count, fullCount, err)
		}
		verb = "mirroring"
		if options.DryRun {
			verb = "mirroring (dry-run)"
		}
	}

	logger.WithFields(logrus.Fields{"copied-image-count": count, "all-image-count": fullCount}).Info(fmt.Sprintf("✅ Finished %s images.", verb))
	return nil
}

func loadImages(ctx context.Context, logger *logrus.Logger, options *MirrorImagesOptions, userAgent string) error {
	logger.WithField("archive-path", options.LoadFrom).Info("🚀 Loading images…")
	if err := images.LoadImages(ctx, logger, options.LoadFrom, options.DryRun, options.Registry, userAgent); err != nil {
		return fmt.Errorf("failed to load images: %w", err)
	}

	logger.Info("✅ Finished loading images.")
	return nil
}

func staticImages() []string {
	return []string{
		resources.WEBTerminalImage}
}
