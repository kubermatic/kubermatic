// +build ee

package mastercontrollermanager

import (
	"context"
	"flag"
	"fmt"

	"go.uber.org/zap"

	eemigrations "github.com/kubermatic/kubermatic/api/pkg/ee/crd/migrations/master"
	eeprovider "github.com/kubermatic/kubermatic/api/pkg/ee/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/validation/seed"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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

func SeedKubeconfigGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, kubeconfig string) (provider.SeedKubeconfigGetter, error) {
	if dynamicDatacenters {
		return provider.SeedKubeconfigGetterFactory(ctx, client)
	}

	return eeprovider.SeedKubeconfigGetterFactory(kubeconfig)
}

func SetupSeedValidationWebhook(
	ctx context.Context,
	mgr manager.Manager,
	log *zap.SugaredLogger,
	webhookOpt seed.WebhookOpts,
	namespace string,
	seedsGetter provider.SeedsGetter,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
	kubeconfig string,
	workerName string,
) error {
	migrationOptions, err := getMigrationOptions(kubeconfig)
	if err != nil {
		return err
	}

	server, err := webhookOpt.Server(
		ctx,
		log,
		namespace,
		workerName,
		seedsGetter,
		provider.SeedClientGetterFactory(seedKubeconfigGetter),
		migrationOptions.SeedMigrationEnabled())
	if err != nil {
		return fmt.Errorf("failed to create server: %v", err)
	}

	if err := mgr.Add(server); err != nil {
		return fmt.Errorf("failed to add server to mgr: %v", err)
	}

	return nil
}

func getMigrationOptions(kubeconfig string) (*eemigrations.Options, error) {
	var (
		clientConfig *clientcmdapi.Config
		err          error
	)

	if len(kubeconfig) > 0 {
		clientConfig, err = clientcmd.LoadFromFile(kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to read the kubeconfig: %v", err)
		}
	}

	return &eemigrations.Options{
		DatacentersFile:    datacentersFile,
		DynamicDatacenters: dynamicDatacenters,
		Kubeconfig:         clientConfig,
	}, nil
}

func RunMigrations(ctx context.Context, client ctrlruntimeclient.Client, log *zap.SugaredLogger, namespace string, kubeconfig string) error {
	options, err := getMigrationOptions(kubeconfig)
	if err != nil {
		return err
	}

	if options.MigrationEnabled() {
		log.Info("executing migrations...")

		if err := eemigrations.RunAll(ctx, log, client, namespace, *options); err != nil {
			return fmt.Errorf("failed to run migrations: %v", err)
		}

		log.Info("migrations executed successfully")
	}

	return nil
}
