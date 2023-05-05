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
	"io/ioutil"
	"os"

	installinit "k8c.io/kubermatic/v2/pkg/install/init"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type InitOptions struct {
	DebugLogPath string

	Interactive bool
}

func InitCommand() *cobra.Command {
	opt := InitOptions{}

	cmd := &cobra.Command{
		Use:          "init",
		Short:        "Run an interactive configurazion wizard",
		RunE:         InitFunc(&opt),
		SilenceUsage: true,
	}

	cmd.PersistentFlags().StringVar(&opt.DebugLogPath, "debug-log-path", "", "file location for debug logging")
	cmd.PersistentFlags().BoolVarP(&opt.Interactive, "interactive", "i", false, "interactive mode to walk through options required for generating configuration files")

	return cmd
}

func InitFunc(opt *InitOptions) cobraFuncE {
	return func(cmd *cobra.Command, args []string) error {
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

			return installinit.Run(logger)
		} else {
			return nil
		}
	}
}
