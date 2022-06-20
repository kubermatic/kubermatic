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
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type MirrorImagesOptions struct {
	Options

	ConfigFile     string
	VersionFilter  string
	Registry       string
	DryRun         bool
	AddonsPath     string
	AddonsImage    string
	HelmValuesFile string
	HelmBinary     string
}

func MirrorImagesCommand(logger *logrus.Logger) *cobra.Command {
	opt := MirrorImagesOptions{}

	cmd := &cobra.Command{
		Use:   "mirror-images",
		Short: "mirror images used by KKP to a private image registry",
		Long:  "",
		PreRun: func(cmd *cobra.Command, args []string) {
			options.CopyInto(&opt.Options)

			if opt.ConfigFile == "" {
				opt.ConfigFile = os.Getenv("CONFIG_YAML")
			}

			if opt.HelmValuesFile == "" {
				opt.HelmValuesFile = os.Getenv("HELM_VALUES")
			}

			if opt.HelmBinary == "" {
				opt.HelmBinary = os.Getenv("HELM_BINARY")
			}
		},

		RunE:         MirrorImagesFunc(logger, &opt),
		SilenceUsage: true,
	}

	cmd.PersistentFlags().StringVar(&opt.ConfigFile, "config", "", "Path to the KubermaticConfiguration YAML file")
	cmd.PersistentFlags().StringVar(&opt.VersionFilter, "version-filter", "", "Version constraint which can be used to filter for specific versions")
	cmd.PersistentFlags().StringVar(&opt.Registry, "registry", "", "Address of the registry to push to, for example localhost:5000")
	cmd.PersistentFlags().BoolVar(&opt.DryRun, "dry-run", false, "Only print the names of found images")
	cmd.PersistentFlags().StringVar(&opt.AddonsPath, "addons-path", "", "Address of the registry to push to, for example localhost:5000")
	cmd.PersistentFlags().StringVar(&opt.AddonsImage, "addons-image", "", "Docker image containing KKP addons, if not given, falls back to the Docker image configured in the KubermaticConfiguration")
	cmd.PersistentFlags().StringVar(&opt.HelmValuesFile, "helm-values-file", "", "Use this values.yaml file when rendering Helm charts")
	cmd.PersistentFlags().StringVar(&opt.HelmBinary, "helm-binary", "helm", "Helm 3.x binary to use for rendering charts")

	return cmd
}

func MirrorImagesFunc(logger *logrus.Logger, options *MirrorImagesOptions) cobraFuncE {
	return handleErrors(logger, func(cmd *cobra.Command, args []string) error {
		return nil
	})
}
