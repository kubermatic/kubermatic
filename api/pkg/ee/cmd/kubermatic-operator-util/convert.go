// +build ee

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
