package util

import (
	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
)

func Hello(log *zap.SugaredLogger, app string, verbose bool) {
	log = log.With("version", resources.KUBERMATICGITTAG)
	if verbose {
		log = log.With("commit", resources.KUBERMATICCOMMIT)
	}

	log.Infof("Starting Kubermatic %s (%s)...", app, resources.KubermaticEdition)
}
