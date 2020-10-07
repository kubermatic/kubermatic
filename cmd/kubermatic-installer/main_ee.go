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
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	eeinstaller "k8c.io/kubermatic/v2/pkg/ee/cmd/kubermatic-installer"
)

func commands(logger *logrus.Logger) []cli.Command {
	return []cli.Command{
		VersionCommand(logger),
		DeployCommand(logger),
		ConvertKubeconfigCommand(logger),
		eeinstaller.ConvertDatacentersCommand(logger),
		eeinstaller.ConvertHelmValuesCommand(logger),
	}
}

func flags() []cli.Flag {
	return []cli.Flag{
		verboseFlag,
		chartsDirectoryFlag,
	}
}
