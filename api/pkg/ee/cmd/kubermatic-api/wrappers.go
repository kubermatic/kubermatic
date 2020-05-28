// +build ee

package kubermaticapi

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

func SeedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (provider.SeedsGetter, error) {
	return eeprovider.SeedsGetterFactory(ctx, client, datacentersFile, namespace, dynamicDatacenters)
}

func SeedKubeconfigGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, kubeconfig string) (provider.SeedKubeconfigGetter, error) {
	if dynamicDatacenters {
		return provider.SeedKubeconfigGetterFactory(ctx, client)
	}

	return eeprovider.SeedKubeconfigGetterFactory(kubeconfig)
}
