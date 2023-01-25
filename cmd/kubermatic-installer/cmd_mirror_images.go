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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/images"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/apimachinery/pkg/util/sets"
)

type MirrorImagesOptions struct {
	Options

	Registry                  string
	Config                    string
	VersionFilter             string
	RegistryPrefix            string
	IgnoreRepositoryOverrides bool
	DryRun                    bool

	AddonsPath  string
	AddonsImage string

	HelmValuesFile string
	HelmTimeout    time.Duration
	HelmBinary     string

	// TODO(embik): deprecated, remove with 2.23
	DockerBinary string
}

func MirrorImagesCommand(logger *logrus.Logger, versions kubermaticversion.Versions) *cobra.Command {
	opt := MirrorImagesOptions{
		HelmTimeout: 5 * time.Minute,
		HelmBinary:  "helm",
	}

	cmd := &cobra.Command{
		Use:   "mirror-images [registry]",
		Short: "Mirror images used by KKP to a private image registry",
		Long:  "Uses the docker CLI to download all container images used by KKP, re-tags them and pushes them to a user-defined registry",
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
		},

		RunE:         MirrorImagesFunc(logger, versions, &opt),
		SilenceUsage: true,
	}

	cmd.PersistentFlags().StringVar(&opt.Config, "config", "", "Path to the KubermaticConfiguration YAML file")
	cmd.PersistentFlags().StringVar(&opt.VersionFilter, "version-filter", "", "Version constraint which can be used to filter for specific versions")
	cmd.PersistentFlags().StringVar(&opt.RegistryPrefix, "registry-prefix", "", "Check source registries against this prefix and only include images that match it")
	cmd.PersistentFlags().BoolVar(&opt.DryRun, "dry-run", false, "Only print the names of source and destination images")
	cmd.PersistentFlags().BoolVar(&opt.IgnoreRepositoryOverrides, "ignore-repository-overrides", true, "Ignore any configured registry overrides in the referenced KubermaticConfiguration to re-use a configuration that already specifies overrides (note that custom tags will still be observed and that this does not affect Helm charts configured via values.yaml; defaults to true)")

	cmd.PersistentFlags().StringVar(&opt.AddonsPath, "addons-path", "", "Path to a local directory containing KKP addons. Takes precedence over --addons-image")
	cmd.PersistentFlags().StringVar(&opt.AddonsImage, "addons-image", "", "Docker image containing KKP addons, if not given, falls back to the Docker image configured in the KubermaticConfiguration")

	cmd.PersistentFlags().DurationVar(&opt.HelmTimeout, "helm-timeout", opt.HelmTimeout, "time to wait for Helm operations to finish")
	cmd.PersistentFlags().StringVar(&opt.HelmValuesFile, "helm-values", "", "Use this values.yaml when rendering Helm charts")
	cmd.PersistentFlags().StringVar(&opt.HelmBinary, "helm-binary", opt.HelmBinary, "Helm 3.x binary to use for rendering charts")

	cmd.PersistentFlags().StringVar(&opt.DockerBinary, "docker-binary", opt.DockerBinary, "deprecated: docker CLI compatible binary to use for pulling and pushing images (this flag has no effect anymore and will be removed in the future)")

	return cmd
}

