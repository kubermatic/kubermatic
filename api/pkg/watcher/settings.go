package watcher

import (
	"fmt"
	"reflect"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"code.cloudfoundry.org/go-pubsub"
	"k8s.io/apimachinery/pkg/watch"
)

// ResourceWatcher watches resources and notifies its subscribers about any changes.
type ResourceWatcher struct {
	settingsProvider  provider.SettingsProvider
	settingsWatcher   watch.Interface
	settingsPublisher *pubsub.PubSub
}

// ResourceWatcher returns a new resource watcher.
func NewResourceWatcher(settingsProvider provider.SettingsProvider) (*ResourceWatcher, error) {
	settingsWatcher, err := settingsProvider.WatchGlobalSettings()
	if err != nil {
		return nil, err
	}

	settingsPublisher := pubsub.New()
	go watchSettings(settingsWatcher, settingsPublisher)

	return &ResourceWatcher{
		settingsProvider:  settingsProvider,
		settingsWatcher:   settingsWatcher,
		settingsPublisher: settingsPublisher,
	}, nil
}

// watchSettings runs in the background and publishes information about settings updates.
func watchSettings(input watch.Interface, settingsPublisher *pubsub.PubSub) {
	defer input.Stop()

	for {
		select {
		case event, ok := <-input.ResultChan():
			if !ok {
				fmt.Printf("settings watch ended with timeout")
				return
			}

			settings, ok := event.Object.(*kubermaticv1.KubermaticSetting)
			if !ok {
				fmt.Printf("expected settings got %s", reflect.TypeOf(event.Object))
			}

			if settings.Name == kubermaticv1.GlobalSettingsName {
				if event.Type == watch.Added || event.Type == watch.Modified {
					settingsPublisher.Publish(settings, pubsub.LinearTreeTraverser([]uint64{1}))
				} else if event.Type == watch.Deleted {
					settingsPublisher.Publish(nil, pubsub.LinearTreeTraverser([]uint64{1}))
				}
			}
		}
	}
}

// SubscribeSettings allows to register subscription handler which will be invoked on each settings change.
func (watcher *ResourceWatcher) SubscribeSettings(subscription pubsub.Subscription) {
	watcher.settingsPublisher.Subscribe(subscription)
}
