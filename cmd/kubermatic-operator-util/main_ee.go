// +build ee

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
	"github.com/urfave/cli"

	util "github.com/kubermatic/kubermatic/api/pkg/ee/cmd/kubermatic-operator-util"
)

func extraCommands() []cli.Command {
	return []cli.Command{
		{
			Name:      "convert",
			Usage:     "Convert a Helm values.yaml to a KubermaticConfiguration manifest (YAML)",
			Action:    util.ConvertAction,
			ArgsUsage: "VALUES_FILE",
		},
	}
}
