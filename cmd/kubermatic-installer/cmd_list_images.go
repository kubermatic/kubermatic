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

	"k8c.io/kubermatic/v2/pkg/install/images"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/apimachinery/pkg/util/sets"
)

type ListImagesOptions struct {
	Options
	ImageCollectionOptions

	Format string
}

const (
	formatPlain = "plain"
	formatJSON  = "json"
)

func ListImagesCommand(logger *logrus.Logger, versions kubermaticversion.Versions) *cobra.Command {
	opt := ListImagesOptions{
		ImageCollectionOptions: ImageCollectionOptions{
			HelmTimeout: 5 * time.Minute,
			HelmBinary:  "helm",
		},
		Format: formatPlain,
	}

	cmd := &cobra.Command{
		Use:   "list-images",
		Short: "List all container images and OCI Helm chart references used by KKP",
		Long: `Lists every container image and OCI Helm chart reference that KKP would deploy,
without mirroring anything. The output goes to stdout (one ref per line by default),
progress and diagnostic logs go to stderr.

Discovery works by rendering the same deployments, addons, Helm charts and
applications that mirror-images uses, so the list is best-effort complete for the
code paths exercised with the given configuration and defaults; it is not a
guaranteed exhaustive inventory.

Network access is required: application charts are always downloaded, and vendored
charts under --charts-directory fetch their dependencies unless they are packaged
(tgz) with the dependencies bundled. When --addons-path is set, the addons Docker
image is not pulled. Runtime is dominated by chart downloads and rendering and is
typically measured in minutes, not seconds.

Use --format json to get per-image source attribution (reconciler, addon, chart,
application, etc.) and to distinguish container images from OCI Helm chart refs.`,
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

	cmd.PersistentFlags().StringVar(&opt.Config, "config", "", "Path to the KubermaticConfiguration YAML file")
	cmd.PersistentFlags().StringVar(&opt.VersionFilter, "version-filter", "", "Version constraint which can be used to filter for specific versions")
	cmd.PersistentFlags().StringArrayVar(&opt.ProviderFilter, "provider-filter", nil, fmt.Sprintf("Cloud providers to list images for. Valid values are: %s. Can be specified multiple times. If not specified, images for all providers will be listed", strings.Join(allSupportedProviderNames(), ", ")))
	cmd.PersistentFlags().StringVar(&opt.RegistryPrefix, "registry-prefix", "", "Check source registries against this prefix and only include images that match it")
	cmd.PersistentFlags().BoolVar(&opt.IgnoreRepositoryOverrides, "ignore-repository-overrides", true, "Ignore any configured registry overrides and tag suffixes in the referenced KubermaticConfiguration to reuse a configuration that already specifies overrides (note that the development-only dockerTag override is still observed and that this does not affect Helm charts configured via values.yaml; defaults to true)")

	cmd.PersistentFlags().StringVar(&opt.AddonsPath, "addons-path", "", "Path to a local directory containing KKP addons. Takes precedence over --addons-image and avoids pulling the addons Docker image")
	cmd.PersistentFlags().StringVar(&opt.AddonsImage, "addons-image", "", "Docker image containing KKP addons, if not given, falls back to the Docker image configured in the KubermaticConfiguration")

	cmd.PersistentFlags().DurationVar(&opt.HelmTimeout, "helm-timeout", opt.HelmTimeout, "time to wait for Helm operations to finish")
	cmd.PersistentFlags().StringVar(&opt.HelmValuesFile, "helm-values", "", "Use this values.yaml when rendering Helm charts")
	cmd.PersistentFlags().StringVar(&opt.HelmBinary, "helm-binary", opt.HelmBinary, "Helm 3.x or 4.x binary to use for rendering charts")

	cmd.PersistentFlags().IntVarP(&opt.Concurrency, "concurrency", "c", defaultHelmConcurrency(), "Number of Helm charts to render in parallel. Set to 1 for the historical sequential behavior. Chart dependency registration is serialized internally, so this is safe for concurrent helm repo add.")

	cmd.PersistentFlags().StringVar(&opt.Format, "format", opt.Format, fmt.Sprintf("Output format. Supported formats: %q (one ref per line on stdout), %q (array of objects with ref, type and sources)", formatPlain, formatJSON))

	return cmd
}

func ListImagesFunc(logger *logrus.Logger, versions kubermaticversion.Versions, options *ListImagesOptions) cobraFuncE {
	return handleErrors(logger, func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if err := validateListImagesFormat(options.Format); err != nil {
			return err
		}

		collection, err := collectAllImages(ctx, logger, versions, &options.ImageCollectionOptions, options.ChartsDirectory)
		if err != nil {
			return err
		}

		return printCollection(cmd.OutOrStdout(), collection, options.Format)
	})
}

func validateListImagesFormat(format string) error {
	switch format {
	case formatPlain, formatJSON:
		return nil
	default:
		return fmt.Errorf("invalid --format %q, supported formats are %q and %q", format, formatPlain, formatJSON)
	}
}

// printCollection writes the collection to out in the requested format. plain
// emits one ref per line with no decoration; json emits a deterministically
// ordered array of {ref, type, sources} objects.
func printCollection(out io.Writer, collection *images.Collection, format string) error {
	switch format {
	case formatPlain:
		for _, ref := range collection.RefList() {
			if _, err := fmt.Fprintln(out, ref); err != nil {
				return fmt.Errorf("failed to write output: %w", err)
			}
		}
		return nil
	case formatJSON:
		entries := collection.Sorted()
		output := make([]listImageEntry, 0, len(entries))
		for _, entry := range entries {
			sources := sets.List(entry.Sources)
			if len(sources) == 0 {
				sources = nil
			}
			output = append(output, listImageEntry{
				Ref:     entry.Ref,
				Type:    string(entry.Type),
				Sources: sources,
			})
		}

		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(output); err != nil {
			return fmt.Errorf("failed to write JSON output: %w", err)
		}
		return nil
	default:
		// format is validated before collection runs; reaching here is a programming error
		return fmt.Errorf("unsupported format %q", format)
	}
}

// listImageEntry is the JSON representation of a single collected ref.
type listImageEntry struct {
	Ref     string   `json:"ref"`
	Type    string   `json:"type"`
	Sources []string `json:"sources"`
}
