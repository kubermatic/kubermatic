/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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
	"io/fs"
	"io/ioutil"
	"os"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	installinit "k8c.io/kubermatic/v2/pkg/install/init"
	"k8c.io/kubermatic/v2/pkg/install/init/generator"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type InitOptions struct {
	DebugLogPath string
	OutputDir    string

	Interactive bool

	// these options are only relevant if non-interactive config generation is used.
	Hostname        string
	ExposeStrategy  string
	GenerateSecrets bool
}

func InitCommand(cmdLogger *logrus.Logger) *cobra.Command {
	opt := InitOptions{}

	cmd := &cobra.Command{
		Use:          "init",
		Short:        "Run an interactive configurazion wizard",
		RunE:         InitFunc(&opt, cmdLogger),
		SilenceUsage: true,
		// TODO(embik): make this command GA once it's in a good shape.
		Hidden: true,
	}

	cmd.PersistentFlags().StringVarP(&opt.OutputDir, "output-dir", "d", ".", "directory to write generated configuration files to")
	cmd.PersistentFlags().StringVar(&opt.DebugLogPath, "debug-log-path", "", "file location for debug logging")
	cmd.PersistentFlags().BoolVarP(&opt.Interactive, "interactive", "i", false, "interactive mode to walk through options required for generating configuration files")

	cmd.PersistentFlags().StringVar(&opt.Hostname, "hostname", "", "DNS suffix used to construct hostnames")
	cmd.PersistentFlags().StringVar(&opt.ExposeStrategy, "expose-strategy", "", "Expose Strategy to use for the generated configuration")
	cmd.PersistentFlags().BoolVar(&opt.GenerateSecrets, "generate-secrets", true, "Generate secrets required for KKP configuration")

	return cmd
}

func InitFunc(opt *InitOptions, cmdLogger *logrus.Logger) cobraFuncE {
	return handleErrors(cmdLogger, func(cmd *cobra.Command, args []string) error {
		stat, err := os.Stat(opt.OutputDir)
		if !os.IsNotExist(err) && err != nil {
			return fmt.Errorf("failed to check output directory: %v", err)
		}

		if errors.Is(err, fs.ErrNotExist) {
			if err := os.MkdirAll(opt.OutputDir, os.ModePerm); err != nil {
				return fmt.Errorf("failed to create output directory: %v", err)
			}
		}

		if !stat.IsDir() {
			return fmt.Errorf("output directory is not a directory")
		}

		if opt.Interactive {
			logger := logrus.New()
			if opt.DebugLogPath != "" {
				logFile, err := os.Create(opt.DebugLogPath)
				if err != nil {
					return err
				}
				logger.SetOutput(logFile)
				logger.SetLevel(logrus.DebugLevel)
				logger.SetFormatter(&logrus.JSONFormatter{})
				defer logFile.Close()
			} else {
				// if no debug-log-path is set, we do not want to log anything.
				logger.SetOutput(ioutil.Discard)
			}

			return installinit.Run(logger, opt.OutputDir)
		} else {
			config := generator.Config{
				DNS:             opt.Hostname,
				ExposeStrategy:  kubermaticv1.ExposeStrategy(opt.ExposeStrategy),
				GenerateSecrets: opt.GenerateSecrets,
			}
			return generator.Generate(config, opt.OutputDir, cmdLogger)
		}
	})
}
