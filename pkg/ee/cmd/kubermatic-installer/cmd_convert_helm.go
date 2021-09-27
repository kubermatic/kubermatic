//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2020 Loodse GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package kubermaticinstaller

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	eeconversion "k8c.io/kubermatic/v2/pkg/ee/conversion"
	yamlutil "k8c.io/kubermatic/v2/pkg/util/yaml"
)

var (
	skipDatacentersFlag = cli.BoolFlag{
		Name:  "skip-datacenters",
		Usage: "Do not extract and convert included datacenters structure",
	}

	skipPresetsFlag = cli.BoolFlag{
		Name:  "skip-presets",
		Usage: "Do not extract and convert included presets structure",
	}
)

func ConvertHelmValuesCommand(logger *logrus.Logger) cli.Command {
	return cli.Command{
		Name:      "convert-helm-values",
		Usage:     "Converts legacy Helm values.yaml into new CRD-based configuration",
		Action:    ConvertHelmValuesAction(logger),
		ArgsUsage: "VALUES_YAML_FILE",
		Flags: []cli.Flag{
			skipDatacentersFlag,
			skipPresetsFlag,
			targetNamespaceFlag,
			unpauseSeedsFlag,
		},
	}
}

func ConvertHelmValuesAction(logger *logrus.Logger) cli.ActionFunc {
	return handleErrors(logger, setupLogger(logger, func(ctx *cli.Context) error {
		valuesFile := ctx.Args().First()
		if valuesFile == "" {
			return errors.New("no values.yaml file argument given")
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
			return fmt.Errorf("failed to read '%s': %v", valuesFile, err)
		}

		opt := eeconversion.Options{
			Namespace:      ctx.String(targetNamespaceFlag.Name),
			IncludeSeeds:   !ctx.Bool(skipDatacentersFlag.Name),
			IncludePresets: !ctx.Bool(skipPresetsFlag.Name),
			PauseSeeds:     !ctx.Bool(unpauseSeedsFlag.Name),
		}

		resources, err := eeconversion.HelmValuesFileToCRDs(content, opt)
		if err != nil {
			return fmt.Errorf("failed to convert: %v", err)
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
	}))
}
