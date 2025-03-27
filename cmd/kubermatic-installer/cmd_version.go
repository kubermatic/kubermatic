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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
)

type VersionOptions struct {
	Options

	Short bool
}

func VersionCommand(logger *logrus.Logger, versions kubermatic.Versions) *cobra.Command {
	opt := VersionOptions{}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the installer version",
		RunE:  VersionFunc(logger, versions, &opt),
		PreRun: func(cmd *cobra.Command, args []string) {
			options.CopyInto(&opt.Options)
		},
		SilenceUsage: true,
	}

	cmd.PersistentFlags().BoolVarP(&opt.Short, "short", "s", false, "omit git and chart information")

	return cmd
}

func VersionFunc(logger *logrus.Logger, versions kubermatic.Versions, opt *VersionOptions) cobraFuncE {
	return handleErrors(logger, func(cmd *cobra.Command, args []string) error {
		name := fmt.Sprintf("Kubermatic %s Installer", versions.KubermaticEdition)

		if opt.Short {
			fmt.Printf("%s %s\n", name, versions.GitVersion)
			return nil
		}

		charts, err := loadCharts(opt.ChartsDirectory)
		if err != nil {
			return fmt.Errorf("failed to determine installer chart state: %w", err)
		}

		nameWidth := len(name)
		versionWidth := len("Version")
		appVersionWidth := len("App Version")

		if l := len(versions.GitVersion); l > versionWidth {
			versionWidth = l
		}

		if l := len(versions.GitVersion); l > appVersionWidth {
			appVersionWidth = l
		}

		for _, chart := range charts {
			if l := len(chart.Name); l > nameWidth {
				nameWidth = l
			}

			if l := len(chart.Version.String()); l > versionWidth {
				versionWidth = l
			}

			if l := len(chart.AppVersion); l > appVersionWidth {
				appVersionWidth = l
			}
		}

		format := fmt.Sprintf("%%-%ds | %%-%ds | %%-%ds\n", nameWidth, versionWidth, appVersionWidth)

		fmt.Printf(format, "Component", "Version", "App Version")
		fmt.Printf("%s-+-%s-+-%s-\n", strings.Repeat("-", nameWidth), strings.Repeat("-", versionWidth), strings.Repeat("-", appVersionWidth))
		fmt.Printf(format, name, versions.GitVersion, versions.GitVersion)

		for _, chart := range charts {
			fmt.Printf(format, chart.Name, chart.Version, chart.AppVersion)
		}

		return nil
	})
}

// HelmCharts is used to sort Helm charts by their name.
type HelmCharts []helm.Chart

func (a HelmCharts) Len() int { return len(a) }

func (a HelmCharts) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a HelmCharts) Less(i, j int) bool { return a[i].Name < a[j].Name }

func loadCharts(chartDirectory string) ([]helm.Chart, error) {
	charts := HelmCharts{}

	err := filepath.Walk(chartDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			chartFile := filepath.Join(path, "Chart.yaml")

			if _, err := os.Stat(chartFile); err == nil {
				chart, err := helm.LoadChart(path)
				if err != nil {
					return fmt.Errorf("failed to read %s: %w", chartFile, err)
				}

				charts = append(charts, *chart)

				return filepath.SkipDir
			}
		}

		return nil
	})

	sort.Sort(charts)

	return charts, err
}
