package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/urfave/cli"
	"go.uber.org/zap"
	yaml3 "gopkg.in/yaml.v3"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/conversion"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/test-infra/pkg/genyaml"
	"sigs.k8s.io/yaml"
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
		{
			Name:      "defaults",
			Usage:     "Outputs a KubermaticConfiguration with all default values, optionally applied to a given configuration manifest (YAML)",
			Action:    defaultsAction,
			ArgsUsage: "[MANIFEST_FILE]",
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

	for i, resource := range resources {
		if err := printYAML(resource); err != nil {
			return cli.NewExitError(fmt.Errorf("failed to create YAML: %v", err), 1)
		}

		if i < len(resources)-1 {
			fmt.Println("\n---")
		}
	}

	return nil
}

func defaultsAction(ctx *cli.Context) error {
	config := &operatorv1alpha1.KubermaticConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "operator.kubermatic.io/v1alpha1",
			Kind:       "KubermaticConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubermatic",
			Namespace: "kubermatic",
		},
	}

	configFile := ctx.Args().First()
	if configFile != "" {
		content, err := ioutil.ReadFile(configFile)
		if err != nil {
			return cli.NewExitError(fmt.Errorf("failed to read file %s: %v", configFile, err), 1)
		}

		if err := yaml.Unmarshal(content, &config); err != nil {
			return cli.NewExitError(fmt.Errorf("failed to parse file %s as YAML: %v", configFile, err), 1)
		}
	}

	logger := zap.NewNop().Sugar()
	defaulted, err := common.DefaultConfiguration(config, logger)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed to create KubermaticConfiguration: %v", err), 1)
	}

	if err := printYAML(defaulted); err != nil {
		return cli.NewExitError(fmt.Errorf("failed to create YAML: %v", err), 1)
	}

	return nil
}

func printYAML(resource interface{}) error {
	encoder := yaml3.NewEncoder(os.Stdout)
	encoder.SetIndent(2)

	// genyaml is smart enough to not output a creationTimestamp when marshalling as YAML
	return genyaml.NewCommentMap().EncodeYaml(resource, encoder)
}
