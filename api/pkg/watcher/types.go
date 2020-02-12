package watcher

import (
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"code.cloudfoundry.org/go-pubsub"
)

type Providers struct {
	SettingsProvider provider.SettingsProvider
	SettingsWatcher  SettingsWatcher
}

type SettingsWatcher interface {
	Subscribe(subscription pubsub.Subscription)
}
