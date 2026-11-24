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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/apimachinery/pkg/util/sets"
)

type ListImagesOptions struct {
	ImageCollectionOptions

	ShowSource   bool
	ChartsOnly   bool
	OutputFormat string
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

	cmd.Flags().BoolVar(&opt.ShowSource, "show-source", false, "Show the origin of each image in an additional tab-separated column")
	cmd.Flags().BoolVar(&opt.ChartsOnly, "charts", false, "Only list the Helm charts collected from the ApplicationDefinitions")
	cmd.Flags().StringVarP(&opt.OutputFormat, "output", "o", "", "Output format for the image list. Supported formats: json (defaults to one image per line)")

	return cmd
}

func ListImagesFunc(logger *logrus.Logger, versions kubermaticversion.Versions, options *ListImagesOptions) cobraFuncE {
	return handleErrors(logger, func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if options.OutputFormat != "" && options.OutputFormat != "json" {
			return fmt.Errorf("invalid output format %q, supported formats: json", options.OutputFormat)
		}

		kubermaticConfig, err := loadAndDefaultKubermaticConfiguration(&options.ImageCollectionOptions)
		if err != nil {
			return fmt.Errorf("failed to get KubermaticConfiguration: %w", err)
		}

		collected, err := collectImages(ctx, logger, versions, kubermaticConfig, &options.ImageCollectionOptions)
		if err != nil {
			return err
		}

		return collected.writeImages(cmd.OutOrStdout(), options)
	})
}

func printImages(w io.Writer, imageSet sets.Set[string]) {
	for _, image := range sets.List(imageSet) {
		fmt.Fprintln(w, image)
	}
}

type jsonOrigin struct {
	Origin  string `json:"origin"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type jsonChartRecord struct {
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	ChartVersion string `json:"chartVersion"`
	Origin       string `json:"origin"`
	Source       string `json:"source"`
}

type jsonImageRecord struct {
	Kind    string       `json:"kind"`
	Image   string       `json:"image"`
	Origins []jsonOrigin `json:"origins"`
}

func (c *ImageCollection) writeImages(w io.Writer, options *ListImagesOptions) error {
	switch {
	case options.OutputFormat == "json":
		return c.writeJSON(w, options.ChartsOnly)
	case options.ChartsOnly:
		for _, chart := range c.sortedCharts() {
			if image := chart.flatImage(); image != "" {
				c.writeImageLine(w, image, options.ShowSource)
			}
		}
	case options.ShowSource:
		for _, image := range c.sortedImages() {
			c.writeImageLine(w, image, true)
		}
	default:
		printImages(w, c.flatImageSet())
	}

	return nil
}

func (c *ImageCollection) writeImageLine(w io.Writer, image string, showSource bool) {
	if !showSource {
		fmt.Fprintln(w, image)
		return
	}

	labels := make([]string, 0, len(c.origins[image]))
	for _, origin := range c.origins[image] {
		labels = append(labels, origin.label())
	}

	fmt.Fprintf(w, "%s\t%s\n", image, strings.Join(labels, ","))
}

func (o ImageOrigin) label() string {
	switch o.Kind {
	case originReconciler:
		return originReconciler + "@" + o.Version
	case originAddon, originApplicationDefinition:
		return o.Kind + "/" + o.Name
	default:
		return o.Kind
	}
}

func (c *ImageCollection) writeJSON(w io.Writer, chartsOnly bool) error {
	encoder := json.NewEncoder(w)

	for _, chart := range c.sortedCharts() {
		record := jsonChartRecord{
			Kind:         "chart",
			Name:         chart.Name,
			ChartVersion: chart.ChartVersion,
			Origin:       originApplicationDefinition,
			Source:       chart.Source,
		}

		if err := encoder.Encode(record); err != nil {
			return err
		}
	}

	if chartsOnly {
		return nil
	}

	for _, image := range c.sortedImages() {
		origins := make([]jsonOrigin, 0, len(c.origins[image]))
		for _, origin := range c.origins[image] {
			origins = append(origins, jsonOrigin{Origin: origin.Kind, Name: origin.Name, Version: origin.Version})
		}

		if err := encoder.Encode(jsonImageRecord{Kind: "image", Image: image, Origins: origins}); err != nil {
			return err
		}
	}

	return nil
}
