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

package seedcontrollermanager

import (
	"context"
	"flag"

	eeprovider "k8c.io/kubermatic/v2/pkg/ee/provider"
	"k8c.io/kubermatic/v2/pkg/provider"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	dynamicDatacenters = false
	datacentersFile    = ""
)

func AddFlags(fs *flag.FlagSet) {
	fs.BoolVar(&dynamicDatacenters, "dynamic-datacenters", false, "Whether to enable dynamic datacenters. Enabling this and defining the datcenters flag will enable the migration of the datacenters defined in datancenters.yaml to Seed custom resources.")
	fs.StringVar(&datacentersFile, "datacenters", "", "The datacenters.yaml file path.")
}

func SeedGetterFactory(ctx context.Context, client ctrlruntimeclient.Reader, seedName string, namespace string) (provider.SeedGetter, error) {
	if dynamicDatacenters {
		return provider.SeedGetterFactory(ctx, client, seedName, namespace)
	}

	return eeprovider.SeedGetterFactory(ctx, client, datacentersFile, seedName)
}
