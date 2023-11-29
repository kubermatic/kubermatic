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

	addonutil "k8c.io/kubermatic/v2/pkg/addon"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
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
	Versions                  kubermaticversion.Versions
	VersionFilter             string
	RegistryPrefix            string
	IgnoreRepositoryOverrides bool
	Archive                   bool
	ArchivePath               string
	LoadFrom                  string
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
	cmd.PersistentFlags().BoolVar(&opt.IgnoreRepositoryOverrides, "ignore-repository-overrides", true, "Ignore any configured registry overrides in the referenced KubermaticConfiguration to reuse a configuration that already specifies overrides (note that custom tags will still be observed and that this does not affect Helm charts configured via values.yaml; defaults to true)")

	cmd.PersistentFlags().StringVar(&opt.AddonsPath, "addons-path", "", "Path to a local directory containing KKP addons. Takes precedence over --addons-image")
	cmd.PersistentFlags().StringVar(&opt.AddonsImage, "addons-image", "", "Docker image containing KKP addons, if not given, falls back to the Docker image configured in the KubermaticConfiguration")

	cmd.PersistentFlags().DurationVar(&opt.HelmTimeout, "helm-timeout", opt.HelmTimeout, "time to wait for Helm operations to finish")
	cmd.PersistentFlags().StringVar(&opt.HelmValuesFile, "helm-values", "", "Use this values.yaml when rendering Helm charts")
	cmd.PersistentFlags().StringVar(&opt.HelmBinary, "helm-binary", opt.HelmBinary, "Helm 3.x binary to use for rendering charts")

	cmd.PersistentFlags().StringVar(&opt.DockerBinary, "docker-binary", opt.DockerBinary, "deprecated: docker CLI compatible binary to use for pulling and pushing images (this flag has no effect anymore and will be removed in the future)")

	return cmd
}

func getKubermaticConfiguration(logger *logrus.Logger, options *MirrorImagesOptions) (*kubermaticv1.KubermaticConfiguration, error) {
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

		tag := options.Versions.Kubermatic
		if suffix != "" {
			tag = fmt.Sprintf("%s-%s", options.Versions.Kubermatic, suffix)
		}

		addonsImage = kubermaticConfig.Spec.UserCluster.Addons.DockerRepository + ":" + tag
	}

	tempDir, err := images.ExtractAddons(ctx, logger, addonsImage)
	if err != nil {
		return "", fmt.Errorf("failed to create local addons path: %w", err)
	}

	return tempDir, nil
}

func MirrorImagesFunc(logger *logrus.Logger, versions kubermaticversion.Versions, options *MirrorImagesOptions) cobraFuncE {
	return handleErrors(logger, func(cmd *cobra.Command, args []string) error {
		if options.DockerBinary != "" {
			logger.Warn("--docker-binary is deprecated and no longer has any effect; it will be removed with KKP 2.23")
		}

		ctx := cmd.Context()
		userAgent := fmt.Sprintf("kubermatic-installer/%s", versions.Kubermatic)

		if options.LoadFrom == "" {
			kubermaticConfig, err := getKubermaticConfiguration(logger, options)
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

						// Collect images without & with Konnectivity, as Konnecctivity / OpenVPN can be switched in clusters
						// at any time. Remove the non-Konnectivity option once OpenVPN option is finally removed.

						imagesWithoutKonnectivity, err := images.GetImagesForVersion(
							versionLogger,
							clusterVersion,
							cloudSpec,
							cniPlugin,
							false,
							kubermaticConfig,
							allAddons,
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
							allAddons,
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

			if options.ChartsDirectory != "" {
				chartsLogger := logger.WithField("charts-directory", options.ChartsDirectory)
				chartsLogger.Info("🚀 Rendering Helm charts…")

				// Because charts can specify a desired kubeVersion and the helm render default is hardcoded to 1.20, we need to set a custom kubeVersion.
				// Otherwise some charts would fail to render (e.g. consul).
				// Since we are just rendering from the client-side, it makes sense to use the latest kubeVersion we support.
				latestClusterVersion := clusterVersions[len(clusterVersions)-1]
				images, err := images.GetImagesForHelmCharts(ctx, chartsLogger, kubermaticConfig, helmClient, options.ChartsDirectory, options.HelmValuesFile, options.RegistryPrefix, latestClusterVersion.Version.Original())
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

			if options.Archive && options.ArchivePath == "" {
				currentPath, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}
				options.ArchivePath = fmt.Sprintf("%s/kubermatic-v%s-images.tar.gz", currentPath, options.Versions.Kubermatic)
			}

			var verb string
			var count, fullCount int
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
				count, fullCount, err = images.CopyImages(ctx, logger, options.DryRun, sets.List(imageSet), options.Registry, userAgent)
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
		} else {
			logger.WithField("archive-path", options.LoadFrom).Info("🚀 Loading images…")
			if err := images.LoadImages(ctx, logger, options.LoadFrom, options.DryRun, options.Registry, userAgent); err != nil {
				return fmt.Errorf("failed to load images: %w", err)
			}

			logger.Info("✅ Finished loading images.")
			return nil
		}
	})
}
