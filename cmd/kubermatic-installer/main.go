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

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

type Options struct {
	Verbose         bool
	LogFormat       log.LogrusFormat
	ChartsDirectory string
}

func (o *Options) CopyInto(other *Options) {
	other.ChartsDirectory = o.ChartsDirectory
	other.Verbose = o.Verbose
}

var options = &Options{}

func main() {
	logger := log.NewLogrus()
	versions := kubermatic.GetVersions()

	rootCmd := &cobra.Command{
		Use:           "kubermatic-installer",
		Short:         "Installs and updates Kubermatic Kubernetes Platform",
		Version:       versions.GitVersion,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if env := os.Getenv("KUBERMATIC_CHARTS_DIRECTORY"); env != "" {
				options.ChartsDirectory = env
			}

			if options.ChartsDirectory == "" {
				options.ChartsDirectory = "charts"
			}

			if options.Verbose {
				logger.SetLevel(logrus.DebugLevel)
			}

			switch options.LogFormat {
			case log.LogrusFormatJSON:
				logger.SetFormatter(&logrus.JSONFormatter{})
			case "", log.LogrusFormatConsole:
				logger.SetFormatter(&logrus.TextFormatter{})
			}
		},
	}

	// cobra does not make any distinction between "error that happened because of bad flags"
	// and "error that happens because of something going bad inside the RunE function", and
	// so would always show the Usage, no matter what error occurred. Tow ork around this, we
	// set SilenceUsages on all commands and manually print the error using the FlagErrorFunc.
	rootCmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		if err := c.Usage(); err != nil {
			return err
		}

		// ensure we exit with code 1 later on
		return err
	})

	rootCmd.PersistentFlags().BoolVarP(&options.Verbose, "verbose", "v", options.Verbose, "enable more verbose output")
	rootCmd.PersistentFlags().VarP(&options.LogFormat, "output", "o", fmt.Sprintf("write logs in a specific output format. supported formats: %s", log.AvailableLogrusFormats.String()))
	rootCmd.PersistentFlags().StringVar(&options.ChartsDirectory, "charts-directory", "", "filesystem path to the Kubermatic Helm charts (defaults to charts/)")

	addCommands(rootCmd, logger, versions)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
