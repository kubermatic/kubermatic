package common

import (
	"code.cloudfoundry.org/go-pubsub"
)

type SettingsWatcher interface {
	Subscribe(subscription pubsub.Subscription)
}
