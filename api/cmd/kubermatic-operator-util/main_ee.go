// +build ee

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