func MirrorImagesFunc(logger *logrus.Logger, versions kubermaticversion.Versions, options *MirrorImagesOptions) cobraFuncE {
	return handleErrors(logger, func(cmd *cobra.Command, args []string) error {
		if options.DockerBinary != "" {
			logger.Warn("--docker-binary is deprecated and no longer has any effect; it will be removed with KKP 2.23")
		}

		if options.Registry == "" {
			return errors.New("no target registry was passed")
		}

		if options.AddonsImage != "" && options.AddonsPath != "" {
			return errors.New("--addons-image and --addons-path must not be set at the same time")
		}

		// error out early if there is no useful Helm binary
		helmClient, err := helm.NewCLI(options.HelmBinary, "", "", options.HelmTimeout, logger)
		if err != nil {
			return fmt.Errorf("failed to create Helm client: %w", err)
		}

		helmVersion, err := helmClient.Version()
		if err != nil {
			return fmt.Errorf("failed to check Helm version: %w", err)
		}

		if helmVersion.LessThan(MinHelmVersion) {
			return fmt.Errorf(
				"the installer requires Helm >= %s, but detected %q as %s (use --helm-binary or $HELM_BINARY to override)",
				MinHelmVersion,
				options.HelmBinary,
				helmVersion,
			)
		}

		caBundle, err := certificates.NewCABundleFromFile(filepath.Join(options.ChartsDirectory, "kubermatic-operator/static/ca-bundle.pem"))
		if err != nil {
			return fmt.Errorf("failed to load CA bundle: %w", err)
		}

		config, _, err := loadKubermaticConfiguration(options.Config)
		if err != nil {
			return fmt.Errorf("failed to load KubermaticConfiguration: %w", err)
		}

		if config == nil {
			return errors.New("please specify your KubermaticConfiguration via --config")
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
			config.Spec.UserCluster.DNATControllerDockerRepository = ""
			config.Spec.UserCluster.EtcdLauncherDockerRepository = ""
			config.Spec.UserCluster.Addons.DockerRepository = ""
			config.Spec.VerticalPodAutoscaler.Recommender.DockerRepository = ""
			config.Spec.VerticalPodAutoscaler.Updater.DockerRepository = ""
			config.Spec.VerticalPodAutoscaler.AdmissionController.DockerRepository = ""
		}

		kubermaticConfig, err := defaulting.DefaultConfiguration(config, zap.NewNop().Sugar())
		if err != nil {
			return fmt.Errorf("failed to default KubermaticConfiguration: %w", err)
		}

		clusterVersions, err := images.GetVersions(logger, kubermaticConfig, options.VersionFilter)
		if err != nil {
			return fmt.Errorf("failed to load versions: %w", err)
		}

		ctx := cmd.Context()

		// if no local addons path is given, use the configured addons
		// Docker image and extract the addons from there
		if options.AddonsPath == "" {
			addonsImage := options.AddonsImage
			if addonsImage == "" {
				suffix := kubermaticConfig.Spec.UserCluster.Addons.DockerTagSuffix

				tag := versions.Kubermatic
				if suffix != "" {
					tag = fmt.Sprintf("%s-%s", versions.Kubermatic, suffix)
				}

				addonsImage = kubermaticConfig.Spec.UserCluster.Addons.DockerRepository + ":" + tag
			}

			if addonsImage != "" {
				tempDir, err := images.ExtractAddons(ctx, logger, addonsImage)
				if err != nil {
					return fmt.Errorf("failed to create local addons path: %w", err)
				}
				defer os.RemoveAll(tempDir)

				options.AddonsPath = tempDir
			}
		}

		logger.Info("🚀 Collecting images…")

		// Using a set here for deduplication
		imageSet := sets.New[string]()
		for _, clusterVersion := range clusterVersions {
			for _, cloudSpec := range images.GetCloudSpecs() {
				for _, cniPlugin := range images.GetCNIPlugins() {
					versionLogger := logger.WithFields(logrus.Fields{
						"version":     clusterVersion.Version.String(),
						"provider":    cloudSpec.ProviderName,
						"cni-plugin":  string(cniPlugin.Type),
						"cni-version": cniPlugin.Version,
					},
					)

					versionLogger.Debug("Collecting images…")

					// Collect images without & with Konnectivity, as Konnecctivity / OpenVPN can be switched in clusters
					// at any time. Remove the non-Konnectivity option once OpenVPN option is finally removed.

					imagesWithoutKonnectivity, err := images.GetImagesForVersion(
						versionLogger,
						clusterVersion,
						cloudSpec,
						cniPlugin,
						false,
						kubermaticConfig,
						options.AddonsPath,
						versions,
						caBundle,
						options.RegistryPrefix,
					)
					if err != nil {
						return fmt.Errorf("failed to get images: %w", err)
					}
					imageSet.Insert(imagesWithoutKonnectivity...)

					imagesWithKonnectivity, err := images.GetImagesForVersion(
						versionLogger,
						clusterVersion,
						cloudSpec,
						cniPlugin,
						true,
						kubermaticConfig,
						options.AddonsPath,
						versions,
						caBundle,
						options.RegistryPrefix,
					)
					if err != nil {
						return fmt.Errorf("failed to get images: %w", err)
					}
					imageSet.Insert(imagesWithKonnectivity...)
				}
			}
		}

		if options.ChartsDirectory != "" {
			chartsLogger := logger.WithField("charts-directory", options.ChartsDirectory)
			chartsLogger.Info("🚀 Rendering Helm charts…")

			images, err := images.GetImagesForHelmCharts(ctx, chartsLogger, kubermaticConfig, helmClient, options.ChartsDirectory, options.HelmValuesFile, options.RegistryPrefix)
			if err != nil {
				return fmt.Errorf("failed to get images: %w", err)
			}
			imageSet.Insert(images...)
		}

		logger.Info("🚀 Rendering system Applications Helm charts…")
		appImages, err := images.GetImagesFromSystemApplicationDefinitions(logger, kubermaticConfig, helmClient, options.HelmTimeout, options.RegistryPrefix)
		if err != nil {
			return fmt.Errorf("failed to get images for system Applications: %w", err)
		}
		imageSet.Insert(appImages...)

		userAgent := fmt.Sprintf("kubermatic-installer/%s", versions.Kubermatic)

		copiedCount, fullCount, err := images.ProcessImages(ctx, logger, options.DryRun, sets.List(imageSet), options.Registry, userAgent)
		if err != nil {
			return fmt.Errorf("failed to mirror all images (successfully copied %d/%d): %w", copiedCount, fullCount, err)
		}

		verb := "mirroring"
		if options.DryRun {
			verb = "listing"
		}

		logger.WithFields(logrus.Fields{"copied-image-count": copiedCount, "all-image-count": fullCount}).Info(fmt.Sprintf("✅ Finished %s images.", verb))

		return nil
	})
}
