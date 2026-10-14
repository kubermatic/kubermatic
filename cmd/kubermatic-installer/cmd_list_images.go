/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/apimachinery/pkg/util/sets"
)

type ListImagesOptions struct {
	ImageCollectionOptions
}

func ListImagesCommand(logger *logrus.Logger, versions kubermaticversion.Versions) *cobra.Command {
	opt := ListImagesOptions{
		ImageCollectionOptions: ImageCollectionOptions{
			HelmTimeout: 5 * time.Minute,
			HelmBinary:  "helm",
		},
	}

	cmd := &cobra.Command{
		Use:   "list-images",
		Short: "Print all container images used by KKP",
		Long:  "Collects all container images used by KKP and prints them, sorted and deduplicated, one per line to stdout",
		Args:  cobra.NoArgs,
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

			opt.Versions = versions
		},

		RunE:         ListImagesFunc(logger, versions, &opt),
		SilenceUsage: true,
	}

	addImageCollectionFlags(cmd, &opt.ImageCollectionOptions, "list")

	return cmd
}

func ListImagesFunc(logger *logrus.Logger, versions kubermaticversion.Versions, options *ListImagesOptions) cobraFuncE {
	return handleErrors(logger, func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		kubermaticConfig, err := loadAndDefaultKubermaticConfiguration(&options.ImageCollectionOptions)
		if err != nil {
			return fmt.Errorf("failed to get KubermaticConfiguration: %w", err)
		}

		imageSet, err := collectImages(ctx, logger, versions, kubermaticConfig, &options.ImageCollectionOptions)
		if err != nil {
			return err
		}

		printImages(cmd.OutOrStdout(), imageSet)

		return nil
	})
}

func printImages(w io.Writer, imageSet sets.Set[string]) {
	for _, image := range sets.List(imageSet) {
		fmt.Fprintln(w, image)
	}
}
