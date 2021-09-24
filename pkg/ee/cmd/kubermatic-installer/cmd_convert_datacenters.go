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
	eeprovider "k8c.io/kubermatic/v2/pkg/ee/provider"
	yamlutil "k8c.io/kubermatic/v2/pkg/util/yaml"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/yaml"
)

var (
	kubeconfigFlag = cli.StringFlag{
		Name:  "kubeconfig",
		Usage: "Kubeconfig file with contexts for every seed datacenter",
	}
)

func ConvertDatacentersCommand(logger *logrus.Logger) cli.Command {
	return cli.Command{
		Name:      "convert-datacenters",
		Usage:     "Converts legacy datacenters.yaml into Seed resources",
		Action:    ConvertDatacentersAction(logger),
		ArgsUsage: "DATACENTERS_FILE",
		Flags: []cli.Flag{
			kubeconfigFlag,
			targetNamespaceFlag,
			unpauseSeedsFlag,
		},
	}
}

func ConvertDatacentersAction(logger *logrus.Logger) cli.ActionFunc {
	return handleErrors(logger, setupLogger(logger, func(ctx *cli.Context) error {
		inputFile := ctx.Args().First()
		if inputFile == "" {
			return errors.New("no datacenters.yaml file argument given")
		}

		datacenters, err := loadDatacenters(inputFile)
		if err != nil {
			return err
		}

		var kubeconfig *clientcmdapi.Config

		kubeconfigFile := ctx.String(kubeconfigFlag.Name)
		if kubeconfigFile != "" {
			kubeconfigBytes, err := ioutil.ReadFile(kubeconfigFile)
			if err != nil {
				return fmt.Errorf("failed to read kubeconfig file %q: %v", kubeconfigFile, err)
			}

			kubeconfig, err = clientcmd.Load(kubeconfigBytes)
			if err != nil {
				return fmt.Errorf("failed to parse kubeconfig file: %v", err)
			}
		} else {
			logger.Warnf(
				"No kubeconfig (--%s) given, cannot automatically create matching "+
					"Secrets for each Seed. Please ensure to manually create kubeconfig "+
					"Secrets and to reference them in the Seeds.",
				kubeconfigFlag.Name)
		}

		namespace := ctx.String(targetNamespaceFlag.Name)
		pauseSeeds := !ctx.Bool(unpauseSeedsFlag.Name)

		resources, err := eeconversion.ConvertDatacenters(datacenters, kubeconfig, namespace, pauseSeeds)
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

type datacentersFile struct {
	Datacenters map[string]eeprovider.DatacenterMeta `json:"datacenters"`
}

func loadDatacenters(input string) (map[string]eeprovider.DatacenterMeta, error) {
	var (
		content []byte
		err     error
	)

	if input == "-" {
		content, err = ioutil.ReadAll(os.Stdin)
	} else {
		content, err = ioutil.ReadFile(input)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read '%s': %v", input, err)
	}

	dcFile := datacentersFile{}
	if err := yaml.UnmarshalStrict(content, &dcFile); err != nil {
		return nil, fmt.Errorf("failed to parse datacenters.yaml: %v", err)
	}

	return dcFile.Datacenters, nil
}
