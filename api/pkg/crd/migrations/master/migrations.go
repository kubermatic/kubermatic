package master

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type MigrationOptions struct {
	DatacentersFile    string
	DynamicDatacenters bool
}

func (o MigrationOptions) SeedMigrationEnabled() bool {
	return o.DatacentersFile != "" && o.DynamicDatacenters
}

// RunAll runs all migrations that should be run inside a master cluster.
func RunAll(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, kubermaticNamespace string, opt MigrationOptions) error {
	if opt.SeedMigrationEnabled() {
		log.Info("datacenters given and dynamic datacenters enabled, attempting to migrate datacenters to Seeds")
		if err := migrateDatacenters(ctx, log, client, kubermaticNamespace, opt.DatacentersFile); err != nil {
			return fmt.Errorf("failed to migrate datacenters.yaml: %v", err)
		}
		log.Info("migration completed successfully")
	}

	return nil
}

// migrateDatacenters creates Seed CRs based on the given datacenters file.
// Seeds are only ever created and never updated/reconciled to match the
// datacenters.yaml, because the validation webhook prevents any modifications
// while the migration is enabled.
func migrateDatacenters(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, kubermaticNamespace string, dcFile string) error {
	seeds, err := provider.LoadSeeds(dcFile)
	if err != nil {
		return fmt.Errorf("failed to load %s: %v", dcFile, err)
	}

	for name, seed := range seeds {
		log := log.With("seed", seed)

		seed.Namespace = kubermaticNamespace

		key, err := ctrlruntimeclient.ObjectKeyFromObject(seed)
		if err != nil {
			return fmt.Errorf("failed to create object key for seed %s: %v", name, err)
		}

		log.Info("checking for Seed existence...")
		existingSeed := kubermaticv1.Seed{}
		err = client.Get(ctx, key, &existingSeed)
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return fmt.Errorf("failed to get Seed %s: %v", name, err)
			}

			log.Info("creating Seed CR...")
			if err := client.Create(ctx, seed); err != nil {
				return fmt.Errorf("failed to create seed %s: %v", name, err)
			}
		}
	}

	return nil
}
