// +build ee

package master

import (
	"context"

	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/crd/migrations/master/options"
	eemastermigrations "github.com/kubermatic/kubermatic/api/pkg/ee/crd/migrations/master"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// RunAll runs all migrations that should be run inside a master cluster.
func RunAll(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, kubermaticNamespace string, opt options.MigrationOptions) error {
	return eemastermigrations.RunAll(ctx, log, client, kubermaticNamespace, opt)
}
