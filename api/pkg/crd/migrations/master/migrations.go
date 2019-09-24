package master

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type MigrationOptions struct {
	DatacentersFile    string
	DynamicDatacenters bool
}

func (o MigrationOptions) SeedMigrationEnabled() bool {
	return o.DatacentersFile != "" && o.DynamicDatacenters
}

type migrationContext struct {
	ctx                 context.Context
	log                 *zap.SugaredLogger
	workerName          string
	kubermaticNamespace string
	client              ctrlruntimeclient.Client
}

// RunAll runs all migrations that should be run inside a master cluster.
func RunAll(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, workerName string, kubermaticNamespace string, opt MigrationOptions) error {
	migrationCtx := &migrationContext{
		ctx:                 ctx,
		log:                 log,
		client:              client,
		workerName:          workerName,
		kubermaticNamespace: kubermaticNamespace,
	}

	if opt.SeedMigrationEnabled() {
		log.Info("datacenters given and dynamic datacenters enabled, attempting to migrate datacenters to Seeds")
		if err := migrateDatacenters(migrationCtx, opt.DatacentersFile); err != nil {
			return fmt.Errorf("failed to migrate datacenters.yaml: %v", err)
		}
		log.Info("migration completed successfully")
	}

	return nil
}

func migrateDatacenters(ctx *migrationContext, dcFile string) error {
	labelSelector, err := workerlabel.LabelSelector(ctx.workerName)
	if err != nil {
		return fmt.Errorf("failed to create workername label selector: %v", err)
	}

	// check if we performed/attempted a migration at an earlier point in time and if so, do not run it a second time;
	// in case of errors, the cluster operator has to remove broken Seeds to restart the migration
	existingSeeds := kubermaticv1.SeedList{}
	if err := ctx.client.List(ctx.ctx, &ctrlruntimeclient.ListOptions{
		Namespace:     ctx.kubermaticNamespace,
		LabelSelector: labelSelector,
	}, &existingSeeds); err != nil {
		return fmt.Errorf("failed to list existing Seeds in %s: %v", ctx.kubermaticNamespace, err)
	}

	if len(existingSeeds.Items) > 0 {
		ctx.log.Warn("migration enabled, but existing Seed CRs found; refusing to migrate again")
		return nil
	}

	seeds, err := provider.LoadSeeds(dcFile)
	if err != nil {
		return fmt.Errorf("failed to load %s: %v", dcFile, err)
	}

	for name, seed := range seeds {
		seed.Namespace = ctx.kubermaticNamespace
		if err := ctx.client.Create(ctx.ctx, seed); err != nil {
			return fmt.Errorf("failed to create seed %s: %v", name, seed)
		}
		ctx.log.Infow("Seed created", "name", name)
	}

	return nil
}
