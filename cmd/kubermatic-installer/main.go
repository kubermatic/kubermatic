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
	"math/rand"
	"os"
	"time"

	"github.com/urfave/cli"

	"k8c.io/kubermatic/v2/pkg/log"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"
)

var (
	verboseFlag = cli.BoolFlag{
		Name:  "verbose, v",
		Usage: "enable more verbose output",
	}

	chartsDirectoryFlag = cli.StringFlag{
		Name:   "charts-directory",
		Value:  "charts",
		Usage:  "filesystem path to the Kubermatic Helm charts",
		EnvVar: "KUBERMATIC_CHARTS_DIRECTORY",
	}
)

func main() {
	rand.Seed(time.Now().UnixNano())

	logger := log.NewLogrus()
	versions := kubermaticversion.NewDefaultVersions()

	app := cli.NewApp()
	app.Name = "kubermatic-installer"
	app.Usage = "Installs and updates Kubermatic Kubernetes Platform"
	app.Version = versions.Kubermatic
	app.HideVersion = true
	app.Flags = flags()
	app.Commands = commands(logger, versions)

	_ = app.Run(os.Args)
}
