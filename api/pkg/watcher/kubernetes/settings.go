package kubernetes

import (
	"reflect"

	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"code.cloudfoundry.org/go-pubsub"
	"k8s.io/apimachinery/pkg/watch"
)

// SettingsWatcher watches settings and notifies its subscribers about any changes.
type SettingsWatcher struct {
	provider  provider.SettingsProvider
	watcher   watch.Interface
	publisher *pubsub.PubSub
}

// SettingsWatcher returns a new resource watcher.
func NewSettingsWatcher(provider provider.SettingsProvider) (*SettingsWatcher, error) {
	watcher, err := provider.WatchGlobalSettings()
	if err != nil {
		return nil, err
	}

	publisher := pubsub.New()
	go run(watcher, publisher)

	return &SettingsWatcher{
		provider:  provider,
		watcher:   watcher,
		publisher: publisher,
	}, nil
}

// run and publish information about settings updates.
func run(input watch.Interface, settingsPublisher *pubsub.PubSub) {
	defer input.Stop()

	for event := range input.ResultChan() {
		settings, ok := event.Object.(*v1.KubermaticSetting)
		if !ok {
			log.Logger.Debugf("expected settings got %s", reflect.TypeOf(event.Object))
		}

		if settings != nil && settings.Name == v1.GlobalSettingsName {
			if event.Type == watch.Added || event.Type == watch.Modified {
				settingsPublisher.Publish(settings, pubsub.LinearTreeTraverser([]uint64{}))
			} else if event.Type == watch.Deleted {
				settingsPublisher.Publish(nil, pubsub.LinearTreeTraverser([]uint64{}))
			}
		}
	}
}

// Subscribe allows to register subscription handler which will be invoked on each settings change.
func (watcher *SettingsWatcher) Subscribe(subscription pubsub.Subscription) {
	watcher.publisher.Subscribe(subscription)
}
