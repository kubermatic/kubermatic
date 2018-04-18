package main

import (
	"log"
	"os"

	"github.com/urfave/cli"
	_ "github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()

	outputDirFlag := cli.StringFlag{
		Name:  "output-dir, o",
		Value: "./",
		Usage: "Path to a directory in which the configs should be saved",
	}

	app.Commands = []cli.Command{
		{
			Name:   "create-cluster-configs",
			Usage:  "generates Container Linux configs for the given config",
			Action: generateContainerLinuxConfigs,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "config, c",
					Value: "config.yaml",
					Usage: "Path to the configuration file",
				},
				cli.StringFlag{
					Name:  "template, t",
					Value: "template.yaml",
					Usage: "Path to the container linux config template to use",
				},
				outputDirFlag,
			},
		},
		{
			Name:   "create-ca",
			Usage:  "creates a ca",
			Action: createCA,
			Flags: []cli.Flag{
				outputDirFlag,
				cli.StringFlag{
					Name:  "common-name",
					Value: "cluster",
					Usage: "Common name to use for the CA",
				},
			},
		},
		{
			Name:   "create-service-account-key",
			Usage:  "creates a service account key",
			Action: createSA,
			Flags: []cli.Flag{
				outputDirFlag,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
