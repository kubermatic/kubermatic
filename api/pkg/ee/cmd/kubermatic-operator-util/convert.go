// +build ee

package util

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/urfave/cli"

	"github.com/kubermatic/kubermatic/api/pkg/ee/conversion"
	yamlutil "github.com/kubermatic/kubermatic/api/pkg/util/yaml"
)

func ConvertAction(ctx *cli.Context) error {
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
		if err := yamlutil.Encode(resource, os.Stdout); err != nil {
			return cli.NewExitError(fmt.Errorf("failed to create YAML: %v", err), 1)
		}

		if i < len(resources)-1 {
			fmt.Println("\n---")
		}
	}

	return nil
}
