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

	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/images"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/apimachinery/pkg/util/sets"
)

type MirrorImagesOptions struct {
	Options

	Config         string
	VersionFilter  string
	Registry       string
	DryRun         bool
	AddonsPath     string
	AddonsImage    string
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
	cmd.PersistentFlags().BoolVar(&opt.DryRun, "dry-run", false, "Only print the names of found images")

	cmd.PersistentFlags().StringVar(&opt.AddonsPath, "addons-path", "", "Address of the registry to push to, for example localhost:5000")
	cmd.PersistentFlags().StringVar(&opt.AddonsImage, "addons-image", "", "Docker image containing KKP addons, if not given, falls back to the Docker image configured in the KubermaticConfiguration")

	cmd.PersistentFlags().DurationVar(&opt.HelmTimeout, "helm-timeout", opt.HelmTimeout, "time to wait for Helm operations to finish")
	cmd.PersistentFlags().StringVar(&opt.HelmValuesFile, "helm-values", "", "Use this values.yaml when rendering Helm charts")
	cmd.PersistentFlags().StringVar(&opt.HelmBinary, "helm-binary", opt.HelmBinary, "Helm 3.x binary to use for rendering charts")

	// these flags are deprecated but retained to ensure compatibility with `image-loader` flags,
	// except for `--versions-file`, that flag was already deprecated in `image-loader`.
	cmd.PersistentFlags().StringVar(&opt.Config, "configuration-file", "", "Path to the KubermaticConfiguration YAML file (deprecated, use --config instead)")
	cmd.PersistentFlags().StringVar(&opt.HelmValuesFile, "helm-values-file", "", "Use this values.yaml file when rendering Helm charts (deprecated, use --helm-values instead)")
	cmd.PersistentFlags().StringVar(&opt.Registry, "registry", "", "Address of the registry to push to, for example localhost:5000 (deprecated, pass registry as argument instead)")

	// TODO(embik): enable this for KKP 2.22 so the flags above cannot be used anymore
	// cmd.PersistentFlags().MarkDeprecated("configuration-file", "use --config instead")
	// cmd.PersistentFlags().MarkDeprecated("helm-values-file", "use --helm-values instead")
	// cmd.PersistentFlags().MarkDeprecated("registry", "pass registry as argument instead")

	return cmd
}

func MirrorImagesFunc(logger *logrus.Logger, versions kubermaticversion.Versions, options *MirrorImagesOptions) cobraFuncE {
	return handleErrors(logger, func(cmd *cobra.Command, args []string) error {
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
			return fmt.Errorf("failed loading CA bundle: %w", err)
		}

		kubermaticConfig, _, err := loadKubermaticConfiguration(options.Config)
		if err != nil {
			return fmt.Errorf("failed to load KubermaticConfiguration: %w", err)
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
				addonsImage = kubermaticConfig.Spec.UserCluster.Addons.DockerRepository + ":" + versions.Kubermatic
			}

			if addonsImage != "" {
				tempDir, err := images.ExtractAddonsFromDockerImage(ctx, logger, addonsImage)
				if err != nil {
					return fmt.Errorf("failed to create local addons path: %w", err)
				}
				defer os.RemoveAll(tempDir)

				options.AddonsPath = tempDir
			}
		}

		// Using a set here for deduplication
		imageSet := sets.NewString()
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

					versionLogger.Info("Collecting images...")
					images, err := images.GetImagesForVersion(
						versionLogger,
						clusterVersion,
						cloudSpec,
						cniPlugin,
						kubermaticConfig,
						options.AddonsPath,
						versions,
						caBundle,
					)
					if err != nil {
						return fmt.Errorf("failed to get images: %w", err)
					}
					imageSet.Insert(images...)
				}
			}
		}

		if options.ChartsDirectory != "" {
			chartsLogger := logger.WithField("charts-directory", options.ChartsDirectory)
			chartsLogger.Info("Rendering Helm charts")

			images, err := images.GetImagesForHelmCharts(ctx, chartsLogger, kubermaticConfig, helmClient, options.ChartsDirectory, options.HelmValuesFile)
			if err != nil {
				return fmt.Errorf("failed to get images: %w", err)
			}
			imageSet.Insert(images...)
		}

		if err := images.ProcessImages(ctx, logger, options.DryRun, imageSet.List(), options.Registry); err != nil {
			return fmt.Errorf("failed to process images: %w", err)
		}

		return nil
	})
}
