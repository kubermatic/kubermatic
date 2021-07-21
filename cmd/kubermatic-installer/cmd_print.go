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
	"strings"

	"github.com/urfave/cli"

	"k8c.io/kubermatic/v2/docs"
)

func PrintCommand() cli.Command {
	return cli.Command{
		Name:        "print",
		Usage:       "Prints example configuration manifest",
		UsageText:   "kubermatic-installer print <resource>",
		Description: "Print an example configuration manifest with defaults for the given resource.\n   Supported resources are \"seed\" and \"kubermaticconfiguration\".",
		ArgsUsage:   "Supported resources are \"seed\" and \"kubermaticconfiguration\".",
		Action:      PrintAction(),
	}
}

func PrintAction() cli.ActionFunc {
	return func(ctx *cli.Context) error {

		arg := strings.ToLower(ctx.Args().First())

		switch arg {
		case "", "config", "kubermaticconfiguration":
			println(docs.ExampleKubermaticConfiguration)
		case "seed":
			println(docs.ExampleSeedConfiguration)
		default:
			println(ctx.Command.ArgsUsage)
		}

		return nil
	}
}
