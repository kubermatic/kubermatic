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
	installinit "k8c.io/kubermatic/v2/pkg/install/init"

	"github.com/spf13/cobra"
)

func InitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "init",
		Short:        "Run an interactive configurazion wizard",
		RunE:         InitFunc(),
		SilenceUsage: true,
	}

	return cmd
}

func InitFunc() cobraFuncE {
	return func(cmd *cobra.Command, args []string) error {
		return installinit.Run()
	}
}
