// +build ee

package seedcontrollermanager

import (
	"context"
	"flag"

	eeprovider "github.com/kubermatic/kubermatic/api/pkg/ee/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

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

func SeedGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, seedName string, namespace string) (provider.SeedGetter, error) {
	if dynamicDatacenters {
		return provider.SeedGetterFactory(ctx, client, seedName, namespace)
	}

	return eeprovider.SeedGetterFactory(ctx, client, datacentersFile, seedName)
}
