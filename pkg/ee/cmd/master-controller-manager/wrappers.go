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

package mastercontrollermanager

import (
	"context"
	"flag"
	"fmt"

	"go.uber.org/zap"

	eeprovider "github.com/kubermatic/kubermatic/pkg/ee/provider"
	"github.com/kubermatic/kubermatic/pkg/provider"
	"github.com/kubermatic/kubermatic/pkg/validation/seed"
	seedvalidation "github.com/kubermatic/kubermatic/pkg/validation/seed"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	dynamicDatacenters = false
	datacentersFile    = ""
)

func AddFlags(fs *flag.FlagSet) {
	fs.BoolVar(&dynamicDatacenters, "dynamic-datacenters", false, "Whether to enable dynamic datacenters. Enabling this and defining the datcenters flag will enable the migration of the datacenters defined in datancenters.yaml to Seed custom resources.")
	fs.StringVar(&datacentersFile, "datacenters", "", "The datacenters.yaml file path.")
}

func SeedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (provider.SeedsGetter, error) {
	return eeprovider.SeedsGetterFactory(ctx, client, datacentersFile, namespace, dynamicDatacenters)
}

func SeedKubeconfigGetterFactory(ctx context.Context, client ctrlruntimeclient.Client) (provider.SeedKubeconfigGetter, error) {
	if dynamicDatacenters {
		return provider.SeedKubeconfigGetterFactory(ctx, client)
	}

	return eeprovider.SeedKubeconfigGetter, nil
}

func SetupSeedValidationWebhook(
	ctx context.Context,
	mgr manager.Manager,
	log *zap.SugaredLogger,
	webhookOpt seed.WebhookOpts,
	namespace string,
	seedsGetter provider.SeedsGetter,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
	workerName string,
) error {
	// Creates a new default validator
	validator, err := seedvalidation.NewDefaultSeedValidator(
		workerName,
		seedsGetter,
		provider.SeedClientGetterFactory(seedKubeconfigGetter),
	)
	if err != nil {
		return fmt.Errorf("failed to create seed validator webhook server: %v", err)
	}
	server, err := webhookOpt.Server(
		ctx,
		log,
		namespace,
		validator.Validate,
	)
	if err != nil {
		return fmt.Errorf("failed to create server: %v", err)
	}

	if err := mgr.Add(server); err != nil {
		return fmt.Errorf("failed to add server to mgr: %v", err)
	}

	return nil
}
