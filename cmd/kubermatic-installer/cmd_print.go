/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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
	"strings"

	"github.com/spf13/cobra"

	"k8c.io/kubermatic/v3/docs"
)

func PrintCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "print [config | kubermaticconfiguration | datacenter]",
		Short: "Print example configuration manifest",
		Long:  "Prints an example configuration manifest with defaults for the given resource.\n   Supported resources are \"datacenter\" and \"kubermaticconfiguration\".",
		RunE:  PrintFunc(),
	}

	return cmd
}

func PrintFunc() cobraFuncE {
	return func(cmd *cobra.Command, args []string) error {
		arg := ""
		if len(args) > 0 {
			arg = strings.ToLower(args[0])
		}

		switch arg {
		case "", "config", "kubermaticconfiguration":
			fmt.Println(docs.ExampleKubermaticConfiguration)
		case "datacenter":
			fmt.Println(docs.ExampleDatacenterConfiguration)
		default:
			return cmd.Usage()
		}

		return nil
	}
}
