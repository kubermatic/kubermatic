package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/urfave/cli"
	"go.uber.org/zap"
	"k8s.io/test-infra/pkg/genyaml"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/conversion"
	"github.com/kubermatic/kubermatic/api/pkg/log"
)

var logger *zap.SugaredLogger

func main() {
	app := cli.NewApp()
	app.Name = "Kubermatic Operator Utility"
	app.Version = "v1.0.0"

	defaultLogFormat := log.FormatConsole

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "log-debug",
			Usage: "Enables more verbose logging",
		},
		cli.GenericFlag{
			Name:  "log-format",
			Value: &defaultLogFormat,
			Usage: fmt.Sprintf("Use one of [%v] to change the log output", log.AvailableFormats),
		},
	}

	app.Commands = []cli.Command{
		{
			Name:      "convert",
			Usage:     "Convert a Helm values.yaml to a KubermaticConfiguration manifest (YAML)",
			Action:    convertAction,
			ArgsUsage: "VALUES_FILE",
		},
	}

	// setup logging
	app.Before = func(c *cli.Context) error {
		format := c.GlobalGeneric("log-format").(*log.Format)
		rawLog := log.New(c.GlobalBool("log-debug"), *format)
		logger = rawLog.Sugar()

		return nil
	}

	err := app.Run(os.Args)
	// Only log failures when the logger has been setup, otherwise
	// we know it's been a CLI parsing failure and the cli package
	// has already output the error and printed the usage hints.
	if err != nil && logger != nil {
		logger.Fatalw("Failed to run command", zap.Error(err))
	}
}

func convertAction(ctx *cli.Context) error {
	valuesFile := ctx.Args().First()
	if valuesFile == "" {
		return cli.NewExitError("no values.yaml file given", 2)
	}

	var (
		content []byte
		err     error
	)

	if valuesFile == "-" {
		content, err = ioutil.ReadAll(os.Stdin)
	} else {
		content, err = ioutil.ReadFile(valuesFile)
	}
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed to read '%s': %v", valuesFile, err), 1)
	}

	resources, err := conversion.HelmValuesFileToCRDs(content, "kubermatic")
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	// genyaml is smart enough to not output a creationTimestamp when marshalling as YAML
	cm := genyaml.NewCommentMap()

	for i, resource := range resources {
		output, err := cm.GenYaml(resource)
		if err != nil {
			return cli.NewExitError(fmt.Errorf("failed to create YAML: %v", err), 1)
		}

		// reduce indentation
		output = strings.Replace(output, "    ", "  ", -1)

		fmt.Print(output)

		if i < len(resources)-1 {
			fmt.Println("\n---")
		}
	}

	return nil
}
