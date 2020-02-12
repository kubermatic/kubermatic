package common

import (
	"code.cloudfoundry.org/go-pubsub"
)

type ResourceWatcher interface {
	SubscribeSettings(subscription pubsub.Subscription)
}
